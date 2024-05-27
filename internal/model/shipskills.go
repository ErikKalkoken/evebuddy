package model

type CharacterShipAbility struct {
	Type   EntityShort[int32]
	Group  EntityShort[int32]
	CanFly bool
}

type ShipSkill struct {
	ID          int64
	Rank        uint
	ShipTypeID  int32
	SkillTypeID int32
	SkillLevel  uint
}
