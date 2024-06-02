package character

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/antihax/goesi"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCharacterSectionIfChanged(t *testing.T) {
	db, r, factory := testutil.New()
	s := New(r, nil, nil, nil, nil, nil)
	ctx := context.Background()
	t.Run("should report as changed and run update when new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		token := factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		section := model.CharacterSectionImplants
		hasUpdated := false
		accessToken := ""
		arg := UpdateCharacterSectionParams{CharacterID: c.ID, Section: section}
		// when
		changed, err := s.updateCharacterSectionIfChanged(ctx, arg,
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
			x, err := r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.LastUpdatedAt, 5*time.Second)
				assert.True(t, x.IsOK())
			}
		}
	})
	t.Run("should report as changed and run update when data has changed and store update and reset error", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		section := model.CharacterSectionImplants
		x1 := factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     section,
			Error:       "error",
		})
		hasUpdated := false
		arg := UpdateCharacterSectionParams{CharacterID: c.ID, Section: section}
		// when
		changed, err := s.updateCharacterSectionIfChanged(ctx, arg,
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
			x2, err := r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.LastUpdatedAt, x1.LastUpdatedAt)
				assert.True(t, x2.IsOK())
			}
		}
	})
	t.Run("should report as unchanged and not run update when data has not changed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		section := model.CharacterSectionImplants
		x1 := factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     section,
			Data:        "old",
		})
		hasUpdated := false
		arg := UpdateCharacterSectionParams{CharacterID: c.ID, Section: section}
		// when
		changed, err := s.updateCharacterSectionIfChanged(ctx, arg,
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
			x2, err := r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.LastUpdatedAt, x1.LastUpdatedAt)
				assert.True(t, x2.IsOK())
			}
		}
	})
}

func TestCharacterSectionUpdateMethods(t *testing.T) {
	db, r, factory := testutil.New()
	s := New(r, nil, nil, nil, nil, nil)
	ctx := context.Background()
	t.Run("Can report when section update is expired", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		updateAt := time.Now().Add(-3 * time.Hour)
		factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID:   c.ID,
			Section:       model.CharacterSectionSkillqueue,
			LastUpdatedAt: updateAt,
		})
		// when
		x, err := s.characterSectionIsUpdateExpired(ctx, UpdateCharacterSectionParams{
			CharacterID: c.ID, Section: model.CharacterSectionSkillqueue,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}
	})
	t.Run("Can retrieve updated at for section", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		updateAt := time.Now().Add(3 * time.Hour)
		o := factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID:   c.ID,
			Section:       model.CharacterSectionSkillqueue,
			LastUpdatedAt: updateAt,
		})
		// when
		x, err := s.characterSectionUpdatedAt(ctx, UpdateCharacterSectionParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionSkillqueue,
		})
		// then
		if assert.NoError(t, err) {

			assert.Equal(t, o.LastUpdatedAt.UTC(), x.UTC())
		}
	})
}
