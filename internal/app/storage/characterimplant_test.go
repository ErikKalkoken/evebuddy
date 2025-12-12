package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCharacterImplant(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		eveType := factory.CreateEveType()
		arg := storage.CreateCharacterImplantParams{
			EveTypeID:   eveType.ID,
			CharacterID: c.ID,
		}
		// when
		err := st.CreateCharacterImplant(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCharacterImplant(ctx, c.ID, arg.EveTypeID)
			if assert.NoError(t, err) {
				assert.Equal(t, eveType, x.EveType)
			}
		}
	})
	t.Run("can replace implants", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		eveType := factory.CreateEveType()
		arg := storage.CreateCharacterImplantParams{
			EveTypeID:   eveType.ID,
			CharacterID: c.ID,
		}
		// when
		err := st.ReplaceCharacterImplants(ctx, c.ID, []storage.CreateCharacterImplantParams{arg})
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCharacterImplant(ctx, c.ID, arg.EveTypeID)
			if assert.NoError(t, err) {
				assert.Equal(t, eveType, x.EveType)
			}
		}
	})
	t.Run("can list implants for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		x1 := factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		x2 := factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		// when
		oo, err := st.ListCharacterImplants(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.Collect(xiter.MapSlice(oo, func(x *app.CharacterImplant) int32 {
				return x.EveType.ID
			}))
			want := set.Of(x1.EveType.ID, x2.EveType.ID)
			xassert.EqualSet(t, want, got)
		}
	})

	t.Run("can list all implants", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x1 := factory.CreateCharacterImplant()
		x2 := factory.CreateCharacterImplant()
		// when
		oo, err := st.ListAllCharacterImplants(ctx)
		// then
		if assert.NoError(t, err) {
			got := set.Collect(xiter.MapSlice(oo, func(x *app.CharacterImplant) int32 {
				return x.EveType.ID
			}))
			want := set.Of(x1.EveType.ID, x2.EveType.ID)
			xassert.EqualSet(t, want, got)
		}
	})
}
