package web

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/andrew-womeldorf/pbtest/migrations"
	"github.com/andrew-womeldorf/pbtest/internal/authz"
	"github.com/andrew-womeldorf/pbtest/internal/store/pbstore"
)

func loadTestPolicies(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile("../../policies/policy.cedar")
	if err != nil {
		t.Fatalf("read policy: %v", err)
	}
	authz.LoadPolicies(fstest.MapFS{
		"policies/policy.cedar": &fstest.MapFile{Data: data},
	})
}

func setupProfileTest(t *testing.T) (*tests.TestApp, *Handler, func()) {
	t.Helper()
	loadTestPolicies(t)
	app, cleanup := newTestApp(t)
	h := &Handler{app: app, store: pbstore.New(app)}
	return app, h, cleanup
}

// TestProfilePage_ShowsUserInfo asserts that GET /profile renders the user's
// name and email, and does not render any password value.
func TestProfilePage_ShowsUserInfo(t *testing.T) {
	app, h, cleanup := setupProfileTest(t)
	defer cleanup()

	user := newTestUser(t, app, "user@test.com", "password12345")
	user.Set("name", "Alice")
	if err := app.Save(user); err != nil {
		t.Fatalf("Save user: %v", err)
	}

	e, rec := newTestEvent(app, http.MethodGet, "/profile", "", "")
	e.Set("userID", user.Id)
	e.Set("userVerified", false)

	if err := h.profilePage(e); err != nil {
		t.Fatalf("profilePage: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Alice") {
		t.Error("profile page should show user name")
	}
	if !strings.Contains(body, "user@test.com") {
		t.Error("profile page should show user email")
	}
	if strings.Contains(body, "password12345") {
		t.Error("profile page must not expose password")
	}
}

// TestProfilePage_UpdateName asserts that POST /profile/name updates the
// user's display name.
func TestProfilePage_UpdateName(t *testing.T) {
	app, h, cleanup := setupProfileTest(t)
	defer cleanup()

	user := newTestUser(t, app, "user@test.com", "password12345")

	e, rec := newFormEvent(app, http.MethodPost, "/profile/name", url.Values{
		"name": {"New Name"},
	})
	e.Set("userID", user.Id)
	e.Set("userVerified", false)

	if err := h.updateName(e); err != nil {
		t.Fatalf("updateName: %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	refreshed, err := app.FindRecordById("users", user.Id)
	if err != nil {
		t.Fatalf("FindRecordById: %v", err)
	}
	if got := refreshed.GetString("name"); got != "New Name" {
		t.Errorf("name = %q, want %q", got, "New Name")
	}
}

// TestProfilePage_ChangePassword asserts that POST /profile/password changes
// the password when current_password is correct.
func TestProfilePage_ChangePassword(t *testing.T) {
	app, h, cleanup := setupProfileTest(t)
	defer cleanup()

	user := newTestUser(t, app, "user@test.com", "oldpassword123")

	e, rec := newFormEvent(app, http.MethodPost, "/profile/password", url.Values{
		"currentPassword": {"oldpassword123"},
		"newPassword":     {"newpassword456"},
		"confirmPassword": {"newpassword456"},
	})
	e.Set("userID", user.Id)
	e.Set("userVerified", false)

	if err := h.changePassword(e); err != nil {
		t.Fatalf("changePassword: %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	// Verify the new password works.
	refreshed, err := app.FindRecordById("users", user.Id)
	if err != nil {
		t.Fatalf("FindRecordById: %v", err)
	}
	if !refreshed.ValidatePassword("newpassword456") {
		t.Error("new password should be valid after change")
	}
}

// TestProfilePage_ChangePassword_WrongCurrent asserts that an incorrect
// current password is rejected.
func TestProfilePage_ChangePassword_WrongCurrent(t *testing.T) {
	app, h, cleanup := setupProfileTest(t)
	defer cleanup()

	user := newTestUser(t, app, "user@test.com", "correctpassword123")

	e, rec := newFormEvent(app, http.MethodPost, "/profile/password", url.Values{
		"currentPassword": {"wrongpassword"},
		"newPassword":     {"newpassword456"},
		"confirmPassword": {"newpassword456"},
	})
	e.Set("userID", user.Id)
	e.Set("userVerified", false)

	if err := h.changePassword(e); err != nil {
		t.Fatalf("changePassword: %v", err)
	}

	// Should re-render profile page with error, not redirect.
	if rec.Code == http.StatusFound {
		t.Error("wrong current password should not redirect")
	}

	// Password must be unchanged.
	refreshed, _ := app.FindRecordById("users", user.Id)
	if !refreshed.ValidatePassword("correctpassword123") {
		t.Error("password should be unchanged after failed attempt")
	}
}

// TestProfilePage_ChangePassword_Mismatch asserts that mismatched new
// passwords are rejected before touching the database.
func TestProfilePage_ChangePassword_Mismatch(t *testing.T) {
	app, h, cleanup := setupProfileTest(t)
	defer cleanup()

	user := newTestUser(t, app, "user@test.com", "password12345")

	e, rec := newFormEvent(app, http.MethodPost, "/profile/password", url.Values{
		"currentPassword": {"password12345"},
		"newPassword":     {"newpassword456"},
		"confirmPassword": {"different789"},
	})
	e.Set("userID", user.Id)
	e.Set("userVerified", false)

	if err := h.changePassword(e); err != nil {
		t.Fatalf("changePassword: %v", err)
	}

	if rec.Code == http.StatusFound {
		t.Error("mismatched passwords should not redirect")
	}
}
