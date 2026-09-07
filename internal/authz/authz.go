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

func Authorize(userID, action string, resourceUID cedar.EntityUID, entities cedar.EntityMap) error {
	principalUID := cedar.EntityUID{Type: "Library::User", ID: cedar.String(userID)}

	entities[principalUID] = cedar.Entity{
		UID: principalUID,
		Attributes: cedar.NewRecord(cedar.RecordMap{
			"id": cedar.String(userID),
		}),
	}

	req := cedar.Request{
		Principal: principalUID,
		Action:    cedar.EntityUID{Type: "Library::Action", ID: cedar.String(action)},
		Resource:  resourceUID,
	}

	slog.Info("cedar authz", slog.Any("request", req), slog.Any("entities", entities))
	decision, diagnostic := cedar.Authorize(policySet, entities, req)
	slog.Info("cedar decision", slog.Any("decision", decision), slog.Any("diagnostic", diagnostic))

	if decision != cedar.Allow {
		return connect.NewError(connect.CodePermissionDenied, errors.New("cedar policy denied access"))
	}
	return nil
}

// Interceptor returns a ConnectRPC interceptor that handles authentication.
// It validates the Bearer token, extracts the user ID, and stores it in context.
// Authorization is handled per-handler via the Authorize function.
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

			ctx = context.WithValue(ctx, userIDKey, userRecord.Id)
			return next(ctx, req)
		}
	}
}
