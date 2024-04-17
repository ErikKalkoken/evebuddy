package storage_test

import (
	"context"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
	"example/evebuddy/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMailList(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		c := factory.CreateCharacter()
		l := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList})
		// when
		err := r.CreateMailList(ctx, c.ID, l.ID)
		// then
		assert.NoError(t, err)
	})
	t.Run("can fetch all mail lists for a character", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		c := factory.CreateCharacter()
		e1 := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList, Name: "alpha"})
		assert.NoError(t, r.CreateMailList(ctx, c.ID, e1.ID))
		e2 := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList, Name: "bravo"})
		assert.NoError(t, r.CreateMailList(ctx, c.ID, e2.ID))
		// when
		ll, err := r.ListMailListsOrdered(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ll, 2)
			o := ll[0]
			assert.Equal(t, o.Name, "alpha")
		}
	})
}
