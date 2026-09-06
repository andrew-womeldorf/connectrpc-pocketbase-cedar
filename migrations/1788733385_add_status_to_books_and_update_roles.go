package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add status field to books
		books, err := app.FindCollectionByNameOrId("books")
		if err != nil {
			return err
		}

		books.Fields.Add(&core.SelectField{
			Name:     "status",
			Required: true,
			Values:   []string{"draft", "published"},
		})

		if err := app.Save(books); err != nil {
			return err
		}

		// Set existing books to published
		if _, err := app.DB().NewQuery("UPDATE books SET status = 'published' WHERE status = ''").Execute(); err != nil {
			return err
		}

		// Update user roles: rename "librarian" to "author"
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		roleField := users.Fields.GetByName("role")
		if sf, ok := roleField.(*core.SelectField); ok {
			sf.Values = []string{"member", "author"}
		}

		if err := app.Save(users); err != nil {
			return err
		}

		// Migrate existing librarian users to author
		if _, err := app.DB().NewQuery("UPDATE users SET role = 'author' WHERE role = 'librarian'").Execute(); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Revert user roles: rename "author" back to "librarian"
		if _, err := app.DB().NewQuery("UPDATE users SET role = 'librarian' WHERE role = 'author'").Execute(); err != nil {
			return err
		}

		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		roleField := users.Fields.GetByName("role")
		if sf, ok := roleField.(*core.SelectField); ok {
			sf.Values = []string{"member", "librarian"}
		}

		if err := app.Save(users); err != nil {
			return err
		}

		// Remove status field from books
		books, err := app.FindCollectionByNameOrId("books")
		if err != nil {
			return err
		}

		books.Fields.RemoveByName("status")

		if err := app.Save(books); err != nil {
			return err
		}

		return nil
	})
}
