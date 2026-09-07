package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		users.MFA.Enabled = true
		users.MFA.Duration = 600

		users.OTP.Enabled = true
		users.OTP.Duration = 180
		users.OTP.Length = 8

		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		users.MFA.Enabled = false
		users.OTP.Enabled = false

		return app.Save(users)
	})
}
