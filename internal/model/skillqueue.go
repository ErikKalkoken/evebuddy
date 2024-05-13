package model

import (
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/types"
)

type SkillqueueItem struct {
	GroupName        string
	FinishDate       time.Time
	FinishedLevel    int
	LevelEndSP       int
	LevelStartSP     int
	MyCharacterID    int32
	QueuePosition    int
	StartDate        time.Time
	SkillName        string
	SkillDescription string
	TrainingStartSP  int
}

// Name returns the name of a skillqueue item.
func (q *SkillqueueItem) Name() string {
	return fmt.Sprintf("%s %s", q.SkillName, romanLetter(q.FinishedLevel))
}

func (q *SkillqueueItem) IsActive() bool {
	now := time.Now()
	return !q.StartDate.IsZero() && q.StartDate.Before(now) && q.FinishDate.After(now)
}

func (q *SkillqueueItem) IsCompleted() bool {
	return q.CompletionP() == 1
}

func (q *SkillqueueItem) CompletionP() float64 {
	d := q.Duration()
	if !d.Valid {
		return 0
	}
	duration := d.Duration
	now := time.Now()
	if q.FinishDate.Before(now) {
		return 1
	}
	if q.StartDate.After(now) {
		return 0
	}
	if duration == 0 {
		return 0
	}
	remaining := q.FinishDate.Sub(now)
	c := remaining.Seconds() / duration.Seconds()
	base := float64(q.LevelEndSP-q.TrainingStartSP) / float64(q.LevelEndSP-q.LevelStartSP)
	return 1 - (c * base)
}

func (q *SkillqueueItem) Duration() types.NullDuration {
	var d types.NullDuration
	if q.StartDate.IsZero() || q.FinishDate.IsZero() {
		return d
	}
	d.Valid = true
	d.Duration = q.FinishDate.Sub(q.StartDate)
	return d
}

func (q *SkillqueueItem) Remaining() types.NullDuration {
	var d types.NullDuration
	if q.StartDate.IsZero() || q.FinishDate.IsZero() {
		return d
	}
	d.Valid = true
	remainingP := 1 - q.CompletionP()
	d.Duration = time.Duration(float64(q.Duration().Duration) * remainingP)
	return d
}

func romanLetter(v int) string {
	m := map[int]string{
		1: "I",
		2: "II",
		3: "III",
		4: "IV",
		5: "V",
	}
	r, ok := m[v]
	if !ok {
		panic(fmt.Sprintf("invalid value: %d", v))
	}
	return r
}
