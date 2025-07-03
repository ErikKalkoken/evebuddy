package app

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CharacterSkill struct {
	ActiveSkillLevel   int
	CharacterID        int32
	EveType            *EveType
	ID                 int64
	SkillPointsInSkill int
	TrainedSkillLevel  int
}

func SkillDisplayName[N int | int32 | int64 | uint | uint32 | uint64](name string, level N) string {
	return fmt.Sprintf("%s %s", name, ihumanize.RomanLetter(level))
}

// CharacterActiveSkillLevel represents the active level of a character's skill.
type CharacterActiveSkillLevel struct {
	CharacterID int32
	Level       int
	TypeID      int32
}

type ListCharacterSkillGroupProgress struct {
	GroupID   int32
	GroupName string
	Total     float64
	Trained   float64
}

type ListSkillProgress struct {
	ActiveSkillLevel  int
	TrainedSkillLevel int
	TypeID            int32
	TypeDescription   string
	TypeName          string
}

type CharacterShipSkill struct {
	ActiveSkillLevel  optional.Optional[int]
	ID                int64
	CharacterID       int32
	Rank              uint
	ShipTypeID        int32
	SkillTypeID       int32
	SkillName         string
	SkillLevel        uint
	TrainedSkillLevel optional.Optional[int]
}

type CharacterServiceSkillqueue interface {
	ListSkillqueueItems(context.Context, int32) ([]*CharacterSkillqueueItem, error)
}

// TODO: Is the mutex still needed with Fyne 2.6 ??

// CharacterSkillqueue represents the skillqueue of a character.
// This type is safe to use concurrently.
type CharacterSkillqueue struct {
	mu          sync.RWMutex
	characterID int32
	items       []*CharacterSkillqueueItem
}

// NewCharacterSkillqueue returns a new skill queue for a character.
func NewCharacterSkillqueue() *CharacterSkillqueue {
	sq := &CharacterSkillqueue{items: make([]*CharacterSkillqueueItem, 0)}
	return sq
}

// CharacterID returns the character ID related to a queue.
func (sq *CharacterSkillqueue) CharacterID() int32 {
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

// Last returns the last skill in the queue or nil if the queue is empty.
func (sq *CharacterSkillqueue) Last() *CharacterSkillqueueItem {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	length := len(sq.items)
	if length == 0 {
		return nil
	}
	return sq.items[length-1]
}

// CompletionP returns the completion percentage of the current skill in training (if any).
func (sq *CharacterSkillqueue) CompletionP() optional.Optional[float64] {
	c := sq.Active()
	if c == nil {
		return optional.Optional[float64]{}
	}
	return optional.From(c.CompletionP())
}

// IsActive reports whether training is active.
func (sq *CharacterSkillqueue) IsActive() bool {
	c := sq.Active()
	if c == nil {
		return false
	}
	return sq.RemainingTime().ValueOrZero() > 0
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
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	var count int
	var isActive bool
	for _, item := range sq.items {
		if !item.Remaining().IsEmpty() {
			isActive = true
			count++
		}
	}
	if !isActive {
		return optional.Optional[int]{}
	}
	return optional.From(count)
}

// RemainingTime returns the total remaining training time of all skills in the queue.
func (sq *CharacterSkillqueue) RemainingTime() optional.Optional[time.Duration] {
	zero := optional.Optional[time.Duration]{}
	t := sq.FinishDate()
	if t.IsEmpty() {
		return zero
	}
	d := time.Until(t.MustValue())
	if d < 0 {
		return zero
	}
	return optional.From(d)
}

// FinishDate returns the finish date of last skill in the queue.
func (sq *CharacterSkillqueue) FinishDate() optional.Optional[time.Time] {
	last := sq.Last()
	if last == nil || last.FinishDate.IsZero() {
		return optional.Optional[time.Time]{}
	}
	return optional.From(last.FinishDate)
}

// Update replaces the content of the queue with a new version from the service.
func (sq *CharacterSkillqueue) Update(ctx context.Context, cs CharacterServiceSkillqueue, characterID int32) error {
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

func (sq *CharacterSkillqueue) fetchItems(ctx context.Context, cs CharacterServiceSkillqueue, characterID int32) ([]*CharacterSkillqueueItem, error) {
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
	CharacterID      int32
	GroupName        string
	FinishDate       time.Time
	FinishedLevel    int
	LevelEndSP       int
	LevelStartSP     int
	ID               int64
	QueuePosition    int
	StartDate        time.Time
	SkillID          int32
	SkillName        string
	SkillDescription string
	TrainingStartSP  int
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
	return !qi.StartDate.IsZero() && qi.StartDate.Before(now) && qi.FinishDate.After(now)
}

func (qi CharacterSkillqueueItem) IsCompleted() bool {
	return qi.CompletionP() == 1
}

func (qi CharacterSkillqueueItem) CompletionP() float64 {
	d := qi.Duration()
	if d.IsEmpty() {
		return 0
	}
	duration := d.ValueOrZero()
	now := time.Now()
	if qi.FinishDate.Before(now) {
		return 1
	}
	if qi.StartDate.After(now) {
		return 0
	}
	if duration == 0 {
		return 0
	}
	remaining := qi.FinishDate.Sub(now)
	c := remaining.Seconds() / duration.Seconds()
	base := float64(qi.LevelEndSP-qi.TrainingStartSP) / float64(qi.LevelEndSP-qi.LevelStartSP)
	return 1 - (c * base)
}

func (qi CharacterSkillqueueItem) Duration() optional.Optional[time.Duration] {
	if qi.StartDate.IsZero() || qi.FinishDate.IsZero() {
		return optional.Optional[time.Duration]{}
	}
	return optional.From(qi.FinishDate.Sub(qi.StartDate))
}

func (qi CharacterSkillqueueItem) Remaining() optional.Optional[time.Duration] {
	if qi.StartDate.IsZero() || qi.FinishDate.IsZero() {
		return optional.Optional[time.Duration]{}
	}
	remainingP := 1 - qi.CompletionP()
	d := qi.Duration()
	return optional.From(time.Duration(float64(d.ValueOrZero()) * remainingP))
}

func (qi CharacterSkillqueueItem) FinishDateEstimate() optional.Optional[time.Time] {
	if !qi.FinishDate.IsZero() {
		return optional.From(qi.FinishDate)
	}
	d := qi.Remaining()
	if d.IsEmpty() {
		return optional.Optional[time.Time]{}
	}
	d2 := d.MustValue()
	if d2 == 0 {
		return optional.From(time.Time{})
	}
	return optional.From(time.Now().UTC().Add(d2))
}
