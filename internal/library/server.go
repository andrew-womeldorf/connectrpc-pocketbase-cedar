package library

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/pocketbase/pocketbase/core"

	"github.com/andrew-womeldorf/pbtest/gen/library/v1/libraryv1connect"
	"github.com/andrew-womeldorf/pbtest/internal/store"
)

var _ libraryv1connect.LibraryServiceHandler = (*Server)(nil)

type Server struct {
	app   core.App
	store store.Store
}

func NewServer(app core.App, s store.Store) *Server {
	return &Server{app: app, store: s}
}

func storeErr(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
