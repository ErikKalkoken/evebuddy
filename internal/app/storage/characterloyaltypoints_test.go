package storage_test

import (
	"context"
	"slices"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
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
		xassert.EqualOptional(t, faction, o.Faction)
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
		o1 := factory.CreateCharacterLoyaltyPointEntry()
		o2 := factory.CreateCharacterLoyaltyPointEntry()
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
}

func TestReplaceCharacterLoyaltyPointEntries(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	loyaltyPoints := func(t *testing.T, characterID int64) map[int64]int64 {
		t.Helper()
		oo, err := st.ListCharacterLoyaltyPointEntries(ctx, characterID)
		require.NoError(t, err)
		m := make(map[int64]int64)
		for _, o := range oo {
			m[o.Corporation.ID] = o.LoyaltyPoints
		}
		return m
	}
	t.Run("can replace entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		kept := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID:   c.ID,
			LoyaltyPoints: 100,
		})
		updated := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID:   c.ID,
			LoyaltyPoints: 200,
		})
		factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID: c.ID,
		})
		other := factory.CreateCharacterLoyaltyPointEntry()
		added := factory.CreateEveCorporation()
		// when
		changed, err := st.ReplaceCharacterLoyaltyPointEntries(ctx, c.ID, map[int64]int64{
			kept.Corporation.ID:    100,
			updated.Corporation.ID: 250,
			added.ID:               300,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		want := map[int64]int64{
			kept.Corporation.ID:    100,
			updated.Corporation.ID: 250,
			added.ID:               300,
		}
		xassert.Equal(t, want, loyaltyPoints(t, c.ID))
		want2 := map[int64]int64{other.Corporation.ID: other.LoyaltyPoints}
		xassert.Equal(t, want2, loyaltyPoints(t, other.CharacterID))
	})
	t.Run("should report no change when entries are the same", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID:   c.ID,
			LoyaltyPoints: 100,
		})
		// when
		changed, err := st.ReplaceCharacterLoyaltyPointEntries(ctx, c.ID, map[int64]int64{
			o.Corporation.ID: 100,
		})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		want := map[int64]int64{o.Corporation.ID: 100}
		xassert.Equal(t, want, loyaltyPoints(t, c.ID))
	})
	t.Run("should remove all entries when replaced with empty map", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID: c.ID,
		})
		// when
		changed, err := st.ReplaceCharacterLoyaltyPointEntries(ctx, c.ID, map[int64]int64{})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Empty(t, loyaltyPoints(t, c.ID))
	})
	t.Run("should remove entries in chunks", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		old := st.MaxIDsPerQuery
		st.MaxIDsPerQuery = 1
		defer func() {
			st.MaxIDsPerQuery = old
		}()
		c := factory.CreateCharacter()
		factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID: c.ID,
		})
		o3 := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID:   c.ID,
			LoyaltyPoints: 100,
		})
		// when
		changed, err := st.ReplaceCharacterLoyaltyPointEntries(ctx, c.ID, map[int64]int64{
			o3.Corporation.ID: 100,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		want := map[int64]int64{o3.Corporation.ID: 100}
		xassert.Equal(t, want, loyaltyPoints(t, c.ID))
	})
	t.Run("should keep entries unchanged when replace fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID:   c.ID,
			LoyaltyPoints: 100,
		})
		o2 := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID:   c.ID,
			LoyaltyPoints: 200,
		})
		const unknownID = 42 // has no eve corporation
		// when
		_, err := st.ReplaceCharacterLoyaltyPointEntries(ctx, c.ID, map[int64]int64{
			o1.Corporation.ID: 100,
			unknownID:         300,
		})
		// then
		require.Error(t, err)
		want := map[int64]int64{
			o1.Corporation.ID: 100,
			o2.Corporation.ID: 200,
		}
		xassert.Equal(t, want, loyaltyPoints(t, c.ID))
	})
	t.Run("should return error when replacing with invalid IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		corporation := factory.CreateEveCorporation()
		cases := []struct {
			name        string
			characterID int64
			points      map[int64]int64
		}{
			{"zero character ID", 0, map[int64]int64{corporation.ID: 100}},
			{"zero corporation ID", c.ID, map[int64]int64{corporation.ID: 100, 0: 200}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// when
				_, err := st.ReplaceCharacterLoyaltyPointEntries(ctx, tc.characterID, tc.points)
				// then
				assert.ErrorIs(t, err, app.ErrInvalid)
			})
		}
	})
}
