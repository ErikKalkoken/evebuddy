package service

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestCharacterUpdateStatusCache(t *testing.T) {
	characterID := int32(42)
	section := model.CharacterSectionImplants
	cache := newCharacterUpdateStatusCache()
	t.Run("can set full status", func(t *testing.T) {
		// given
		myDate := time.Now()
		myError := "error"
		// when
		cache.set(characterID, section, myError, myDate)
		// then
		x, y := cache.get(characterID, section)
		assert.Equal(t, myDate, y)
		assert.Equal(t, myError, x)
	})
	t.Run("can set error only", func(t *testing.T) {
		// given
		myDate := time.Now().Add(-1 * time.Hour)
		cache.set(characterID, section, "old-error", myDate)
		// when
		cache.setError(characterID, section, "new-error")
		// then
		x, y := cache.get(characterID, section)
		assert.Equal(t, myDate, y)
		assert.Equal(t, "new-error", x)
	})
}

func TestCharacterUpdateStatusCacheInit(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	// ctx := context.Background()
	section := model.CharacterSectionImplants
	t.Run("should init", func(t *testing.T) {
		// given
		c := factory.CreateCharacter()
		myDate := time.Now().Add(-1 * time.Hour)
		factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID:   c.ID,
			Section:       section,
			LastUpdatedAt: myDate,
			Error:         "my-error",
		})
		cache := newCharacterUpdateStatusCache()
		// when
		cache.initCache(r)
		// then
		x, y := cache.get(c.ID, section)
		assert.Equal(t, myDate.UTC(), y.UTC())
		assert.Equal(t, "my-error", x)

	})
}
