package app

import (
	"iter"
	"maps"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

// Character is an Eve Online character owned by the user.
type Character struct {
	AssetValue        optional.Optional[float64]
	EveCharacter      *EveCharacter
	Home              *EveLocation
	ID                int32
	IsTrainingWatched bool
	LastCloneJumpAt   optional.Optional[time.Time]
	LastLoginAt       optional.Optional[time.Time]
	Location          *EveLocation
	Ship              *EveType
	TotalSP           optional.Optional[int]
	UnallocatedSP     optional.Optional[int]
	WalletBalance     optional.Optional[float64]
	// Calculated fields
	NextCloneJump optional.Optional[time.Time] // zero time == now
}

type CharacterAttributes struct {
	ID            int64
	BonusRemaps   int
	CharacterID   int32
	Charisma      int
	Intelligence  int
	LastRemapDate time.Time
	Memory        int
	Perception    int
	Willpower     int
}

type CharacterImplant struct {
	CharacterID int32
	EveType     *EveType
	ID          int64
	SlotNum     int // 0 = unknown
}

// A Role is the in-game role of a character in a corporation.
type Role uint

const (
	RoleUndefined Role = iota
	RoleAccountant
	RoleAccountTake1
	RoleAccountTake2
	RoleAccountTake3
	RoleAccountTake4
	RoleAccountTake5
	RoleAccountTake6
	RoleAccountTake7
	RoleAuditor
	RoleBrandManager
	RoleCommunicationsOfficer
	RoleConfigEquipment
	RoleConfigStarbaseEquipment
	RoleContainerTake1
	RoleContainerTake2
	RoleContainerTake3
	RoleContainerTake4
	RoleContainerTake5
	RoleContainerTake6
	RoleContainerTake7
	RoleContractManager
	RoleDeliveriesContainerTake
	RoleDeliveriesQuery
	RoleDeliveriesTake
	RoleDiplomat
	RoleDirector
	RoleFactoryManager
	RoleFittingManager
	RoleHangarQuery1
	RoleHangarQuery2
	RoleHangarQuery3
	RoleHangarQuery4
	RoleHangarQuery5
	RoleHangarQuery6
	RoleHangarQuery7
	RoleHangarTake1
	RoleHangarTake2
	RoleHangarTake3
	RoleHangarTake4
	RoleHangarTake5
	RoleHangarTake6
	RoleHangarTake7
	RoleJuniorAccountant
	RolePersonnelManager
	RoleProjectManager
	RoleRentFactoryFacility
	RoleRentOffice
	RoleRentResearchFacility
	RoleSecurityOfficer
	RoleSkillPlanManager
	RoleStarbaseDefenseOperator
	RoleStarbaseFuelTechnician
	RoleStationManager
	RoleTrader
)

func (cp Role) String() string {
	s := role2String[cp]
	return s
}

func (cp Role) Display() string {
	titler := cases.Title(language.English)
	return titler.String(cp.String())
}

func RolesAll() iter.Seq[Role] {
	return maps.Keys(role2String)
}

var role2String = map[Role]string{
	RoleAuditor:                 "auditor",
	RoleConfigEquipment:         "config equipment",
	RoleContainerTake7:          "container take 7",
	RoleFactoryManager:          "factory manager",
	RoleHangarQuery3:            "hangar query 3",
	RoleHangarQuery6:            "hangar query 6",
	RoleHangarTake1:             "hangar take 1",
	RoleHangarTake4:             "hangar take 4",
	RoleContainerTake3:          "container take 3",
	RoleContractManager:         "contract manager",
	RoleDeliveriesTake:          "deliveries take",
	RoleHangarTake7:             "hangar take 7",
	RoleRentFactoryFacility:     "rent factory facility",
	RoleRentOffice:              "rent office",
	RoleStarbaseDefenseOperator: "starbase defense operator",
	RoleAccountTake3:            "account take 3",
	RoleAccountTake7:            "account take 7",
	RoleAccountant:              "accountant",
	RoleContainerTake2:          "container take 2",
	RoleDeliveriesQuery:         "deliveries query",
	RoleHangarQuery2:            "hangar query 2",
	RoleHangarQuery7:            "hangar query 7",
	RolePersonnelManager:        "personnel manager",
	RoleAccountTake2:            "account take 2",
	RoleAccountTake4:            "account take 4",
	RoleBrandManager:            "brand manager",
	RoleContainerTake4:          "container take 4",
	RoleDeliveriesContainerTake: "deliveries container take",
	RoleHangarQuery5:            "hangar query 5",
	RoleHangarTake5:             "hangar take 5",
	RoleHangarTake6:             "hangar take 6",
	RoleConfigStarbaseEquipment: "config starbase equipment",
	RoleHangarTake3:             "hangar take 3",
	RoleStationManager:          "station manager",
	RoleAccountTake1:            "account take 1",
	RoleContainerTake1:          "container take 1",
	RoleDiplomat:                "diplomat",
	RoleHangarTake2:             "hangar take 2",
	RoleProjectManager:          "project manager",
	RoleRentResearchFacility:    "rent research facility",
	RoleSecurityOfficer:         "security officer",
	RoleSkillPlanManager:        "skill plan manager",
	RoleFittingManager:          "fitting manager",
	RoleAccountTake5:            "account take 5",
	RoleAccountTake6:            "account take 6",
	RoleCommunicationsOfficer:   "communications officer",
	RoleDirector:                "director",
	RoleHangarQuery1:            "hangar query 1",
	RoleStarbaseFuelTechnician:  "starbase fuel technician",
	RoleTrader:                  "trader",
	RoleContainerTake5:          "container take 5",
	RoleContainerTake6:          "container take 6",
	RoleHangarQuery4:            "hangar query 4",
	RoleJuniorAccountant:        "junior accountant",
}

type CharacterRole struct {
	CharacterID int32
	Role        Role
	Granted     bool
}

type CharacterJumpClone struct {
	CharacterID int32
	ID          int64
	Implants    []*CharacterJumpCloneImplant
	CloneID     int32
	Location    *EveLocationShort
	Name        string
	Region      *EntityShort[int32]
}

type CharacterJumpClone2 struct {
	Character     *EntityShort[int32]
	ImplantsCount int
	ID            int64
	CloneID       int32
	Location      *EveLocation
}

type CharacterJumpCloneImplant struct {
	ID      int64
	EveType *EveType
	SlotNum int // 0 = unknown
}

type CharacterPlanet struct {
	ID           int64
	CharacterID  int32
	EvePlanet    *EvePlanet
	LastUpdate   time.Time
	LastNotified optional.Optional[time.Time] // expiry time that was last notified
	Pins         []*PlanetPin
	UpgradeLevel int
}

func (cp CharacterPlanet) NameRichText() []widget.RichTextSegment {
	return slices.Concat(
		cp.EvePlanet.SolarSystem.SecurityStatusRichText(),
		iwidget.RichTextSegmentsFromText("  "+cp.EvePlanet.Name),
	)
}

// ExtractedTypes returns a list of unique types currently being extracted.
func (cp CharacterPlanet) ExtractedTypes() []*EveType {
	types := make(map[int32]*EveType)
	for pp := range cp.ActiveExtractors() {
		types[pp.ExtractorProductType.ID] = pp.ExtractorProductType
	}
	return slices.Collect(maps.Values(types))
}

func (cp CharacterPlanet) ActiveExtractors() iter.Seq[*PlanetPin] {
	return xiter.Filter(slices.Values(cp.Pins), func(o *PlanetPin) bool {
		return o.IsExtracting()
	})
}

func (cp CharacterPlanet) ExtractedTypeNames() []string {
	return extractedStringsSorted(cp.ExtractedTypes(), func(a *EveType) string {
		return a.Name
	})
}

// ExtractionsExpiryTime returns the final expiry time for all extractions.
// When no expiry data is found it will return a zero time.
func (cp CharacterPlanet) ExtractionsExpiryTime() time.Time {
	expireTimes := make([]time.Time, 0)
	for pp := range cp.ActiveExtractors() {
		if pp.ExpiryTime.IsEmpty() {
			continue
		}
		expireTimes = append(expireTimes, pp.ExpiryTime.ValueOrZero())
	}
	if len(expireTimes) == 0 {
		return time.Time{}
	}
	slices.SortFunc(expireTimes, func(a, b time.Time) int {
		return b.Compare(a) // sort descending
	})
	return expireTimes[0]
}

func (cp CharacterPlanet) ActiveProducers() iter.Seq[*PlanetPin] {
	return xiter.Filter(slices.Values(cp.Pins), func(o *PlanetPin) bool {
		return o.IsProducing()
	})
}

// ProducedSchematics returns a list of unique schematics currently in production.
func (cp CharacterPlanet) ProducedSchematics() []*EveSchematic {
	schematics := make(map[int32]*EveSchematic)
	for pp := range cp.ActiveProducers() {
		schematics[pp.Schematic.ID] = pp.Schematic
	}
	return slices.Collect(maps.Values(schematics))
}

func (cp CharacterPlanet) ProducedSchematicNames() []string {
	return extractedStringsSorted(cp.ProducedSchematics(), func(a *EveSchematic) string {
		return a.Name
	})
}

func (cp CharacterPlanet) IsExpired() bool {
	due := cp.ExtractionsExpiryTime()
	if due.IsZero() {
		return false
	}
	return due.Before(time.Now())
}

func (cp CharacterPlanet) DueRichText() []widget.RichTextSegment {
	if cp.IsExpired() {
		return iwidget.RichTextSegmentsFromText("OFFLINE", widget.RichTextStyle{ColorName: theme.ColorNameError})
	}
	due := cp.ExtractionsExpiryTime()
	if due.IsZero() {
		return iwidget.RichTextSegmentsFromText("-")
	}
	return iwidget.RichTextSegmentsFromText(due.Format(DateTimeFormat))
}

func extractedStringsSorted[T any](s []T, extract func(a T) string) []string {
	s2 := make([]string, 0)
	for _, x := range s {
		s2 = append(s2, extract(x))
	}
	slices.Sort(s2)
	return s2
}

type PlanetPin struct {
	ID                   int64
	ExpiryTime           optional.Optional[time.Time]
	ExtractorProductType *EveType
	FactorySchematic     *EveSchematic
	InstallTime          optional.Optional[time.Time]
	LastCycleStart       optional.Optional[time.Time]
	Schematic            *EveSchematic
	Type                 *EveType
}

func (pp PlanetPin) IsExtracting() bool {
	return pp.Type.Group.ID == EveGroupExtractorControlUnits && pp.ExtractorProductType != nil
}

func (pp PlanetPin) IsProducing() bool {
	return pp.Type.Group.ID == EveGroupProcessors && pp.Schematic != nil
}

type CharacterShipAbility struct {
	Type   EntityShort[int32]
	Group  EntityShort[int32]
	CanFly bool
}

type CharacterWalletJournalEntry struct {
	Amount        float64
	Balance       float64
	CharacterID   int32
	ContextID     int64
	ContextIDType string
	Date          time.Time
	Description   string
	FirstParty    *EveEntity
	ID            int64
	Reason        string
	RefID         int64
	RefType       string
	SecondParty   *EveEntity
	Tax           float64
	TaxReceiver   *EveEntity
}

func (we CharacterWalletJournalEntry) RefTypeDisplay() string {
	titler := cases.Title(language.English)
	return titler.String(strings.ReplaceAll(we.RefType, "_", " "))
}

type CharacterWalletTransaction struct {
	CharacterID   int32
	Client        *EveEntity
	Date          time.Time
	ID            int64
	IsBuy         bool
	IsPersonal    bool
	JournalRefID  int64
	Location      *EveLocationShort
	Region        *EntityShort[int32]
	Quantity      int32
	TransactionID int64
	Type          *EveType
	UnitPrice     float64
}

func (wt *CharacterWalletTransaction) Total() float64 {
	x := wt.UnitPrice * float64(wt.Quantity)
	if wt.IsBuy {
		return -1 * x
	}
	return x
}

type SearchCategory string

const (
	SearchAgent         SearchCategory = "agent"
	SearchAlliance      SearchCategory = "alliance"
	SearchCharacter     SearchCategory = "character"
	SearchConstellation SearchCategory = "constellation"
	SearchCorporation   SearchCategory = "corporation"
	SearchFaction       SearchCategory = "faction"
	SearchRegion        SearchCategory = "region"
	SearchSolarSystem   SearchCategory = "solar_system"
	SearchStation       SearchCategory = "station"
	SearchType          SearchCategory = "inventory_type"
)

func (x SearchCategory) String() string {
	titler := cases.Title(language.English)
	return titler.String(strings.ReplaceAll(string(x), "_", " "))
}

// SearchCategories returns all available search categories
func SearchCategories() []SearchCategory {
	return []SearchCategory{
		SearchAgent,
		SearchAlliance,
		SearchCharacter,
		SearchConstellation,
		SearchCorporation,
		SearchFaction,
		SearchRegion,
		SearchSolarSystem,
		SearchStation,
		SearchType,
	}
}
