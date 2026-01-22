package app

import (
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// LocationFlag represents a location flag for assets.
type LocationFlag uint

const (
	FlagUndefined LocationFlag = iota
	FlagAssetSafety
	FlagAutoFit
	FlagBonus
	FlagBooster
	FlagBoosterBay
	FlagCapsule
	FlagCapsuleerDeliveries
	FlagCargo
	FlagCorpDeliveries
	FlagCorporationGoalDeliveries
	FlagCorpSAG1
	FlagCorpSAG2
	FlagCorpSAG3
	FlagCorpSAG4
	FlagCorpSAG5
	FlagCorpSAG6
	FlagCorpSAG7
	FlagCorpseBay
	FlagCrateLoot
	FlagDeliveries
	FlagDroneBay
	FlagDustBattle
	FlagDustDatabank
	FlagExpeditionHold
	FlagFighterBay
	FlagFighterTube0
	FlagFighterTube1
	FlagFighterTube2
	FlagFighterTube3
	FlagFighterTube4
	FlagFleetHangar
	FlagFrigateEscapeBay
	FlagHangar
	FlagHangarAll
	FlagHiddenModifiers
	FlagHiSlot0
	FlagHiSlot1
	FlagHiSlot2
	FlagHiSlot3
	FlagHiSlot4
	FlagHiSlot5
	FlagHiSlot6
	FlagHiSlot7
	FlagImplant
	FlagImpounded
	FlagInfrastructureHangar
	FlagJunkyardReprocessed
	FlagJunkyardTrashed
	FlagLocked
	FlagLoSlot0
	FlagLoSlot1
	FlagLoSlot2
	FlagLoSlot3
	FlagLoSlot4
	FlagLoSlot5
	FlagLoSlot6
	FlagLoSlot7
	FlagMedSlot0
	FlagMedSlot1
	FlagMedSlot2
	FlagMedSlot3
	FlagMedSlot4
	FlagMedSlot5
	FlagMedSlot6
	FlagMedSlot7
	FlagMobileDepotHold
	FlagMoonMaterialBay
	FlagOfficeFolder
	FlagPilot
	FlagPlanetSurface
	FlagQuafeBay
	FlagQuantumCoreRoom
	FlagReward
	FlagRigSlot0
	FlagRigSlot1
	FlagRigSlot2
	FlagRigSlot3
	FlagRigSlot4
	FlagRigSlot5
	FlagRigSlot6
	FlagRigSlot7
	FlagSecondaryStorage
	FlagServiceSlot0
	FlagServiceSlot1
	FlagServiceSlot2
	FlagServiceSlot3
	FlagServiceSlot4
	FlagServiceSlot5
	FlagServiceSlot6
	FlagServiceSlot7
	FlagShipHangar
	FlagShipOffline
	FlagSkill
	FlagSkillInTraining
	FlagSpecializedAmmoHold
	FlagSpecializedAsteroidHold
	FlagSpecializedCommandCenterHold
	FlagSpecializedFuelBay
	FlagSpecializedGasHold
	FlagSpecializedIceHold
	FlagSpecializedIndustrialShipHold
	FlagSpecializedLargeShipHold
	FlagSpecializedMaterialBay
	FlagSpecializedMediumShipHold
	FlagSpecializedMineralHold
	FlagSpecializedOreHold
	FlagSpecializedPlanetaryCommoditiesHold
	FlagSpecializedSalvageHold
	FlagSpecializedShipHold
	FlagSpecializedSmallShipHold
	FlagStructureActive
	FlagStructureDeedBay
	FlagStructureFuel
	FlagStructureInactive
	FlagStructureOffline
	FlagSubSystemBay
	FlagSubSystemSlot0
	FlagSubSystemSlot1
	FlagSubSystemSlot2
	FlagSubSystemSlot3
	FlagSubSystemSlot4
	FlagSubSystemSlot5
	FlagSubSystemSlot6
	FlagSubSystemSlot7
	FlagUnlocked
	FlagWallet
	FlagWardrobe
	FlagUnknown
)

// LocationType represents a location type for assets.
type LocationType uint

const (
	TypeUndefined LocationType = iota
	TypeItem
	TypeOther
	TypeSolarSystem
	TypeStation
	TypeUnknown
)

// InventoryTypeVariant represents a variant of inventory types for asset items.
type InventoryTypeVariant uint

const (
	VariantRegular InventoryTypeVariant = iota
	VariantBPO
	VariantBPC
	VariantSKIN
)

// Asset represents a generic asset in Eve Online.
type Asset struct {
	IsBlueprintCopy bool
	IsSingleton     bool
	ItemID          int64
	LocationFlag    LocationFlag
	LocationID      int64
	LocationType    LocationType
	Name            string
	Price           optional.Optional[float64]
	Quantity        int
	Type            *EveType
}

// ID returns the item ID.
func (ca Asset) ID() int64 {
	return ca.ItemID
}

func (ca Asset) Unwrap() Asset {
	return ca
}

func (ca Asset) CanHaveName() bool {
	if !ca.IsSingleton {
		return false
	}
	switch ca.Type.Group.Category.ID {
	case
		EveCategoryDeployable,
		EveCategoryShip,
		EveCategoryStructure:
		return true
	}
	switch ca.Type.Group.ID {
	case
		EveGroupAuditLogSecureContainer,
		EveGroupBiomass,
		EveGroupCargoContainer,
		EveGroupFreightContainer,
		EveGroupSecureCargoContainer:
		return true
	}
	return false
}

func (ca Asset) DisplayName() string {
	if ca.Name != "" {
		return ca.Name
	}
	s := ca.TypeName()
	if ca.IsBlueprintCopy {
		s += " (Copy)"
	}
	return s
}

func (ca Asset) DisplayName2() string {
	if ca.Name != "" {
		return fmt.Sprintf("%s \"%s\"", ca.TypeName(), ca.Name)
	}
	s := ca.TypeName()
	if ca.IsBlueprintCopy {
		s += " (Copy)"
	}
	return s
}

func (ca Asset) IsBPO() bool {
	return ca.Type.Group.Category.ID == EveCategoryBlueprint && !ca.IsBlueprintCopy
}

func (ca Asset) IsSKIN() bool {
	return ca.Type.Group.Category.ID == EveCategorySKINs
}

func (ca Asset) IsContainer() bool {
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
	case EveGroupFreightContainer,
		EveGroupAuditLogSecureContainer,
		EveGroupCargoContainer,
		EveGroupSecureCargoContainer:
		return true
	}
	return false
}

func (ca Asset) LocationCategory() LocationFlag {
	return FlagUndefined
}

func (ca Asset) IsInAssetSafety() bool {
	return ca.LocationFlag == FlagAssetSafety
}

func (ca Asset) IsInAnyCargoHold() bool {
	switch ca.LocationFlag {
	case
		FlagCargo,
		FlagFleetHangar,
		FlagMobileDepotHold,
		FlagMoonMaterialBay,
		FlagQuafeBay,
		FlagSpecializedAmmoHold,
		FlagSpecializedAsteroidHold,
		FlagSpecializedCommandCenterHold,
		FlagSpecializedGasHold,
		FlagSpecializedIceHold,
		FlagSpecializedIndustrialShipHold,
		FlagSpecializedLargeShipHold,
		FlagSpecializedMaterialBay,
		FlagSpecializedMediumShipHold,
		FlagSpecializedMineralHold,
		FlagSpecializedOreHold,
		FlagSpecializedPlanetaryCommoditiesHold,
		FlagSpecializedSalvageHold,
		FlagSpecializedShipHold,
		FlagSpecializedSmallShipHold,
		FlagStructureDeedBay:
		return true
	}
	return false
}

func (ca Asset) IsInDroneBay() bool {
	return ca.LocationFlag == FlagDroneBay
}

func (ca Asset) IsInFighterBay() bool {
	switch ca.LocationFlag {
	case
		FlagFighterBay,
		FlagFighterTube0,
		FlagFighterTube1,
		FlagFighterTube2,
		FlagFighterTube3,
		FlagFighterTube4:
		return true
	}
	return false
}

func (ca Asset) IsInFrigateEscapeBay() bool {
	return ca.LocationFlag == FlagFrigateEscapeBay
}

func (ca Asset) IsInFuelBay() bool {
	return ca.LocationFlag == FlagSpecializedFuelBay
}

func (ca Asset) IsInSpace() bool {
	return ca.LocationType == TypeSolarSystem
}

func (ca Asset) IsInHangar() bool {
	return ca.LocationFlag == FlagHangar
}

func (ca Asset) IsFitted() bool {
	switch ca.LocationFlag {
	case
		FlagHiSlot0,
		FlagHiSlot1,
		FlagHiSlot2,
		FlagHiSlot3,
		FlagHiSlot4,
		FlagHiSlot5,
		FlagHiSlot6,
		FlagHiSlot7:
		return true
	case
		FlagMedSlot0,
		FlagMedSlot1,
		FlagMedSlot2,
		FlagMedSlot3,
		FlagMedSlot4,
		FlagMedSlot5,
		FlagMedSlot6,
		FlagMedSlot7:
		return true
	case
		FlagLoSlot0,
		FlagLoSlot1,
		FlagLoSlot2,
		FlagLoSlot3,
		FlagLoSlot4,
		FlagLoSlot5,
		FlagLoSlot6,
		FlagLoSlot7:
		return true
	case
		FlagRigSlot0,
		FlagRigSlot1,
		FlagRigSlot2,
		FlagRigSlot3,
		FlagRigSlot4,
		FlagRigSlot5,
		FlagRigSlot6,
		FlagRigSlot7:
		return true
	case
		FlagSubSystemSlot0,
		FlagSubSystemSlot1,
		FlagSubSystemSlot2,
		FlagSubSystemSlot3,
		FlagSubSystemSlot4,
		FlagSubSystemSlot5,
		FlagSubSystemSlot6,
		FlagSubSystemSlot7:
	}
	return false
}

func (ca Asset) IsShipOther() bool {
	return !ca.IsInAnyCargoHold() &&
		!ca.IsInDroneBay() &&
		!ca.IsInFighterBay() &&
		!ca.IsInFuelBay() &&
		!ca.IsFitted() &&
		!ca.IsInFrigateEscapeBay()
}

// QuantityFiltered returns the quantity for items which are not inside a container
// and reports whether this item should be counted.
func (ca Asset) QuantityFiltered() (int, bool) {
	if ca.IsFitted() ||
		ca.IsInDroneBay() ||
		ca.IsInFrigateEscapeBay() ||
		ca.IsInFighterBay() ||
		ca.IsInFuelBay() ||
		ca.IsInAnyCargoHold() {
		return 0, false
	}
	return int(ca.Quantity), true
}

func (ca Asset) TypeName() string {
	if ca.Type == nil {
		return ""
	}
	return ca.Type.Name
}

func (ca Asset) Variant() InventoryTypeVariant {
	if ca.IsSKIN() {
		return VariantSKIN
	}
	if ca.IsBPO() {
		return VariantBPO
	}
	if ca.IsBlueprintCopy {
		return VariantBPC
	}
	return VariantRegular
}

type CharacterAsset struct {
	Asset
	CharacterID int32
}

type CorporationAsset struct {
	Asset
	CorporationID int32
}
