package model

type CharacterAsset struct {
	ID              int64
	CharacterID     int32
	EveCategoryID   int32
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

func (ca CharacterAsset) NamePlus() string {
	if ca.Name != "" {
		return ca.Name
	}
	return ca.EveType.Name
}

func (ca CharacterAsset) IsBPO() bool {
	return ca.EveCategoryID == EveCategoryBlueprint && !ca.IsBlueprintCopy
}

func (ca CharacterAsset) IsSKIN() bool {
	return ca.EveCategoryID == EveCategorySKINs
}

type CharacterAssetLocation struct {
	ID           int64
	CharacterID  int32
	Location     *EntityShort[int64]
	LocationType string
	SolarSystem  *EntityShort[int32]
}
