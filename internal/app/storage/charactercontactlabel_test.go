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
	t.Run("can delete labels", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		l2 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		// when
		err := st.DeleteCharacterContactLabels(ctx, c.ID, set.Of(l1.LabelID))
		// then
		require.NoError(t, err)
		got, err := st.ListCharacterContactLabelIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(l2.LabelID)
		xassert.Equal(t, want, got)
	})
}
