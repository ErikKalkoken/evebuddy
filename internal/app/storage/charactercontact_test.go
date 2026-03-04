package storage_test

import (
	"context"
	"maps"
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
		label := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		arg := storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			ContactID:   contact.ID,
			Standing:    -1.5,
			IsBlocked:   optional.New(true),
			IsWatched:   optional.New(false),
			LabelIDs:    []int64{label.LabelID},
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
		xassert.Equal(t, set.Of(label.Name), o.Labels)
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		label1 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		label2 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		o1 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{label1.LabelID},
		})
		arg := storage.UpdateOrCreateCharacterContactParams{
			CharacterID: o1.CharacterID,
			ContactID:   o1.Contact.ID,
			Standing:    -1.5,
			IsBlocked:   optional.New(true),
			IsWatched:   optional.New(false),
			LabelIDs:    []int64{label2.LabelID},
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
		xassert.Equal(t, set.Of(label2.Name), o2.Labels)
	})
	t.Run("should ignore missing label IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		contact := factory.CreateEveEntityCharacter()
		arg := storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			ContactID:   contact.ID,
			Standing:    -1.5,
			LabelIDs:    []int64{1},
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
		xassert.Equal(t, 0, o.Labels.Size())
	})
	t.Run("can list contacts for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		label1 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		contact1 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{label1.LabelID},
		})
		contact2 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterContact()
		// when
		oo, err := st.ListCharacterContacts(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(contact1.Contact.ID, contact2.Contact.ID)
		oo2 := maps.Collect(xiter.MapSlice2(oo, func(x *app.CharacterContact) (int64, *app.CharacterContact) {
			return x.Contact.ID, x
		}))
		xassert.Equal(t, want, set.Collect(maps.Keys(oo2)))
		o := oo2[contact1.Contact.ID]
		got2 := o.Labels
		want2 := set.Of(label1.Name)
		xassert.Equal(t, want2, got2)
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
