package repository_test

import (
	"context"
	"database/sql"
	"example/evebuddy/internal/factory"
	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/repository"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveEntity(t *testing.T) {
	// setup
	db, q, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create EveEntity", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		factory.CreateEveEntity(repository.CreateEveEntityParams{
			ID:   42,
			Name: "Dummy",
		})
		// when
		e, err := q.GetEveEntity(ctx, 42)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, e.Name, "Dummy")
		}
	})
	t.Run("can return all existing IDs", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		e1 := factory.CreateEveEntity()
		e2 := factory.CreateEveEntity()
		// when
		r, err := q.ListEveEntityIDs(ctx)
		// then
		if assert.NoError(t, err) {
			gotIDs := set.NewFromSlice([]int64{e1.ID, e2.ID})
			wantIDs := set.NewFromSlice(r)
			assert.Equal(t, wantIDs, gotIDs)
		}
	})
	t.Run("should return all character names in order", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		factory.CreateEveEntity(repository.CreateEveEntityParams{Name: "Yalpha2", Category: repository.EveEntityCharacter})
		factory.CreateEveEntity(repository.CreateEveEntityParams{Name: "Xalpha1", Category: repository.EveEntityCharacter})
		factory.CreateEveEntity(repository.CreateEveEntityParams{Name: "charlie", Category: repository.EveEntityCharacter})
		factory.CreateEveEntity(repository.CreateEveEntityParams{Name: "other", Category: repository.EveEntityCharacter})
		// when
		ee, err := q.ListEveEntitiesByPartialName(ctx, "%ALPHA%")
		// then
		if assert.NoError(t, err) {
			var got []string
			for _, e := range ee {
				got = append(got, e.Name)
			}
			want := []string{"Xalpha1", "Yalpha2"}
			assert.Equal(t, want, got)
		}
	})
}

func setUpDB() (*sql.DB, *repository.Queries, factory.Factory) {
	db, err := repository.ConnectDB(":memory:", true)
	if err != nil {
		panic(err)
	}
	q := repository.New(db)
	factory := factory.New(q)
	return db, q, factory
}
