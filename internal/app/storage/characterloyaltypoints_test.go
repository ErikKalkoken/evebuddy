package storage_test

import (
	"context"
	"slices"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCharacterLoyaltyPointEntry(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		faction := factory.CreateEveEntity(app.EveEntity{
			Category: app.EveEntityFaction,
		})
		corporation := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
			FactionID: optional.New(faction.ID),
		})
		arg := storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID:   c.ID,
			CorporationID: corporation.ID,
			LoyaltyPoints: 10,
		}
		// when
		err := st.UpdateOrCreateCharacterLoyaltyPointEntry(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterLoyaltyPointEntry(ctx, c.ID, corporation.ID)
		require.NoError(t, err)
		xassert.Equal(t, corporation.ID, o.Corporation.ID)
		xassert.Equal(t, corporation.Name, o.Corporation.Name)
		xassert.Equal(t, 10, o.LoyaltyPoints)
		xassert.Equal(t, faction.ID, o.Faction.MustValue().ID)
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateCharacterLoyaltyPointEntry()
		arg := storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID:   o1.CharacterID,
			CorporationID: o1.Corporation.ID,
			LoyaltyPoints: 10_000,
		}
		// when
		err := st.UpdateOrCreateCharacterLoyaltyPointEntry(ctx, arg)
		// then
		require.NoError(t, err)
		o2, err := st.GetCharacterLoyaltyPointEntry(ctx, o1.CharacterID, o1.Corporation.ID)
		require.NoError(t, err)
		xassert.Equal(t, o1.CharacterID, o2.CharacterID)
		xassert.Equal(t, o1.Corporation.ID, o2.Corporation.ID)
		xassert.Equal(t, 10_000, o2.LoyaltyPoints)
	})
	t.Run("can list entries for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID: c.ID,
		})
		o2 := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterLoyaltyPointEntry()
		// when
		s, err := st.ListCharacterLoyaltyPointEntries(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(o1.Corporation.ID, o2.Corporation.ID)
		got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterLoyaltyPointEntry) int64 {
			return x.Corporation.ID
		}))
		xassert.Equal(t, want, got)
	})
	t.Run("can list entry IDs for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID: c.ID,
		})
		o2 := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterLoyaltyPointEntry()
		// when
		got, err := st.ListCharacterLoyaltyPointEntryIDs(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(o1.Corporation.ID, o2.Corporation.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("can list all entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{})
		o2 := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{})
		factory.CreateCharacterLoyaltyPointEntry()
		// when
		s, err := st.ListAllCharacterLoyaltyPointEntries(ctx)
		// then
		require.NoError(t, err)
		want := set.Of(o1.Corporation.ID, o2.Corporation.ID)
		got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterLoyaltyPointEntry) int64 {
			return x.Corporation.ID
		}))
		xassert.Equal(t, want, got)
	})
	t.Run("can delete entries for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID: c.ID,
		})
		o2 := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID: c.ID,
		})
		// when
		err := st.DeleteCharacterLoyaltyPointEntries(ctx, c.ID, set.Of(o2.Corporation.ID))
		// then
		require.NoError(t, err)
		want := set.Of(o1.Corporation.ID)
		got, err := st.ListCharacterLoyaltyPointEntryIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, want, got)
	})
}
