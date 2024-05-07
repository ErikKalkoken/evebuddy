package model_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSkillqueueItemCompletion(t *testing.T) {
	now := time.Now()
	t.Run("should calculate when in progress", func(t *testing.T) {
		q := model.SkillqueueItem{
			StartDate:  now.Add(time.Hour * -1),
			FinishDate: now.Add(time.Hour * +3),
		}
		assert.InDelta(t, 0.25, q.CompletionP(), 0.01)
	})
	t.Run("should return 0 when starting in the future", func(t *testing.T) {
		q := model.SkillqueueItem{
			StartDate:  now.Add(time.Hour * +1),
			FinishDate: now.Add(time.Hour * +3),
		}
		assert.Equal(t, 0.0, q.CompletionP())
	})
	t.Run("should return 1 when finished in the past", func(t *testing.T) {
		q := model.SkillqueueItem{
			StartDate:  now.Add(time.Hour * -3),
			FinishDate: now.Add(time.Hour * -1),
		}
		assert.Equal(t, 1.0, q.CompletionP())
	})
}
