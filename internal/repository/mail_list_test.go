package repository_test

import (
	"context"
	"example/evebuddy/internal/repository"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMailList(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		l := factory.CreateEveEntity(repository.EveEntity{Category: repository.EveEntityMailList})
		// when
		err := r.CreateMailList(ctx, c.ID, l.ID)
		// then
		assert.NoError(t, err)
	})
	t.Run("can fetch all mail lists for a character", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		e1 := factory.CreateEveEntity(repository.EveEntity{Category: repository.EveEntityMailList, Name: "alpha"})
		assert.NoError(t, r.CreateMailList(ctx, c.ID, e1.ID))
		e2 := factory.CreateEveEntity(repository.EveEntity{Category: repository.EveEntityMailList, Name: "bravo"})
		assert.NoError(t, r.CreateMailList(ctx, c.ID, e2.ID))
		// when
		ll, err := r.ListMailLists(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ll, 2)
			o := ll[0]
			assert.Equal(t, o.Name, "alpha")
		}
	})
}
