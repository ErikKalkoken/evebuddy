package storage_test

import (
	"context"
	"maps"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCharacterContactLabel(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		// when
		err := st.UpdateOrCreateCharacterContactLabel(ctx, storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
			LabelID:     42,
			Name:        "Alpha",
		})
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterContactLabel(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, c.ID, got.CharacterID)
		xassert.Equal(t, 42, got.LabelID)
		xassert.Equal(t, "Alpha", got.Name)
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		// when
		err := st.UpdateOrCreateCharacterContactLabel(ctx, storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
			LabelID:     o1.LabelID,
			Name:        "Alpha",
		})
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterContactLabel(ctx, c.ID, o1.LabelID)
		require.NoError(t, err)
		xassert.Equal(t, "Alpha", got.Name)
	})
	t.Run("can list labels for a character contact", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		l2 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterContactLabel()
		// when
		oo, err := st.ListCharacterContactLabels(ctx, c.ID)
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.CharacterContactLabel) int64 {
			return x.LabelID
		}))
		want := set.Of(l1.LabelID, l2.LabelID)
		xassert.Equal(t, want, got)
	})
	t.Run("can list label IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		l2 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterContactLabel()
		// when
		got, err := st.ListCharacterContactLabelIDs(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(l1.LabelID, l2.LabelID)
		xassert.Equal(t, want, got)
	})
	t.Run("can replace labels", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		kept := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
			Name:        "Alpha",
		})
		renamed := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
			Name:        "Bravo",
		})
		factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		other := factory.CreateCharacterContactLabel()
		// when
		changed, err := st.ReplaceCharacterContactLabels(ctx, c.ID, map[int64]string{
			kept.LabelID:    "Alpha",
			renamed.LabelID: "Charlie",
			42:              "Delta",
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		oo, err := st.ListCharacterContactLabels(ctx, c.ID)
		require.NoError(t, err)
		got := maps.Collect(xiter.MapSlice2(oo, func(x *app.CharacterContactLabel) (int64, string) {
			return x.LabelID, x.Name
		}))
		want := map[int64]string{
			kept.LabelID:    "Alpha",
			renamed.LabelID: "Charlie",
			42:              "Delta",
		}
		xassert.Equal(t, want, got)
		got2, err := st.ListCharacterContactLabelIDs(ctx, other.CharacterID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(other.LabelID), got2)
	})
	t.Run("should report no change when labels are the same", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		l := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
			Name:        "Alpha",
		})
		// when
		changed, err := st.ReplaceCharacterContactLabels(ctx, c.ID, map[int64]string{
			l.LabelID: "Alpha",
		})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
	})
	t.Run("should remove all labels when replaced with empty map", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		// when
		changed, err := st.ReplaceCharacterContactLabels(ctx, c.ID, map[int64]string{})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCharacterContactLabelIDs(ctx, c.ID)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
	t.Run("should remove labels in chunks", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		old := st.MaxIDsPerQuery
		st.MaxIDsPerQuery = 1
		defer func() {
			st.MaxIDsPerQuery = old
		}()
		c := factory.CreateCharacter()
		factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		l3 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		// when
		changed, err := st.ReplaceCharacterContactLabels(ctx, c.ID, map[int64]string{
			l3.LabelID: l3.Name,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCharacterContactLabelIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(l3.LabelID), got)
	})
	t.Run("should remove removed labels from contacts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
			Name:        "Alpha",
		})
		l2 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
			Name:        "Bravo",
		})
		contact := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{l1.LabelID, l2.LabelID},
		})
		// when
		_, err := st.ReplaceCharacterContactLabels(ctx, c.ID, map[int64]string{
			l2.LabelID: "Bravo",
		})
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterContact(ctx, c.ID, contact.Contact.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of("Bravo"), got.Labels)
	})
	t.Run("should return error when replacing with invalid IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		cases := []struct {
			name        string
			characterID int64
			labels      map[int64]string
		}{
			{"zero character ID", 0, map[int64]string{1: "Alpha"}},
			{"zero label ID", c.ID, map[int64]string{1: "Alpha", 0: "Bravo"}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// when
				_, err := st.ReplaceCharacterContactLabels(ctx, tc.characterID, tc.labels)
				// then
				assert.ErrorIs(t, err, app.ErrInvalid)
			})
		}
	})
}
