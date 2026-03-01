package storage_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
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
			TypeID:      eveType.ID,
			CharacterID: c.ID,
		}
		// when
		err := st.CreateCharacterImplant(ctx, arg)
		// then
		require.NoError(t, err)
		x, err := st.GetCharacterImplant(ctx, c.ID, arg.TypeID)
		require.NoError(t, err)
		xassert.Equal(t, eveType, x.EveType)
	})
	t.Run("can replace implants", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		et := factory.CreateEveType()
		// when
		err := st.ReplaceCharacterImplants(ctx, c.ID, set.Of(et.ID))
		// then
		require.NoError(t, err)
		x, err := st.GetCharacterImplant(ctx, c.ID, et.ID)
		require.NoError(t, err)
		xassert.Equal(t, et, x.EveType)
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
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.CharacterImplant) int64 {
			return x.EveType.ID
		}))
		want := set.Of(x1.EveType.ID, x2.EveType.ID)
		xassert.Equal(t, want, got)
	})

	t.Run("can list implant IDs for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		x1 := factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		x2 := factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		// when
		got, err := st.ListCharacterImplantIDs(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(x1.EveType.ID, x2.EveType.ID)
		xassert.Equal(t, want, got)
	})

	t.Run("can list all implants", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x1 := factory.CreateCharacterImplant()
		x2 := factory.CreateCharacterImplant()
		// when
		oo, err := st.ListAllCharacterImplants(ctx)
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.CharacterImplant) int64 {
			return x.EveType.ID
		}))
		want := set.Of(x1.EveType.ID, x2.EveType.ID)
		xassert.Equal(t, want, got)
	})
}
