package app

import (
	"fmt"
	"strings"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type EveTypeVariant uint

const (
	VariantRegular EveTypeVariant = iota
	VariantBPO
	VariantBPC
	VariantSKIN
)

type CharacterAsset struct {
	CharacterID     int32
	ID              int64
	IsBlueprintCopy bool
	IsSingleton     bool
	ItemID          int64
	LocationFlag    string
	LocationID      int64
	LocationType    string
	Name            string
	Price           optional.Optional[float64]
	Quantity        int32
	Type            *EveType
}

func (ca CharacterAsset) DisplayName() string {
	if ca.Name != "" {
		return ca.Name
	}
	s := ca.TypeName()
	if ca.IsBlueprintCopy {
		s += " (Copy)"
	}
	return s
}

func (ca CharacterAsset) DisplayName2() string {
	if ca.Name != "" {
		return fmt.Sprintf("%s \"%s\"", ca.TypeName(), ca.Name)
	}
	s := ca.TypeName()
	if ca.IsBlueprintCopy {
		s += " (Copy)"
	}
	return s
}

func (ca CharacterAsset) TypeName() string {
	if ca.Type == nil {
		return ""
	}
	return ca.Type.Name
}

func (ca CharacterAsset) IsBPO() bool {
	return ca.Type.Group.Category.ID == EveCategoryBlueprint && !ca.IsBlueprintCopy
}

func (ca CharacterAsset) IsSKIN() bool {
	return ca.Type.Group.Category.ID == EveCategorySKINs
}

func (ca CharacterAsset) IsContainer() bool {
	if !ca.IsSingleton {
		return false
	}
	if ca.Type.IsShip() {
		return true
	}
	if ca.Type.ID == EveTypeAssetSafetyWrap {
		return true
	}
	switch ca.Type.Group.ID {
	case EveGroupAuditLogFreightContainer,
		EveGroupAuditLogSecureCargoContainer,
		EveGroupCargoContainer,
		EveGroupSecureCargoContainer:
		return true
	}
	return false
}

func (ca CharacterAsset) InAssetSafety() bool {
	return ca.LocationFlag == "AssetSafety"
}

func (ca CharacterAsset) InCargoBay() bool {
	return ca.LocationFlag == "Cargo"
}

func (ca CharacterAsset) InDroneBay() bool {
	return ca.LocationFlag == "DroneBay"
}

func (ca CharacterAsset) InFighterBay() bool {
	return ca.LocationFlag == "FighterBay" || strings.HasPrefix(ca.LocationFlag, "FighterTube")
}

func (ca CharacterAsset) InFrigateEscapeBay() bool {
	return ca.LocationFlag == "FrigateEscapeBay"
}

func (ca CharacterAsset) IsInFuelBay() bool {
	return ca.LocationFlag == "SpecializedFuelBay"
}

func (ca CharacterAsset) IsInSpace() bool {
	return ca.LocationType == "solar_system"
}

func (ca CharacterAsset) IsInHangar() bool {
	return ca.LocationFlag == "Hangar"
}

func (ca CharacterAsset) IsFitted() bool {
	switch s := ca.LocationFlag; {
	case strings.HasPrefix(s, "HiSlot"):
		return true
	case strings.HasPrefix(s, "MedSlot"):
		return true
	case strings.HasPrefix(s, "LoSlot"):
		return true
	case strings.HasPrefix(s, "RigSlot"):
		return true
	case strings.HasPrefix(s, "SubSystemSlot"):
		return true
	}
	return false
}

func (ca CharacterAsset) IsShipOther() bool {
	return !ca.InCargoBay() &&
		!ca.InDroneBay() &&
		!ca.InFighterBay() &&
		!ca.IsInFuelBay() &&
		!ca.IsFitted() &&
		!ca.InFrigateEscapeBay()
}

func (ca CharacterAsset) Variant() EveTypeVariant {
	if ca.IsSKIN() {
		return VariantSKIN
	} else if ca.IsBPO() {
		return VariantBPO
	} else if ca.IsBlueprintCopy {
		return VariantBPC
	}
	return VariantRegular
}
