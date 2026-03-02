package characterservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateCharacterContactsESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create new entries from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c.ID,
		})
		contact := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contacts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"contact_id":   contact.ID,
				"contact_type": "character",
				"standing":     -1.5,
			}}),
		)

		// when
		changed, err := s.updateContactsESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterContacts,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCharacterContact(ctx, c.ID, contact.ID)
		require.NoError(t, err)
		xassert.Equal(t, -1.5, o.Standing)
	})
	t.Run("should update existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c.ID,
		})
		contact := factory.CreateEveEntityCharacter()
		factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
			ContactID:   contact.ID,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contacts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"contact_id":   contact.ID,
				"contact_type": "character",
				"standing":     -1.5,
				"is_blocked":   true,
				"is_watched":   true,
			}}),
		)

		// when
		changed, err := s.updateContactsESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterContacts,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCharacterContact(ctx, c.ID, contact.ID)
		require.NoError(t, err)
		xassert.Equal(t, -1.5, o.Standing)
		xassert.Equal(t, true, o.IsBlocked.MustValue())
		xassert.Equal(t, true, o.IsWatched.MustValue())
	})
	t.Run("should delete obsolete entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: c.ID,
		})
		contact := factory.CreateEveEntityCharacter()
		factory.CreateCharacterContact(storage.UpdateOrCreateCharacterContactParams{
			CharacterID: c.ID,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contacts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"contact_id":   contact.ID,
				"contact_type": "character",
				"standing":     -1.5,
			}}),
		)

		// when
		changed, err := s.updateContactsESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterContacts,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		ids, err := st.ListCharacterContactIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(contact.ID), ids)
	})
}
