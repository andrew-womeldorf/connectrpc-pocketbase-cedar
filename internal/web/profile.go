package web

import (
	"net/http"

	"github.com/cedar-policy/cedar-go"
	"github.com/pocketbase/pocketbase/core"

	"github.com/andrew-womeldorf/pbtest/internal/authz"
)

func userSelfEntities(userID string) (cedar.EntityUID, cedar.EntityMap) {
	uid := cedar.EntityUID{Type: "Library::User", ID: cedar.String(userID)}
	return uid, cedar.EntityMap{uid: cedar.Entity{UID: uid}}
}

func (h *Handler) profilePage(e *core.RequestEvent) error {
	userID := e.Get("userID").(string)
	userVerified, _ := e.Get("userVerified").(bool)

	resourceUID, entities := userSelfEntities(userID)
	if err := authz.Authorize(userID, userVerified, "GetMe", resourceUID, entities); err != nil {
		return e.Redirect(http.StatusFound, "/books")
	}

	record, err := h.app.FindRecordById("users", userID)
	if err != nil {
		return e.InternalServerError("", err)
	}

	return h.render(e, "profile.html", pageData{
		"Name":     record.GetString("name"),
		"Email":    record.Email(),
		"Verified": record.Verified(),
	})
}

func (h *Handler) updateName(e *core.RequestEvent) error {
	userID := e.Get("userID").(string)
	userVerified, _ := e.Get("userVerified").(bool)

	resourceUID, entities := userSelfEntities(userID)
	if err := authz.Authorize(userID, userVerified, "UpdateMe", resourceUID, entities); err != nil {
		return e.Redirect(http.StatusFound, "/profile")
	}

	name := e.Request.FormValue("name")
	if name == "" {
		return h.renderWithFlash(e, "profile.html", "Name cannot be empty.", nil)
	}

	record, err := h.app.FindRecordById("users", userID)
	if err != nil {
		return e.InternalServerError("", err)
	}

	record.Set("name", name)
	if err := h.app.Save(record); err != nil {
		return h.renderWithFlash(e, "profile.html", "Could not update name: "+err.Error(), nil)
	}

	return e.Redirect(http.StatusFound, "/profile")
}

func (h *Handler) changePassword(e *core.RequestEvent) error {
	userID := e.Get("userID").(string)
	userVerified, _ := e.Get("userVerified").(bool)

	resourceUID, entities := userSelfEntities(userID)
	if err := authz.Authorize(userID, userVerified, "UpdateMe", resourceUID, entities); err != nil {
		return e.Redirect(http.StatusFound, "/profile")
	}

	currentPassword := e.Request.FormValue("currentPassword")
	newPassword := e.Request.FormValue("newPassword")
	confirmPassword := e.Request.FormValue("confirmPassword")

	renderErr := func(msg string) error {
		return h.renderWithFlash(e, "profile.html", msg, nil)
	}

	if newPassword != confirmPassword {
		return renderErr("New passwords do not match.")
	}

	record, err := h.app.FindRecordById("users", userID)
	if err != nil {
		return e.InternalServerError("", err)
	}

	if !record.ValidatePassword(currentPassword) {
		return renderErr("Current password is incorrect.")
	}

	record.SetPassword(newPassword)
	if err := h.app.Save(record); err != nil {
		return renderErr("Could not update password: " + err.Error())
	}

	return e.Redirect(http.StatusFound, "/profile")
}
