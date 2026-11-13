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
	t.Run("should report no change when entries are unchanged", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		corporation := factory.CreateEveCorporation()
		factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID:   c.ID,
			CorporationID: corporation.ID,
			LoyaltyPoints: 10_000,
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
		assert.False(t, changed)
		o, err := st.GetCharacterLoyaltyPointEntry(ctx, c.ID, corporation.ID)
		require.NoError(t, err)
		xassert.Equal(t, 10_000, o.LoyaltyPoints)
	})
	t.Run("should create missing corporation from ESI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		corporationID := factory.CreateEveEntityCorporation().ID // without eve corporation
		ceo := factory.CreateEveEntityCharacter()
		station := factory.CreateEveEntityWithCategory(app.EveEntityStation)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/loyalty/points", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"corporation_id": corporationID,
				"loyalty_points": 10_000,
			}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d", corporationID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"ceo_id":          ceo.ID,
				"creator_id":      ceo.ID,
				"date_founded":    "2020-01-01T00:00:00Z",
				"description":     "",
				"home_station_id": station.ID,
				"member_count":    42,
				"name":            "Alpha Corp",
				"shares":          1000,
				"ticker":          "ALPHA",
				"url":             "",
				"war_eligible":    false,
				"tax_rates": map[string]float64{
					"isk":           0.1,
					"loyalty_point": 0,
				},
				"state":         "active",
				"friendly_fire": "legal",
				"type":          "player_owned",
			}),
		)

		// when
		changed, err := s.updateLoyaltyPointEntriesESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterLoyaltyPoints,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCharacterLoyaltyPointEntry(ctx, c.ID, corporationID)
		require.NoError(t, err)
		xassert.Equal(t, 10_000, o.LoyaltyPoints)
		xassert.Equal(t, "Alpha Corp", o.Corporation.Name)
	})
	t.Run("should skip entries when their corporation can not be fetched", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		corporation := factory.CreateEveCorporation()
		failingID := factory.CreateEveEntityCorporation().ID // without eve corporation
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/loyalty/points", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"corporation_id": corporation.ID,
				"loyalty_points": 10_000,
			}, {
				"corporation_id": failingID,
				"loyalty_points": 20_000,
			}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d", failingID),
			httpmock.NewJsonResponderOrPanic(404, map[string]any{"error": "not found"}),
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

func TestListLoyaltyPointEntries(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can list loyalty point entries for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		e := factory.CreateCharacterLoyaltyPointEntry(storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{CharacterID: c.ID})
		factory.CreateCharacterLoyaltyPointEntry() // entry for another character
		// when
		got, err := s.ListLoyaltyPointEntries(ctx, c.ID)
		// then
		require.NoError(t, err)
		if assert.Len(t, got, 1) {
			assert.Equal(t, e.Corporation.ID, got[0].Corporation.ID)
		}
	})
}

func TestListAllLoyaltyPointEntries(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can list loyalty point entries across all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateCharacterLoyaltyPointEntry()
		factory.CreateCharacterLoyaltyPointEntry()
		// when
		got, err := s.ListAllLoyaltyPointEntries(ctx)
		// then
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
}
