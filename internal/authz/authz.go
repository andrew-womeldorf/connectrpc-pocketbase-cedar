package authz

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
	"github.com/cedar-policy/cedar-go"
	"github.com/pocketbase/pocketbase/core"

	libraryv1 "gitlab.com/andrew.womeldorf/pbtest/gen/library/v1"
)

type contextKey string

const userIDKey contextKey = "userID"

var policySet *cedar.PolicySet

func LoadPolicies(policyFS fs.FS) {
	data, err := fs.ReadFile(policyFS, "policies/policy.cedar")
	if err != nil {
		log.Fatalf("Failed to read Cedar policies: %v", err)
	}

	policySet, err = cedar.NewPolicySetFromBytes("policies.cedar", data)
	if err != nil {
		log.Fatalf("Failed to parse Cedar policies: %v", err)
	}
}

func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

func Interceptor(app core.App) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
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

			procedure := req.Spec().Procedure
			actionName := procedure[strings.LastIndex(procedure, "/")+1:]

			var resourceID string
			switch msg := req.Any().(type) {
			case *libraryv1.GetBookRequest:
				resourceID = msg.GetId()
			case *libraryv1.UpdateBookRequest:
				resourceID = msg.GetId()
			case *libraryv1.DeleteBookRequest:
				resourceID = msg.GetId()
			}

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

			ctx = context.WithValue(ctx, userIDKey, userID)
			return next(ctx, req)
		}
	}
}
