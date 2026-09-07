package web

import (
	"net/http"

	"github.com/cedar-policy/cedar-go"
	"github.com/pocketbase/pocketbase/core"

	"github.com/andrew-womeldorf/pbtest/internal/authz"
	"github.com/andrew-womeldorf/pbtest/internal/store"
)

type bookView struct {
	ID         string
	Title      string
	Author     string
	AuthorName string
	Status     string
}

func (h *Handler) bookView(b *store.Book) bookView {
	return bookView{
		ID:         b.ID,
		Title:      b.Title,
		Author:     b.Author,
		AuthorName: h.userName(b.Author),
		Status:     b.Status,
	}
}

func bookEntities(bookID, bookAuthor, bookStatus string) (cedar.EntityUID, cedar.EntityMap) {
	resourceUID := cedar.EntityUID{Type: "Library::Book", ID: cedar.String(bookID)}
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

func (h *Handler) listBooks(e *core.RequestEvent) error {
	userID := e.Get("userID").(string)

	resourceUID, entities := bookEntities("", "", "")
	if err := authz.Authorize(userID, "ListBooks", resourceUID, entities); err != nil {
		return e.Redirect(http.StatusFound, "/login")
	}

	books, err := h.store.ListBooks(e.Request.Context())
	if err != nil {
		return e.InternalServerError("", err)
	}

	var views []bookView
	for _, b := range books {
		if b.Status == "draft" && b.Author != userID {
			continue
		}
		views = append(views, h.bookView(b))
	}

	return h.render(e, "books.html", pageData{"Books": views})
}

func (h *Handler) showBook(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	userID := e.Get("userID").(string)

	book, err := h.store.GetBook(e.Request.Context(), id)
	if err != nil {
		return e.NotFoundError("", err)
	}

	resourceUID, entities := bookEntities(book.ID, book.Author, book.Status)
	if err := authz.Authorize(userID, "GetBook", resourceUID, entities); err != nil {
		return e.NotFoundError("", nil)
	}

	reviews, err := h.store.ListReviews(e.Request.Context(), book.ID)
	if err != nil {
		return e.InternalServerError("", err)
	}

	type reviewView struct {
		ID           string
		Content      string
		Rating       int
		Reviewer     string
		ReviewerName string
		CanDelete    bool
	}

	var reviewViews []reviewView
	for _, r := range reviews {
		reviewViews = append(reviewViews, reviewView{
			ID:           r.ID,
			Content:      r.Content,
			Rating:       r.Rating,
			Reviewer:     r.Reviewer,
			ReviewerName: h.userName(r.Reviewer),
			CanDelete:    r.Reviewer == userID || book.Author == userID,
		})
	}

	bv := h.bookView(book)
	return h.render(e, "book.html", pageData{
		"Book":      bv,
		"Reviews":   reviewViews,
		"IsOwner":   book.Author == userID,
		"CanReview": book.Author != userID,
	})
}

func (h *Handler) newBook(e *core.RequestEvent) error {
	return h.render(e, "book_form.html", nil)
}

func (h *Handler) createBook(e *core.RequestEvent) error {
	userID := e.Get("userID").(string)
	title := e.Request.FormValue("title")

	resourceUID, entities := bookEntities("", "", "")
	if err := authz.Authorize(userID, "CreateBook", resourceUID, entities); err != nil {
		return h.renderWithFlash(e, "book_form.html", "Not authorized.", nil)
	}

	book, err := h.store.CreateBook(e.Request.Context(), &store.Book{
		Title:  title,
		Author: userID,
		Status: "draft",
	})
	if err != nil {
		return h.renderWithFlash(e, "book_form.html", "Could not create book: "+err.Error(), nil)
	}

	return e.Redirect(http.StatusFound, "/books/"+book.ID)
}

func (h *Handler) editBook(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	userID := e.Get("userID").(string)

	book, err := h.store.GetBook(e.Request.Context(), id)
	if err != nil {
		return e.NotFoundError("", err)
	}

	if book.Author != userID {
		return e.NotFoundError("", nil)
	}

	bv := h.bookView(book)
	return h.render(e, "book_form.html", pageData{"Book": bv})
}

func (h *Handler) updateBook(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	userID := e.Get("userID").(string)

	book, err := h.store.GetBook(e.Request.Context(), id)
	if err != nil {
		return e.NotFoundError("", err)
	}

	resourceUID, entities := bookEntities(book.ID, book.Author, book.Status)
	if err := authz.Authorize(userID, "UpdateBook", resourceUID, entities); err != nil {
		return e.NotFoundError("", nil)
	}

	title := e.Request.FormValue("title")
	status := e.Request.FormValue("status")

	var update store.BookUpdate
	if title != "" {
		update.Title = &title
	}
	if status != "" {
		update.Status = &status
	}

	if _, err := h.store.UpdateBook(e.Request.Context(), book.ID, update); err != nil {
		bv := h.bookView(book)
		return h.renderWithFlash(e, "book_form.html", "Could not update book: "+err.Error(), pageData{"Book": bv})
	}

	return e.Redirect(http.StatusFound, "/books/"+book.ID)
}

func (h *Handler) deleteBook(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	userID := e.Get("userID").(string)

	book, err := h.store.GetBook(e.Request.Context(), id)
	if err != nil {
		return e.NotFoundError("", err)
	}

	resourceUID, entities := bookEntities(book.ID, book.Author, book.Status)
	if err := authz.Authorize(userID, "DeleteBook", resourceUID, entities); err != nil {
		return e.NotFoundError("", nil)
	}

	if err := h.store.DeleteBook(e.Request.Context(), book.ID); err != nil {
		return e.InternalServerError("", err)
	}

	return e.Redirect(http.StatusFound, "/books")
}

func (h *Handler) publishBook(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	userID := e.Get("userID").(string)

	book, err := h.store.GetBook(e.Request.Context(), id)
	if err != nil {
		return e.NotFoundError("", err)
	}

	resourceUID, entities := bookEntities(book.ID, book.Author, book.Status)
	if err := authz.Authorize(userID, "UpdateBook", resourceUID, entities); err != nil {
		return e.NotFoundError("", nil)
	}

	status := "published"
	if _, err := h.store.UpdateBook(e.Request.Context(), book.ID, store.BookUpdate{Status: &status}); err != nil {
		return e.InternalServerError("", err)
	}

	return e.Redirect(http.StatusFound, "/books/"+book.ID)
}
