package model

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

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
	SkillName        string
	SkillDescription string
	TrainingStartSP  int
}

func (qi CharacterSkillqueueItem) IsActive() bool {
	now := time.Now()
	return !qi.StartDate.IsZero() && qi.StartDate.Before(now) && qi.FinishDate.After(now)
}

func (qi CharacterSkillqueueItem) IsCompleted() bool {
	return qi.CompletionP() == 1
}

func (qi CharacterSkillqueueItem) CompletionP() float64 {
	d := qi.Duration()
	if !d.Valid {
		return 0
	}
	duration := d.Duration
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

func (qi CharacterSkillqueueItem) Duration() optional.Duration {
	var d optional.Duration
	if qi.StartDate.IsZero() || qi.FinishDate.IsZero() {
		return d
	}
	d.Valid = true
	d.Duration = qi.FinishDate.Sub(qi.StartDate)
	return d
}

func (qi CharacterSkillqueueItem) Remaining() optional.Duration {
	var d optional.Duration
	if qi.StartDate.IsZero() || qi.FinishDate.IsZero() {
		return d
	}
	d.Valid = true
	remainingP := 1 - qi.CompletionP()
	d.Duration = time.Duration(float64(qi.Duration().Duration) * remainingP)
	return d
}
