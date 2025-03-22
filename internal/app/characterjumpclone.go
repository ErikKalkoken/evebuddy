package app

type CharacterJumpClone struct {
	CharacterID int32
	ID          int64
	Implants    []*CharacterJumpCloneImplant
	JumpCloneID int32
	Location    *EntityShort[int64]
	Name        string
	Region      *EntityShort[int32]
}

type CharacterJumpClone2 struct {
	Character   *EntityShort[int32]
	ID          int64
	JumpCloneID int32
	Location    *EveLocation
}

type CharacterJumpCloneImplant struct {
	ID      int64
	EveType *EveType
	SlotNum int // 0 = unknown
}
