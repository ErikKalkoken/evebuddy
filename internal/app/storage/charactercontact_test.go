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
		xassert.EqualOptional(t, true, o.IsBlocked)
		xassert.EqualOptional(t, false, o.IsWatched)
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
		xassert.EqualOptional(t, true, o2.IsBlocked)
		xassert.EqualOptional(t, false, o2.IsWatched)
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
		label2 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		label3 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		contact1 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{label1.LabelID, label2.LabelID},
		})
		contact2 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{label3.LabelID},
		})
		contact3 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		other := factory.CreateCharacter()
		otherLabel := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: other.ID,
		})
		factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: other.ID,
			ContactID:   contact1.Contact.ID,
			LabelIDs:    []int64{otherLabel.LabelID},
		})
		// when
		oo, err := st.ListCharacterContacts(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(contact1.Contact.ID, contact2.Contact.ID, contact3.Contact.ID)
		oo2 := maps.Collect(xiter.MapSlice2(oo, func(x *app.CharacterContact) (int64, *app.CharacterContact) {
			return x.Contact.ID, x
		}))
		xassert.Equal(t, want, set.Collect(maps.Keys(oo2)))
		xassert.Equal(t, set.Of(label1.Name, label2.Name), oo2[contact1.Contact.ID].Labels)
		xassert.Equal(t, set.Of(label3.Name), oo2[contact2.Contact.ID].Labels)
		xassert.Equal(t, set.Of[string](), oo2[contact3.Contact.ID].Labels)
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
	t.Run("can create contacts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		contact := factory.CreateEveEntityCharacter()
		label := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
			Name:        "Alpha",
		})
		// when
		changed, err := st.ReplaceCharacterContacts(ctx, c.ID, []storage.ReplaceCharacterContactParams{{
			ContactID: contact.ID,
			IsBlocked: optional.New(true),
			IsWatched: optional.New(false),
			LabelIDs:  []int64{label.LabelID},
			Standing:  -1.5,
		}})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.GetCharacterContact(ctx, c.ID, contact.ID)
		require.NoError(t, err)
		xassert.Equal(t, optional.New(true), got.IsBlocked)
		xassert.Equal(t, optional.New(false), got.IsWatched)
		xassert.Equal(t, set.Of("Alpha"), got.Labels)
		xassert.Equal(t, -1.5, got.Standing)
	})
	t.Run("can update fields of existing contacts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		label := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
			Name:        "Alpha",
		})
		o := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{label.LabelID},
			Standing:    5,
		})
		// when
		changed, err := st.ReplaceCharacterContacts(ctx, c.ID, []storage.ReplaceCharacterContactParams{{
			ContactID: o.Contact.ID,
			IsBlocked: optional.New(true),
			IsWatched: optional.New(true),
			LabelIDs:  []int64{label.LabelID},
			Standing:  -10,
		}})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.GetCharacterContact(ctx, c.ID, o.Contact.ID)
		require.NoError(t, err)
		xassert.Equal(t, optional.New(true), got.IsBlocked)
		xassert.Equal(t, optional.New(true), got.IsWatched)
		xassert.Equal(t, set.Of("Alpha"), got.Labels)
		xassert.Equal(t, -10.0, got.Standing)
	})
	t.Run("can update labels of existing contacts", func(t *testing.T) {
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
		l3 := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
			Name:        "Charlie",
		})
		o := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{l1.LabelID, l2.LabelID},
			Standing:    5,
		})
		// when
		changed, err := st.ReplaceCharacterContacts(ctx, c.ID, []storage.ReplaceCharacterContactParams{{
			ContactID: o.Contact.ID,
			LabelIDs:  []int64{l2.LabelID, l3.LabelID},
			Standing:  5,
		}})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.GetCharacterContact(ctx, c.ID, o.Contact.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of("Bravo", "Charlie"), got.Labels)
	})
	t.Run("can delete obsolete contacts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		kept := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			Standing:    5,
		})
		factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		other := factory.CreateCharacterContact()
		// when
		changed, err := st.ReplaceCharacterContacts(ctx, c.ID, []storage.ReplaceCharacterContactParams{{
			ContactID: kept.Contact.ID,
			Standing:  5,
		}})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCharacterContactIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(kept.Contact.ID), got)
		got2, err := st.ListCharacterContactIDs(ctx, other.CharacterID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(other.Contact.ID), got2)
	})
	t.Run("should remove all contacts when replaced with empty slice", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		// when
		changed, err := st.ReplaceCharacterContacts(ctx, c.ID, nil)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCharacterContactIDs(ctx, c.ID)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
	t.Run("should remove contacts in chunks", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		old := st.MaxIDsPerQuery
		st.MaxIDsPerQuery = 1
		defer func() {
			st.MaxIDsPerQuery = old
		}()
		c := factory.CreateCharacter()
		factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		o3 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			Standing:    5,
		})
		// when
		changed, err := st.ReplaceCharacterContacts(ctx, c.ID, []storage.ReplaceCharacterContactParams{{
			ContactID: o3.Contact.ID,
			Standing:  5,
		}})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCharacterContactIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(o3.Contact.ID), got)
	})
	t.Run("should report no change when contacts are the same", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		label := factory.CreateCharacterContactLabel(storage.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: c.ID,
		})
		o := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			IsBlocked:   optional.New(false),
			LabelIDs:    []int64{label.LabelID},
			Standing:    5,
		})
		// when
		changed, err := st.ReplaceCharacterContacts(ctx, c.ID, []storage.ReplaceCharacterContactParams{{
			ContactID: o.Contact.ID,
			IsBlocked: optional.New(false),
			LabelIDs:  []int64{label.LabelID},
			Standing:  5,
		}})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
	})
	t.Run("should create placeholder for missing label", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			Standing:    5,
		})
		// when
		changed, err := st.ReplaceCharacterContacts(ctx, c.ID, []storage.ReplaceCharacterContactParams{{
			ContactID: o.Contact.ID,
			LabelIDs:  []int64{42},
			Standing:  5,
		}})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.GetCharacterContact(ctx, c.ID, o.Contact.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of("Label 42"), got.Labels)
		got2, err := st.GetCharacterContactLabel(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, "Label 42", got2.Name)
	})
	t.Run("should create placeholder only once for label missing on several contacts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		e1 := factory.CreateEveEntityCharacter()
		e2 := factory.CreateEveEntityCharacter()
		// when
		_, err := st.ReplaceCharacterContacts(ctx, c.ID, []storage.ReplaceCharacterContactParams{
			{ContactID: e1.ID, LabelIDs: []int64{42}},
			{ContactID: e2.ID, LabelIDs: []int64{42}},
		})
		// then
		require.NoError(t, err)
		got, err := st.ListCharacterContactLabelIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](42), got)
		for _, id := range []int64{e1.ID, e2.ID} {
			got2, err := st.GetCharacterContact(ctx, c.ID, id)
			require.NoError(t, err)
			xassert.Equal(t, set.Of("Label 42"), got2.Labels)
		}
	})
	t.Run("should keep contacts unchanged when replace fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			Standing:    5,
		})
		o2 := factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		const unknownID = 42 // has no eve entity
		// when
		_, err := st.ReplaceCharacterContacts(ctx, c.ID, []storage.ReplaceCharacterContactParams{
			{ContactID: o1.Contact.ID, Standing: -10},
			{ContactID: unknownID},
		})
		// then
		require.Error(t, err)
		got, err := st.ListCharacterContactIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(o1.Contact.ID, o2.Contact.ID), got)
		got2, err := st.GetCharacterContact(ctx, c.ID, o1.Contact.ID)
		require.NoError(t, err)
		xassert.Equal(t, 5.0, got2.Standing)
	})
	t.Run("should return error when replacing with invalid IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		e := factory.CreateEveEntityCharacter()
		cases := []struct {
			name        string
			characterID int64
			contacts    []storage.ReplaceCharacterContactParams
		}{
			{"zero character ID", 0, []storage.ReplaceCharacterContactParams{{ContactID: e.ID}}},
			{"zero contact ID", c.ID, []storage.ReplaceCharacterContactParams{{ContactID: e.ID}, {ContactID: 0}}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// when
				_, err := st.ReplaceCharacterContacts(ctx, tc.characterID, tc.contacts)
				// then
				assert.ErrorIs(t, err, app.ErrInvalid)
			})
		}
	})
}
