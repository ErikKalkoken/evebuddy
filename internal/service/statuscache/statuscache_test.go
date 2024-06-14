package statuscache

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/cache"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestCharacterGetUpdateStatusSummary(t *testing.T) {
	cache := cache.New()
	cs := New(cache)
	cc := []*model.CharacterShort{{ID: 1, Name: "First"}, {ID: 2, Name: "Second"}}
	cs.setCharacters(cc)
	t.Run("should report when all sections are up-to-date", func(t *testing.T) {
		// given
		for _, c := range cc {
			for _, section := range model.CharacterSections {
				o := &model.CharacterSectionStatus{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
					UpdatedAt:    time.Now(),
				}
				cs.CharacterSectionSet(o)
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
				o := &model.CharacterSectionStatus{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
					UpdatedAt:    time.Now(),
				}
				cs.CharacterSectionSet(o)
			}
		}
		o := &model.CharacterSectionStatus{
			CharacterID:  cc[0].ID,
			Section:      model.SectionLocation,
			ErrorMessage: "error",
		}
		cs.CharacterSectionSet(o)
		// when
		_, c := cs.Summary()
		// then
		assert.Equal(t, 1, c)
	})
	t.Run("should report current progress", func(t *testing.T) {
		// given
		for _, c := range cc {
			for _, section := range model.CharacterSections {
				o := &model.CharacterSectionStatus{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
					UpdatedAt:    time.Now(),
				}
				cs.CharacterSectionSet(o)
			}
		}
		o := &model.CharacterSectionStatus{
			CharacterID: cc[0].ID,
			Section:     model.SectionLocation,
			CompletedAt: time.Now().Add(-1 * time.Hour),
		}
		cs.CharacterSectionSet(o)
		// when
		p, c := cs.Summary()
		// then
		assert.Less(t, p, float32(1.0))
		assert.Equal(t, 0, c)
	})
}
