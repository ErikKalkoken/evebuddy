package app

import (
	"math"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
)

const (
	EveCategoryBlueprint  = 9
	EveCategoryDrone      = 18
	EveCategoryDeployable = 22
	EveCategoryFighter    = 87
	EveCategoryOrbitals   = 46
	EveCategoryShip       = 6
	EveCategorySkill      = 16
	EveCategorySKINs      = 91
	EveCategoryStarbase   = 23
	EveCategoryStation    = 3
	EveCategoryStructure  = 65
)

// EveCategory is a category in Eve Online.
type EveCategory struct {
	ID          int32
	IsPublished bool
	Name        string
}

const (
	EveGroupAuditLogFreightContainer     = 649
	EveGroupAuditLogSecureCargoContainer = 448
	EveGroupBlackOps                     = 898
	EveGroupCapitalIndustrialShip        = 883
	EveGroupCargoContainer               = 12
	EveGroupCarrier                      = 547
	EveGroupDreadnought                  = 485
	EveGroupExtractorControlUnits        = 1063
	EveGroupForceAuxiliary               = 1538
	EveGroupJumpFreighter                = 902
	EveGroupPlanet                       = 7
	EveGroupProcessors                   = 1028
	EveGroupSecureCargoContainer         = 340
	EveGroupSuperCarrier                 = 659
	EveGroupTitan                        = 30
)

// EveGroup is a group in Eve Online.
type EveGroup struct {
	ID          int32
	Category    *EveCategory
	IsPublished bool
	Name        string
}

const (
	EveTypeAssetSafetyWrap             = 60
	EveTypeIHUB                        = 32458
	EveTypeInfomorphSynchronizing      = 33399
	EveTypeInterplanetaryConsolidation = 2495
	EveTypePlanetTemperate             = 11
	EveTypeSolarSystem                 = 5
	EveTypeTCU                         = 32226

	EveTypeIndustry                    = 3380
	EveTypeMassProduction              = 3387
	EveTypeAdvancedMassProduction      = 24625
	EveTypeLaboratoryOperation         = 3406
	EveTypeAdvancedLaboratoryOperation = 24624
	EveTypeMassReactions               = 45748
	EveTypeAdvancedMassReactions       = 45749
)

// EveType is a type in Eve Online.
type EveType struct {
	ID             int32
	Group          *EveGroup
	Capacity       float32
	Description    string
	GraphicID      int32
	IconID         int32
	IsPublished    bool
	MarketGroupID  int32
	Mass           float32
	Name           string
	PackagedVolume float32
	PortionSize    int
	Radius         float32
	Volume         float32
}

func (et EveType) DescriptionPlain() string {
	return evehtml.ToPlain(et.Description)
}

func (et EveType) IsBlueprint() bool {
	return et.Group.Category.ID == EveCategoryBlueprint
}

func (et EveType) IsShip() bool {
	return et.Group.Category.ID == EveCategoryShip
}

func (et EveType) IsSKIN() bool {
	return et.Group.Category.ID == EveCategorySKINs
}

func (et EveType) IsTradeable() bool {
	return et.MarketGroupID != 0
}

func (et EveType) HasFuelBay() bool {
	if et.Group.Category.ID != EveCategoryShip {
		return false
	}
	switch et.Group.ID {
	case EveGroupBlackOps,
		EveGroupCapitalIndustrialShip,
		EveGroupCarrier,
		EveGroupDreadnought,
		EveGroupForceAuxiliary,
		EveGroupJumpFreighter,
		EveGroupSuperCarrier,
		EveGroupTitan:
		return true
	}
	return false
}

func (et EveType) HasRender() bool {
	switch et.Group.Category.ID {
	case
		EveCategoryDrone,
		EveCategoryDeployable,
		EveCategoryFighter,
		EveCategoryShip,
		EveCategoryStation,
		EveCategoryStructure,
		EveCategoryStarbase:
		return true
	}
	return false
}

// Icon returns the icon for a type from the eveicon package
// and whether and icon exists for this type.
func (et EveType) Icon() (fyne.Resource, bool) {
	if et.IconID == 0 {
		return nil, false
	}
	res, ok := eveicon.FromID(et.IconID)
	if !ok {
		return nil, false
	}
	return res, true
}

func (et EveType) EveEntity() *EveEntity {
	return &EveEntity{ID: et.ID, Name: et.Name, Category: EveEntityInventoryType}
}

type EveTypeDogmaAttribute struct {
	EveType        *EveType
	DogmaAttribute *EveDogmaAttribute
	Value          float32
}

const (
	EveDogmaAttributeArmorEMDamageResistance             = 267
	EveDogmaAttributeArmorExplosiveDamageResistance      = 268
	EveDogmaAttributeArmorHitpoints                      = 265
	EveDogmaAttributeArmorKineticDamageResistance        = 269
	EveDogmaAttributeArmorThermalDamageResistance        = 270
	EveDogmaAttributeCalibration                         = 1132
	EveDogmaAttributeCapacitorCapacity                   = 482
	EveDogmaAttributeCapacitorRechargeTime               = 55
	EveDogmaAttributeCapacitorWarfareResistance          = 2045
	EveDogmaAttributeCapacitorWarfareResistanceBonus     = 2267
	EveDogmaAttributeCapacity                            = 38
	EveDogmaAttributeCargoScanResistance                 = 188
	EveDogmaAttributeCharisma                            = 164
	EveDogmaAttributeCPUOutput                           = 48
	EveDogmaAttributeCPUusage                            = 50
	EveDogmaAttributeDroneBandwidth                      = 1271
	EveDogmaAttributeDroneCapacity                       = 283
	EveDogmaAttributeECMResistance                       = 2253
	EveDogmaAttributeEntosisAssistanceImpedance          = 2754
	EveDogmaAttributeFighterHangarCapacity               = 2055
	EveDogmaAttributeFighterSquadronLaunchTubes          = 2216
	EveDogmaAttributeFuelBayCapacity                     = 1549
	EveDogmaAttributeGravimetricSensorStrength           = 211
	EveDogmaAttributeHeavyFighterSquadronLimit           = 2219
	EveDogmaAttributeHighSlots                           = 14
	EveDogmaAttributeHullEmDamageResistance              = 974
	EveDogmaAttributeHullExplosiveDamageResistance       = 975
	EveDogmaAttributeHullKinDamageResistance             = 976
	EveDogmaAttributeHullThermDamageResistance           = 977
	EveDogmaAttributeImplantSlot                         = 331
	EveDogmaAttributeInertiaModifier                     = 70
	EveDogmaAttributeInertiaMultiplier                   = 70
	EveDogmaAttributeIntelligence                        = 165
	EveDogmaAttributeJumpDriveCapacitorNeed              = 898
	EveDogmaAttributeJumpDriveConsumptionAmount          = 868
	EveDogmaAttributeJumpDriveFuelNeed                   = 866
	EveDogmaAttributeLadarSensorStrength                 = 209
	EveDogmaAttributeLauncherHardpoints                  = 101
	EveDogmaAttributeLightFighterSquadronLimit           = 2217
	EveDogmaAttributeLowSlots                            = 12
	EveDogmaAttributeMagnetometricSensorStrength         = 210
	EveDogmaAttributeMass                                = 4
	EveDogmaAttributeMaximumJumpRange                    = 867
	EveDogmaAttributeMaximumLockedTargets                = 192
	EveDogmaAttributeMaximumTargetingRange               = 76
	EveDogmaAttributeMaximumVelocity                     = 37
	EveDogmaAttributeMaxVelocity                         = 37
	EveDogmaAttributeMediumSlots                         = 13
	EveDogmaAttributeMemory                              = 166
	EveDogmaAttributeMetaLevel                           = 633
	EveDogmaAttributeOnboardJumpDrive                    = 861
	EveDogmaAttributePerception                          = 167
	EveDogmaAttributePowergridOutput                     = 11
	EveDogmaAttributePowergridUsage                      = 30
	EveDogmaAttributePrimaryAttribute                    = 180
	EveDogmaAttributePrimarySkillID                      = 182
	EveDogmaAttributePrimarySkillLevel                   = 277
	EveDogmaAttributeQuaternarySkillID                   = 1285
	EveDogmaAttributeQuaternarySkillLevel                = 1286
	EveDogmaAttributeQuinarySkillID                      = 1289
	EveDogmaAttributeQuinarySkillLevel                   = 1287
	EveDogmaAttributeRADARSensorStrength                 = 208
	EveDogmaAttributeRemoteElectronicAssistanceImpedance = 2135
	EveDogmaAttributeRemoteLogisticsImpedance            = 2116
	EveDogmaAttributeReprocessingSkillType               = 790
	EveDogmaAttributeRequiredSkill1                      = 182
	EveDogmaAttributeRigSlots                            = 1154
	EveDogmaAttributeServiceSlots                        = 2056
	EveDogmaAttributeScanResolution                      = 564
	EveDogmaAttributeSecondaryAttribute                  = 181
	EveDogmaAttributeSecondarySkillID                    = 183
	EveDogmaAttributeSecondarySkillLevel                 = 278
	EveDogmaAttributeSenarySkillID                       = 1290
	EveDogmaAttributeSenarySkillLevel                    = 1288
	EveDogmaAttributeSensorWarfareResistance             = 2112
	EveDogmaAttributeShieldCapacity                      = 263
	EveDogmaAttributeShieldEMDamageResistance            = 271
	EveDogmaAttributeShieldExplosiveDamageResistance     = 272
	EveDogmaAttributeShieldKineticDamageResistance       = 273
	EveDogmaAttributeShieldRechargeTime                  = 479
	EveDogmaAttributeShieldThermalDamageResistance       = 274
	EveDogmaAttributeShipWarpSpeed                       = 1281
	EveDogmaAttributeSignatureRadius                     = 552
	EveDogmaAttributeStasisWebifierResistance            = 2115
	EveDogmaAttributeStasisWebifierResistanceBonus       = 3422
	EveDogmaAttributeStructureEMDamageResistance         = 113
	EveDogmaAttributeStructureExplosiveDamageResistance  = 111
	EveDogmaAttributeStructureHitpoints                  = 9
	EveDogmaAttributeStructureKineticDamageResistance    = 109
	EveDogmaAttributeStructureThermalDamageResistance    = 110
	EveDogmaAttributeSupportFighterSquadronLimit         = 2218
	EveDogmaAttributeTargetPainterResistance             = 2114
	EveDogmaAttributeTechLevel                           = 422
	EveDogmaAttributeTertiarySkillID                     = 184
	EveDogmaAttributeTertiarySkillLevel                  = 279
	EveDogmaAttributeTurretHardpoints                    = 102
	EveDogmaAttributeWarpSpeedMultiplier                 = 600
	EveDogmaAttributeWeaponDisruptionResistance          = 2113
	EveDogmaAttributeWillpower                           = 168
	EveDogmaAttributeCharismaModifier                    = 175
	EveDogmaAttributeIntelligenceModifier                = 176
	EveDogmaAttributeMemoryModifier                      = 177
	EveDogmaAttributePerceptionModifier                  = 178
	EveDogmaAttributeWillpowerModifier                   = 179
	EveDogmaAttributeTrainingTimeMultiplier              = 275
)

type EveUnitID uint

const (
	EveUnitNone                           EveUnitID = 0
	EveUnitLength                         EveUnitID = 1
	EveUnitMass                           EveUnitID = 2
	EveUnitTime                           EveUnitID = 3
	EveUnitElectricCurrent                EveUnitID = 4
	EveUnitTemperature                    EveUnitID = 5
	EveUnitAmountOfSubstance              EveUnitID = 6
	EveUnitLuminousIntensity              EveUnitID = 7
	EveUnitArea                           EveUnitID = 8
	EveUnitVolume                         EveUnitID = 9
	EveUnitSpeed                          EveUnitID = 10
	EveUnitAcceleration                   EveUnitID = 11
	EveUnitWaveNumber                     EveUnitID = 12
	EveUnitMassDensity                    EveUnitID = 13
	EveUnitSpecificVolume                 EveUnitID = 14
	EveUnitCurrentDensity                 EveUnitID = 15
	EveUnitMagneticFieldStrength          EveUnitID = 16
	EveUnitAmountOfSubstanceConcentration EveUnitID = 17
	EveUnitLuminance                      EveUnitID = 18
	EveUnitMassFraction                   EveUnitID = 19
	EveUnitMilliseconds                   EveUnitID = 101
	EveUnitMillimeters                    EveUnitID = 102
	EveUnitMegaPascals                    EveUnitID = 103
	EveUnitMultiplier                     EveUnitID = 104
	EveUnitPercentage                     EveUnitID = 105
	EveUnitTeraflops                      EveUnitID = 106
	EveUnitMegaWatts                      EveUnitID = 107
	EveUnitInverseAbsolutePercent         EveUnitID = 108
	EveUnitModifierPercent                EveUnitID = 109
	EveUnitInverseModifierPercent         EveUnitID = 111
	EveUnitRadiansPerSecond               EveUnitID = 112
	EveUnitHitpoints                      EveUnitID = 113
	EveUnitCapacitorUnits                 EveUnitID = 114
	EveUnitGroupID                        EveUnitID = 115
	EveUnitTypeID                         EveUnitID = 116
	EveUnitSizeClass                      EveUnitID = 117
	EveUnitOreUnits                       EveUnitID = 118
	EveUnitAttributeID                    EveUnitID = 119
	EveUnitAttributePoints                EveUnitID = 120
	EveUnitRealPercent                    EveUnitID = 121
	EveUnitFittingSlots                   EveUnitID = 122
	EveUnitTrueTime                       EveUnitID = 123
	EveUnitModifierRelativePercent        EveUnitID = 124
	EveUnitNewton                         EveUnitID = 125
	EveUnitLightYear                      EveUnitID = 126
	EveUnitAbsolutePercent                EveUnitID = 127
	EveUnitDroneBandwidth                 EveUnitID = 128
	EveUnitHours                          EveUnitID = 129
	EveUnitMoney                          EveUnitID = 133
	EveUnitLogisticalCapacity             EveUnitID = 134
	EveUnitAstronomicalUnit               EveUnitID = 135
	EveUnitSlot                           EveUnitID = 136
	EveUnitBoolean                        EveUnitID = 137
	EveUnitUnits                          EveUnitID = 138
	EveUnitBonus                          EveUnitID = 139
	EveUnitLevel                          EveUnitID = 140
	EveUnitHardpoints                     EveUnitID = 141
	EveUnitSex                            EveUnitID = 142
	EveUnitDatetime                       EveUnitID = 143
	EveUnitWarpSpeed                      EveUnitID = 144 // inferred
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
	Unit         EveUnitID
}

type EveMarketPrice struct {
	TypeID        int32
	AdjustedPrice float64
	AveragePrice  float64
}

// Equal reports whether two objects are equal.
func (x EveMarketPrice) Equal(other EveMarketPrice) bool {
	return x.TypeID == other.TypeID &&
		math.Abs(float64(x.AdjustedPrice-other.AdjustedPrice)) < 0.001 &&
		math.Abs(float64(x.AveragePrice-other.AveragePrice)) < 0.001
}
