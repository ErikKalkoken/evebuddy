package app_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestSkillqueueItemCompletion(t *testing.T) {
	t.Run("should calculate when started at 0", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
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
		q := app.CharacterSkillqueueItem{
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
		q := app.CharacterSkillqueueItem{
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
		q := app.CharacterSkillqueueItem{
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
		q := app.CharacterSkillqueueItem{
			StartDate:  now.Add(time.Hour * +1),
			FinishDate: now.Add(time.Hour * +3),
		}
		assert.Equal(t, 0.0, q.CompletionP())
	})
	t.Run("should return 1 when finished in the past", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			StartDate:  now.Add(time.Hour * -3),
			FinishDate: now.Add(time.Hour * -1),
		}
		assert.Equal(t, 1.0, q.CompletionP())
	})
}

func TestSkillqueueItemDuration(t *testing.T) {
	t.Run("should return duration when possible", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			StartDate:  now.Add(time.Hour * +1),
			FinishDate: now.Add(time.Hour * +3),
		}
		d := q.Duration()
		assert.True(t, d.Valid)
		assert.Equal(t, 2*time.Hour, d.Duration)
	})
	t.Run("should return null when duration can not be calculated 1", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			StartDate: now.Add(time.Hour * +1),
		}
		d := q.Duration()
		assert.False(t, d.Valid)
	})
	t.Run("should return null when duration can not be calculated 2", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			FinishDate: now.Add(time.Hour * +1),
		}
		d := q.Duration()
		assert.False(t, d.Valid)
	})
	t.Run("should return null when duration can not be calculated 3", func(t *testing.T) {
		q := app.CharacterSkillqueueItem{}
		d := q.Duration()
		assert.False(t, d.Valid)
	})
}

func makeItem(startDate, finishDate time.Time) app.CharacterSkillqueueItem {
	return app.CharacterSkillqueueItem{
		StartDate:       startDate,
		FinishDate:      finishDate,
		LevelStartSP:    0,
		LevelEndSP:      1000,
		TrainingStartSP: 0,
	}
}

func TestSkillqueueItemRemaining(t *testing.T) {
	t.Run("should return correct value when finish in the future", func(t *testing.T) {
		now := time.Now()
		q := makeItem(now, now.Add(time.Hour*+3))
		d := q.Remaining()
		assert.True(t, d.Valid)
		assert.InDelta(t, 3*time.Hour, d.Duration, 10000)
	})
	t.Run("should return correct value when start and finish in the future", func(t *testing.T) {
		now := time.Now()
		q := makeItem(now.Add(time.Hour*+1), now.Add(time.Hour*+3))
		d := q.Remaining()
		assert.True(t, d.Valid)
		assert.InDelta(t, 2*time.Hour, d.Duration, 10000)
	})
	t.Run("should return 0 remaining when completed", func(t *testing.T) {
		now := time.Now()
		q := makeItem(now.Add(time.Hour*-3), now.Add(time.Hour*-2))
		d := q.Remaining()
		assert.True(t, d.Valid)
		assert.Equal(t, time.Duration(0), d.Duration)
	})
	t.Run("should return null when no finish date", func(t *testing.T) {
		now := time.Now()
		q := makeItem(now, time.Time{})
		d := q.Remaining()
		assert.False(t, d.Valid)
	})
	t.Run("should return null when no start date", func(t *testing.T) {
		now := time.Now()
		q := makeItem(time.Time{}, now.Add(time.Hour*+2))
		d := q.Remaining()
		assert.False(t, d.Valid)
	})
}
