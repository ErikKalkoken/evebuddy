package characterservice

import (
	"context"
	"testing"
	"time"

	"github.com/antihax/goesi"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

// TODO: Add tests for UpdateSectionIfNeeded()

func TestUpdateSectionIfChanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should report as changed and run update when new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		token := factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		section := app.SectionCharacterImplants
		var hasUpdated bool
		accessToken := ""
		arg := app.CharacterSectionUpdateParams{CharacterID: c.ID, Section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg,
			func(ctx context.Context, characterID int32) (any, error) {
				accessToken = ctx.Value(goesi.ContextAccessToken).(string)
				return "any", nil
			},
			func(ctx context.Context, characterID int32, data any) error {
				hasUpdated = true
				return nil
			})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			assert.Equal(t, accessToken, token.AccessToken)
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
		testutil.TruncateTables(db)
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
		arg := app.CharacterSectionUpdateParams{CharacterID: c.ID, Section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg,
			func(ctx context.Context, characterID int32) (any, error) {
				return "any", nil
			},
			func(ctx context.Context, characterID int32, data any) error {
				hasUpdated = true
				return nil
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
		testutil.TruncateTables(db)
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
		arg := app.CharacterSectionUpdateParams{CharacterID: c.ID, Section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg,
			func(ctx context.Context, characterID int32) (any, error) {
				return "old", nil
			},
			func(ctx context.Context, characterID int32, data any) error {
				hasUpdated = true
				return nil
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
	t.Run("should update when data has not changed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
		arg := app.CharacterSectionUpdateParams{CharacterID: c.ID, Section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg,
			func(ctx context.Context, characterID int32) (any, error) {
				return "old", nil
			},
			func(ctx context.Context, characterID int32, data any) error {
				hasUpdated = true
				return nil
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
	s := NewFake(st)
	ctx := context.Background()
	t.Run("report true when section has changed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAssets,
		})
		// when
		got, err := s.hasSectionChanged(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAssets,
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
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		got, err := s.hasSectionChanged(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAssets,
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
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		status := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAssets,
		})
		// when
		got, err := s.hasSectionChanged(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAssets,
		}, status.ContentHash,
		)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.False(t, got)
	})
}
