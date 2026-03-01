package characterservice_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCharacterService_HasSection(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := characterservice.NewFake(st)
	section := app.SectionCharacterImplants
	ctx := context.Background()
	t.Run("should report true when exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		// when
		got, err := s.HasSection(ctx, c.ID, section)
		// then
		require.NoError(t, err)
		assert.True(t, got)
	})
	t.Run("should report false when not exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		// when
		got, err := s.HasSection(ctx, c.ID, section)
		// then
		require.NoError(t, err)
		assert.False(t, got)
	})
	t.Run("should report false when exist, but incomplete", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		_, err := st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
			CompletedAt: &sql.NullTime{},
		})
		require.NoError(t, err)
		// when
		got, err := s.HasSection(ctx, c.ID, section)
		// then
		require.NoError(t, err)
		assert.False(t, got)
	})
}

func TestCharacterService_UpdateSectionIfNeeded(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := characterservice.NewFake(st)
	ctx := context.Background()
	const section = app.SectionCharacterAssets
	t.Run("should report true when changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		et := factory.CreateEveType()
		es := factory.CreateEveLocationStation()
		data := []map[string]any{{
			"is_blueprint_copy": true,
			"is_singleton":      true,
			"item_id":           1000000016835,
			"location_flag":     "Hangar",
			"location_id":       es.ID,
			"location_type":     "station",
			"quantity":          1,
			"type_id":           et.ID,
		}}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     section,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
		require.NoError(t, err)
		assert.False(t, x.HasError())
	})

	t.Run("should not update and report false when not changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		et := factory.CreateEveType()
		es := factory.CreateEveLocationStation()
		data := []map[string]any{{
			"is_blueprint_copy": true,
			"is_singleton":      true,
			"item_id":           1000000016835,
			"location_flag":     "Hangar",
			"location_id":       es.ID,
			"location_type":     "station",
			"quantity":          1,
			"type_id":           et.ID,
		}}
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
			CompletedAt: time.Now().Add(-6 * time.Hour),
			Data:        data,
		})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		// when
		changed, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     section,
		})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now(), x.CompletedAt, 5*time.Second)

		xassert.Equal(t, 1, httpmock.GetTotalCallCount())
		ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0, ids.Size())
	})
	t.Run("should not fetch or update when not expired and report false", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		et := factory.CreateEveType()
		es := factory.CreateEveLocationStation()
		data := []map[string]any{{
			"is_blueprint_copy": true,
			"is_singleton":      true,
			"item_id":           1000000016835,
			"location_flag":     "Hangar",
			"location_id":       es.ID,
			"location_type":     "station",
			"quantity":          1,
			"type_id":           et.ID,
		}}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		// when
		changed, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     section,
		})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		xassert.Equal(t, 0, httpmock.GetTotalCallCount())
		ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0, ids.Size())
	})
	t.Run("should record when update failed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets", c.ID),
			httpmock.NewJsonResponderOrPanic(500, map[string]string{"error": "dummy error"}))
		// when
		_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     section,
		})
		// then
		require.Error(t, err)
		x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
		require.NoError(t, err)
		assert.True(t, x.HasError())
		xassert.Equal(t, "500 Internal Server Error", x.ErrorMessage)
	})
	t.Run("should fetch and update when not expired and force update requested", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		et := factory.CreateEveType()
		es := factory.CreateEveLocationStation()
		data := []map[string]any{{
			"is_blueprint_copy": true,
			"is_singleton":      true,
			"item_id":           1000000016835,
			"location_flag":     "Hangar",
			"location_id":       es.ID,
			"location_type":     "station",
			"quantity":          1,
			"type_id":           et.ID,
		}}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		// when
		_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     section,
			ForceUpdate: true,
		})
		// then
		require.NoError(t, err)
		xassert.Equal(t, 1, httpmock.GetTotalCallCount())
		ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 1, ids.Size())
	})
	t.Run("should update when not changed and force update requested", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		et := factory.CreateEveType()
		es := factory.CreateEveLocationStation()
		data := []map[string]any{{
			"is_blueprint_copy": true,
			"is_singleton":      true,
			"item_id":           1000000016835,
			"location_flag":     "Hangar",
			"location_id":       es.ID,
			"location_type":     "station",
			"quantity":          1,
			"type_id":           et.ID,
		}}
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
			CompletedAt: time.Now().Add(-6 * time.Hour),
			Data:        data,
		})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     section,
			ForceUpdate: true,
		})
		// then
		require.NoError(t, err)
		x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now(), x.CompletedAt, 5*time.Second)
		xassert.Equal(t, 1, httpmock.GetTotalCallCount())
		ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 1, ids.Size())
	})
	t.Run("should update when last update failed and error has timed out", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		et := factory.CreateEveType()
		es := factory.CreateEveLocationStation()
		data := []map[string]any{{
			"is_blueprint_copy": true,
			"is_singleton":      true,
			"item_id":           1000000016835,
			"location_flag":     "Hangar",
			"location_id":       es.ID,
			"location_type":     "station",
			"quantity":          1,
			"type_id":           et.ID,
		}}
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      section,
			CompletedAt:  time.Now().Add(-6 * time.Hour),
			Data:         "old",
			ErrorMessage: "error",
			UpdatedAt:    time.Now().UTC().Add(-1 * time.Hour),
		})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     section,
		})
		// then
		require.NoError(t, err)
		x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now(), x.CompletedAt, 5*time.Second)
		xassert.Equal(t, 1, httpmock.GetTotalCallCount())
		ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 1, ids.Size())
	})
	t.Run("should not update when last update failed but below error timeout", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		et := factory.CreateEveType()
		es := factory.CreateEveLocationStation()
		data := []map[string]any{{
			"is_blueprint_copy": true,
			"is_singleton":      true,
			"item_id":           1000000016835,
			"location_flag":     "Hangar",
			"location_id":       es.ID,
			"location_type":     "station",
			"quantity":          1,
			"type_id":           et.ID,
		}}
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      section,
			CompletedAt:  time.Now().Add(-6 * time.Hour),
			Data:         "old",
			ErrorMessage: "error",
			UpdatedAt:    time.Now().UTC(),
		})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/assets", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     section,
		})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		xassert.Equal(t, 0, httpmock.GetTotalCallCount())
		ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0, ids.Size())
	})
}
