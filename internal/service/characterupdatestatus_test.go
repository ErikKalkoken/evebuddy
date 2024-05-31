package service

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestCharacterGetUpdateStatusSummary(t *testing.T) {
	db, r, _ := testutil.New()
	defer db.Close()
	s := NewService(r)
	characterIDs := []int32{1, 2}
	s.characterStatus.SetCharacterIDs(characterIDs)
	t.Run("should report when all sections are up-to-date", func(t *testing.T) {
		// given
		for _, characterID := range characterIDs {
			for _, section := range model.CharacterSections {
				s.characterStatus.Set(characterID, section, "", time.Now())
			}
		}
		// when
		p, ok := s.CharacterGetUpdateStatusSummary()
		// then
		if assert.True(t, ok) {
			assert.Equal(t, float32(1.0), p)
		}
	})
	t.Run("should report when there is an error", func(t *testing.T) {
		// given
		for _, characterID := range characterIDs {
			for _, section := range model.CharacterSections {
				s.characterStatus.Set(characterID, section, "", time.Now())
			}
		}
		s.characterStatus.SetError(characterIDs[1], model.CharacterSectionLocation, "error")
		// when
		_, ok := s.CharacterGetUpdateStatusSummary()
		// then
		assert.False(t, ok)
	})
	t.Run("should report current progress", func(t *testing.T) {
		// given
		for _, characterID := range characterIDs {
			for _, section := range model.CharacterSections {
				s.characterStatus.Set(characterID, section, "", time.Now())
			}
		}
		s.characterStatus.Set(characterIDs[1], model.CharacterSectionLocation, "", time.Now().Add(-1*time.Hour))
		// when
		p, ok := s.CharacterGetUpdateStatusSummary()
		// then
		if assert.True(t, ok) {
			assert.Less(t, p, float32(1.0))
		}
	})
}
