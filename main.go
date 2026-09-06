package main

import (
	"embed"
	"log"
	"log/slog"
	"os"

	"connectrpc.com/connect"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"

	"gitlab.com/andrew.womeldorf/pbtest/gen/library/v1/libraryv1connect"
	"gitlab.com/andrew.womeldorf/pbtest/internal/authz"
	"gitlab.com/andrew.womeldorf/pbtest/internal/library"

	_ "gitlab.com/andrew.womeldorf/pbtest/migrations"
)

//go:embed policies/policies.cedar
var policiesFS embed.FS

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	authz.LoadPolicies(policiesFS)

	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: osutils.IsProbablyGoRun(),
	})

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		interceptors := connect.WithInterceptors(authz.Interceptor(app))
		path, handler := libraryv1connect.NewLibraryServiceHandler(library.NewServer(app), interceptors)

		e.Router.Any(path+"{path...}", func(e *core.RequestEvent) error {
			handler.ServeHTTP(e.Response, e.Request)
			return nil
		})

		return e.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
