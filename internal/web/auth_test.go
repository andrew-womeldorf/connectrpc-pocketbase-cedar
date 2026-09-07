package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/andrew-womeldorf/pbtest/migrations"
	"github.com/andrew-womeldorf/pbtest/internal/store/pbstore"
)

func newTestApp(t *testing.T) (*tests.TestApp, func()) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("NewTestApp: %v", err)
	}
	return app, func() { app.Cleanup() }
}

func newTestUser(t *testing.T, app core.App, email, password string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("FindCollection: %v", err)
	}
	r := core.NewRecord(col)
	r.SetEmail(email)
	r.SetPassword(password)
	if err := app.Save(r); err != nil {
		t.Fatalf("Save user: %v", err)
	}
	return r
}

func newTestEvent(app core.App, method, target string, body string, contentType string) (*core.RequestEvent, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	} else {
		reqBody = strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, reqBody)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	e := &core.RequestEvent{}
	e.App = app
	e.Request = req
	e.Response = rec
	return e, rec
}

func newFormEvent(app core.App, method, target string, values url.Values) (*core.RequestEvent, *httptest.ResponseRecorder) {
	return newTestEvent(app, method, target, values.Encode(), "application/x-www-form-urlencoded")
}

// TestLoginSubmit_MFARequired asserts that submitting the login form with valid
// credentials redirects to the MFA page (not directly to /books) when MFA is
// enabled on the users collection.
func TestLoginSubmit_MFARequired(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	newTestUser(t, app, "user@test.com", "password12345")

	e, rec := newFormEvent(app, http.MethodPost, "/login", url.Values{
		"email":    {"user@test.com"},
		"password": {"password12345"},
	})

	h := &Handler{app: app, store: pbstore.New(app)}
	if err := h.loginSubmit(e); err != nil {
		t.Fatalf("loginSubmit: %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "/mfa") {
		t.Errorf("Location = %q, want redirect to /mfa", location)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == authCookieName {
			t.Error("auth cookie was set but should not be (MFA not completed)")
		}
	}
}

// TestLoginSubmit_MFARequired_SendsOTP asserts that submitting the login form
// causes an OTP email to be sent to the user.
func TestLoginSubmit_MFARequired_SendsOTP(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	newTestUser(t, app, "user@test.com", "password12345")

	e, _ := newFormEvent(app, http.MethodPost, "/login", url.Values{
		"email":    {"user@test.com"},
		"password": {"password12345"},
	})

	h := &Handler{app: app, store: pbstore.New(app)}
	if err := h.loginSubmit(e); err != nil {
		t.Fatalf("loginSubmit: %v", err)
	}

	if app.TestMailer.TotalSend() < 1 {
		t.Error("expected OTP email to be sent, got none")
	}
}

// TestMFASubmit_ValidOTP asserts the full MFA flow: after login, submitting a
// valid OTP issues an auth cookie and redirects to /books.
func TestMFASubmit_ValidOTP(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	newTestUser(t, app, "user@test.com", "password12345")

	h := &Handler{app: app, store: pbstore.New(app)}

	// Step 1: login with password
	e, rec := newFormEvent(app, http.MethodPost, "/login", url.Values{
		"email":    {"user@test.com"},
		"password": {"password12345"},
	})
	if err := h.loginSubmit(e); err != nil {
		t.Fatalf("loginSubmit: %v", err)
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "/mfa") {
		t.Fatalf("expected redirect to /mfa, got %q", location)
	}

	// Parse mfaId and otpId from the redirect URL
	redirectURL, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	mfaID := redirectURL.Query().Get("mfaId")
	otpID := redirectURL.Query().Get("otpId")
	if mfaID == "" || otpID == "" {
		t.Fatalf("mfaId=%q otpId=%q, both must be non-empty", mfaID, otpID)
	}

	// Extract OTP code from the test mailer
	messages := app.TestMailer.Messages()
	if len(messages) == 0 {
		t.Fatal("no OTP email sent")
	}
	otpCode := extractOTPFromEmail(t, messages[len(messages)-1].HTML)

	// Step 2: submit OTP
	e2, rec2 := newFormEvent(app, http.MethodPost, "/mfa", url.Values{
		"mfaId": {mfaID},
		"otpId": {otpID},
		"code":  {otpCode},
	})
	if err := h.mfaSubmit(e2); err != nil {
		t.Fatalf("mfaSubmit: %v", err)
	}

	if rec2.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec2.Code, http.StatusFound)
	}
	if loc := rec2.Header().Get("Location"); loc != "/books" {
		t.Errorf("Location = %q, want /books", loc)
	}

	var authCookie *http.Cookie
	for _, c := range rec2.Result().Cookies() {
		if c.Name == authCookieName {
			authCookie = c
		}
	}
	if authCookie == nil || authCookie.Value == "" {
		t.Error("auth cookie not set after successful MFA")
	}
}

// TestMFASubmit_InvalidOTP asserts that submitting a wrong OTP code does not
// issue an auth token and re-renders the MFA page with an error.
func TestMFASubmit_InvalidOTP(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	newTestUser(t, app, "user@test.com", "password12345")

	h := &Handler{app: app, store: pbstore.New(app)}

	// Login to get mfaId and otpId
	e, rec := newFormEvent(app, http.MethodPost, "/login", url.Values{
		"email":    {"user@test.com"},
		"password": {"password12345"},
	})
	if err := h.loginSubmit(e); err != nil {
		t.Fatalf("loginSubmit: %v", err)
	}
	location := rec.Header().Get("Location")
	redirectURL, _ := url.Parse(location)
	mfaID := redirectURL.Query().Get("mfaId")
	otpID := redirectURL.Query().Get("otpId")

	// Submit wrong OTP
	e2, rec2 := newFormEvent(app, http.MethodPost, "/mfa", url.Values{
		"mfaId": {mfaID},
		"otpId": {otpID},
		"code":  {"00000000"}, // wrong code
	})
	if err := h.mfaSubmit(e2); err != nil {
		t.Fatalf("mfaSubmit: %v", err)
	}

	// Should NOT redirect to /books
	if rec2.Code == http.StatusFound && rec2.Header().Get("Location") == "/books" {
		t.Error("invalid OTP was accepted")
	}
	for _, c := range rec2.Result().Cookies() {
		if c.Name == authCookieName && c.MaxAge > 0 {
			t.Error("auth cookie was set with invalid OTP")
		}
	}
}

// TestRegisterSubmit_SendsVerificationEmail asserts that registering a new
// account triggers a verification email to be sent.
func TestRegisterSubmit_SendsVerificationEmail(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	e, _ := newFormEvent(app, http.MethodPost, "/register", url.Values{
		"name":            {"Test User"},
		"email":           {"newuser@test.com"},
		"password":        {"password12345"},
		"passwordConfirm": {"password12345"},
	})

	h := &Handler{app: app, store: pbstore.New(app)}
	if err := h.registerSubmit(e); err != nil {
		t.Fatalf("registerSubmit: %v", err)
	}

	if app.TestMailer.TotalSend() < 1 {
		t.Error("expected verification email to be sent, got none")
	}

	msg := app.TestMailer.Messages()[0]
	if !strings.Contains(msg.HTML, "/verify?token=") {
		t.Errorf("verification email body missing /verify?token= link; got: %s", msg.HTML)
	}
}

// TestRegistrationAutoLogsIn asserts that registering immediately issues an auth
// cookie and redirects to /books, without requiring a separate login step.
func TestRegistrationAutoLogsIn(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	e, rec := newFormEvent(app, http.MethodPost, "/register", url.Values{
		"name":            {"New User"},
		"email":           {"newuser@test.com"},
		"password":        {"password12345"},
		"passwordConfirm": {"password12345"},
	})

	h := &Handler{app: app, store: pbstore.New(app)}
	if err := h.registerSubmit(e); err != nil {
		t.Fatalf("registerSubmit: %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != "/books" {
		t.Errorf("Location = %q, want /books", loc)
	}

	var authCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == authCookieName {
			authCookie = c
		}
	}
	if authCookie == nil || authCookie.Value == "" {
		t.Error("auth cookie not set after registration")
	}
}

// TestRegistrationUserStartsUnverified asserts that a newly registered user is
// not yet verified — they must click the email link or complete an OTP login.
func TestRegistrationUserStartsUnverified(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	e, _ := newFormEvent(app, http.MethodPost, "/register", url.Values{
		"name":            {"New User"},
		"email":           {"newuser@test.com"},
		"password":        {"password12345"},
		"passwordConfirm": {"password12345"},
	})

	h := &Handler{app: app, store: pbstore.New(app)}
	if err := h.registerSubmit(e); err != nil {
		t.Fatalf("registerSubmit: %v", err)
	}

	record, err := app.FindAuthRecordByEmail("users", "newuser@test.com")
	if err != nil {
		t.Fatalf("FindAuthRecordByEmail: %v", err)
	}
	if record.Verified() {
		t.Error("user should start unverified after registration")
	}
}

// TestVerifyToken asserts that clicking the verification link marks the user as
// verified.
func TestVerifyToken(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	user := newTestUser(t, app, "user@test.com", "password12345")
	if user.Verified() {
		t.Fatal("user should start unverified")
	}

	token, err := user.NewVerificationToken()
	if err != nil {
		t.Fatalf("NewVerificationToken: %v", err)
	}

	e, rec := newTestEvent(app, http.MethodGet, "/verify?token="+token, "", "")

	h := &Handler{app: app, store: pbstore.New(app)}
	if err := h.verifySubmit(e); err != nil {
		t.Fatalf("verifySubmit: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (verification success page)", rec.Code)
	}

	// The user record must now be verified.
	refreshed, err := app.FindRecordById("users", user.Id)
	if err != nil {
		t.Fatalf("FindRecordById: %v", err)
	}
	if !refreshed.Verified() {
		t.Error("user should be verified after clicking verification link")
	}
}

// TestVerifyToken_InvalidToken asserts that an invalid token renders an error.
func TestVerifyToken_InvalidToken(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	e, rec := newTestEvent(app, http.MethodGet, "/verify?token=bogustoken", "", "")

	h := &Handler{app: app, store: pbstore.New(app)}
	if err := h.verifySubmit(e); err != nil {
		t.Fatalf("verifySubmit: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (error page)", rec.Code)
	}
}

// extractOTPFromEmail parses an 8-digit OTP code from an HTML email body.
func extractOTPFromEmail(t *testing.T, body string) string {
	t.Helper()
	// OTP email body contains: <strong>{OTP}</strong>
	const open = "<strong>"
	const close = "</strong>"
	start := strings.Index(body, open)
	if start == -1 {
		t.Fatalf("could not find OTP in email body: %s", body)
	}
	start += len(open)
	end := strings.Index(body[start:], close)
	if end == -1 {
		t.Fatalf("could not find closing tag in email body: %s", body)
	}
	code := body[start : start+end]
	if len(code) != 8 {
		t.Fatalf("OTP code %q: want 8 digits", code)
	}
	return code
}
