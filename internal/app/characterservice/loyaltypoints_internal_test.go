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

func TestUpdateCharacterLoyaltyPointEntriesESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("should create new entries from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		corporation := factory.CreateEveCorporation()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/loyalty/points", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"corporation_id": corporation.ID,
				"loyalty_points": 10_000,
			}}),
		)

		// when
		changed, err := s.updateLoyaltyPointEntriesESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterLoyaltyPoints,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCharacterLoyaltyPointEntry(ctx, c.ID, corporation.ID)
		require.NoError(t, err)
		xassert.Equal(t, 10_000, o.LoyaltyPoints)
	})
	t.Run("should update existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		corporation := factory.CreateEveCorporation()
		factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID:   c.ID,
			CorporationID: corporation.ID,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/loyalty/points", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"corporation_id": corporation.ID,
				"loyalty_points": 10_000,
			}}),
		)

		// when
		changed, err := s.updateLoyaltyPointEntriesESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterLoyaltyPoints,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCharacterLoyaltyPointEntry(ctx, c.ID, corporation.ID)
		require.NoError(t, err)
		xassert.Equal(t, 10_000, o.LoyaltyPoints)
	})
	t.Run("should delete obsolete entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		corporation := factory.CreateEveCorporation()
		factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID: c.ID,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/loyalty/points", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"corporation_id": corporation.ID,
				"loyalty_points": 10_000,
			}}),
		)

		// when
		changed, err := s.updateLoyaltyPointEntriesESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterLoyaltyPoints,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		ids, err := st.ListCharacterLoyaltyPointEntryIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(corporation.ID), ids)
	})
}
