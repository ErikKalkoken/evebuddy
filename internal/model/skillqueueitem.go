package model

import (
	"fmt"
	"time"
)

type SkillqueueItem struct {
	GroupName       string
	FinishDate      time.Time
	FinishedLevel   int
	LevelEndSP      int
	LevelStartSP    int
	MyCharacterID   int32
	QueuePosition   int
	StartDate       time.Time
	SkillName       string
	TrainingStartSP int
}

// Name returns the name of a skillqueue item.
func (q *SkillqueueItem) Name() string {
	return fmt.Sprintf("%s %s", q.SkillName, romanLetter(q.FinishedLevel))
}

func (q *SkillqueueItem) IsActive() bool {
	now := time.Now()
	return q.StartDate.Before(now) && q.FinishDate.After(now)
}

func (q *SkillqueueItem) CompletionP() float64 {
	now := time.Now()
	if q.FinishDate.Before(now) {
		return 1
	}
	if q.StartDate.After(now) {
		return 0
	}
	if q.Duration() == 0 {
		return 0
	}
	remaining := q.FinishDate.Sub(now)
	return 1 - remaining.Seconds()/q.Duration().Seconds()
}

func (q *SkillqueueItem) Duration() time.Duration {
	return q.FinishDate.Sub(q.StartDate)
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
