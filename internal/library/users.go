package library

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/cedar-policy/cedar-go"
	"github.com/pocketbase/pocketbase/core"

	libraryv1 "github.com/andrew-womeldorf/pbtest/gen/library/v1"
	"github.com/andrew-womeldorf/pbtest/internal/authz"
)

func recordToUser(r *core.Record) *libraryv1.User {
	return &libraryv1.User{
		Id:       r.Id,
		Name:     r.GetString("name"),
		Email:    r.Email(),
		Verified: r.Verified(),
	}
}

// userSelfEntities builds a Cedar entity map where the resource is the
// requesting user. Used for GetMe/UpdateMe to enforce resource == principal.
func userSelfEntities(userID string) (cedar.EntityUID, cedar.EntityMap) {
	uid := cedar.EntityUID{Type: "Library::User", ID: cedar.String(userID)}
	return uid, cedar.EntityMap{uid: cedar.Entity{UID: uid}}
}

func (s *Server) GetMe(ctx context.Context, req *connect.Request[libraryv1.GetMeRequest]) (*connect.Response[libraryv1.GetMeResponse], error) {
	userID := authz.UserIDFromContext(ctx)
	userVerified := authz.UserVerifiedFromContext(ctx)

	resourceUID, entities := userSelfEntities(userID)
	if err := authz.Authorize(userID, userVerified, "GetMe", resourceUID, entities); err != nil {
		return nil, err
	}

	record, err := s.app.FindRecordById("users", userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}

	return connect.NewResponse(&libraryv1.GetMeResponse{
		User: recordToUser(record),
	}), nil
}

func (s *Server) UpdateMe(ctx context.Context, req *connect.Request[libraryv1.UpdateMeRequest]) (*connect.Response[libraryv1.UpdateMeResponse], error) {
	userID := authz.UserIDFromContext(ctx)
	userVerified := authz.UserVerifiedFromContext(ctx)

	resourceUID, entities := userSelfEntities(userID)
	if err := authz.Authorize(userID, userVerified, "UpdateMe", resourceUID, entities); err != nil {
		return nil, err
	}

	record, err := s.app.FindRecordById("users", userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}

	msg := req.Msg
	if msg.Name != nil {
		record.Set("name", msg.GetName())
	}

	if msg.NewPassword != nil {
		if msg.CurrentPassword == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("current_password is required when changing password"))
		}
		if !record.ValidatePassword(msg.GetCurrentPassword()) {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("current password is incorrect"))
		}
		record.SetPassword(msg.GetNewPassword())
	}

	if err := s.app.Save(record); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&libraryv1.UpdateMeResponse{
		User: recordToUser(record),
	}), nil
}
