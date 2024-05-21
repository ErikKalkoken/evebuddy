package service

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestHasCharacterSectionChanged(t *testing.T) {
	db, r, factory := testutil.New()
	s := NewService(r)
	ctx := context.Background()
	t.Run("should report as changed when new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		section := model.CharacterSectionImplants
		// when
		changed, err := s.recordCharacterSectionUpdate(ctx, c.ID, section, "new")
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.LastUpdatedAt.Time, 5*time.Second)
				assert.True(t, x.IsOK())
			}
		}
	})
	t.Run("should report as changed when data has changed and store update and reset error", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		section := model.CharacterSectionImplants
		x1 := factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     section,
			Error:       "error",
		})
		// when
		changed, err := s.recordCharacterSectionUpdate(ctx, c.ID, section, "new")
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x2, err := r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.LastUpdatedAt.Time, x1.LastUpdatedAt.Time)
				assert.True(t, x2.IsOK())
			}
		}
	})
	t.Run("should report as unchanged when data has not changed and store update", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		section := model.CharacterSectionImplants
		x1 := factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     section,
			Data:        "old",
		})
		// when
		changed, err := s.recordCharacterSectionUpdate(ctx, c.ID, section, "old")
		if assert.NoError(t, err) {
			assert.False(t, changed)
			x2, err := r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.LastUpdatedAt.Time, x1.LastUpdatedAt.Time)
				assert.True(t, x2.IsOK())
			}
		}
	})
}
