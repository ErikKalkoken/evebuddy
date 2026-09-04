package app

import (
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
