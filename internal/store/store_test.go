package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/andrew-womeldorf/pbtest/migrations"

	"github.com/andrew-womeldorf/pbtest/internal/store"
	"github.com/andrew-womeldorf/pbtest/internal/store/memory"
	"github.com/andrew-womeldorf/pbtest/internal/store/pbstore"
)

type testEnv struct {
	store   store.Store
	userID  string
	user2ID string
}

type storeTestSuite struct {
	name  string
	setup func(t *testing.T) (testEnv, func())
}

func TestStore(t *testing.T) {
	suites := []storeTestSuite{
		{
			name: "memory",
			setup: func(t *testing.T) (testEnv, func()) {
				return testEnv{
					store:   memory.New(),
					userID:  "user1",
					user2ID: "user2",
				}, func() {}
			},
		},
		{
			name: "pocketbase",
			setup: func(t *testing.T) (testEnv, func()) {
				app, err := tests.NewTestApp()
				if err != nil {
					t.Fatal(err)
				}

				users, err := app.FindCollectionByNameOrId("users")
				if err != nil {
					t.Fatal(err)
				}

				u1 := core.NewRecord(users)
				u1.SetEmail("user1@test.com")
				u1.SetPassword("password123456")
				if err := app.Save(u1); err != nil {
					t.Fatal(err)
				}

				u2 := core.NewRecord(users)
				u2.SetEmail("user2@test.com")
				u2.SetPassword("password123456")
				if err := app.Save(u2); err != nil {
					t.Fatal(err)
				}

				return testEnv{
					store:   pbstore.New(app),
					userID:  u1.Id,
					user2ID: u2.Id,
				}, func() { app.Cleanup() }
			},
		},
	}

	for _, suite := range suites {
		t.Run(suite.name, func(t *testing.T) {
			runStoreTests(t, suite.setup)
		})
	}
}

func runStoreTests(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	t.Run("CreateBook", func(t *testing.T) { testCreateBook(t, setup) })
	t.Run("GetBook", func(t *testing.T) { testGetBook(t, setup) })
	t.Run("GetBookNotFound", func(t *testing.T) { testGetBookNotFound(t, setup) })
	t.Run("ListBooks", func(t *testing.T) { testListBooks(t, setup) })
	t.Run("UpdateBook", func(t *testing.T) { testUpdateBook(t, setup) })
	t.Run("UpdateBookNotFound", func(t *testing.T) { testUpdateBookNotFound(t, setup) })
	t.Run("DeleteBook", func(t *testing.T) { testDeleteBook(t, setup) })
	t.Run("DeleteBookNotFound", func(t *testing.T) { testDeleteBookNotFound(t, setup) })

	t.Run("CreateReview", func(t *testing.T) { testCreateReview(t, setup) })
	t.Run("GetReview", func(t *testing.T) { testGetReview(t, setup) })
	t.Run("GetReviewNotFound", func(t *testing.T) { testGetReviewNotFound(t, setup) })
	t.Run("ListReviews", func(t *testing.T) { testListReviews(t, setup) })
	t.Run("UpdateReview", func(t *testing.T) { testUpdateReview(t, setup) })
	t.Run("UpdateReviewNotFound", func(t *testing.T) { testUpdateReviewNotFound(t, setup) })
	t.Run("DeleteReview", func(t *testing.T) { testDeleteReview(t, setup) })
	t.Run("DeleteReviewNotFound", func(t *testing.T) { testDeleteReviewNotFound(t, setup) })
}

func ptr[T any](v T) *T { return &v }

// Book tests

func testCreateBook(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	ctx := context.Background()
	book, err := env.store.CreateBook(ctx, &store.Book{
		Title: "Test Book", Author: env.userID, Status: "draft",
	})
	if err != nil {
		t.Fatalf("CreateBook: %v", err)
	}
	if book.ID == "" {
		t.Error("expected non-empty ID")
	}
	if book.Title != "Test Book" {
		t.Errorf("Title = %q, want %q", book.Title, "Test Book")
	}
	if book.Author != env.userID {
		t.Errorf("Author = %q, want %q", book.Author, env.userID)
	}
	if book.Status != "draft" {
		t.Errorf("Status = %q, want %q", book.Status, "draft")
	}
}

func testGetBook(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	ctx := context.Background()
	created, err := env.store.CreateBook(ctx, &store.Book{
		Title: "Test Book", Author: env.userID, Status: "published",
	})
	if err != nil {
		t.Fatalf("CreateBook: %v", err)
	}

	got, err := env.store.GetBook(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetBook: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Title != "Test Book" {
		t.Errorf("Title = %q, want %q", got.Title, "Test Book")
	}
	if got.Author != env.userID {
		t.Errorf("Author = %q, want %q", got.Author, env.userID)
	}
}

func testGetBookNotFound(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	_, err := env.store.GetBook(context.Background(), "nonexistent")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func testListBooks(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	ctx := context.Background()
	env.store.CreateBook(ctx, &store.Book{Title: "Book A", Author: env.userID, Status: "draft"})
	env.store.CreateBook(ctx, &store.Book{Title: "Book B", Author: env.user2ID, Status: "published"})

	books, err := env.store.ListBooks(ctx)
	if err != nil {
		t.Fatalf("ListBooks: %v", err)
	}
	if len(books) != 2 {
		t.Errorf("len = %d, want 2", len(books))
	}
}

func testUpdateBook(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	ctx := context.Background()
	created, _ := env.store.CreateBook(ctx, &store.Book{
		Title: "Original", Author: env.userID, Status: "draft",
	})

	updated, err := env.store.UpdateBook(ctx, created.ID, store.BookUpdate{
		Title: ptr("Updated Title"),
	})
	if err != nil {
		t.Fatalf("UpdateBook: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", updated.Title, "Updated Title")
	}
	if updated.Status != "draft" {
		t.Errorf("Status = %q, want %q (unchanged)", updated.Status, "draft")
	}
}

func testUpdateBookNotFound(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	_, err := env.store.UpdateBook(context.Background(), "nonexistent", store.BookUpdate{
		Title: ptr("x"),
	})
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func testDeleteBook(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	ctx := context.Background()
	created, _ := env.store.CreateBook(ctx, &store.Book{
		Title: "To Delete", Author: env.userID, Status: "draft",
	})

	if err := env.store.DeleteBook(ctx, created.ID); err != nil {
		t.Fatalf("DeleteBook: %v", err)
	}

	_, err := env.store.GetBook(ctx, created.ID)
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("GetBook after delete: err = %v, want ErrNotFound", err)
	}
}

func testDeleteBookNotFound(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	err := env.store.DeleteBook(context.Background(), "nonexistent")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// Review tests

func createTestBook(t *testing.T, env testEnv) *store.Book {
	t.Helper()
	book, err := env.store.CreateBook(context.Background(), &store.Book{
		Title: "Review Target", Author: env.userID, Status: "published",
	})
	if err != nil {
		t.Fatalf("setup CreateBook: %v", err)
	}
	return book
}

func testCreateReview(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	ctx := context.Background()
	book := createTestBook(t, env)

	review, err := env.store.CreateReview(ctx, &store.Review{
		BookID: book.ID, Reviewer: env.user2ID, Content: "Great book", Rating: 5,
	})
	if err != nil {
		t.Fatalf("CreateReview: %v", err)
	}
	if review.ID == "" {
		t.Error("expected non-empty ID")
	}
	if review.BookID != book.ID {
		t.Errorf("BookID = %q, want %q", review.BookID, book.ID)
	}
	if review.Rating != 5 {
		t.Errorf("Rating = %d, want 5", review.Rating)
	}
}

func testGetReview(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	ctx := context.Background()
	book := createTestBook(t, env)
	created, _ := env.store.CreateReview(ctx, &store.Review{
		BookID: book.ID, Reviewer: env.user2ID, Content: "Nice", Rating: 4,
	})

	got, err := env.store.GetReview(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetReview: %v", err)
	}
	if got.Content != "Nice" {
		t.Errorf("Content = %q, want %q", got.Content, "Nice")
	}
}

func testGetReviewNotFound(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	_, err := env.store.GetReview(context.Background(), "nonexistent")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func testListReviews(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	ctx := context.Background()
	book1 := createTestBook(t, env)
	book2, _ := env.store.CreateBook(ctx, &store.Book{
		Title: "Other Book", Author: env.user2ID, Status: "published",
	})

	env.store.CreateReview(ctx, &store.Review{BookID: book1.ID, Reviewer: env.user2ID, Content: "a", Rating: 3})
	env.store.CreateReview(ctx, &store.Review{BookID: book1.ID, Reviewer: env.user2ID, Content: "b", Rating: 4})
	env.store.CreateReview(ctx, &store.Review{BookID: book2.ID, Reviewer: env.userID, Content: "c", Rating: 5})

	reviews, err := env.store.ListReviews(ctx, book1.ID)
	if err != nil {
		t.Fatalf("ListReviews: %v", err)
	}
	if len(reviews) != 2 {
		t.Errorf("len = %d, want 2", len(reviews))
	}
}

func testUpdateReview(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	ctx := context.Background()
	book := createTestBook(t, env)
	created, _ := env.store.CreateReview(ctx, &store.Review{
		BookID: book.ID, Reviewer: env.user2ID, Content: "Original", Rating: 3,
	})

	updated, err := env.store.UpdateReview(ctx, created.ID, store.ReviewUpdate{
		Content: ptr("Revised"),
	})
	if err != nil {
		t.Fatalf("UpdateReview: %v", err)
	}
	if updated.Content != "Revised" {
		t.Errorf("Content = %q, want %q", updated.Content, "Revised")
	}
	if updated.Rating != 3 {
		t.Errorf("Rating = %d, want 3 (unchanged)", updated.Rating)
	}
}

func testUpdateReviewNotFound(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	_, err := env.store.UpdateReview(context.Background(), "nonexistent", store.ReviewUpdate{
		Content: ptr("x"),
	})
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func testDeleteReview(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	ctx := context.Background()
	book := createTestBook(t, env)
	created, _ := env.store.CreateReview(ctx, &store.Review{
		BookID: book.ID, Reviewer: env.user2ID, Content: "Delete me", Rating: 1,
	})

	if err := env.store.DeleteReview(ctx, created.ID); err != nil {
		t.Fatalf("DeleteReview: %v", err)
	}

	_, err := env.store.GetReview(ctx, created.ID)
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("GetReview after delete: err = %v, want ErrNotFound", err)
	}
}

func testDeleteReviewNotFound(t *testing.T, setup func(t *testing.T) (testEnv, func())) {
	env, cleanup := setup(t)
	defer cleanup()

	err := env.store.DeleteReview(context.Background(), "nonexistent")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
