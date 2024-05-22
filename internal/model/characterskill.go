package model

import (
	"time"
)

type CharacterAttribute struct {
	ID            int64
	BonusRemaps   int
	CharacterID   int32
	Charisma      int
	Intelligence  int
	LastRemapDate time.Time
	Memory        int
	Perception    int
	Willpower     int
}

type CharacterSkill struct {
	ActiveSkillLevel   int
	CharacterID        int32
	EveType            *EveType
	ID                 int64
	SkillPointsInSkill int
	TrainedSkillLevel  int
}

type ListCharacterSkillGroupProgress struct {
	GroupID   int32
	GroupName string
	Total     float64
	Trained   float64
}

type ListCharacterSkillProgress struct {
	ActiveSkillLevel  int
	TrainedSkillLevel int
	TypeID            int32
	TypeDescription   string
	TypeName          string
}
