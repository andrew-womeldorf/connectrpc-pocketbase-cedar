package web

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"github.com/andrew-womeldorf/pbtest/internal/store"
)

//go:embed views
var viewsFS embed.FS

//go:embed static
var staticFS embed.FS

type Handler struct {
	app   core.App
	store store.Store
}

func RegisterRoutes(r *router.Router[*core.RequestEvent], app core.App, s store.Store) {
	h := &Handler{app: app, store: s}

	staticDir, _ := fs.Sub(staticFS, "static")
	r.GET("/static/{path...}", func(e *core.RequestEvent) error {
		http.StripPrefix("/static/", http.FileServer(http.FS(staticDir))).ServeHTTP(e.Response, e.Request)
		return nil
	})

	r.GET("/login", h.loginPage)
	r.POST("/login", h.loginSubmit)
	r.GET("/register", h.registerPage)
	r.POST("/register", h.registerSubmit)
	r.POST("/logout", h.logout)

	auth := r.Group("").BindFunc(h.requireAuth)
	auth.GET("/books", h.listBooks)
	auth.GET("/books/new", h.newBook)
	auth.POST("/books/new", h.createBook)
	auth.GET("/books/{id}", h.showBook)
	auth.GET("/books/{id}/edit", h.editBook)
	auth.POST("/books/{id}/edit", h.updateBook)
	auth.POST("/books/{id}/delete", h.deleteBook)
	auth.POST("/books/{id}/publish", h.publishBook)
	auth.POST("/books/{id}/reviews", h.createReview)
	auth.POST("/reviews/{id}/delete", h.deleteReview)
}

func (h *Handler) requireAuth(e *core.RequestEvent) error {
	cookie, err := e.Request.Cookie("pb_auth")
	if err != nil || cookie.Value == "" {
		return e.Redirect(http.StatusFound, "/login")
	}

	record, err := h.app.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
	if err != nil {
		http.SetCookie(e.Response, &http.Cookie{
			Name:     "pb_auth",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		return e.Redirect(http.StatusFound, "/login")
	}

	e.Set("userID", record.Id)
	e.Set("userName", record.GetString("name"))
	return e.Next()
}

type pageData map[string]any

func (h *Handler) render(e *core.RequestEvent, page string, data pageData) error {
	if data == nil {
		data = pageData{}
	}
	data["UserID"] = e.Get("userID")
	data["UserName"] = e.Get("userName")

	funcMap := template.FuncMap{
		"stars": func(rating int) string {
			return strings.Repeat("★", rating) + strings.Repeat("☆", 5-rating)
		},
	}

	tmpl, err := template.New("layout.html").Funcs(funcMap).ParseFS(viewsFS, "views/layout.html", "views/"+page)
	if err != nil {
		return e.InternalServerError("", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return e.InternalServerError("", err)
	}

	return e.HTML(http.StatusOK, buf.String())
}

func (h *Handler) renderWithFlash(e *core.RequestEvent, page string, flash string, data pageData) error {
	if data == nil {
		data = pageData{}
	}
	data["Flash"] = flash
	return h.render(e, page, data)
}

func (h *Handler) userName(id string) string {
	if id == "" {
		return "Unknown"
	}
	record, err := h.app.FindRecordById("users", id)
	if err != nil {
		return "Unknown"
	}
	name := record.GetString("name")
	if name == "" {
		return record.GetString("email")
	}
	return name
}
