package model

type CharacterAsset struct {
	ID              int64
	CharacterID     int32
	EveType         *EntityShort[int32]
	IsBlueprintCopy bool
	IsSingleton     bool
	ItemID          int64
	LocationFlag    string
	LocationID      int64
	LocationType    string
	Name            string
	Quantity        int32
}
