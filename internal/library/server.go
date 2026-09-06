package library

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pocketbase/pocketbase/core"

	"gitlab.com/andrew.womeldorf/pbtest/gen/library/v1/libraryv1connect"
	libraryv1 "gitlab.com/andrew.womeldorf/pbtest/gen/library/v1"
	"gitlab.com/andrew.womeldorf/pbtest/internal/authz"
)

var _ libraryv1connect.LibraryServiceHandler = (*Server)(nil)

type Server struct {
	app core.App
}

func NewServer(app core.App) *Server {
	return &Server{app: app}
}

func bookFromRecord(r *core.Record) *libraryv1.Book {
	return &libraryv1.Book{
		Id:     r.Id,
		Title:  r.GetString("title"),
		Author: r.GetString("author"),
		Status: r.GetString("status"),
	}
}

func (s *Server) CreateBook(ctx context.Context, req *connect.Request[libraryv1.CreateBookRequest]) (*connect.Response[libraryv1.CreateBookResponse], error) {
	collection, err := s.app.FindCollectionByNameOrId("books")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	record := core.NewRecord(collection)
	record.Set("title", req.Msg.GetTitle())
	record.Set("author", authz.UserIDFromContext(ctx))
	record.Set("status", "draft")

	if err := s.app.Save(record); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&libraryv1.CreateBookResponse{
		Book: bookFromRecord(record),
	}), nil
}

func (s *Server) GetBook(ctx context.Context, req *connect.Request[libraryv1.GetBookRequest]) (*connect.Response[libraryv1.GetBookResponse], error) {
	record, err := s.app.FindRecordById("books", req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&libraryv1.GetBookResponse{
		Book: bookFromRecord(record),
	}), nil
}

func (s *Server) ListBooks(ctx context.Context, req *connect.Request[libraryv1.ListBooksRequest]) (*connect.Response[libraryv1.ListBooksResponse], error) {
	userID := authz.UserIDFromContext(ctx)

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

func (s *Server) UpdateBook(ctx context.Context, req *connect.Request[libraryv1.UpdateBookRequest]) (*connect.Response[libraryv1.UpdateBookResponse], error) {
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

func (s *Server) DeleteBook(ctx context.Context, req *connect.Request[libraryv1.DeleteBookRequest]) (*connect.Response[libraryv1.DeleteBookResponse], error) {
	record, err := s.app.FindRecordById("books", req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if err := s.app.Delete(record); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&libraryv1.DeleteBookResponse{}), nil
}
