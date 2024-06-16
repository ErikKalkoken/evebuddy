package character

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/antihax/goesi"
	"github.com/stretchr/testify/assert"
)

// TODO: Add tests for UpdateSectionIfNeeded()

func TestUpdateCharacterSectionIfChanged(t *testing.T) {
	db, r, factory := testutil.New()
	s := New(r, nil, nil, nil, nil, nil)
	ctx := context.Background()
	t.Run("should report as changed and run update when new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		token := factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		section := app.SectionImplants
		hasUpdated := false
		accessToken := ""
		arg := UpdateSectionParams{CharacterID: c.ID, Section: section}
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
			x, err := r.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.CompletedAt, 5*time.Second)
				assert.True(t, x.IsOK())
			}
		}
	})
	t.Run("should report as changed and run update when data has changed and store update and reset error", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		section := app.SectionImplants
		x1 := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      section,
			ErrorMessage: "error",
		})
		hasUpdated := false
		arg := UpdateSectionParams{CharacterID: c.ID, Section: section}
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
			x2, err := r.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.CompletedAt, x1.CompletedAt)
				assert.True(t, x2.IsOK())
			}
		}
	})
	t.Run("should report as unchanged and not run update when data has not changed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		section := app.SectionImplants
		x1 := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
			Data:        "old",
		})
		hasUpdated := false
		arg := UpdateSectionParams{CharacterID: c.ID, Section: section}
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
			x2, err := r.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.CompletedAt, x1.CompletedAt)
				assert.True(t, x2.IsOK())
			}
		}
	})
}

func TestCharacterSectionUpdateMethods(t *testing.T) {
	db, r, factory := testutil.New()
	s := New(r, nil, nil, nil, nil, nil)
	ctx := context.Background()
	t.Run("Can report wether a section was updated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionSkillqueue,
			CompletedAt: time.Now(),
		})
		// when
		x, err := s.SectionWasUpdated(ctx, c.ID, app.SectionSkillqueue)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, true, x)
		}
	})
	t.Run("Can report wether a section was updated 2", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		x, err := s.SectionWasUpdated(ctx, c.ID, app.SectionSkillqueue)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, false, x)
		}
	})
}
