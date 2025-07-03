package app_test

import (
	"context"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestCharacterSkillqueue(t *testing.T) {
	characterID := int32(42)
	ctx := context.Background()
	t.Run("can return information about an active skill queue", func(t *testing.T) {
		sq := app.NewCharacterSkillqueue()
		item1 := makeSkillQueueItem(characterID, app.CharacterSkillqueueItem{
			StartDate:     time.Now().Add(-3 * time.Hour),
			FinishDate:    time.Now().Add(3 * time.Hour),
			QueuePosition: 1,
		})
		item2 := makeSkillQueueItem(characterID, app.CharacterSkillqueueItem{
			StartDate:     time.Now().Add(3 * time.Hour),
			FinishDate:    time.Now().Add(7 * time.Hour),
			QueuePosition: 2,
		})
		cs := MyCS{items: []*app.CharacterSkillqueueItem{item1, item2}}
		err := sq.Update(ctx, cs, characterID)
		if assert.NoError(t, err) {
			assert.Equal(t, characterID, sq.CharacterID())
			assert.Equal(t, 2, sq.Size())
			assert.Equal(t, item1, sq.Active())
			assert.Equal(t, item2, sq.Item(1))
			assert.InDelta(t, 0.5, sq.CompletionP().ValueOrZero(), 0.01)
			assert.True(t, sq.IsActive())
			assert.Equal(t, 2, sq.RemainingCount().ValueOrZero())
		}
	})
	t.Run("can return information about an empty skill queue", func(t *testing.T) {
		sq := app.NewCharacterSkillqueue()
		assert.Equal(t, int32(0), sq.CharacterID())
		assert.Equal(t, 0, sq.Size())
		assert.Nil(t, sq.Active())
		assert.Nil(t, sq.Item(1))
		assert.True(t, sq.CompletionP().IsEmpty())
		assert.False(t, sq.IsActive())
		assert.True(t, sq.RemainingCount().IsEmpty())
	})
}

func TestCharacterSkillqueueRemainingTime(t *testing.T) {
	characterID := int32(42)
	ctx := context.Background()
	t.Run("can return correct remainaing time for active skill queue 1", func(t *testing.T) {
		sq := app.NewCharacterSkillqueue()
		item1 := makeSkillQueueItem(characterID, app.CharacterSkillqueueItem{
			StartDate:     time.Now().Add(-3 * time.Hour),
			FinishDate:    time.Now().Add(3 * time.Hour),
			QueuePosition: 1,
		})
		item2 := makeSkillQueueItem(characterID, app.CharacterSkillqueueItem{
			StartDate:     time.Now().Add(3 * time.Hour),
			FinishDate:    time.Now().Add(7 * time.Hour),
			QueuePosition: 2,
		})
		cs := MyCS{items: []*app.CharacterSkillqueueItem{item1, item2}}
		err := sq.Update(ctx, cs, characterID)
		if assert.NoError(t, err) {
			assert.WithinDuration(t, toTime(7*time.Hour), toTime(sq.RemainingTime().ValueOrZero()), 1*time.Second)
			assert.WithinDuration(t, toTime(7*time.Hour), sq.FinishDate().ValueOrZero(), 1*time.Second)
		}
	})
	t.Run("should ignore finished skills in caluclation", func(t *testing.T) {
		sq := app.NewCharacterSkillqueue()
		item0 := makeSkillQueueItem(characterID, app.CharacterSkillqueueItem{
			StartDate:     time.Now().Add(-6 * time.Hour),
			FinishDate:    time.Now().Add(-3 * time.Hour),
			QueuePosition: 1,
		})
		item1 := makeSkillQueueItem(characterID, app.CharacterSkillqueueItem{
			StartDate:     time.Now().Add(-3 * time.Hour),
			FinishDate:    time.Now().Add(3 * time.Hour),
			QueuePosition: 2,
		})
		item2 := makeSkillQueueItem(characterID, app.CharacterSkillqueueItem{
			StartDate:     time.Now().Add(3 * time.Hour),
			FinishDate:    time.Now().Add(7 * time.Hour),
			QueuePosition: 3,
		})
		cs := MyCS{items: []*app.CharacterSkillqueueItem{item0, item1, item2}}
		err := sq.Update(ctx, cs, characterID)
		if assert.NoError(t, err) {
			assert.WithinDuration(t, toTime(7*time.Hour), toTime(sq.RemainingTime().ValueOrZero()), 1*time.Second)
			assert.WithinDuration(t, toTime(7*time.Hour), sq.FinishDate().ValueOrZero(), 1*time.Second)
		}
	})
	t.Run("returns empty remainaing time for empty skill queue", func(t *testing.T) {
		sq := app.NewCharacterSkillqueue()
		assert.True(t, sq.RemainingTime().IsEmpty())
		assert.True(t, sq.FinishDate().IsEmpty())
	})
}

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
	t.Run("should return 0 when empty", func(t *testing.T) {
		q := app.CharacterSkillqueueItem{}
		assert.Equal(t, 0.0, q.CompletionP())
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
		assert.Equal(t, 2*time.Hour, d.MustValue())
	})
	t.Run("should return null when duration can not be calculated 1", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			StartDate: now.Add(time.Hour * +1),
		}
		d := q.Duration()
		assert.True(t, d.IsEmpty())
	})
	t.Run("should return empty when duration can not be calculated 2", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			FinishDate: now.Add(time.Hour * +1),
		}
		d := q.Duration()
		assert.True(t, d.IsEmpty())
	})
	t.Run("should return empty when duration can not be calculated 3", func(t *testing.T) {
		q := app.CharacterSkillqueueItem{}
		d := q.Duration()
		assert.True(t, d.IsEmpty())
	})
}

func TestSkillqueueItemRemaining(t *testing.T) {
	t.Run("should return correct value when finish in the future", func(t *testing.T) {
		now := time.Now()
		q := makeItem(now, now.Add(time.Hour*+3))
		d := q.Remaining()
		assert.InDelta(t, 3*time.Hour, d.MustValue(), 10000)
	})
	t.Run("should return correct value when start and finish in the future", func(t *testing.T) {
		now := time.Now()
		q := makeItem(now.Add(time.Hour*+1), now.Add(time.Hour*+3))
		d := q.Remaining()
		assert.InDelta(t, 2*time.Hour, d.MustValue(), 10000)
	})
	t.Run("should return 0 remaining when completed", func(t *testing.T) {
		now := time.Now()
		q := makeItem(now.Add(time.Hour*-3), now.Add(time.Hour*-2))
		d := q.Remaining()
		assert.Equal(t, time.Duration(0), d.MustValue())
	})
	t.Run("should return null when no finish date", func(t *testing.T) {
		now := time.Now()
		q := makeItem(now, time.Time{})
		d := q.Remaining()
		assert.True(t, d.IsEmpty())
	})
	t.Run("should return null when no start date", func(t *testing.T) {
		now := time.Now()
		q := makeItem(time.Time{}, now.Add(time.Hour*+2))
		d := q.Remaining()
		assert.True(t, d.IsEmpty())
	})
}

type MyCS struct {
	items []*app.CharacterSkillqueueItem
	err   error
}

func (cs MyCS) ListSkillqueueItems(context.Context, int32) ([]*app.CharacterSkillqueueItem, error) {
	if cs.err != nil {
		return nil, cs.err
	}
	return cs.items, nil
}

func makeSkillQueueItem(characterID int32, args ...app.CharacterSkillqueueItem) *app.CharacterSkillqueueItem {
	var arg app.CharacterSkillqueueItem
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.QueuePosition == 0 {
		panic("must define QueuePosition")
	}
	now := time.Now()
	arg.CharacterID = characterID
	if arg.FinishedLevel == 0 {
		arg.FinishedLevel = rand.IntN(5) + 1
	}
	if arg.LevelEndSP == 0 {
		arg.LevelEndSP = rand.IntN(1_000_000)
	}
	if arg.StartDate.IsZero() {
		hours := rand.IntN(10)*24 + 3
		arg.StartDate = now.Add(time.Duration(-hours) * time.Hour)
	}
	if arg.FinishDate.IsZero() {
		hours := rand.IntN(10)*24 + 3
		arg.FinishDate = now.Add(time.Duration(hours) * time.Hour)
	}
	if arg.GroupName == "" {
		arg.GroupName = "Group"
	}
	if arg.SkillName == "" {
		arg.SkillName = "Skill"
	}
	if arg.SkillDescription == "" {
		arg.SkillDescription = "Description"
	}
	return &arg
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

func toTime(d time.Duration) time.Time {
	return time.Now().Add(d)
}
