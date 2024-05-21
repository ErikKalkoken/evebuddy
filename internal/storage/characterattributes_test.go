package storage_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestCharacterAttributes(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		bonusRemaps := sql.NullInt64{Int64: 7, Valid: true}
		lastRemapDate := sql.NullTime{Time: time.Now().UTC(), Valid: true}
		arg := storage.UpdateOrCreateCharacterAttributesParams{
			CharacterID:   c.ID,
			BonusRemaps:   bonusRemaps,
			Charisma:      20,
			Intelligence:  21,
			LastRemapDate: lastRemapDate,
			Memory:        22,
			Perception:    23,
			Willpower:     24,
		}
		// when
		err := r.UpdateOrCreateCharacterAttributes(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCharacterAttributes(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 20, x.Charisma)
				assert.Equal(t, 21, x.Intelligence)
				assert.Equal(t, 22, x.Memory)
				assert.Equal(t, 23, x.Perception)
				assert.Equal(t, 24, x.Willpower)
				assert.Equal(t, bonusRemaps, x.BonusRemaps)
				assert.Equal(t, lastRemapDate, x.LastRemapDate)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterAttributes(storage.UpdateOrCreateCharacterAttributesParams{
			CharacterID: c.ID,
		})
		bonusRemaps := sql.NullInt64{Int64: 7, Valid: true}
		lastRemapDate := sql.NullTime{Time: time.Now().UTC(), Valid: true}
		arg := storage.UpdateOrCreateCharacterAttributesParams{
			CharacterID:   c.ID,
			BonusRemaps:   bonusRemaps,
			Charisma:      20,
			Intelligence:  21,
			LastRemapDate: lastRemapDate,
			Memory:        22,
			Perception:    23,
			Willpower:     24,
		}
		// when
		err := r.UpdateOrCreateCharacterAttributes(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCharacterAttributes(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 20, x.Charisma)
				assert.Equal(t, 21, x.Intelligence)
				assert.Equal(t, 22, x.Memory)
				assert.Equal(t, 23, x.Perception)
				assert.Equal(t, 24, x.Willpower)
				assert.Equal(t, bonusRemaps, x.BonusRemaps)
				assert.Equal(t, lastRemapDate, x.LastRemapDate)
			}
		}
	})

}
