package model

import "time"

type SkillqueueItem struct {
	SkillName       string
	FinishDate      time.Time
	FinishedLevel   int
	LevelEndSP      int
	LevelStartSP    int
	MyCharacterID   int32
	QueuePosition   int
	StartDate       time.Time
	TrainingStartSP int
}
