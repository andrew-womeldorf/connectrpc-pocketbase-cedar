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

	"github.com/andrew-womeldorf/pbtest/gen/library/v1/libraryv1connect"
	"github.com/andrew-womeldorf/pbtest/internal/authz"
	"github.com/andrew-womeldorf/pbtest/internal/library"
	"github.com/andrew-womeldorf/pbtest/internal/store/pbstore"
	"github.com/andrew-womeldorf/pbtest/internal/web"

	_ "github.com/andrew-womeldorf/pbtest/migrations"
)

//go:embed policies/policy.cedar
var policiesFS embed.FS

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	authz.LoadPolicies(policiesFS)

	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: osutils.IsProbablyGoRun(),
	})

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		s := pbstore.New(app)

		interceptors := connect.WithInterceptors(authz.Interceptor(app))
		path, handler := libraryv1connect.NewLibraryServiceHandler(library.NewServer(app, s), interceptors)

		e.Router.Any(path+"{path...}", func(e *core.RequestEvent) error {
			handler.ServeHTTP(e.Response, e.Request)
			return nil
		})

		web.RegisterRoutes(e.Router, app, s)

		return e.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
