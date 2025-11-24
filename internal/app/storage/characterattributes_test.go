package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestCharacterAttributes(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		lastRemapDate := time.Now().UTC()
		arg := storage.UpdateOrCreateCharacterAttributesParams{
			CharacterID:   c.ID,
			BonusRemaps:   7,
			Charisma:      20,
			Intelligence:  21,
			LastRemapDate: lastRemapDate,
			Memory:        22,
			Perception:    23,
			Willpower:     24,
		}
		// when
		err := st.UpdateOrCreateCharacterAttributes(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCharacterAttributes(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 20, x.Charisma)
				assert.Equal(t, 21, x.Intelligence)
				assert.Equal(t, 22, x.Memory)
				assert.Equal(t, 23, x.Perception)
				assert.Equal(t, 24, x.Willpower)
				assert.Equal(t, 7, x.BonusRemaps)
				assert.Equal(t, lastRemapDate, x.LastRemapDate)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterAttributes(storage.UpdateOrCreateCharacterAttributesParams{
			CharacterID: c.ID,
		})
		lastRemapDate := time.Now().UTC()
		arg := storage.UpdateOrCreateCharacterAttributesParams{
			CharacterID:   c.ID,
			BonusRemaps:   7,
			Charisma:      20,
			Intelligence:  21,
			LastRemapDate: lastRemapDate,
			Memory:        22,
			Perception:    23,
			Willpower:     24,
		}
		// when
		err := st.UpdateOrCreateCharacterAttributes(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCharacterAttributes(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 20, x.Charisma)
				assert.Equal(t, 21, x.Intelligence)
				assert.Equal(t, 22, x.Memory)
				assert.Equal(t, 23, x.Perception)
				assert.Equal(t, 24, x.Willpower)
				assert.Equal(t, 7, x.BonusRemaps)
				assert.Equal(t, lastRemapDate, x.LastRemapDate)
			}
		}
	})
	t.Run("returns not found error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetCharacterAttributes(ctx, 1)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}
