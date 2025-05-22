package app

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type EveAlliance struct {
	CreatorCorporation  *EveEntity
	Creator             *EveEntity
	DateFounded         time.Time
	ExecutorCorporation *EveEntity
	Faction             *EveEntity
	ID                  int32
	Name                string
	Ticker              string
}

func (x EveAlliance) ToEveEntity() *EveEntity {
	return &EveEntity{ID: x.ID, Name: x.Name, Category: EveEntityAlliance}
}

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

// TODO: Add Bloodline (e.g. to show in character description)

// An Eve Online character.
type EveCharacter struct {
	Alliance       *EveEntity
	Birthday       time.Time
	Corporation    *EveEntity
	Description    string
	Faction        *EveEntity
	Gender         string
	ID             int32
	Name           string
	Race           *EveRace
	SecurityStatus float64
	Title          string
}

func (ec EveCharacter) AllianceName() string {
	if !ec.HasAlliance() {
		return ""
	}
	return ec.Alliance.Name
}

func (ec EveCharacter) FactionName() string {
	if !ec.HasFaction() {
		return ""
	}
	return ec.Faction.Name
}

// HasAlliance reports whether the character is member of an alliance.
func (ec EveCharacter) HasAlliance() bool {
	return ec.Alliance != nil
}

// HasFaction reports whether the character is member of a faction.
func (ec EveCharacter) HasFaction() bool {
	return ec.Faction != nil
}

func (ec EveCharacter) DescriptionPlain() string {
	return evehtml.ToPlain(ec.Description)
}

func (ec EveCharacter) RaceDescription() string {
	if ec.Race == nil {
		return ""
	}
	return ec.Race.Description
}

func (ec EveCharacter) ToEveEntity() *EveEntity {
	return &EveEntity{ID: ec.ID, Name: ec.Name, Category: EveEntityCharacter}
}

// EveConstellation is a constellation in Eve Online.
type EveConstellation struct {
	ID     int32
	Name   string
	Region *EveRegion
}

func (ec EveConstellation) ToEveEntity() *EveEntity {
	return &EveEntity{ID: ec.ID, Name: ec.Name, Category: EveEntityConstellation}
}

// An Eve Online corporation.
type EveCorporation struct {
	Alliance    *EveEntity
	Ceo         *EveEntity
	Creator     *EveEntity
	DateFounded optional.Optional[time.Time]
	Description string
	Faction     *EveEntity
	HomeStation *EveEntity
	ID          int32
	MemberCount int
	Name        string
	Shares      optional.Optional[int]
	TaxRate     float32
	Ticker      string
	URL         string
	WarEligible bool
	Timestamp   time.Time
}

func (ec EveCorporation) HasAlliance() bool {
	return ec.Alliance != nil
}

func (ec EveCorporation) HasFaction() bool {
	return ec.Faction != nil
}

func (ec EveCorporation) DescriptionPlain() string {
	return evehtml.ToPlain(ec.Description)
}

func (ec EveCorporation) ToEveEntity() *EveEntity {
	return &EveEntity{ID: ec.ID, Name: ec.Name, Category: EveEntityCorporation}
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

const (
	npcCorporationIDBegin = 1_000_000
	npcCorporationIDEnd   = 2_000_000
	npcCharacterIDBegin   = 3_000_000
	npcCharacterIDEnd     = 4_000_000
)

// An EveEntity in EveOnline.
type EveEntity struct {
	Category EveEntityCategory
	ID       int32
	Name     string
}

func (ee EveEntity) CategoryDisplay() string {
	titler := cases.Title(language.English)
	return titler.String(ee.Category.String())
}

// IsCharacter reports whether an entity is a character.
func (ee EveEntity) IsCharacter() bool {
	return ee.Category == EveEntityCharacter
}

// IsNPC reports whether an entity is a NPC.
//
// This function only works for characters and corporations and returns an empty value for anything else..
func (ee EveEntity) IsNPC() optional.Optional[bool] {
	switch ee.Category {
	case EveEntityCharacter:
		return optional.From(ee.ID >= npcCharacterIDBegin && ee.ID < npcCharacterIDEnd)
	case EveEntityCorporation:
		return optional.From(ee.ID >= npcCorporationIDBegin && ee.ID < npcCorporationIDEnd)
	}
	return optional.Optional[bool]{}
}

func (ee *EveEntity) Compare(other *EveEntity) int {
	return cmp.Compare(ee.Name, other.Name)
}

type EveEntityCategory int

// Supported categories of EveEntity
const (
	EveEntityUndefined EveEntityCategory = iota
	EveEntityAlliance
	EveEntityCharacter
	EveEntityConstellation
	EveEntityCorporation
	EveEntityFaction
	EveEntityInventoryType
	EveEntityMailList
	EveEntityRegion
	EveEntitySolarSystem
	EveEntityStation
	EveEntityUnknown
)

// IsKnown reports whether a category is known.
func (eec EveEntityCategory) IsKnown() bool {
	return eec != EveEntityUndefined && eec != EveEntityUnknown
}

func (eec EveEntityCategory) String() string {
	switch eec {
	case EveEntityUndefined:
		return "undefined"
	case EveEntityAlliance:
		return "alliance"
	case EveEntityCharacter:
		return "character"
	case EveEntityConstellation:
		return "constellation"
	case EveEntityCorporation:
		return "corporation"
	case EveEntityFaction:
		return "faction"
	case EveEntityInventoryType:
		return "inventory type"
	case EveEntityMailList:
		return "mailing list"
	case EveEntityRegion:
		return "region"
	case EveEntitySolarSystem:
		return "solar system"
	case EveEntityStation:
		return "station"
	case EveEntityUnknown:
		return "unknown"
	default:
		return "?"
	}
}

// ToEveImage returns the corresponding category string for the EveImage service.
// Will return an empty string when category is not supported.
func (eec EveEntityCategory) ToEveImage() string {
	switch eec {
	case EveEntityAlliance:
		return "alliance"
	case EveEntityCharacter:
		return "character"
	case EveEntityCorporation:
		return "corporation"
	case EveEntityFaction:
		return "faction"
	case EveEntityInventoryType:
		return "inventory_type"
	default:
		return ""
	}
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

type EveLocationVariant int

const (
	EveLocationUnknown EveLocationVariant = iota
	EveLocationAssetSafety
	EveLocationStation
	EveLocationStructure
	EveLocationSolarSystem
)

const LocationUnknownID = 888 // custom ID to signify a location that is not known

// EveLocation is a location in Eve Online.
type EveLocation struct {
	ID          int64
	SolarSystem *EveSolarSystem
	Type        *EveType
	Name        string
	Owner       *EveEntity
	UpdatedAt   time.Time
}

// DisplayName returns a user friendly name.
func (el EveLocation) DisplayName() string {
	if el.Name != "" {
		return el.Name
	}
	return el.alternativeName()
}

func (el EveLocation) DisplayRichText() []widget.RichTextSegment {
	var n string
	if el.Name != "" {
		n = el.Name
	} else {
		n = el.alternativeName()
	}
	if el.SolarSystem == nil {
		return iwidget.NewRichTextSegmentFromText(n)
	}
	return slices.Concat(
		el.SolarSystem.SecurityStatusRichText(),
		iwidget.NewRichTextSegmentFromText(fmt.Sprintf("  %s", n)))
}

// DisplayName2 returns a user friendly name not including the system name.
func (el EveLocation) DisplayName2() string {
	if el.Name != "" {
		if el.Variant() != EveLocationStructure {
			return el.Name
		}
		p := strings.Split(el.Name, " - ")
		if len(p) < 2 {
			return el.Name
		}
		return p[1]
	}
	return el.alternativeName()
}

func (el EveLocation) alternativeName() string {
	switch el.Variant() {
	case EveLocationUnknown:
		return "Unknown"
	case EveLocationAssetSafety:
		return "Asset Safety"
	case EveLocationSolarSystem:
		if el.SolarSystem == nil {
			return fmt.Sprintf("Unknown solar system #%d", el.ID)
		}
		return el.SolarSystem.Name
	case EveLocationStructure:
		return fmt.Sprintf("Unknown structure #%d", el.ID)
	}
	return fmt.Sprintf("Unknown location #%d", el.ID)
}

func (el EveLocation) Variant() EveLocationVariant {
	return LocationVariantFromID(el.ID)
}

func LocationVariantFromID(id int64) EveLocationVariant {
	switch {
	case id == LocationUnknownID:
		return EveLocationUnknown
	case id == 2004:
		return EveLocationAssetSafety
	case id >= 30_000_000 && id < 33_000_000:
		return EveLocationSolarSystem
	case id >= 60_000_000 && id < 64_000_000:
		return EveLocationStation
	case id >= 1_000_000_000_000:
		return EveLocationStructure
	default:
		return EveLocationUnknown
	}
}

func (el EveLocation) ToEveEntity() *EveEntity {
	switch el.Variant() {
	case EveLocationSolarSystem:
		return &EveEntity{ID: int32(el.ID), Name: el.Name, Category: EveEntitySolarSystem}
	case EveLocationStation:
		return &EveEntity{ID: int32(el.ID), Name: el.Name, Category: EveEntityStation}
	}
	return nil
}

func (el EveLocation) ToShort() *EveLocationShort {
	o := &EveLocationShort{
		ID:   el.ID,
		Name: optional.From(el.Name),
	}
	if el.SolarSystem != nil {
		o.SecurityStatus = optional.From(el.SolarSystem.SecurityStatus)
	}
	return o
}

// EveLocationShort is a shortened representation of EveLocation.
type EveLocationShort struct {
	ID             int64
	Name           optional.Optional[string]
	SecurityStatus optional.Optional[float32]
}

func (l EveLocationShort) DisplayName() string {
	return l.Name.ValueOrFallback("?")
}

func (l EveLocationShort) DisplayRichText() []widget.RichTextSegment {
	var s []widget.RichTextSegment
	if !l.SecurityStatus.IsEmpty() {
		secValue := l.SecurityStatus.MustValue()
		secType := NewSolarSystemSecurityTypeFromValue(secValue)
		s = slices.Concat(s, iwidget.NewRichTextSegmentFromText(
			fmt.Sprintf("%.1f", secValue),
			widget.RichTextStyle{ColorName: secType.ToColorName(), Inline: true},
		))
	}
	var name string
	if len(s) > 0 {
		name += "   "
	}
	name += humanize.Optional(l.Name, "?")
	s = slices.Concat(s, iwidget.NewRichTextSegmentFromText(name))
	return s
}

func (l EveLocationShort) SecurityType() optional.Optional[SolarSystemSecurityType] {
	if l.SecurityStatus.IsEmpty() {
		return optional.Optional[SolarSystemSecurityType]{}
	}
	return optional.From(NewSolarSystemSecurityTypeFromValue(l.SecurityStatus.MustValue()))
}

type EveMarketPrice struct {
	TypeID        int32
	AdjustedPrice float64
	AveragePrice  float64
}

// EveMoon is a moon in Eve Online.
type EveMoon struct {
	ID          int32
	Name        string
	SolarSystem *EveSolarSystem
}

var rePlanetType = regexp.MustCompile(`Planet \((\S*)\)`)

// EvePlanet is a planet in Eve Online.
type EvePlanet struct {
	ID          int32
	Name        string
	SolarSystem *EveSolarSystem
	Type        *EveType
}

func (ep EvePlanet) TypeDisplay() string {
	if ep.Type == nil {
		return ""
	}
	m := rePlanetType.FindStringSubmatch(ep.Type.Name)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// TODO: Add race alliance

// EveRace is a race in Eve Online.
type EveRace struct {
	Description string
	Name        string
	ID          int32
}

// FactionID returns the faction ID of a race.
func (er EveRace) FactionID() (int32, bool) {
	m := map[int32]int32{
		1:   500001,
		2:   500002,
		4:   500003,
		8:   500004,
		16:  500005,
		135: 500026,
	}
	factionID, ok := m[er.ID]
	return factionID, ok
}

// EveRegion is a region in Eve Online.
type EveRegion struct {
	Description string
	ID          int32
	Name        string
}

func (er EveRegion) DescriptionPlain() string {
	return evehtml.ToPlain(er.Description)
}

func (er EveRegion) ToEveEntity() *EveEntity {
	return &EveEntity{ID: er.ID, Name: er.Name, Category: EveEntityRegion}
}

// EveRouteHeader describes the header for a route in EVE Online.
type EveRouteHeader struct {
	Origin      *EveSolarSystem
	Destination *EveSolarSystem
	Preference  EveRoutePreference
}

func (x EveRouteHeader) String() string {
	var originID, destinationID int32
	if x.Origin != nil {
		originID = x.Origin.ID
	}
	if x.Destination != nil {
		destinationID = x.Destination.ID
	}
	return fmt.Sprintf("{Origin: %d Destination: %d Preference: %s}", originID, destinationID, x.Preference)
}

// EveRoutePreference represents the calculation preference when requesting a route from ESI.
type EveRoutePreference uint

const (
	RouteShortest EveRoutePreference = iota
	RouteSecure
	RouteInsecure
)

func (x EveRoutePreference) String() string {
	m := map[EveRoutePreference]string{
		RouteShortest: "shortest",
		RouteSecure:   "secure",
		RouteInsecure: "insecure",
	}
	return m[x]
}

func EveRoutePreferenceFromString(s string) EveRoutePreference {
	m := map[string]EveRoutePreference{
		"shortest": RouteShortest,
		"secure":   RouteSecure,
		"insecure": RouteInsecure,
	}
	return m[s]
}

func EveRoutePreferences() []EveRoutePreference {
	return []EveRoutePreference{RouteShortest, RouteSecure, RouteInsecure}
}

// EveSchematic is a schematic for planetary industry in Eve Online.
type EveSchematic struct {
	ID        int32
	CycleTime int
	Name      string
}

func (es EveSchematic) Icon() (fyne.Resource, bool) {
	return eveicon.FromSchematicID(es.ID)
}

type EveShipSkill struct {
	ID          int64
	Rank        uint
	ShipTypeID  int32
	SkillTypeID int32
	SkillName   string
	SkillLevel  uint
}

type SolarSystemSecurityType uint

const (
	NullSec SolarSystemSecurityType = iota
	LowSec
	HighSec
	SuperHighSec
)

// ToImportance returns the importance value for a security type.
func (t SolarSystemSecurityType) ToImportance() widget.Importance {
	switch t {
	case SuperHighSec:
		return widget.HighImportance
	case HighSec:
		return widget.SuccessImportance
	case LowSec:
		return widget.WarningImportance
	case NullSec:
		return widget.DangerImportance
	}
	return widget.MediumImportance
}

func (t SolarSystemSecurityType) ToColorName() fyne.ThemeColorName {
	switch t {
	case SuperHighSec:
		return theme.ColorNamePrimary
	case HighSec:
		return theme.ColorNameSuccess
	case LowSec:
		return theme.ColorNameWarning
	case NullSec:
		return theme.ColorNameError
	}
	return theme.ColorNameForeground
}

func NewSolarSystemSecurityTypeFromValue(v float32) SolarSystemSecurityType {
	switch {
	case v >= 0.9:
		return SuperHighSec
	case v >= 0.45:
		return HighSec
	case v > 0.0:
		return LowSec
	}
	return NullSec
}

// EveSolarSystem is a solar system in Eve Online.
type EveSolarSystem struct {
	Constellation  *EveConstellation
	ID             int32
	Name           string
	SecurityStatus float32
}

func (es EveSolarSystem) IsWormholeSpace() bool {
	return es.ID >= 31000000
}

func (es EveSolarSystem) SecurityType() SolarSystemSecurityType {
	return NewSolarSystemSecurityTypeFromValue(es.SecurityStatus)
}

func (es EveSolarSystem) SecurityStatusDisplay() string {
	return fmt.Sprintf("%.1f", es.SecurityStatus)
}

func (es EveSolarSystem) SecurityStatusRichText() []widget.RichTextSegment {
	return []widget.RichTextSegment{&widget.TextSegment{
		Text: es.SecurityStatusDisplay(),
		Style: widget.RichTextStyle{
			ColorName: es.SecurityType().ToColorName(),
			Inline:    true,
		},
	}}
}

func (es EveSolarSystem) ToEveEntity() *EveEntity {
	return &EveEntity{ID: es.ID, Name: es.Name, Category: EveEntitySolarSystem}
}

func (es EveSolarSystem) DisplayRichText() []widget.RichTextSegment {
	return slices.Concat(
		es.SecurityStatusRichText(),
		iwidget.NewRichTextSegmentFromText(fmt.Sprintf("  %s", es.Name)),
	)
}

func (es EveSolarSystem) DisplayRichTextWithRegion() []widget.RichTextSegment {
	return slices.Concat(
		es.SecurityStatusRichText(),
		iwidget.NewRichTextSegmentFromText(fmt.Sprintf("  %s (%s)", es.Name, es.Constellation.Region.Name)),
	)
}

type EveSolarSystemPlanet struct {
	AsteroidBeltIDs []int32
	MoonIDs         []int32
	PlanetID        int32
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

func (et EveType) ToEveEntity() *EveEntity {
	return &EveEntity{ID: et.ID, Name: et.Name, Category: EveEntityInventoryType}
}

type EveTypeDogmaAttribute struct {
	EveType        *EveType
	DogmaAttribute *EveDogmaAttribute
	Value          float32
}

type MembershipHistoryItem struct {
	EndDate      time.Time
	Days         int
	IsDeleted    bool
	IsOldest     bool
	Organization *EveEntity
	RecordID     int
	StartDate    time.Time
}

func (hi MembershipHistoryItem) OrganizationName() string {
	if hi.Organization != nil {
		return hi.Organization.Name
	}
	return "?"
}
