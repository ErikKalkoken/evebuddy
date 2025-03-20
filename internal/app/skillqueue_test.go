package app_test

import (
	"context"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

type MyCS struct {
	items []*app.CharacterSkillqueueItem
	err   error
}

func (cs MyCS) ListCharacterSkillqueueItems(context.Context, int32) ([]*app.CharacterSkillqueueItem, error) {
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

func TestCharacterSkillqueue(t *testing.T) {
	characterID := int32(42)
	t.Run("can return information about an active skill queue", func(t *testing.T) {
		sq := app.NewCharacterSkillqueue()
		item1 := makeSkillQueueItem(characterID, app.CharacterSkillqueueItem{
			StartDate:  time.Now().Add(-3 * time.Hour),
			FinishDate: time.Now().Add(3 * time.Hour),
		})
		item2 := makeSkillQueueItem(characterID, app.CharacterSkillqueueItem{
			StartDate:  time.Now().Add(3 * time.Hour),
			FinishDate: time.Now().Add(7 * time.Hour),
		})
		cs := MyCS{items: []*app.CharacterSkillqueueItem{item1, item2}}
		err := sq.Update(cs, characterID)
		if assert.NoError(t, err) {
			assert.Equal(t, characterID, sq.CharacterID())
			assert.Equal(t, 2, sq.Size())
			assert.Equal(t, item1, sq.Current())
			assert.Equal(t, item2, sq.Item(1))
			assert.InDelta(t, 0.5, sq.Completion().ValueOrZero(), 0.01)
			assert.WithinDuration(t, toTime(7*time.Hour), toTime(sq.Remaining().ValueOrZero()), 10*time.Second)
			assert.True(t, sq.IsActive())
		}
	})
	t.Run("can return information about an empty skill queue", func(t *testing.T) {
		sq := app.NewCharacterSkillqueue()
		assert.Equal(t, int32(0), sq.CharacterID())
		assert.Equal(t, 0, sq.Size())
		assert.Nil(t, sq.Current())
		assert.Nil(t, sq.Item(1))
		assert.True(t, sq.Completion().IsEmpty())
		assert.False(t, sq.IsActive())
	})
}

func toTime(d time.Duration) time.Time {
	return time.Now().Add(d)
}
