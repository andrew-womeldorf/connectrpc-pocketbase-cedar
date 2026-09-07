package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/andrew-womeldorf/pbtest/internal/store"
)

type Store struct {
	mu      sync.Mutex
	books   map[string]*store.Book
	reviews map[string]*store.Review
	nextID  int
}

func New() *Store {
	return &Store{
		books:   make(map[string]*store.Book),
		reviews: make(map[string]*store.Review),
	}
}

func (s *Store) genID() string {
	s.nextID++
	return fmt.Sprintf("mem_%d", s.nextID)
}

func (s *Store) CreateBook(_ context.Context, book *store.Book) (*store.Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b := *book
	b.ID = s.genID()
	s.books[b.ID] = &b

	out := b
	return &out, nil
}

func (s *Store) GetBook(_ context.Context, id string) (*store.Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.books[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	out := *b
	return &out, nil
}

func (s *Store) ListBooks(_ context.Context) ([]*store.Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	books := make([]*store.Book, 0, len(s.books))
	for _, b := range s.books {
		out := *b
		books = append(books, &out)
	}
	return books, nil
}

func (s *Store) UpdateBook(_ context.Context, id string, update store.BookUpdate) (*store.Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.books[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	if update.Title != nil {
		b.Title = *update.Title
	}
	if update.Status != nil {
		b.Status = *update.Status
	}

	out := *b
	return &out, nil
}

func (s *Store) DeleteBook(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.books[id]; !ok {
		return store.ErrNotFound
	}
	delete(s.books, id)
	return nil
}

func (s *Store) CreateReview(_ context.Context, review *store.Review) (*store.Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := *review
	r.ID = s.genID()
	s.reviews[r.ID] = &r

	out := r
	return &out, nil
}

func (s *Store) GetReview(_ context.Context, id string) (*store.Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.reviews[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	out := *r
	return &out, nil
}

func (s *Store) ListReviews(_ context.Context, bookID string) ([]*store.Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var reviews []*store.Review
	for _, r := range s.reviews {
		if r.BookID == bookID {
			out := *r
			reviews = append(reviews, &out)
		}
	}
	return reviews, nil
}

func (s *Store) UpdateReview(_ context.Context, id string, update store.ReviewUpdate) (*store.Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.reviews[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	if update.Content != nil {
		r.Content = *update.Content
	}
	if update.Rating != nil {
		r.Rating = *update.Rating
	}

	out := *r
	return &out, nil
}

func (s *Store) DeleteReview(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.reviews[id]; !ok {
		return store.ErrNotFound
	}
	delete(s.reviews, id)
	return nil
}
