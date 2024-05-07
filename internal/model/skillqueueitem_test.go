package model_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSkillqueueItemCompletion(t *testing.T) {
	t.Run("should calculate when started at 0", func(t *testing.T) {
		now := time.Now()
		q := model.SkillqueueItem{
			StartDate:       now.Add(time.Hour * -1),
			FinishDate:      now.Add(time.Hour * +3),
			LevelStartSP:    0,
			LevelEndSP:      100,
			TrainingStartSP: 0,
		}
		assert.InDelta(t, 0.25, q.CompletionP(), 0.01)
	})
	t.Run("should calculate when with sp offset 1", func(t *testing.T) {
		now := time.Now()
		q := model.SkillqueueItem{
			StartDate:       now.Add(time.Hour * -1),
			FinishDate:      now.Add(time.Hour * +1),
			LevelStartSP:    0,
			LevelEndSP:      100,
			TrainingStartSP: 50,
		}
		assert.InDelta(t, 0.75, q.CompletionP(), 0.01)
	})
	t.Run("should calculate when with sp offset 2", func(t *testing.T) {
		now := time.Now()
		q := model.SkillqueueItem{
			StartDate:       now.Add(time.Hour * -2),
			FinishDate:      now.Add(time.Hour * +1),
			LevelStartSP:    0,
			LevelEndSP:      100,
			TrainingStartSP: 25,
		}
		assert.InDelta(t, 0.75, q.CompletionP(), 0.01)
	})
	t.Run("should calculate when with sp offset 3", func(t *testing.T) {
		now := time.Now()
		q := model.SkillqueueItem{
			StartDate:       now.Add(time.Hour * -2),
			FinishDate:      now.Add(time.Hour * +1),
			LevelStartSP:    100,
			LevelEndSP:      200,
			TrainingStartSP: 125,
		}
		assert.InDelta(t, 0.75, q.CompletionP(), 0.01)
	})
	t.Run("should return 0 when starting in the future", func(t *testing.T) {
		now := time.Now()
		q := model.SkillqueueItem{
			StartDate:  now.Add(time.Hour * +1),
			FinishDate: now.Add(time.Hour * +3),
		}
		assert.Equal(t, 0.0, q.CompletionP())
	})
	t.Run("should return 1 when finished in the past", func(t *testing.T) {
		now := time.Now()
		q := model.SkillqueueItem{
			StartDate:  now.Add(time.Hour * -3),
			FinishDate: now.Add(time.Hour * -1),
		}
		assert.Equal(t, 1.0, q.CompletionP())
	})
}
