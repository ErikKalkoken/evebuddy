package model

import "time"

type SkillqueueItem struct {
	EveTypeID       int32
	FinishDate      time.Time
	FinishedLevel   int
	LevelEndSP      int
	LevelStartSP    int
	MyCharacterID   int32
	QueuePosition   int
	StartDate       time.Time
	TrainingStartSP int
}
