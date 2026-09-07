package library

import (
	"context"

	"connectrpc.com/connect"
	"github.com/cedar-policy/cedar-go"

	libraryv1 "github.com/andrew-womeldorf/pbtest/gen/library/v1"
	"github.com/andrew-womeldorf/pbtest/internal/authz"
	"github.com/andrew-womeldorf/pbtest/internal/store"
)

func bookToProto(b *store.Book) *libraryv1.Book {
	return &libraryv1.Book{
		Id:     b.ID,
		Title:  b.Title,
		Author: b.Author,
		Status: b.Status,
	}
}

func bookEntities(bookID, bookAuthor, bookStatus string) (cedar.EntityUID, cedar.EntityMap) {
	resourceUID := cedar.EntityUID{Type: "Book", ID: cedar.String(bookID)}
	entities := cedar.EntityMap{
		resourceUID: cedar.Entity{
			UID: resourceUID,
			Attributes: cedar.NewRecord(cedar.RecordMap{
				"author": cedar.String(bookAuthor),
				"status": cedar.String(bookStatus),
			}),
		},
	}
	return resourceUID, entities
}

func (s *Server) CreateBook(ctx context.Context, req *connect.Request[libraryv1.CreateBookRequest]) (*connect.Response[libraryv1.CreateBookResponse], error) {
	userID := authz.UserIDFromContext(ctx)

	resourceUID, entities := bookEntities("", "", "")
	if err := authz.Authorize(userID, "CreateBook", resourceUID, entities); err != nil {
		return nil, err
	}

	book, err := s.store.CreateBook(ctx, &store.Book{
		Title:  req.Msg.GetTitle(),
		Author: userID,
		Status: "draft",
	})
	if err != nil {
		return nil, storeErr(err)
	}

	return connect.NewResponse(&libraryv1.CreateBookResponse{
		Book: bookToProto(book),
	}), nil
}

func (s *Server) GetBook(ctx context.Context, req *connect.Request[libraryv1.GetBookRequest]) (*connect.Response[libraryv1.GetBookResponse], error) {
	book, err := s.store.GetBook(ctx, req.Msg.GetId())
	if err != nil {
		return nil, storeErr(err)
	}

	userID := authz.UserIDFromContext(ctx)
	resourceUID, entities := bookEntities(book.ID, book.Author, book.Status)
	if err := authz.Authorize(userID, "GetBook", resourceUID, entities); err != nil {
		return nil, err
	}

	return connect.NewResponse(&libraryv1.GetBookResponse{
		Book: bookToProto(book),
	}), nil
}

func (s *Server) ListBooks(ctx context.Context, req *connect.Request[libraryv1.ListBooksRequest]) (*connect.Response[libraryv1.ListBooksResponse], error) {
	userID := authz.UserIDFromContext(ctx)

	resourceUID, entities := bookEntities("", "", "")
	if err := authz.Authorize(userID, "ListBooks", resourceUID, entities); err != nil {
		return nil, err
	}

	books, err := s.store.ListBooks(ctx)
	if err != nil {
		return nil, storeErr(err)
	}

	var result []*libraryv1.Book
	for _, b := range books {
		if b.Status == "draft" && b.Author != userID {
			continue
		}
		result = append(result, bookToProto(b))
	}

	return connect.NewResponse(&libraryv1.ListBooksResponse{
		Books: result,
	}), nil
}

func (s *Server) UpdateBook(ctx context.Context, req *connect.Request[libraryv1.UpdateBookRequest]) (*connect.Response[libraryv1.UpdateBookResponse], error) {
	book, err := s.store.GetBook(ctx, req.Msg.GetId())
	if err != nil {
		return nil, storeErr(err)
	}

	userID := authz.UserIDFromContext(ctx)
	resourceUID, entities := bookEntities(book.ID, book.Author, book.Status)
	if err := authz.Authorize(userID, "UpdateBook", resourceUID, entities); err != nil {
		return nil, err
	}

	var update store.BookUpdate
	if t := req.Msg.GetTitle(); t != "" {
		update.Title = &t
	}
	if st := req.Msg.GetStatus(); st != "" {
		update.Status = &st
	}

	updated, err := s.store.UpdateBook(ctx, book.ID, update)
	if err != nil {
		return nil, storeErr(err)
	}

	return connect.NewResponse(&libraryv1.UpdateBookResponse{
		Book: bookToProto(updated),
	}), nil
}

func (s *Server) DeleteBook(ctx context.Context, req *connect.Request[libraryv1.DeleteBookRequest]) (*connect.Response[libraryv1.DeleteBookResponse], error) {
	book, err := s.store.GetBook(ctx, req.Msg.GetId())
	if err != nil {
		return nil, storeErr(err)
	}

	userID := authz.UserIDFromContext(ctx)
	resourceUID, entities := bookEntities(book.ID, book.Author, book.Status)
	if err := authz.Authorize(userID, "DeleteBook", resourceUID, entities); err != nil {
		return nil, err
	}

	if err := s.store.DeleteBook(ctx, book.ID); err != nil {
		return nil, storeErr(err)
	}

	return connect.NewResponse(&libraryv1.DeleteBookResponse{}), nil
}
