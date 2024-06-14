package statuscache

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
	section := model.SectionImplants
	cache := cache.New()
	statusCache := New(cache)
	t.Run("can get and set a character status", func(t *testing.T) {
		// given
		o := &model.CharacterUpdateStatus{
			CharacterID:  characterID,
			Section:      section,
			CompletedAt:  time.Now(),
			StartedAt:    time.Now().Add(-30 * time.Second),
			ErrorMessage: "error",
			UpdatedAt:    time.Now().Add(-5 * time.Second),
		}
		cc := []*model.CharacterShort{{ID: 42, Name: "Alpha"}}
		statusCache.setCharacters(cc)
		// when
		statusCache.Set(o)
		// then
		v := statusCache.Get(characterID, section)
		assert.Equal(t, v.CharacterID, o.CharacterID)
		assert.Equal(t, v.CharacterName, "Alpha")
		assert.Equal(t, v.CompletedAt.UTC(), o.CompletedAt.UTC())
		assert.Equal(t, v.ErrorMessage, o.ErrorMessage)
		assert.Equal(t, v.Section, o.Section)
		assert.Equal(t, v.StartedAt.UTC(), o.StartedAt.UTC())
		assert.Equal(t, v.UpdateAt.UTC(), o.UpdatedAt.UTC())
	})
}

func TestCharacterUpdateStatusCacheInit(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	cache := cache.New()
	// ctx := context.Background()
	section := model.SectionImplants
	t.Run("should init", func(t *testing.T) {
		// given
		c := factory.CreateCharacter()
		completedAt := time.Now().Add(-1 * time.Hour)
		factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     section,
			CompletedAt: completedAt,
			Error:       "my-error",
		})
		statusCache := New(cache)
		// when
		statusCache.InitCache(r)
		// then
		v := statusCache.Get(c.ID, section)
		assert.Equal(t, completedAt.UTC(), v.CompletedAt.UTC())
		assert.Equal(t, "my-error", v.ErrorMessage)
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
				o := &model.CharacterUpdateStatus{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
					UpdatedAt:    time.Now(),
				}
				cs.Set(o)
			}
		}
		// when
		p, count := cs.Summary()
		// then
		assert.Equal(t, float32(1.0), p)
		assert.Equal(t, 0, count)
	})
	t.Run("should report when there is an error", func(t *testing.T) {
		// given
		for _, c := range cc {
			for _, section := range model.CharacterSections {
				o := &model.CharacterUpdateStatus{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
					UpdatedAt:    time.Now(),
				}
				cs.Set(o)
			}
		}
		o := &model.CharacterUpdateStatus{
			CharacterID:  cc[0].ID,
			Section:      model.SectionLocation,
			ErrorMessage: "error",
		}
		cs.Set(o)
		// when
		_, c := cs.Summary()
		// then
		assert.Equal(t, 1, c)
	})
	t.Run("should report current progress", func(t *testing.T) {
		// given
		for _, c := range cc {
			for _, section := range model.CharacterSections {
				o := &model.CharacterUpdateStatus{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
					UpdatedAt:    time.Now(),
				}
				cs.Set(o)
			}
		}
		o := &model.CharacterUpdateStatus{
			CharacterID: cc[0].ID,
			Section:     model.SectionLocation,
			CompletedAt: time.Now().Add(-1 * time.Hour),
		}
		cs.Set(o)
		// when
		p, c := cs.Summary()
		// then
		assert.Less(t, p, float32(1.0))
		assert.Equal(t, 0, c)
	})
}
