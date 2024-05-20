package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestMailList(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		l := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList})
		// when
		err := r.CreateCharacterMailList(ctx, c.ID, l.ID)
		// then
		assert.NoError(t, err)
	})
	t.Run("can fetch all mail lists for a character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		e1 := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList, Name: "alpha"})
		assert.NoError(t, r.CreateCharacterMailList(ctx, c.ID, e1.ID))
		e2 := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList, Name: "bravo"})
		assert.NoError(t, r.CreateCharacterMailList(ctx, c.ID, e2.ID))
		// when
		ll, err := r.ListCharacterMailListsOrdered(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ll, 2)
			o := ll[0]
			assert.Equal(t, o.Name, "alpha")
		}
	})
	t.Run("can delete obsolete mail lists for a character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		e1 := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList})
		mustNotFail(r.CreateCharacterMailList(ctx, c1.ID, e1.ID))
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c1.ID, RecipientIDs: []int32{e1.ID}})
		e2 := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList})
		mustNotFail(r.CreateCharacterMailList(ctx, c1.ID, e2.ID))
		c2 := factory.CreateCharacter()
		e3 := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList})
		mustNotFail(r.CreateCharacterMailList(ctx, c2.ID, e3.ID))
		// when
		err := r.DeleteObsoleteCharacterMailLists(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			lists, err := r.ListCharacterMailListsOrdered(ctx, c1.ID)
			if assert.NoError(t, err) {
				got := set.New[int32]()
				for _, l := range lists {
					got.Add(l.ID)
				}
				want := set.NewFromSlice([]int32{e1.ID})
				assert.Equal(t, want, got)
			}
			lists, err = r.ListCharacterMailListsOrdered(ctx, c2.ID)
			if assert.NoError(t, err) {
				got := set.New[int32]()
				for _, l := range lists {
					got.Add(l.ID)
				}
				want := set.NewFromSlice([]int32{e3.ID})
				assert.Equal(t, want, got)
			}
		}
	})
}

func mustNotFail(err error) {
	if err != nil {
		panic(err)
	}
}
