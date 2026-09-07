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

	// I'm curious about this choice. Pocketbase generates a JWT, which in the
	// MDN docs is also recommended, but I want to dive in more around the
	// stateless token. Do we really want that? Can we revoke that globally?
	// Are opaque tokens better? How? What's PASETO?
	token, err := record.NewAuthToken()
	if err != nil {
		return h.renderWithFlash(e, "login.html", "Could not create session.", nil)
	}

	// oi, it's been a long time since i've done legit browser work... time to
	// relearn the things i didn't truely understand the first time i learned
	// them.
	//
	// - https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Cookies
	// - https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Set-Cookie
	//
	// Domain:
	//   - Don't set. Limits the cookie to only same-origin, and not
	//     subdomains. Setting it to an incorrect domain is not a server error.
	//     The server will send the Set-Cookie header, and the client will
	//     receive it, but I do not see the cookie in cookie storage. Perhaps
	//     it would be available if i visited the incorrect domain?
	// SameSite:
	//   - Lax == allow the cookie to be sent when navigating to the site from
	//     not-the-site. i.e. google.com -> mysite.com cookie is sent
	//   - Strict == allow the cookie to be sent only within the site
	//     navigation. i.e. goole.com -> mysite.com cookie is NOT sent
	//
	// learn about cookie prefixes:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Set-Cookie#cookie_prefixes
	//
	// cookie prefixes can help protect against session fixation:
	// https://owasp.org/www-community/attacks/Session_fixation
	http.SetCookie(e.Response, &http.Cookie{
		// __Host-Http == defense in depth
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		// TODO: Revisit MaxAge alongside JWTs. JWT has its own expiration. how
		// do we reconcile these two timestamps, if we even use JWT-esque
		// tokens anymore...
		MaxAge:   60 * 60 * 24 * 14,
		// HttpOnly: true == inaccessible via js document.cookie.
		// Required `true` with cookie prefixes __Http and __Host-Http.
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure: true == only sent *from the client* over https (localhost
		// fine, except on safari) server will send the cookie to the client
		// over http...?
		// Required `true` with any cookie prefixes.
		Secure: true,
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
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure: true,
	})

	return e.Redirect(http.StatusFound, "/login")
}
