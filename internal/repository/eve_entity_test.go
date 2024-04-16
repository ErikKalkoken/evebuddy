package repository_test

import (
	"context"
	"example/evebuddy/internal/repository"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveEntity(t *testing.T) {
	db, r, _ := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return error when no object found", func(t *testing.T) {
		_, err := r.GetEveEntityByNameAndCategory(ctx, "dummy", repository.EveEntityAlliance)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}

// func TestEveEntity(t *testing.T) {
// 	// setup
// 	db, q, factory := setUpDB()
// 	defer db.Close()
// 	ctx := context.Background()
// 	t.Run("can create EveEntity", func(t *testing.T) {
// 		// given
// 		sqlc.TruncateTables(db)
// 		factory.CreateEveEntity(sqlc.CreateEveEntityParams{
// 			ID:   42,
// 			Name: "Dummy",
// 		})
// 		// when
// 		e, err := q.GetEveEntity(ctx, 42)
// 		// then
// 		if assert.NoError(t, err) {
// 			assert.Equal(t, e.Name, "Dummy")
// 		}
// 	})
// 	t.Run("can return all existing IDs", func(t *testing.T) {
// 		// given
// 		sqlc.TruncateTables(db)
// 		e1 := factory.CreateEveEntity()
// 		e2 := factory.CreateEveEntity()
// 		// when
// 		r, err := q.ListEveEntityIDs(ctx)
// 		// then
// 		if assert.NoError(t, err) {
// 			gotIDs := set.NewFromSlice([]int64{e1.ID, e2.ID})
// 			wantIDs := set.NewFromSlice(r)
// 			assert.Equal(t, wantIDs, gotIDs)
// 		}
// 	})
// 	t.Run("should return all character names in order", func(t *testing.T) {
// 		// given
// 		sqlc.TruncateTables(db)
// 		factory.CreateEveEntity(sqlc.CreateEveEntityParams{Name: "Yalpha2", Category: sqlc.EveEntityCharacter})
// 		factory.CreateEveEntity(sqlc.CreateEveEntityParams{Name: "Xalpha1", Category: sqlc.EveEntityCharacter})
// 		factory.CreateEveEntity(sqlc.CreateEveEntityParams{Name: "charlie", Category: sqlc.EveEntityCharacter})
// 		factory.CreateEveEntity(sqlc.CreateEveEntityParams{Name: "other", Category: sqlc.EveEntityCharacter})
// 		// when
// 		ee, err := q.ListEveEntitiesByPartialName(ctx, "%ALPHA%")
// 		// then
// 		if assert.NoError(t, err) {
// 			var got []string
// 			for _, e := range ee {
// 				got = append(got, e.Name)
// 			}
// 			want := []string{"Xalpha1", "Yalpha2"}
// 			assert.Equal(t, want, got)
// 		}
// 	})
// }

// func setUpDB() (*sql.DB, *sqlc.Queries, factory.Factory) {
// 	db, err := sqlc.ConnectDB(":memory:", true)
// 	if err != nil {
// 		panic(err)
// 	}
// 	q := sqlc.New(db)
// 	factory := factory.New(q)
// 	return db, q, factory
// }
