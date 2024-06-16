package app

import "github.com/ErikKalkoken/evebuddy/internal/optional"

type CharacterShipAbility struct {
	Type   EntityShort[int32]
	Group  EntityShort[int32]
	CanFly bool
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

type CharacterShipSkill struct {
	ActiveSkillLevel  optional.Int
	ID                int64
	CharacterID       int32
	Rank              uint
	ShipTypeID        int32
	SkillTypeID       int32
	SkillName         string
	SkillLevel        uint
	TrainedSkillLevel optional.Int
}
