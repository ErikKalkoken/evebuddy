package characterstatus

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
	statusCache := New(cache)
	t.Run("can set full status", func(t *testing.T) {
		// given
		myDate := time.Now()
		myError := "error"
		// when
		statusCache.SetStatus(characterID, section, myError, myDate)
		// then
		x, y := statusCache.GetStatus(characterID, section)
		assert.Equal(t, myDate, y)
		assert.Equal(t, myError, x)
	})
	t.Run("can set error only", func(t *testing.T) {
		// given
		myDate := time.Now().Add(-1 * time.Hour)
		statusCache.SetStatus(characterID, section, "old-error", myDate)
		// when
		statusCache.SetError(characterID, section, "new-error")
		// then
		x, y := statusCache.GetStatus(characterID, section)
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
		statusCache := New(cache)
		// when
		statusCache.InitCache(r)
		// then
		x, y := statusCache.GetStatus(c.ID, section)
		assert.Equal(t, myDate.UTC(), y.UTC())
		assert.Equal(t, "my-error", x)
		ids := statusCache.getCharacters()
		assert.Equal(t, []int32{c.ID}, ids)

	})
}

func TestCharacterUpdateStatusCacheCharacterIDs(t *testing.T) {
	cache := cache.New()
	statusCache := New(cache)
	t.Run("can get and set characterIDs", func(t *testing.T) {
		// given
		ids := []int32{1, 2, 3}
		// when
		statusCache.setCharacters(ids)
		// then
		assert.Equal(t, ids, statusCache.getCharacters())
	})
}

func TestCharacterGetUpdateStatusSummary(t *testing.T) {
	cache := cache.New()
	cs := New(cache)
	characterIDs := []int32{1, 2}
	cs.setCharacters(characterIDs)
	t.Run("should report when all sections are up-to-date", func(t *testing.T) {
		// given
		for _, characterID := range characterIDs {
			for _, section := range model.CharacterSections {
				cs.SetStatus(characterID, section, "", time.Now())
			}
		}
		// when
		p, ok := cs.StatusSummary()
		// then
		if assert.True(t, ok) {
			assert.Equal(t, float32(1.0), p)
		}
	})
	t.Run("should report when there is an error", func(t *testing.T) {
		// given
		for _, characterID := range characterIDs {
			for _, section := range model.CharacterSections {
				cs.SetStatus(characterID, section, "", time.Now())
			}
		}
		cs.SetError(characterIDs[1], model.CharacterSectionLocation, "error")
		// when
		_, ok := cs.StatusSummary()
		// then
		assert.False(t, ok)
	})
	t.Run("should report current progress", func(t *testing.T) {
		// given
		for _, characterID := range characterIDs {
			for _, section := range model.CharacterSections {
				cs.SetStatus(characterID, section, "", time.Now())
			}
		}
		cs.SetStatus(characterIDs[1], model.CharacterSectionLocation, "", time.Now().Add(-1*time.Hour))
		// when
		p, ok := cs.StatusSummary()
		// then
		if assert.True(t, ok) {
			assert.Less(t, p, float32(1.0))
		}
	})
}
