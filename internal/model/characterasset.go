package model

import (
	"database/sql"
	"fmt"
)

type EveTypeVariant uint

const (
	VariantRegular EveTypeVariant = iota
	VariantBPO
	VariantBPC
	VariantSKIN
)

type CharacterAsset struct {
	ID              int64
	CharacterID     int32
	EveType         *EveType
	IsBlueprintCopy bool
	IsSingleton     bool
	ItemID          int64
	LocationFlag    string
	LocationID      int64
	LocationType    string
	Name            string
	Quantity        int32
	Price           sql.NullFloat64
}

func (ca CharacterAsset) DisplayName() string {
	if ca.Name != "" {
		return ca.Name
	}
	s := ca.EveType.Name
	if ca.IsBlueprintCopy {
		s += " (Copy)"
	}
	return s
}

func (ca CharacterAsset) DisplayName2() string {
	if ca.Name != "" {
		return fmt.Sprintf("%s \"%s\"", ca.EveType.Name, ca.Name)
	}
	s := ca.EveType.Name
	if ca.IsBlueprintCopy {
		s += " (Copy)"
	}
	return s
}

func (ca CharacterAsset) IsBPO() bool {
	return ca.EveType.Group.Category.ID == EveCategoryBlueprint && !ca.IsBlueprintCopy
}

func (ca CharacterAsset) IsSKIN() bool {
	return ca.EveType.Group.Category.ID == EveCategorySKINs
}

func (ca CharacterAsset) IsContainer() bool {
	if !ca.IsSingleton {
		return false
	}
	if ca.EveType.Group.Category.ID == EveCategoryShip {
		return true
	}
	switch ca.EveType.Group.ID {
	case EveGroupAuditLogFreightContainer,
		EveGroupAuditLogSecureCargoContainer,
		EveGroupCargoContainer,
		EveGroupSecureCargoContainer:
		return true
	}
	return false
}

func (ca CharacterAsset) Variant() EveTypeVariant {
	if ca.IsSKIN() {
		return VariantSKIN
	} else if ca.IsBPO() {
		return VariantBPO
	} else if ca.IsBlueprintCopy {
		return VariantBPO
	}
	return VariantRegular
}

type CharacterAssetLocation struct {
	ID             int64
	CharacterID    int32
	Location       *EntityShort[int64]
	LocationType   string
	SolarSystem    *EntityShort[int32]
	SecurityStatus sql.NullFloat64
}
