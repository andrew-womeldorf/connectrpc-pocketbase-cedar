package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		books, err := app.FindCollectionByNameOrId("books")
		if err != nil {
			return err
		}

		// Copy owner values into author before changing the field type,
		// so existing books retain their author association.
		if _, err := app.DB().NewQuery("UPDATE books SET author = owner WHERE owner != ''").Execute(); err != nil {
			return err
		}

		// Remove the old text author field and owner relation field
		books.Fields.RemoveByName("author")
		books.Fields.RemoveByName("owner")

		// Add author as a relation to users
		books.Fields.Add(&core.RelationField{
			Name:         "author",
			CollectionId: "_pb_users_auth_",
			MaxSelect:    1,
			Required:     true,
		})

		if err := app.Save(books); err != nil {
			return err
		}

		// Remove role field from users
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		users.Fields.RemoveByName("role")

		return app.Save(users)
	}, func(app core.App) error {
		// Restore role field on users
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		users.Fields.Add(&core.SelectField{
			Name:   "role",
			Values: []string{"member", "author"},
		})

		if err := app.Save(users); err != nil {
			return err
		}

		books, err := app.FindCollectionByNameOrId("books")
		if err != nil {
			return err
		}

		// Replace author relation with text field and restore owner relation
		books.Fields.RemoveByName("author")

		books.Fields.Add(&core.TextField{
			Name: "author",
		})
		books.Fields.Add(&core.RelationField{
			Name:         "owner",
			CollectionId: "_pb_users_auth_",
			MaxSelect:    1,
		})

		return app.Save(books)
	})
}
