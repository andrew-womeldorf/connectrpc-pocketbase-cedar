package web

import (
	"net/http"
	"strconv"

	"github.com/cedar-policy/cedar-go"
	"github.com/pocketbase/pocketbase/core"

	"github.com/andrew-womeldorf/pbtest/internal/authz"
	"github.com/andrew-womeldorf/pbtest/internal/store"
)

func reviewEntities(reviewID, reviewer, bookAuthor string) (cedar.EntityUID, cedar.EntityMap) {
	resourceUID := cedar.EntityUID{Type: "Library::Review", ID: cedar.String(reviewID)}
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

func (h *Handler) createReview(e *core.RequestEvent) error {
	bookID := e.Request.PathValue("id")
	userID := e.Get("userID").(string)
	userVerified, _ := e.Get("userVerified").(bool)

	book, err := h.store.GetBook(e.Request.Context(), bookID)
	if err != nil {
		return e.NotFoundError("", err)
	}

	resourceUID, entities := reviewEntities("", userID, book.Author)
	if err := authz.Authorize(userID, userVerified, "CreateReview", resourceUID, entities); err != nil {
		return e.Redirect(http.StatusFound, "/books/"+bookID)
	}

	ratingStr := e.Request.FormValue("rating")
	rating, err := strconv.Atoi(ratingStr)
	if err != nil || rating < 1 || rating > 5 {
		return e.Redirect(http.StatusFound, "/books/"+bookID)
	}

	content := e.Request.FormValue("content")

	if _, err := h.store.CreateReview(e.Request.Context(), &store.Review{
		BookID:   book.ID,
		Reviewer: userID,
		Content:  content,
		Rating:   rating,
	}); err != nil {
		return e.Redirect(http.StatusFound, "/books/"+bookID)
	}

	return e.Redirect(http.StatusFound, "/books/"+bookID)
}

func (h *Handler) deleteReview(e *core.RequestEvent) error {
	reviewID := e.Request.PathValue("id")
	userID := e.Get("userID").(string)
	userVerified, _ := e.Get("userVerified").(bool)

	review, err := h.store.GetReview(e.Request.Context(), reviewID)
	if err != nil {
		return e.NotFoundError("", err)
	}

	book, err := h.store.GetBook(e.Request.Context(), review.BookID)
	if err != nil {
		return e.InternalServerError("", err)
	}

	resourceUID, entities := reviewEntities(review.ID, review.Reviewer, book.Author)
	if err := authz.Authorize(userID, userVerified, "DeleteReview", resourceUID, entities); err != nil {
		return e.Redirect(http.StatusFound, "/books/"+review.BookID)
	}

	if err := h.store.DeleteReview(e.Request.Context(), review.ID); err != nil {
		return e.InternalServerError("", err)
	}

	return e.Redirect(http.StatusFound, "/books/"+review.BookID)
}
