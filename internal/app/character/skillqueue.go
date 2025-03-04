package character

import (
	"context"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CharacterServiceSkillqueue interface {
	ListCharacterSkillqueueItems(context.Context, int32) ([]*app.CharacterSkillqueueItem, error)
}

// CharacterSkillqueue represents the skillqueue of a character.
type CharacterSkillqueue struct {
	characterID int32
	items       []*app.CharacterSkillqueueItem
}

func NewCharacterSkillqueue() CharacterSkillqueue {
	sq := CharacterSkillqueue{}
	return sq
}

func (sq *CharacterSkillqueue) CharacterID() int32 {
	return sq.characterID
}

func (sq *CharacterSkillqueue) Current() *app.CharacterSkillqueueItem {
	for _, item := range sq.items {
		if item.IsActive() {
			return item
		}
	}
	return nil
}

func (sq *CharacterSkillqueue) Completion() optional.Optional[float64] {
	c := sq.Current()
	if c == nil {
		return optional.Optional[float64]{}
	}
	return optional.New(c.CompletionP())
}

func (sq *CharacterSkillqueue) IsActive() bool {
	c := sq.Current()
	if c == nil {
		return false
	}
	return sq.Remaining().ValueOrZero() > 0
}

func (sq *CharacterSkillqueue) Item(id int) *app.CharacterSkillqueueItem {
	if id < 0 || id >= len(sq.items) {
		return nil
	}
	return sq.items[id]
}

func (sq *CharacterSkillqueue) Size() int {
	return len(sq.items)
}

func (sq *CharacterSkillqueue) Remaining() optional.Optional[time.Duration] {
	var r optional.Optional[time.Duration]
	for _, item := range sq.items {
		r = optional.New(r.ValueOrZero() + item.Remaining().ValueOrZero())
	}
	return r
}

func (sq *CharacterSkillqueue) Update(cs CharacterServiceSkillqueue, characterID int32) error {
	if characterID == 0 {
		sq.items = []*app.CharacterSkillqueueItem{}
		return nil
	}
	items, err := cs.ListCharacterSkillqueueItems(context.Background(), characterID)
	if err != nil {
		return err
	}
	sq.items = items
	sq.characterID = characterID
	return nil
}
