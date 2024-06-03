package model

const (
	EveDogmaAttributeImplantSlot          = 331
	EveDogmaAttributePrimarySkillID       = 182
	EveDogmaAttributePrimarySkillLevel    = 277
	EveDogmaAttributeSecondarySkillID     = 183
	EveDogmaAttributeSecondarySkillLevel  = 278
	EveDogmaAttributeTertiarySkillID      = 184
	EveDogmaAttributeTertiarySkillLevel   = 279
	EveDogmaAttributeQuaternarySkillID    = 1285
	EveDogmaAttributeQuaternarySkillLevel = 1286
	EveDogmaAttributeQuinarySkillID       = 1289
	EveDogmaAttributeQuinarySkillLevel    = 1287
	EveDogmaAttributeSenarySkillID        = 1290
	EveDogmaAttributeSenarySkillLevel     = 1288
)

type EveDogmaAttribute struct {
	ID           int32
	DefaultValue float32
	Description  string
	DisplayName  string
	IconID       int32
	Name         string
	IsHighGood   bool
	IsPublished  bool
	IsStackable  bool
	UnitID       int32
}

type EveDogmaAttributeForType struct {
	EveTypeID      int32
	DogmaAttribute *EveDogmaAttribute
	Value          float32
}
