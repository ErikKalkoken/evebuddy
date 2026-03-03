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

func TestCharacterContact(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		contact := factory.CreateEveEntityCharacter()
		arg := storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			ContactID:   contact.ID,
			Standing:    -1.5,
		}
		// when
		err := st.UpdateOrCreateCharacterContact(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterContact(ctx, c.ID, contact.ID)
		require.NoError(t, err)
		xassert.Equal(t, contact.ID, o.Contact.ID)
		xassert.Equal(t, contact.Name, o.Contact.Name)
		xassert.Equal(t, -1.5, o.Standing)
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		contact := factory.CreateEveEntityCharacter()
		arg := storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			ContactID:   contact.ID,
			Standing:    -1.5,
			IsBlocked:   optional.New(true),
			IsWatched:   optional.New(false),
		}
		// when
		err := st.UpdateOrCreateCharacterContact(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterContact(ctx, c.ID, contact.ID)
		require.NoError(t, err)
		xassert.Equal(t, contact.ID, o.Contact.ID)
		xassert.Equal(t, contact.Name, o.Contact.Name)
		xassert.Equal(t, -1.5, o.Standing)
		xassert.Equal(t, true, o.IsBlocked.MustValue())
		xassert.Equal(t, false, o.IsWatched.MustValue())
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateCharacterContact()
		arg := storage.UpdateOrCreateCharacterContactParams{
			CharacterID: o1.CharacterID,
			ContactID:   o1.Contact.ID,
			Standing:    -1.5,
			IsBlocked:   optional.New(true),
			IsWatched:   optional.New(false),
		}
		// when
		err := st.UpdateOrCreateCharacterContact(ctx, arg)
		// then
		require.NoError(t, err)
		o2, err := st.GetCharacterContact(ctx, o1.CharacterID, o1.Contact.ID)
		require.NoError(t, err)
		xassert.Equal(t, o1.CharacterID, o2.CharacterID)
		xassert.Equal(t, o1.Contact.ID, o2.Contact.ID)
		xassert.Equal(t, -1.5, o2.Standing)
		xassert.Equal(t, true, o2.IsBlocked.MustValue())
		xassert.Equal(t, false, o2.IsWatched.MustValue())
	})
	t.Run("can list entries for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		o2 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterContact()
		// when
		s, err := st.ListCharacterContacts(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(o1.Contact.ID, o2.Contact.ID)
		got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterContact) int64 {
			return x.Contact.ID
		}))
		xassert.Equal(t, want, got)
	})
	t.Run("can list entry IDs for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		o2 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterContact()
		// when
		got, err := st.ListCharacterContactIDs(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(o1.Contact.ID, o2.Contact.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("can delete entries for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		o2 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		// when
		err := st.DeleteCharacterContacts(ctx, c.ID, set.Of(o2.Contact.ID))
		// then
		require.NoError(t, err)
		want := set.Of(o1.Contact.ID)
		got, err := st.ListCharacterContactIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, want, got)
	})
}

func TestCharacterContactLabel(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		// when
		err := st.CreateCharacterContactLabel(ctx, storage.CreateCharacterContactLabelParams{
			CharacterID: c.ID,
			LabelID:     42,
			Name:        "Alpha",
		})
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterContactLabel(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, "Alpha", got)
	})
	t.Run("can list labels", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateCharacterContactLabel()
		l2 := factory.CreateCharacterContactLabel()
		// when
		got, err := st.ListCharacterContactLabels(ctx, c.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, set.Of(l1, l2), got)
	})
	t.Run("can delete labels", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateCharacterContactLabel()
		l2 := factory.CreateCharacterContactLabel()
		// when
		err := st.DeleteCharacterContactLabels(ctx, c.ID, set.Of(l1))
		// then
		require.NoError(t, err)
		got, err := st.ListCharacterContactLabels(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(l2), got)
	})
}
