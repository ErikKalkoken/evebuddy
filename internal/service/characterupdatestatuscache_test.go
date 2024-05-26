package service

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/cache"
	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestCharacterUpdateStatusCache(t *testing.T) {
	characterID := int32(42)
	section := model.CharacterSectionImplants
	cache := cache.New()
	statusCache := newCharacterUpdateStatusCache(cache)
	t.Run("can set full status", func(t *testing.T) {
		// given
		myDate := time.Now()
		myError := "error"
		// when
		statusCache.setStatus(characterID, section, myError, myDate)
		// then
		x, y := statusCache.getStatus(characterID, section)
		assert.Equal(t, myDate, y)
		assert.Equal(t, myError, x)
	})
	t.Run("can set error only", func(t *testing.T) {
		// given
		myDate := time.Now().Add(-1 * time.Hour)
		statusCache.setStatus(characterID, section, "old-error", myDate)
		// when
		statusCache.setStatusError(characterID, section, "new-error")
		// then
		x, y := statusCache.getStatus(characterID, section)
		assert.Equal(t, myDate, y)
		assert.Equal(t, "new-error", x)
	})
}

func TestCharacterUpdateStatusCacheInit(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	cache := cache.New()
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
		statusCache := newCharacterUpdateStatusCache(cache)
		// when
		statusCache.initCache(r)
		// then
		x, y := statusCache.getStatus(c.ID, section)
		assert.Equal(t, myDate.UTC(), y.UTC())
		assert.Equal(t, "my-error", x)
		ids := statusCache.getCharacterIDs()
		assert.Equal(t, []int32{c.ID}, ids)

	})
}

func TestCharacterUpdateStatusCacheCharacterIDs(t *testing.T) {
	cache := cache.New()
	statusCache := newCharacterUpdateStatusCache(cache)
	t.Run("can get and set characterIDs", func(t *testing.T) {
		// given
		ids := []int32{1, 2, 3}
		// when
		statusCache.setCharacterIDs(ids)
		// then
		assert.Equal(t, ids, statusCache.getCharacterIDs())
	})
}
