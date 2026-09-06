package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/cedar-policy/cedar-go"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"

	libraryv1 "gitlab.com/andrew.womeldorf/pbtest/gen/library/v1"
	"gitlab.com/andrew.womeldorf/pbtest/gen/library/v1/libraryv1connect"

	_ "gitlab.com/andrew.womeldorf/pbtest/migrations"
)

// ---------------------------------------------------------
// 1. CEDAR POLICIES
// ---------------------------------------------------------
const cedarPolicies = `
@id("allow-create-book")
permit (
    principal,
    action == Action::"CreateBook",
    resource
);

@id("allow-get-published-book")
permit (
    principal,
    action == Action::"GetBook",
    resource
)
when {
    resource.status == "published"
};

@id("allow-get-own-draft")
permit (
    principal,
    action == Action::"GetBook",
    resource
)
when {
    resource.status == "draft" && resource.author == principal.id
};

@id("allow-list-books")
permit (
    principal,
    action == Action::"ListBooks",
    resource
);

@id("allow-update-own-book")
permit (
    principal,
    action == Action::"UpdateBook",
    resource
)
when {
    resource.author == principal.id
};

@id("allow-delete-own-book")
permit (
    principal,
    action == Action::"DeleteBook",
    resource
)
when {
    resource.author == principal.id
};
`

var policySet *cedar.PolicySet

func init() {
	var err error
	policySet, err = cedar.NewPolicySetFromBytes("policies.cedar", []byte(cedarPolicies))
	if err != nil {
		log.Fatalf("Failed to parse Cedar policies: %v", err)
	}
}

// ---------------------------------------------------------
// 2. CONNECT-RPC SERVER IMPLEMENTATION
// ---------------------------------------------------------
type libraryServer struct {
	app core.App
}

type contextKey string

const userIDKey contextKey = "userID"

func bookFromRecord(r *core.Record) *libraryv1.Book {
	return &libraryv1.Book{
		Id:     r.Id,
		Title:  r.GetString("title"),
		Author: r.GetString("author"),
		Status: r.GetString("status"),
	}
}

func (s *libraryServer) CreateBook(ctx context.Context, req *connect.Request[libraryv1.CreateBookRequest]) (*connect.Response[libraryv1.CreateBookResponse], error) {
	collection, err := s.app.FindCollectionByNameOrId("books")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	record := core.NewRecord(collection)
	record.Set("title", req.Msg.GetTitle())
	record.Set("author", ctx.Value(userIDKey))
	record.Set("status", "draft")

	if err := s.app.Save(record); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&libraryv1.CreateBookResponse{
		Book: bookFromRecord(record),
	}), nil
}

func (s *libraryServer) GetBook(ctx context.Context, req *connect.Request[libraryv1.GetBookRequest]) (*connect.Response[libraryv1.GetBookResponse], error) {
	record, err := s.app.FindRecordById("books", req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&libraryv1.GetBookResponse{
		Book: bookFromRecord(record),
	}), nil
}

func (s *libraryServer) ListBooks(ctx context.Context, req *connect.Request[libraryv1.ListBooksRequest]) (*connect.Response[libraryv1.ListBooksResponse], error) {
	userID, _ := ctx.Value(userIDKey).(string)

	records, err := s.app.FindAllRecords("books")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var books []*libraryv1.Book
	for _, r := range records {
		if r.GetString("status") == "draft" && r.GetString("author") != userID {
			continue
		}
		books = append(books, bookFromRecord(r))
	}

	return connect.NewResponse(&libraryv1.ListBooksResponse{
		Books: books,
	}), nil
}

func (s *libraryServer) UpdateBook(ctx context.Context, req *connect.Request[libraryv1.UpdateBookRequest]) (*connect.Response[libraryv1.UpdateBookResponse], error) {
	record, err := s.app.FindRecordById("books", req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if req.Msg.GetTitle() != "" {
		record.Set("title", req.Msg.GetTitle())
	}
	if req.Msg.GetStatus() != "" {
		record.Set("status", req.Msg.GetStatus())
	}

	if err := s.app.Save(record); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&libraryv1.UpdateBookResponse{
		Book: bookFromRecord(record),
	}), nil
}

func (s *libraryServer) DeleteBook(ctx context.Context, req *connect.Request[libraryv1.DeleteBookRequest]) (*connect.Response[libraryv1.DeleteBookResponse], error) {
	record, err := s.app.FindRecordById("books", req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if err := s.app.Delete(record); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&libraryv1.DeleteBookResponse{}), nil
}

// ---------------------------------------------------------
// 3. AUTH & CEDAR INTERCEPTOR (Middleware for ConnectRPC)
// ---------------------------------------------------------
func authZInterceptor(app core.App) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {

			// A. AUTHENTICATION (PocketBase)
			authHeader := req.Header().Get("Authorization")
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing token"))
			}

			userRecord, err := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid token"))
			}
			userID := userRecord.Id

			// B. AUTHORIZATION (Cedar)
			procedure := req.Spec().Procedure
			actionName := procedure[strings.LastIndex(procedure, "/")+1:]

			// Extract resource ID from any request type that has one
			var resourceID string
			switch msg := req.Any().(type) {
			case *libraryv1.GetBookRequest:
				resourceID = msg.GetId()
			case *libraryv1.UpdateBookRequest:
				resourceID = msg.GetId()
			case *libraryv1.DeleteBookRequest:
				resourceID = msg.GetId()
			}

			// Look up book attributes for resource-level policies
			bookAuthor := ""
			bookStatus := ""
			if resourceID != "" {
				if book, err := app.FindRecordById("books", resourceID); err == nil {
					bookAuthor = book.GetString("author")
					bookStatus = book.GetString("status")
				}
			}

			principalUID := cedar.EntityUID{Type: "User", ID: cedar.String(userID)}
			actionUID := cedar.EntityUID{Type: "Action", ID: cedar.String(actionName)}
			resourceUID := cedar.EntityUID{Type: "Book", ID: cedar.String(resourceID)}

			entities := cedar.EntityMap{
				principalUID: cedar.Entity{
					UID: principalUID,
					Attributes: cedar.NewRecord(cedar.RecordMap{
						"id": cedar.String(userID),
					}),
				},
				resourceUID: cedar.Entity{
					UID: resourceUID,
					Attributes: cedar.NewRecord(cedar.RecordMap{
						"author": cedar.String(bookAuthor),
						"status": cedar.String(bookStatus),
					}),
				},
			}

			cedarReq := cedar.Request{
				Principal: principalUID,
				Action:    actionUID,
				Resource:  resourceUID,
			}

			slog.Info("cedar authz", slog.Any("request", cedarReq), slog.Any("entities", entities))
			decision, diagnostic := policySet.IsAuthorized(entities, cedarReq)
			slog.Info("cedar decision", slog.Any("decision", decision), slog.Any("diagnostic", diagnostic))
			if decision != cedar.Allow {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("cedar policy denied access"))
			}

			// Pass user ID to handlers via context
			ctx = context.WithValue(ctx, userIDKey, userID)
			return next(ctx, req)
		}
	}
}

// ---------------------------------------------------------
// 4. MAIN ENTRY POINT
// ---------------------------------------------------------
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: osutils.IsProbablyGoRun(),
	})

	// Intercept the setup phase to register ConnectRPC routes onto PocketBase's Mux
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {

		// 1. Set up the ConnectRPC interceptor pipeline
		interceptors := connect.WithInterceptors(authZInterceptor(app))

		// 2. Instantiate your ConnectRPC server, passing the PB app for DB access
		libServer := &libraryServer{app: app}

		// 3. Generate the Connect path and handler
		path, handler := libraryv1connect.NewLibraryServiceHandler(libServer, interceptors)

		e.Router.Any(path+"{path...}", func(e *core.RequestEvent) error {
			handler.ServeHTTP(e.Response, e.Request)
			return nil
		})

		return e.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
