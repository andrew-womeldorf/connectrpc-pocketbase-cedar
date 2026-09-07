package library

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/cedar-policy/cedar-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	libraryv1 "gitlab.com/andrew.womeldorf/pbtest/gen/library/v1"
	"gitlab.com/andrew.womeldorf/pbtest/internal/authz"
)

func reviewFromRecord(r *core.Record) *libraryv1.Review {
	return &libraryv1.Review{
		Id:       r.Id,
		BookId:   r.GetString("book"),
		Reviewer: r.GetString("reviewer"),
		Content:  r.GetString("content"),
		Rating:   int32(r.GetInt("rating")),
	}
}

func reviewEntities(reviewID, reviewer, bookAuthor string) (cedar.EntityUID, cedar.EntityMap) {
	resourceUID := cedar.EntityUID{Type: "Review", ID: cedar.String(reviewID)}
	entities := cedar.EntityMap{
		resourceUID: cedar.Entity{
			UID: resourceUID,
			Attributes: cedar.NewRecord(cedar.RecordMap{
				"reviewer":    cedar.String(reviewer),
				"book_author": cedar.String(bookAuthor),
			}),
		},
	}
	return resourceUID, entities
}

func (s *Server) CreateReview(ctx context.Context, req *connect.Request[libraryv1.CreateReviewRequest]) (*connect.Response[libraryv1.CreateReviewResponse], error) {
	userID := authz.UserIDFromContext(ctx)

	book, err := s.app.FindRecordById("books", req.Msg.GetBookId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("book not found"))
	}

	resourceUID, entities := reviewEntities("", userID, book.GetString("author"))
	if err := authz.Authorize(userID, "CreateReview", resourceUID, entities); err != nil {
		return nil, err
	}

	rating := req.Msg.GetRating()
	if rating < 1 || rating > 5 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rating must be between 1 and 5"))
	}

	collection, err := s.app.FindCollectionByNameOrId("reviews")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	record := core.NewRecord(collection)
	record.Set("book", book.Id)
	record.Set("reviewer", userID)
	record.Set("content", req.Msg.GetContent())
	record.Set("rating", rating)

	if err := s.app.Save(record); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&libraryv1.CreateReviewResponse{
		Review: reviewFromRecord(record),
	}), nil
}

func (s *Server) GetReview(ctx context.Context, req *connect.Request[libraryv1.GetReviewRequest]) (*connect.Response[libraryv1.GetReviewResponse], error) {
	review, err := s.app.FindRecordById("reviews", req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("review not found"))
	}

	book, err := s.app.FindRecordById("books", review.GetString("book"))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	userID := authz.UserIDFromContext(ctx)
	resourceUID, entities := reviewEntities(review.Id, review.GetString("reviewer"), book.GetString("author"))
	if err := authz.Authorize(userID, "GetReview", resourceUID, entities); err != nil {
		return nil, err
	}

	return connect.NewResponse(&libraryv1.GetReviewResponse{
		Review: reviewFromRecord(review),
	}), nil
}

func (s *Server) ListReviews(ctx context.Context, req *connect.Request[libraryv1.ListReviewsRequest]) (*connect.Response[libraryv1.ListReviewsResponse], error) {
	userID := authz.UserIDFromContext(ctx)

	resourceUID, entities := reviewEntities("", "", "")
	if err := authz.Authorize(userID, "ListReviews", resourceUID, entities); err != nil {
		return nil, err
	}

	records, err := s.app.FindAllRecords("reviews", dbx.NewExp("book = {:bookId}", dbx.Params{"bookId": req.Msg.GetBookId()}))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var reviews []*libraryv1.Review
	for _, r := range records {
		reviews = append(reviews, reviewFromRecord(r))
	}

	return connect.NewResponse(&libraryv1.ListReviewsResponse{
		Reviews: reviews,
	}), nil
}

func (s *Server) UpdateReview(ctx context.Context, req *connect.Request[libraryv1.UpdateReviewRequest]) (*connect.Response[libraryv1.UpdateReviewResponse], error) {
	review, err := s.app.FindRecordById("reviews", req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("review not found"))
	}

	book, err := s.app.FindRecordById("books", review.GetString("book"))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	userID := authz.UserIDFromContext(ctx)
	resourceUID, entities := reviewEntities(review.Id, review.GetString("reviewer"), book.GetString("author"))
	if err := authz.Authorize(userID, "UpdateReview", resourceUID, entities); err != nil {
		return nil, err
	}

	if req.Msg.GetContent() != "" {
		review.Set("content", req.Msg.GetContent())
	}
	if req.Msg.GetRating() != 0 {
		rating := req.Msg.GetRating()
		if rating < 1 || rating > 5 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rating must be between 1 and 5"))
		}
		review.Set("rating", rating)
	}

	if err := s.app.Save(review); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&libraryv1.UpdateReviewResponse{
		Review: reviewFromRecord(review),
	}), nil
}

func (s *Server) DeleteReview(ctx context.Context, req *connect.Request[libraryv1.DeleteReviewRequest]) (*connect.Response[libraryv1.DeleteReviewResponse], error) {
	review, err := s.app.FindRecordById("reviews", req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("review not found"))
	}

	book, err := s.app.FindRecordById("books", review.GetString("book"))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	userID := authz.UserIDFromContext(ctx)
	resourceUID, entities := reviewEntities(review.Id, review.GetString("reviewer"), book.GetString("author"))
	if err := authz.Authorize(userID, "DeleteReview", resourceUID, entities); err != nil {
		return nil, err
	}

	if err := s.app.Delete(review); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&libraryv1.DeleteReviewResponse{}), nil
}
