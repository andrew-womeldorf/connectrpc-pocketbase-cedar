package library

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/cedar-policy/cedar-go"

	libraryv1 "github.com/andrew-womeldorf/pbtest/gen/library/v1"
	"github.com/andrew-womeldorf/pbtest/internal/authz"
	"github.com/andrew-womeldorf/pbtest/internal/store"
)

func reviewToProto(r *store.Review) *libraryv1.Review {
	return &libraryv1.Review{
		Id:       r.ID,
		BookId:   r.BookID,
		Reviewer: r.Reviewer,
		Content:  r.Content,
		Rating:   int32(r.Rating),
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

	book, err := s.store.GetBook(ctx, req.Msg.GetBookId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("book not found"))
	}

	resourceUID, entities := reviewEntities("", userID, book.Author)
	if err := authz.Authorize(userID, "CreateReview", resourceUID, entities); err != nil {
		return nil, err
	}

	rating := req.Msg.GetRating()
	if rating < 1 || rating > 5 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rating must be between 1 and 5"))
	}

	review, err := s.store.CreateReview(ctx, &store.Review{
		BookID:   book.ID,
		Reviewer: userID,
		Content:  req.Msg.GetContent(),
		Rating:   int(rating),
	})
	if err != nil {
		return nil, storeErr(err)
	}

	return connect.NewResponse(&libraryv1.CreateReviewResponse{
		Review: reviewToProto(review),
	}), nil
}

func (s *Server) GetReview(ctx context.Context, req *connect.Request[libraryv1.GetReviewRequest]) (*connect.Response[libraryv1.GetReviewResponse], error) {
	review, err := s.store.GetReview(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("review not found"))
	}

	book, err := s.store.GetBook(ctx, review.BookID)
	if err != nil {
		return nil, storeErr(err)
	}

	userID := authz.UserIDFromContext(ctx)
	resourceUID, entities := reviewEntities(review.ID, review.Reviewer, book.Author)
	if err := authz.Authorize(userID, "GetReview", resourceUID, entities); err != nil {
		return nil, err
	}

	return connect.NewResponse(&libraryv1.GetReviewResponse{
		Review: reviewToProto(review),
	}), nil
}

func (s *Server) ListReviews(ctx context.Context, req *connect.Request[libraryv1.ListReviewsRequest]) (*connect.Response[libraryv1.ListReviewsResponse], error) {
	userID := authz.UserIDFromContext(ctx)

	resourceUID, entities := reviewEntities("", "", "")
	if err := authz.Authorize(userID, "ListReviews", resourceUID, entities); err != nil {
		return nil, err
	}

	reviews, err := s.store.ListReviews(ctx, req.Msg.GetBookId())
	if err != nil {
		return nil, storeErr(err)
	}

	var result []*libraryv1.Review
	for _, r := range reviews {
		result = append(result, reviewToProto(r))
	}

	return connect.NewResponse(&libraryv1.ListReviewsResponse{
		Reviews: result,
	}), nil
}

func (s *Server) UpdateReview(ctx context.Context, req *connect.Request[libraryv1.UpdateReviewRequest]) (*connect.Response[libraryv1.UpdateReviewResponse], error) {
	review, err := s.store.GetReview(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("review not found"))
	}

	book, err := s.store.GetBook(ctx, review.BookID)
	if err != nil {
		return nil, storeErr(err)
	}

	userID := authz.UserIDFromContext(ctx)
	resourceUID, entities := reviewEntities(review.ID, review.Reviewer, book.Author)
	if err := authz.Authorize(userID, "UpdateReview", resourceUID, entities); err != nil {
		return nil, err
	}

	var update store.ReviewUpdate
	if c := req.Msg.GetContent(); c != "" {
		update.Content = &c
	}
	if r := req.Msg.GetRating(); r != 0 {
		rating := int(r)
		if rating < 1 || rating > 5 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rating must be between 1 and 5"))
		}
		update.Rating = &rating
	}

	updated, err := s.store.UpdateReview(ctx, review.ID, update)
	if err != nil {
		return nil, storeErr(err)
	}

	return connect.NewResponse(&libraryv1.UpdateReviewResponse{
		Review: reviewToProto(updated),
	}), nil
}

func (s *Server) DeleteReview(ctx context.Context, req *connect.Request[libraryv1.DeleteReviewRequest]) (*connect.Response[libraryv1.DeleteReviewResponse], error) {
	review, err := s.store.GetReview(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("review not found"))
	}

	book, err := s.store.GetBook(ctx, review.BookID)
	if err != nil {
		return nil, storeErr(err)
	}

	userID := authz.UserIDFromContext(ctx)
	resourceUID, entities := reviewEntities(review.ID, review.Reviewer, book.Author)
	if err := authz.Authorize(userID, "DeleteReview", resourceUID, entities); err != nil {
		return nil, err
	}

	if err := s.store.DeleteReview(ctx, review.ID); err != nil {
		return nil, storeErr(err)
	}

	return connect.NewResponse(&libraryv1.DeleteReviewResponse{}), nil
}
