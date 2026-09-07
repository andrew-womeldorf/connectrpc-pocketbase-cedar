package library

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/andrew-womeldorf/pbtest/gen/library/v1/libraryv1connect"
	"github.com/andrew-womeldorf/pbtest/internal/store"
)

var _ libraryv1connect.LibraryServiceHandler = (*Server)(nil)

type Server struct {
	store store.Store
}

func NewServer(s store.Store) *Server {
	return &Server{store: s}
}

func storeErr(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
