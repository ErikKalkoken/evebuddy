package app

import (
	"cmp"
	"context"
	"fmt"
	"iter"
	"slices"
	"strings"
	"sync"
	"time"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CharacterSkill struct {
	ActiveSkillLevel   int64
	CharacterID        int64
	SkillPointsInSkill int64
	TrainedSkillLevel  int64
	Type               *EveType
}

type CharacterSkill2 struct {
	ActiveSkillLevel   int64
	CharacterID        int64
	HasPrerequisites   bool
	Skill              *EveSkill
	SkillPointsInSkill int64
	TrainedSkillLevel  int64
}

func SkillDisplayName[N int | int64 | uint | uint32 | uint64](name string, level N) string {
	return fmt.Sprintf("%s %s", name, ihumanize.RomanLetter(level))
}

// CharacterActiveSkillLevel represents the active level of a character's skill.
type CharacterActiveSkillLevel struct {
	CharacterID int64
	Level       int
	TypeID      int64
}

type CharacterShipSkill struct {
	ActiveSkillLevel  optional.Optional[int]
	ID                int64
	CharacterID       int64
	Rank              uint
	ShipTypeID        int64
	SkillTypeID       int64
	SkillName         string
	SkillLevel        uint
	TrainedSkillLevel optional.Optional[int]
}

type CharacterServiceSkillqueue interface {
	ListSkillqueueItems(context.Context, int64) ([]*CharacterSkillqueueItem, error)
}

// TODO: Is the mutex still needed with Fyne 2.6 ??

// CharacterSkillqueue represents the skillqueue of a character.
// This type is safe to use concurrently.
type CharacterSkillqueue struct {
	mu          sync.RWMutex
	characterID int64
	items       []*CharacterSkillqueueItem
}

// NewCharacterSkillqueue returns a new skill queue for a character.
func NewCharacterSkillqueue() *CharacterSkillqueue {
	sq := &CharacterSkillqueue{items: make([]*CharacterSkillqueueItem, 0)}
	return sq
}

func (sq *CharacterSkillqueue) All() iter.Seq[*CharacterSkillqueueItem] {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	return slices.Values(sq.items)
}

// CharacterID returns the character ID related to a queue.
func (sq *CharacterSkillqueue) CharacterID() int64 {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	return sq.characterID
}

// Active returns the skill currently in training or nil if training is inactive.
func (sq *CharacterSkillqueue) Active() *CharacterSkillqueueItem {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	for _, item := range sq.items {
		if item.IsActive() {
			return item
		}
	}
	return nil
}

// Last tries to return the last skill in the queue and reports if a skill was returned.
func (sq *CharacterSkillqueue) Last() (*CharacterSkillqueueItem, bool) {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	length := len(sq.items)
	if length == 0 {
		return nil, false
	}
	return sq.items[length-1], true
}

// CompletionP returns the completion percentage of the current skill in training (if any).
func (sq *CharacterSkillqueue) CompletionP() optional.Optional[float64] {
	c := sq.Active()
	if c == nil {
		return optional.Optional[float64]{}
	}
	return optional.New(c.CompletionP())
}

// IsActive reports whether training is active.
// An empty queue will be reported as inactive.
func (sq *CharacterSkillqueue) IsActive() bool {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	for _, qi := range sq.items {
		if v, ok := qi.FinishDate.Value(); ok && v.After(time.Now().UTC()) {
			return true
		}
	}
	return false
}

// Item returns the item on position id in the queue.
// It returns nil when the position is invalid.
func (sq *CharacterSkillqueue) Item(id int) *CharacterSkillqueueItem {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	if id < 0 || id >= len(sq.items) {
		return nil
	}
	return sq.items[id]
}

// Size returns the total number of items in the queue.
func (sq *CharacterSkillqueue) Size() int {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	return len(sq.items)
}

// RemainingCount returns the number of skills to be trained.
func (sq *CharacterSkillqueue) RemainingCount() optional.Optional[int] {
	if !sq.IsActive() {
		return optional.Optional[int]{}
	}
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	var count int
	for _, item := range sq.items {
		if !item.Remaining().IsEmpty() {
			count++
		}
	}
	return optional.New(count)
}

// RemainingTime returns the total remaining training time of all skills in the queue.
func (sq *CharacterSkillqueue) RemainingTime() optional.Optional[time.Duration] {
	var zero optional.Optional[time.Duration]
	t := sq.FinishDate()
	v, ok := t.Value()
	if !ok {
		return zero
	}
	d := time.Until(v)
	if d < 0 {
		return zero
	}
	return optional.New(d)
}

// FinishDate returns the finish date of last skill in the queue.
func (sq *CharacterSkillqueue) FinishDate() optional.Optional[time.Time] {
	if last, ok := sq.Last(); ok {
		return last.FinishDate
	}
	return optional.Optional[time.Time]{}
}

// Update replaces the content of the queue with a new version from the service.
func (sq *CharacterSkillqueue) Update(ctx context.Context, cs CharacterServiceSkillqueue, characterID int64) error {
	items, err := sq.fetchItems(ctx, cs, characterID)
	if err != nil {
		return err
	}
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.items = items
	sq.characterID = characterID
	return nil
}

func (sq *CharacterSkillqueue) fetchItems(ctx context.Context, cs CharacterServiceSkillqueue, characterID int64) ([]*CharacterSkillqueueItem, error) {
	if characterID == 0 {
		items := make([]*CharacterSkillqueueItem, 0)
		return items, nil
	}
	items, err := cs.ListSkillqueueItems(ctx, characterID)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b *CharacterSkillqueueItem) int {
		return cmp.Compare(a.QueuePosition, b.QueuePosition)
	})
	return items, nil
}

type CharacterSkillqueueItem struct {
	CharacterID      int64
	GroupName        string
	FinishDate       optional.Optional[time.Time]
	FinishedLevel    int64
	LevelEndSP       optional.Optional[int64]
	LevelStartSP     optional.Optional[int64]
	ID               int64
	QueuePosition    int64
	StartDate        optional.Optional[time.Time]
	SkillID          int64
	SkillName        string
	SkillDescription string
	TrainingStartSP  optional.Optional[int64]
}

func (qi CharacterSkillqueueItem) String() string {
	return fmt.Sprintf("%s %s", qi.SkillName, ihumanize.RomanLetter(qi.FinishedLevel))
}

// StringShortened returns a string where some names are abbreviated.
func (qi CharacterSkillqueueItem) StringShortened() string {
	name := strings.ReplaceAll(qi.SkillName, "Specialization", "Spec.")
	return fmt.Sprintf("%s %s", name, ihumanize.RomanLetter(qi.FinishedLevel))
}

// IsActive reports whether a skill is active.
func (qi CharacterSkillqueueItem) IsActive() bool {
	now := time.Now()
	start, ok1 := qi.StartDate.Value()
	finish, ok2 := qi.FinishDate.Value()
	if !ok1 || !ok2 {
		return false
	}
	return start.Before(now) && finish.After(now)
}

func (qi CharacterSkillqueueItem) IsCompleted() bool {
	return qi.CompletionP() == 1
}

func (qi CharacterSkillqueueItem) CompletionP() float64 {
	start, ok1 := qi.StartDate.Value()
	finish, ok2 := qi.FinishDate.Value()
	if !ok1 || !ok2 {
		return 0
	}
	d := qi.Duration()
	duration, ok := d.Value()
	if !ok {
		return 0
	}
	now := time.Now()
	if finish.Before(now) {
		return 1
	}
	if start.After(now) {
		return 0
	}
	if duration == 0 {
		return 0
	}
	remaining := finish.Sub(now)
	c := remaining.Seconds() / duration.Seconds()
	levelEndSP := qi.LevelEndSP.ValueOrZero()
	base := float64(levelEndSP-qi.TrainingStartSP.ValueOrZero()) / float64(levelEndSP-qi.LevelStartSP.ValueOrZero())
	return 1 - (c * base)
}

func (qi CharacterSkillqueueItem) Duration() optional.Optional[time.Duration] {
	start, ok1 := qi.StartDate.Value()
	finish, ok2 := qi.FinishDate.Value()
	if !ok1 || !ok2 {
		return optional.Optional[time.Duration]{}
	}
	return optional.New(finish.Sub(start))
}

func (qi CharacterSkillqueueItem) Remaining() optional.Optional[time.Duration] {
	p := qi.CompletionP()
	if p == 1 {
		return optional.New(time.Duration(0)) // completed
	}
	if v, ok := qi.FinishDate.Value(); ok && qi.IsActive() {
		return optional.New(time.Until(v))
	}
	return qi.Duration()
}
