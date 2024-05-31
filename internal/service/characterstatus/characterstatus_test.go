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
		cc := statusCache.ListCharacters()
		assert.Len(t, cc, 1)
		assert.Equal(t, cc[0].ID, c.ID)

	})
}

func TestCharacterUpdateStatusCacheCharacterIDs(t *testing.T) {
	cache := cache.New()
	statusCache := New(cache)
	t.Run("can get and set characterIDs", func(t *testing.T) {
		// given
		cc := []*model.CharacterShort{{ID: 1, Name: "First"}, {ID: 2, Name: "Second"}}
		// when
		statusCache.setCharacters(cc)
		// then
		assert.Equal(t, cc, statusCache.ListCharacters())
	})
}

func TestCharacterGetUpdateStatusSummary(t *testing.T) {
	cache := cache.New()
	cs := New(cache)
	cc := []*model.CharacterShort{{ID: 1, Name: "First"}, {ID: 2, Name: "Second"}}
	cs.setCharacters(cc)
	t.Run("should report when all sections are up-to-date", func(t *testing.T) {
		// given
		for _, c := range cc {
			for _, section := range model.CharacterSections {
				cs.SetStatus(c.ID, section, "", time.Now())
			}
		}
		// when
		p, ok := cs.Summary()
		// then
		if assert.True(t, ok) {
			assert.Equal(t, float32(1.0), p)
		}
	})
	t.Run("should report when there is an error", func(t *testing.T) {
		// given
		for _, c := range cc {
			for _, section := range model.CharacterSections {
				cs.SetStatus(c.ID, section, "", time.Now())
			}
		}
		cs.SetError(cc[1].ID, model.CharacterSectionLocation, "error")
		// when
		_, ok := cs.Summary()
		// then
		assert.False(t, ok)
	})
	t.Run("should report current progress", func(t *testing.T) {
		// given
		for _, c := range cc {
			for _, section := range model.CharacterSections {
				cs.SetStatus(c.ID, section, "", time.Now())
			}
		}
		cs.SetStatus(cc[1].ID, model.CharacterSectionLocation, "", time.Now().Add(-1*time.Hour))
		// when
		p, ok := cs.Summary()
		// then
		if assert.True(t, ok) {
			assert.Less(t, p, float32(1.0))
		}
	})
}
