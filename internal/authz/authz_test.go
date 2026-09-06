package authz

import (
	"context"
	"os"
	"testing"
	"testing/fstest"

	"github.com/cedar-policy/cedar-go"
)

func loadTestPolicies(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile("../../policies/policy.cedar")
	if err != nil {
		t.Fatal(err)
	}
	LoadPolicies(fstest.MapFS{
		"policies/policy.cedar": &fstest.MapFile{Data: data},
	})
}

func TestUserIDFromContext(t *testing.T) {
	t.Run("empty context", func(t *testing.T) {
		if got := UserIDFromContext(context.Background()); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("with user ID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey, "user123")
		if got := UserIDFromContext(ctx); got != "user123" {
			t.Errorf("got %q, want %q", got, "user123")
		}
	})
}

func TestCedarAuthorization(t *testing.T) {
	loadTestPolicies(t)

	tests := []struct {
		name       string
		userID     string
		action     string
		bookID     string
		bookAuthor string
		bookStatus string
		want       cedar.Decision
	}{
		{
			name:   "any user can create a book",
			userID: "user1", action: "CreateBook",
			bookID: "new-book", bookAuthor: "", bookStatus: "",
			want: cedar.Allow,
		},
		{
			name:   "any user can get a published book",
			userID: "user2", action: "GetBook",
			bookID: "published-book", bookAuthor: "user1", bookStatus: "published",
			want: cedar.Allow,
		},
		{
			name:   "author can get own draft",
			userID: "user1", action: "GetBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Allow,
		},
		{
			name:   "non-author cannot get someone else's draft",
			userID: "user2", action: "GetBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Deny,
		},
		{
			name:   "any user can list books",
			userID: "user1", action: "ListBooks",
			bookID: "any-book", bookAuthor: "user1", bookStatus: "published",
			want: cedar.Allow,
		},
		{
			name:   "author can update own book",
			userID: "user1", action: "UpdateBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Allow,
		},
		{
			name:   "non-author cannot update someone else's book",
			userID: "user2", action: "UpdateBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Deny,
		},
		{
			name:   "author can delete own book",
			userID: "user1", action: "DeleteBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Allow,
		},
		{
			name:   "non-author cannot delete someone else's book",
			userID: "user2", action: "DeleteBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Deny,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			principalUID := cedar.EntityUID{Type: "User", ID: cedar.String(tt.userID)}
			actionUID := cedar.EntityUID{Type: "Action", ID: cedar.String(tt.action)}
			resourceUID := cedar.EntityUID{Type: "Book", ID: cedar.String(tt.bookID)}

			entities := cedar.EntityMap{
				principalUID: cedar.Entity{
					UID: principalUID,
					Attributes: cedar.NewRecord(cedar.RecordMap{
						"id": cedar.String(tt.userID),
					}),
				},
				resourceUID: cedar.Entity{
					UID: resourceUID,
					Attributes: cedar.NewRecord(cedar.RecordMap{
						"author": cedar.String(tt.bookAuthor),
						"status": cedar.String(tt.bookStatus),
					}),
				},
			}

			req := cedar.Request{
				Principal: principalUID,
				Action:    actionUID,
				Resource:  resourceUID,
			}

			got, _ := policySet.IsAuthorized(entities, req)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
