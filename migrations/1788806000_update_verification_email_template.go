package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		users.VerificationTemplate.Body = strings.ReplaceAll(
			users.VerificationTemplate.Body,
			"/_/#/auth/confirm-verification/{TOKEN}",
			"/verify?token={TOKEN}",
		)

		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		users.VerificationTemplate.Body = strings.ReplaceAll(
			users.VerificationTemplate.Body,
			"/verify?token={TOKEN}",
			"/_/#/auth/confirm-verification/{TOKEN}",
		)

		return app.Save(users)
	})
}
