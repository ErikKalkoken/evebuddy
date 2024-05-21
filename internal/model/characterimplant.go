package model

type CharacterImplant struct {
	CharacterID int32
	EveType     *EveType
	ID          int64
	SlotNum     int // 0 = unknown
}
