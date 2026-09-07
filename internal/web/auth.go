package web

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

func (h *Handler) loginPage(e *core.RequestEvent) error {
	return h.render(e, "login.html", nil)
}

func (h *Handler) loginSubmit(e *core.RequestEvent) error {
	email := e.Request.FormValue("email")
	password := e.Request.FormValue("password")

	record, err := h.app.FindAuthRecordByEmail("users", email)
	if err != nil || !record.ValidatePassword(password) {
		return h.renderWithFlash(e, "login.html", "Invalid email or password.", nil)
	}

	token, err := record.NewAuthToken()
	if err != nil {
		return h.renderWithFlash(e, "login.html", "Could not create session.", nil)
	}

	http.SetCookie(e.Response, &http.Cookie{
		Name:     "pb_auth",
		Value:    token,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 14,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return e.Redirect(http.StatusFound, "/books")
}

func (h *Handler) registerPage(e *core.RequestEvent) error {
	return h.render(e, "register.html", nil)
}

func (h *Handler) registerSubmit(e *core.RequestEvent) error {
	name := e.Request.FormValue("name")
	email := e.Request.FormValue("email")
	password := e.Request.FormValue("password")
	passwordConfirm := e.Request.FormValue("passwordConfirm")

	if password != passwordConfirm {
		return h.renderWithFlash(e, "register.html", "Passwords do not match.", nil)
	}

	collection, err := h.app.FindCollectionByNameOrId("users")
	if err != nil {
		return h.renderWithFlash(e, "register.html", "Registration is not available.", nil)
	}

	record := core.NewRecord(collection)
	record.Set("name", name)
	record.Set("email", email)
	record.Set("password", password)

	if err := h.app.Save(record); err != nil {
		return h.renderWithFlash(e, "register.html", "Could not create account: "+err.Error(), nil)
	}

	return e.Redirect(http.StatusFound, "/login")
}

func (h *Handler) logout(e *core.RequestEvent) error {
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
