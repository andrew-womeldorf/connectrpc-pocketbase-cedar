package pbstore

import (
	"context"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"gitlab.com/andrew.womeldorf/pbtest/internal/store"
)

type Store struct {
	app core.App
}

func New(app core.App) *Store {
	return &Store{app: app}
}

func bookFromRecord(r *core.Record) *store.Book {
	return &store.Book{
		ID:     r.Id,
		Title:  r.GetString("title"),
		Author: r.GetString("author"),
		Status: r.GetString("status"),
	}
}

func reviewFromRecord(r *core.Record) *store.Review {
	return &store.Review{
		ID:       r.Id,
		BookID:   r.GetString("book"),
		Reviewer: r.GetString("reviewer"),
		Content:  r.GetString("content"),
		Rating:   r.GetInt("rating"),
	}
}

func (s *Store) CreateBook(_ context.Context, book *store.Book) (*store.Book, error) {
	collection, err := s.app.FindCollectionByNameOrId("books")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	record.Set("title", book.Title)
	record.Set("author", book.Author)
	record.Set("status", book.Status)

	if err := s.app.Save(record); err != nil {
		return nil, err
	}

	return bookFromRecord(record), nil
}

func (s *Store) GetBook(_ context.Context, id string) (*store.Book, error) {
	record, err := s.app.FindRecordById("books", id)
	if err != nil {
		return nil, store.ErrNotFound
	}
	return bookFromRecord(record), nil
}

func (s *Store) ListBooks(_ context.Context) ([]*store.Book, error) {
	records, err := s.app.FindAllRecords("books")
	if err != nil {
		return nil, err
	}

	books := make([]*store.Book, len(records))
	for i, r := range records {
		books[i] = bookFromRecord(r)
	}
	return books, nil
}

func (s *Store) UpdateBook(_ context.Context, id string, update store.BookUpdate) (*store.Book, error) {
	record, err := s.app.FindRecordById("books", id)
	if err != nil {
		return nil, store.ErrNotFound
	}

	if update.Title != nil {
		record.Set("title", *update.Title)
	}
	if update.Status != nil {
		record.Set("status", *update.Status)
	}

	if err := s.app.Save(record); err != nil {
		return nil, err
	}

	return bookFromRecord(record), nil
}

func (s *Store) DeleteBook(_ context.Context, id string) error {
	record, err := s.app.FindRecordById("books", id)
	if err != nil {
		return store.ErrNotFound
	}
	return s.app.Delete(record)
}

func (s *Store) CreateReview(_ context.Context, review *store.Review) (*store.Review, error) {
	collection, err := s.app.FindCollectionByNameOrId("reviews")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	record.Set("book", review.BookID)
	record.Set("reviewer", review.Reviewer)
	record.Set("content", review.Content)
	record.Set("rating", review.Rating)

	if err := s.app.Save(record); err != nil {
		return nil, err
	}

	return reviewFromRecord(record), nil
}

func (s *Store) GetReview(_ context.Context, id string) (*store.Review, error) {
	record, err := s.app.FindRecordById("reviews", id)
	if err != nil {
		return nil, store.ErrNotFound
	}
	return reviewFromRecord(record), nil
}

func (s *Store) ListReviews(_ context.Context, bookID string) ([]*store.Review, error) {
	records, err := s.app.FindAllRecords("reviews",
		dbx.NewExp("book = {:bookId}", dbx.Params{"bookId": bookID}),
	)
	if err != nil {
		return nil, err
	}

	reviews := make([]*store.Review, len(records))
	for i, r := range records {
		reviews[i] = reviewFromRecord(r)
	}
	return reviews, nil
}

func (s *Store) UpdateReview(_ context.Context, id string, update store.ReviewUpdate) (*store.Review, error) {
	record, err := s.app.FindRecordById("reviews", id)
	if err != nil {
		return nil, store.ErrNotFound
	}

	if update.Content != nil {
		record.Set("content", *update.Content)
	}
	if update.Rating != nil {
		record.Set("rating", *update.Rating)
	}

	if err := s.app.Save(record); err != nil {
		return nil, err
	}

	return reviewFromRecord(record), nil
}

func (s *Store) DeleteReview(_ context.Context, id string) error {
	record, err := s.app.FindRecordById("reviews", id)
	if err != nil {
		return store.ErrNotFound
	}
	return s.app.Delete(record)
}
