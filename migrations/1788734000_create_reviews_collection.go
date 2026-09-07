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

		collection := core.NewBaseCollection("reviews")

		collection.Fields.Add(&core.RelationField{
			Name:         "book",
			CollectionId: books.Id,
			MaxSelect:    1,
			Required:     true,
		})

		collection.Fields.Add(&core.RelationField{
			Name:         "reviewer",
			CollectionId: "_pb_users_auth_",
			MaxSelect:    1,
			Required:     true,
		})

		collection.Fields.Add(&core.TextField{
			Name:     "content",
			Required: true,
		})

		min := 1.0
		max := 5.0
		collection.Fields.Add(&core.NumberField{
			Name:     "rating",
			Required: true,
			Min:      &min,
			Max:      &max,
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("reviews")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
