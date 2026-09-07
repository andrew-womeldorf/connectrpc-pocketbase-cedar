package store

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("not found")

type Book struct {
	ID     string
	Title  string
	Author string
	Status string
}

type BookUpdate struct {
	Title  *string
	Status *string
}

type Review struct {
	ID       string
	BookID   string
	Reviewer string
	Content  string
	Rating   int
}

type ReviewUpdate struct {
	Content *string
	Rating  *int
}

type Store interface {
	CreateBook(ctx context.Context, book *Book) (*Book, error)
	GetBook(ctx context.Context, id string) (*Book, error)
	ListBooks(ctx context.Context) ([]*Book, error)
	UpdateBook(ctx context.Context, id string, update BookUpdate) (*Book, error)
	DeleteBook(ctx context.Context, id string) error

	CreateReview(ctx context.Context, review *Review) (*Review, error)
	GetReview(ctx context.Context, id string) (*Review, error)
	ListReviews(ctx context.Context, bookID string) ([]*Review, error)
	UpdateReview(ctx context.Context, id string, update ReviewUpdate) (*Review, error)
	DeleteReview(ctx context.Context, id string) error
}
