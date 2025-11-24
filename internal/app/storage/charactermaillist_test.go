package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestMailList(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		l := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList})
		// when
		err := st.CreateCharacterMailList(ctx, c.ID, l.ID)
		// then
		assert.NoError(t, err)
	})
	t.Run("can fetch all mail lists for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		e1 := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList, Name: "alpha"})
		assert.NoError(t, st.CreateCharacterMailList(ctx, c.ID, e1.ID))
		e2 := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList, Name: "bravo"})
		assert.NoError(t, st.CreateCharacterMailList(ctx, c.ID, e2.ID))
		// when
		ll, err := st.ListCharacterMailListsOrdered(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ll, 2)
			o := ll[0]
			assert.Equal(t, o.Name, "alpha")
		}
	})
	t.Run("can delete obsolete mail lists for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacterFull()
		e1 := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList})
		if err := st.CreateCharacterMailList(ctx, c1.ID, e1.ID); err != nil {
			t.Fatal(err)
		}
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c1.ID, RecipientIDs: []int32{e1.ID}})
		e2 := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList})
		if err := st.CreateCharacterMailList(ctx, c1.ID, e2.ID); err != nil {
			t.Fatal(err)
		}
		c2 := factory.CreateCharacterFull()
		e3 := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList})
		if err := st.CreateCharacterMailList(ctx, c2.ID, e3.ID); err != nil {
			t.Fatal(err)
		}
		// when
		err := st.DeleteObsoleteCharacterMailLists(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			lists, err := st.ListCharacterMailListsOrdered(ctx, c1.ID)
			if assert.NoError(t, err) {
				got := set.Of[int32]()
				for _, l := range lists {
					got.Add(l.ID)
				}
				want := set.Of([]int32{e1.ID}...)
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
			lists, err = st.ListCharacterMailListsOrdered(ctx, c2.ID)
			if assert.NoError(t, err) {
				got := set.Of[int32]()
				for _, l := range lists {
					got.Add(l.ID)
				}
				want := set.Of(e3.ID)
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
}
