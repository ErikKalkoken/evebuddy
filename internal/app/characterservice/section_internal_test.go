package characterservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fnt-eve/goesi-openapi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

// TODO: Add tests for UpdateSectionIfNeeded()

func TestUpdateSectionIfChanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("should report as changed and run update when new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		token := factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		section := app.SectionCharacterImplants
		var hasUpdated bool
		var tokenSource oauth2.TokenSource
		arg := characterSectionUpdateParams{characterID: c.ID, section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg, false,
			func(ctx context.Context, characterID int64) (any, error) {
				tokenSource = ctx.Value(goesi.ContextOAuth2).(oauth2.TokenSource)
				return "any", nil
			},
			func(ctx context.Context, characterID int64, data any) (bool, error) {
				hasUpdated = true
				return true, nil
			})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			tok, err := tokenSource.Token()
			require.NoError(t, err)
			xassert.Equal(t, tok.AccessToken, token.AccessToken)
			assert.True(t, hasUpdated)
			x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.CompletedAt, 5*time.Second)
				assert.False(t, x.HasError())
			}
		}
	})
	t.Run("should report as changed and run update when data has changed and store update and reset error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		section := app.SectionCharacterImplants
		x1 := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      section,
			ErrorMessage: "error",
			CompletedAt:  time.Now().Add(-5 * time.Second),
		})
		var hasUpdated bool
		arg := characterSectionUpdateParams{characterID: c.ID, section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg, false,
			func(ctx context.Context, characterID int64) (any, error) {
				return "any", nil
			},
			func(ctx context.Context, characterID int64, data any) (bool, error) {
				hasUpdated = true
				return true, nil
			})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			assert.True(t, hasUpdated)
			x2, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.CompletedAt, x1.CompletedAt)
				assert.False(t, x2.HasError())
			}
		}
	})
	t.Run("should report as unchanged and not run update when data has not changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		section := app.SectionCharacterImplants
		x1 := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
			Data:        "old",
			CompletedAt: time.Now().Add(-5 * time.Second),
		})
		hasUpdated := false
		arg := characterSectionUpdateParams{characterID: c.ID, section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg, false,
			func(ctx context.Context, characterID int64) (any, error) {
				return "old", nil
			},
			func(ctx context.Context, characterID int64, data any) (bool, error) {
				hasUpdated = true
				return true, nil
			})
		// then
		if assert.NoError(t, err) {
			assert.False(t, changed)
			assert.False(t, hasUpdated)
			x2, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.CompletedAt, x1.CompletedAt)
				assert.False(t, x2.HasError())
			}
		}
	})
	t.Run("should update when data has not changed and forced", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		section := app.SectionCharacterIndustryJobs
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
			Data:        "old",
			CompletedAt: time.Now().Add(-5 * time.Second),
		})
		var hasUpdated bool
		arg := characterSectionUpdateParams{
			characterID: c.ID,
			section:     section,
			forceUpdate: true,
		}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg, false,
			func(ctx context.Context, characterID int64) (any, error) {
				return "old", nil
			},
			func(ctx context.Context, characterID int64, data any) (bool, error) {
				hasUpdated = true
				return true, nil
			})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			assert.True(t, hasUpdated)
		}
	})
}

func TestHasSectionChanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("report true when section has changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAssets,
		})
		// when
		got, err := s.hasSectionChanged(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterAssets,
		}, "changed",
		)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, got)
	})
	t.Run("report true when section does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		// when
		got, err := s.hasSectionChanged(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterAssets,
		}, "changed",
		)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, got)
	})
	t.Run("report false when section has not changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		status := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAssets,
		})
		// when
		got, err := s.hasSectionChanged(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterAssets,
		}, status.ContentHash,
		)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.False(t, got)
	})
}

func TestCharacterService_UpdateSectionIfNeeded(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
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
		changed, err := s.UpdateSectionIfNeeded(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     section,
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
		changed, err := s.UpdateSectionIfNeeded(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     section,
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
		changed, err := s.UpdateSectionIfNeeded(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     section,
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
		_, err := s.UpdateSectionIfNeeded(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     section,
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
		_, err := s.UpdateSectionIfNeeded(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     section,
			forceUpdate: true,
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
		_, err := s.UpdateSectionIfNeeded(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     section,
			forceUpdate: true,
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
		_, err := s.UpdateSectionIfNeeded(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     section,
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
		changed, err := s.UpdateSectionIfNeeded(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     section,
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
