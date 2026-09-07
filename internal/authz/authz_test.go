package authz

import (
	"context"
	"os"
	"testing"
	"testing/fstest"

	"connectrpc.com/connect"
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

func TestUserVerifiedFromContext(t *testing.T) {
	t.Run("empty context defaults false", func(t *testing.T) {
		if got := UserVerifiedFromContext(context.Background()); got {
			t.Error("expected false for empty context")
		}
	})

	t.Run("with verified true", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userVerifiedKey, true)
		if got := UserVerifiedFromContext(ctx); !got {
			t.Error("expected true")
		}
	})
}

func principalEntity(userID string, verified bool) cedar.Entity {
	uid := cedar.EntityUID{Type: "Library::User", ID: cedar.String(userID)}
	return cedar.Entity{
		UID: uid,
		Attributes: cedar.NewRecord(cedar.RecordMap{
			"id":       cedar.String(userID),
			"verified": cedar.Boolean(verified),
		}),
	}
}

func TestCedarAuthorization(t *testing.T) {
	loadTestPolicies(t)

	tests := []struct {
		name         string
		userID       string
		userVerified bool
		action       string
		bookID       string
		bookAuthor   string
		bookStatus   string
		want         cedar.Decision
	}{
		{
			name:         "verified user can create a book",
			userID:       "user1", userVerified: true, action: "CreateBook",
			bookID: "new-book", bookAuthor: "", bookStatus: "",
			want: cedar.Allow,
		},
		{
			name:         "unverified user cannot create a book",
			userID:       "user1", userVerified: false, action: "CreateBook",
			bookID: "new-book", bookAuthor: "", bookStatus: "",
			want: cedar.Deny,
		},
		{
			name:         "any user can get a published book",
			userID:       "user2", userVerified: false, action: "GetBook",
			bookID: "published-book", bookAuthor: "user1", bookStatus: "published",
			want: cedar.Allow,
		},
		{
			name:         "author can get own draft",
			userID:       "user1", userVerified: false, action: "GetBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Allow,
		},
		{
			name:         "non-author cannot get someone else's draft",
			userID:       "user2", userVerified: true, action: "GetBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Deny,
		},
		{
			name:         "any user can list books",
			userID:       "user1", userVerified: false, action: "ListBooks",
			bookID: "any-book", bookAuthor: "user1", bookStatus: "published",
			want: cedar.Allow,
		},
		{
			name:         "verified author can update own book",
			userID:       "user1", userVerified: true, action: "UpdateBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Allow,
		},
		{
			name:         "unverified author cannot update own book",
			userID:       "user1", userVerified: false, action: "UpdateBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Deny,
		},
		{
			name:         "non-author cannot update someone else's book",
			userID:       "user2", userVerified: true, action: "UpdateBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Deny,
		},
		{
			name:         "verified author can delete own book",
			userID:       "user1", userVerified: true, action: "DeleteBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Allow,
		},
		{
			name:         "unverified author cannot delete own book",
			userID:       "user1", userVerified: false, action: "DeleteBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Deny,
		},
		{
			name:         "non-author cannot delete someone else's book",
			userID:       "user2", userVerified: true, action: "DeleteBook",
			bookID: "draft-book", bookAuthor: "user1", bookStatus: "draft",
			want: cedar.Deny,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			principalUID := cedar.EntityUID{Type: "Library::User", ID: cedar.String(tt.userID)}
			actionUID := cedar.EntityUID{Type: "Library::Action", ID: cedar.String(tt.action)}
			resourceUID := cedar.EntityUID{Type: "Library::Book", ID: cedar.String(tt.bookID)}

			pe := principalEntity(tt.userID, tt.userVerified)
			entities := cedar.EntityMap{
				principalUID: pe,
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

			got, _ := cedar.Authorize(policySet, entities, req)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReviewAuthorization(t *testing.T) {
	loadTestPolicies(t)

	tests := []struct {
		name         string
		userID       string
		userVerified bool
		action       string
		reviewID     string
		reviewer     string
		bookAuthor   string
		want         cedar.Decision
	}{
		{
			name:         "verified user can create a review on someone else's book",
			userID:       "user2", userVerified: true, action: "CreateReview",
			reviewID: "new-review", reviewer: "user2", bookAuthor: "user1",
			want: cedar.Allow,
		},
		{
			name:         "unverified user cannot create a review",
			userID:       "user2", userVerified: false, action: "CreateReview",
			reviewID: "new-review", reviewer: "user2", bookAuthor: "user1",
			want: cedar.Deny,
		},
		{
			name:         "author cannot review own book",
			userID:       "user1", userVerified: true, action: "CreateReview",
			reviewID: "self-review", reviewer: "user1", bookAuthor: "user1",
			want: cedar.Deny,
		},
		{
			name:         "any user can get a review",
			userID:       "user2", userVerified: false, action: "GetReview",
			reviewID: "review1", reviewer: "user1", bookAuthor: "user1",
			want: cedar.Allow,
		},
		{
			name:         "any user can list reviews",
			userID:       "user1", userVerified: false, action: "ListReviews",
			reviewID: "any-review", reviewer: "user2", bookAuthor: "user1",
			want: cedar.Allow,
		},
		{
			name:         "verified reviewer can update own review",
			userID:       "user2", userVerified: true, action: "UpdateReview",
			reviewID: "review1", reviewer: "user2", bookAuthor: "user1",
			want: cedar.Allow,
		},
		{
			name:         "unverified reviewer cannot update own review",
			userID:       "user2", userVerified: false, action: "UpdateReview",
			reviewID: "review1", reviewer: "user2", bookAuthor: "user1",
			want: cedar.Deny,
		},
		{
			name:         "non-reviewer cannot update review",
			userID:       "user1", userVerified: true, action: "UpdateReview",
			reviewID: "review1", reviewer: "user2", bookAuthor: "user1",
			want: cedar.Deny,
		},
		{
			name:         "verified reviewer can delete own review",
			userID:       "user2", userVerified: true, action: "DeleteReview",
			reviewID: "review1", reviewer: "user2", bookAuthor: "user1",
			want: cedar.Allow,
		},
		{
			name:         "unverified reviewer cannot delete own review",
			userID:       "user2", userVerified: false, action: "DeleteReview",
			reviewID: "review1", reviewer: "user2", bookAuthor: "user1",
			want: cedar.Deny,
		},
		{
			name:         "verified book author can moderate (delete) review",
			userID:       "user1", userVerified: true, action: "DeleteReview",
			reviewID: "review1", reviewer: "user2", bookAuthor: "user1",
			want: cedar.Allow,
		},
		{
			name:         "unverified book author cannot moderate review",
			userID:       "user1", userVerified: false, action: "DeleteReview",
			reviewID: "review1", reviewer: "user2", bookAuthor: "user1",
			want: cedar.Deny,
		},
		{
			name:         "unrelated user cannot delete review",
			userID:       "user3", userVerified: true, action: "DeleteReview",
			reviewID: "review1", reviewer: "user2", bookAuthor: "user1",
			want: cedar.Deny,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			principalUID := cedar.EntityUID{Type: "Library::User", ID: cedar.String(tt.userID)}
			actionUID := cedar.EntityUID{Type: "Library::Action", ID: cedar.String(tt.action)}
			resourceUID := cedar.EntityUID{Type: "Library::Review", ID: cedar.String(tt.reviewID)}

			pe := principalEntity(tt.userID, tt.userVerified)
			entities := cedar.EntityMap{
				principalUID: pe,
				resourceUID: cedar.Entity{
					UID: resourceUID,
					Attributes: cedar.NewRecord(cedar.RecordMap{
						"reviewer":    cedar.String(tt.reviewer),
						"book_author": cedar.String(tt.bookAuthor),
					}),
				},
			}

			req := cedar.Request{
				Principal: principalUID,
				Action:    actionUID,
				Resource:  resourceUID,
			}

			got, _ := cedar.Authorize(policySet, entities, req)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserSelfAuthorization(t *testing.T) {
	loadTestPolicies(t)

	tests := []struct {
		name     string
		userID   string
		action   string
		resource string // entity ID of the resource user
		want     cedar.Decision
	}{
		{
			name: "user can get themselves",
			userID: "user1", action: "GetMe", resource: "user1",
			want: cedar.Allow,
		},
		{
			name: "user cannot get another user",
			userID: "user1", action: "GetMe", resource: "user2",
			want: cedar.Deny,
		},
		{
			name: "user can update themselves",
			userID: "user1", action: "UpdateMe", resource: "user1",
			want: cedar.Allow,
		},
		{
			name: "user cannot update another user",
			userID: "user1", action: "UpdateMe", resource: "user2",
			want: cedar.Deny,
		},
		{
			name: "unverified user can get themselves",
			userID: "user1", action: "GetMe", resource: "user1",
			want: cedar.Allow,
		},
		{
			name: "unverified user can update themselves",
			userID: "user1", action: "UpdateMe", resource: "user1",
			want: cedar.Allow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			principalUID := cedar.EntityUID{Type: "Library::User", ID: cedar.String(tt.userID)}
			actionUID := cedar.EntityUID{Type: "Library::Action", ID: cedar.String(tt.action)}
			resourceUID := cedar.EntityUID{Type: "Library::User", ID: cedar.String(tt.resource)}

			// verified=false for unverified cases; the policy shouldn't require it
			pe := principalEntity(tt.userID, false)
			entities := cedar.EntityMap{
				principalUID: pe,
				resourceUID: cedar.Entity{UID: resourceUID},
			}

			req := cedar.Request{
				Principal: principalUID,
				Action:    actionUID,
				Resource:  resourceUID,
			}

			got, _ := cedar.Authorize(policySet, entities, req)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthorize(t *testing.T) {
	loadTestPolicies(t)

	t.Run("verified user can create book", func(t *testing.T) {
		resourceUID := cedar.EntityUID{Type: "Library::Book", ID: cedar.String("")}
		entities := cedar.EntityMap{
			resourceUID: cedar.Entity{
				UID: resourceUID,
				Attributes: cedar.NewRecord(cedar.RecordMap{
					"author": cedar.String(""),
					"status": cedar.String(""),
				}),
			},
		}
		if err := Authorize("user1", true, "CreateBook", resourceUID, entities); err != nil {
			t.Errorf("expected allow, got error: %v", err)
		}
	})

	t.Run("unverified user cannot create book", func(t *testing.T) {
		resourceUID := cedar.EntityUID{Type: "Library::Book", ID: cedar.String("")}
		entities := cedar.EntityMap{
			resourceUID: cedar.Entity{
				UID: resourceUID,
				Attributes: cedar.NewRecord(cedar.RecordMap{
					"author": cedar.String(""),
					"status": cedar.String(""),
				}),
			},
		}
		if err := Authorize("user1", false, "CreateBook", resourceUID, entities); err == nil {
			t.Error("expected deny for unverified user, got nil")
		}
	})

	t.Run("denies unauthorized book delete", func(t *testing.T) {
		resourceUID := cedar.EntityUID{Type: "Library::Book", ID: cedar.String("book1")}
		entities := cedar.EntityMap{
			resourceUID: cedar.Entity{
				UID: resourceUID,
				Attributes: cedar.NewRecord(cedar.RecordMap{
					"author": cedar.String("user1"),
					"status": cedar.String("draft"),
				}),
			},
		}
		err := Authorize("user2", true, "DeleteBook", resourceUID, entities)
		if err == nil {
			t.Error("expected deny, got nil")
		}
		if connectErr, ok := err.(*connect.Error); ok {
			if connectErr.Code() != connect.CodePermissionDenied {
				t.Errorf("expected CodePermissionDenied, got %v", connectErr.Code())
			}
		}
	})

	t.Run("verified book author can moderate review", func(t *testing.T) {
		resourceUID := cedar.EntityUID{Type: "Library::Review", ID: cedar.String("review1")}
		entities := cedar.EntityMap{
			resourceUID: cedar.Entity{
				UID: resourceUID,
				Attributes: cedar.NewRecord(cedar.RecordMap{
					"reviewer":    cedar.String("user2"),
					"book_author": cedar.String("user1"),
				}),
			},
		}
		if err := Authorize("user1", true, "DeleteReview", resourceUID, entities); err != nil {
			t.Errorf("expected allow, got error: %v", err)
		}
	})

	t.Run("denies self-review", func(t *testing.T) {
		resourceUID := cedar.EntityUID{Type: "Library::Review", ID: cedar.String("")}
		entities := cedar.EntityMap{
			resourceUID: cedar.Entity{
				UID: resourceUID,
				Attributes: cedar.NewRecord(cedar.RecordMap{
					"reviewer":    cedar.String("user1"),
					"book_author": cedar.String("user1"),
				}),
			},
		}
		if err := Authorize("user1", true, "CreateReview", resourceUID, entities); err == nil {
			t.Error("expected deny for self-review, got nil")
		}
	})
}
