package web

import (
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/mails"
	"github.com/pocketbase/pocketbase/tools/security"
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

	collection := record.Collection()

	// If MFA is not enabled, issue a token directly.
	if !collection.MFA.Enabled {
		return h.issueToken(e, record)
	}

	// MFA is enabled: password is the first factor. Create an MFA record to
	// track that the password step was completed, then send an OTP for the
	// second factor.
	mfa := core.NewMFA(h.app)
	mfa.SetCollectionRef(collection.Id)
	mfa.SetRecordRef(record.Id)
	mfa.SetMethod(core.MFAMethodPassword)
	if err := h.app.Save(mfa); err != nil {
		return h.renderWithFlash(e, "login.html", "Could not initiate two-factor auth.", nil)
	}

	otpID, err := h.createAndSendOTP(record)
	if err != nil {
		return h.renderWithFlash(e, "login.html", "Could not send verification code.", nil)
	}

	return e.Redirect(http.StatusFound, fmt.Sprintf("/mfa?mfaId=%s&otpId=%s", mfa.Id, otpID))
}

// createAndSendOTP creates an OTP record for the given auth record, sends the
// OTP email, and returns the OTP ID.
func (h *Handler) createAndSendOTP(record *core.Record) (string, error) {
	collection := record.Collection()

	length := collection.OTP.Length
	if length == 0 {
		length = 8
	}
	code := security.RandomStringWithAlphabet(length, "1234567890")

	otp := core.NewOTP(h.app)
	otp.SetCollectionRef(collection.Id)
	otp.SetRecordRef(record.Id)
	otp.SetPassword(code)
	if err := h.app.Save(otp); err != nil {
		return "", err
	}

	if err := mails.SendRecordOTP(h.app, record, otp.Id, code); err != nil {
		return "", err
	}

	return otp.Id, nil
}

func (h *Handler) mfaPage(e *core.RequestEvent) error {
	mfaID := e.Request.URL.Query().Get("mfaId")
	otpID := e.Request.URL.Query().Get("otpId")
	return h.render(e, "mfa.html", pageData{"MFAId": mfaID, "OTPId": otpID})
}

func (h *Handler) mfaSubmit(e *core.RequestEvent) error {
	mfaID := e.Request.FormValue("mfaId")
	otpID := e.Request.FormValue("otpId")
	code := e.Request.FormValue("code")

	renderErr := func(msg string) error {
		return h.renderWithFlash(e, "mfa.html", msg, pageData{"MFAId": mfaID, "OTPId": otpID})
	}

	// Validate the MFA record.
	mfa, err := h.app.FindRecordById(core.CollectionNameMFAs, mfaID)
	if err != nil {
		return renderErr("Invalid or expired session. Please log in again.")
	}
	mfaProxy := &core.MFA{}
	mfaProxy.SetProxyRecord(mfa)

	collection, err := h.app.FindCollectionByNameOrId("users")
	if err != nil {
		return renderErr("Internal error.")
	}

	if mfaProxy.HasExpired(collection.MFA.DurationTime()) {
		_ = h.app.Delete(mfa)
		return renderErr("Session expired. Please log in again.")
	}

	// Validate the OTP record.
	otpRecord, err := h.app.FindRecordById(core.CollectionNameOTPs, otpID)
	if err != nil {
		return renderErr("Invalid or expired code.")
	}
	otpProxy := &core.OTP{}
	otpProxy.SetProxyRecord(otpRecord)

	if otpProxy.HasExpired(collection.OTP.DurationTime()) {
		_ = h.app.Delete(otpRecord)
		return renderErr("Verification code has expired. Please log in again.")
	}

	// OTP and MFA must reference the same user.
	if mfaProxy.RecordRef() != otpProxy.RecordRef() {
		return renderErr("Invalid session.")
	}

	// Validate the OTP code.
	if !otpProxy.ValidatePassword(code) {
		return renderErr("Invalid verification code.")
	}

	// Fetch the auth record.
	authRecord, err := h.app.FindRecordById("users", mfaProxy.RecordRef())
	if err != nil {
		return renderErr("User not found.")
	}

	// If the OTP was sent to this user's email, mark them as verified.
	if otpProxy.SentTo() == authRecord.Email() {
		authRecord.SetVerified(true)
		_ = h.app.Save(authRecord)
	}

	// Clean up MFA and OTP records.
	_ = h.app.Delete(mfa)
	_ = h.app.Delete(otpRecord)

	return h.issueToken(e, authRecord)
}

// issueToken generates an auth JWT, sets the auth cookie, and redirects to /books.
func (h *Handler) issueToken(e *core.RequestEvent, record *core.Record) error {
	token, err := record.NewAuthToken()
	if err != nil {
		return h.renderWithFlash(e, "login.html", "Could not create session.", nil)
	}

	// I'm curious about this choice. Pocketbase generates a JWT, which in the
	// MDN docs is also recommended, but I want to dive in more around the
	// stateless token. Do we really want that? Can we revoke that globally?
	// Are opaque tokens better? How? What's PASETO?
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

	// Send verification email. If this fails, registration still succeeds.
	_ = mails.SendRecordVerification(h.app, record)

	// Authenticate immediately — the user just proved they control the email
	// by registering. They start unverified; verification happens by clicking
	// the email link (or completing an OTP login).
	return h.issueToken(e, record)
}

func (h *Handler) verifySubmit(e *core.RequestEvent) error {
	token := e.Request.URL.Query().Get("token")
	if token == "" {
		return h.renderWithFlash(e, "login.html", "Invalid verification link.", nil)
	}

	record, err := h.app.FindAuthRecordByToken(token, core.TokenTypeVerification)
	if err != nil {
		return h.renderWithFlash(e, "login.html", "Invalid or expired verification link.", nil)
	}

	record.SetVerified(true)
	if err := h.app.Save(record); err != nil {
		return h.renderWithFlash(e, "login.html", "Could not verify email.", nil)
	}

	return h.render(e, "verify.html", nil)
}

func (h *Handler) logout(e *core.RequestEvent) error {
	http.SetCookie(e.Response, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})

	return e.Redirect(http.StatusFound, "/login")
}
