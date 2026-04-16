package app

import (
	"iter"
	"maps"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// Character is an EVE Online character owned by the user.
type Character struct {
	AssetValue        optional.Optional[float64]
	ContractEscrow    optional.Optional[float64]
	EveCharacter      *EveCharacter
	Home              optional.Optional[*EveLocation]
	ID                int64
	IsTrainingWatched bool
	LastCloneJumpAt   optional.Optional[time.Time]
	LastLoginAt       optional.Optional[time.Time]
	Location          optional.Optional[*EveLocation]
	MarketEscrow      optional.Optional[float64]
	Ship              optional.Optional[*EveType]
	TrainedSP         optional.Optional[int64]
	UnallocatedSP     optional.Optional[int64]
	WalletBalance     optional.Optional[float64]
	// Calculated fields
	NextCloneJump optional.Optional[time.Time] // zero time == now
}

func (c *Character) IDOrZero() int64 {
	if c == nil {
		return 0
	}
	return c.ID
}

func (c *Character) NameOrZero() string {
	if c == nil || c.EveCharacter == nil {
		return ""
	}
	return c.EveCharacter.Name
}

type CharacterAttributes struct {
	ID            int64
	BonusRemaps   optional.Optional[int64]
	CharacterID   int64
	Charisma      int64
	Intelligence  int64
	LastRemapDate optional.Optional[time.Time]
	Memory        int64
	Perception    int64
	Willpower     int64
}

type CharacterImplant struct {
	CharacterID int64
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
	return xstrings.Title(cp.String())
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
	CharacterID int64
	Role        Role
	Granted     bool
}

type CharacterJumpClone struct {
	CharacterID int64
	ID          int64
	Implants    []*CharacterJumpCloneImplant
	CloneID     int64
	Location    *EveLocationShort
	Name        optional.Optional[string]
	Region      *EntityShort
}

type CharacterJumpClone2 struct {
	Character     *EntityShort
	ImplantsCount int
	ID            int64
	CloneID       int64
	Location      *EveLocation
}

type CharacterJumpCloneImplant struct {
	ID      int64
	EveType *EveType
	SlotNum int // 0 = unknown
}

type CharacterLoyaltyPointEntry struct {
	ID            int64
	CharacterID   int64
	Corporation   *EntityShort
	Faction       optional.Optional[*EveEntity]
	LoyaltyPoints int64
}

type CharacterPlanet struct {
	ID           int64
	CharacterID  int64
	EvePlanet    *EvePlanet
	LastUpdate   time.Time
	LastNotified optional.Optional[time.Time] // expiry time that was last notified
	Pins         []*PlanetPin
	UpgradeLevel int64
}

func (cp CharacterPlanet) NameRichText() []widget.RichTextSegment {
	return slices.Concat(
		cp.EvePlanet.SolarSystem.SecurityStatusRichText(),
		xwidget.RichTextSegmentsFromText("  "+cp.EvePlanet.Name),
	)
}

// ExtractedTypes returns a list of unique types currently being extracted.
func (cp CharacterPlanet) ExtractedTypes() []*EveType {
	types := make(map[int64]*EveType)
	for pp := range cp.ActiveExtractors() {
		if v, ok := pp.ExtractorProductType.Value(); ok {
			types[v.ID] = v
		}
	}
	return slices.Collect(maps.Values(types))
}

func (cp CharacterPlanet) ActiveExtractors() iter.Seq[*PlanetPin] {
	return xiter.Filter(slices.Values(cp.Pins), func(o *PlanetPin) bool {
		return o.IsExtracting()
	})
}

// ExtractionsEarliestExpiry returns the earliest expiry time of all extractions.
// When no expiry data is found it will return empty.
func (cp CharacterPlanet) ExtractionsEarliestExpiry() optional.Optional[time.Time] {
	times := cp.ExtractionsExpiryTimes()
	if len(times) == 0 {
		return optional.Optional[time.Time]{}
	}
	earliest := slices.MinFunc(times, func(a, b time.Time) int {
		return a.Compare(b)
	})
	return optional.New(earliest)
}

// ExtractionsExpiryTimes returns the expiry times for all extractions.
// When no expiry data is found it will return empty.
func (cp CharacterPlanet) ExtractionsExpiryTimes() []time.Time {
	var s []time.Time
	for pp := range cp.ActiveExtractors() {
		if v, ok := pp.ExpiryTime.Value(); ok && !v.IsZero() {
			s = append(s, v)
		}
	}
	return s
}

func (cp CharacterPlanet) ActiveProducers() iter.Seq[*PlanetPin] {
	return xiter.Filter(slices.Values(cp.Pins), func(o *PlanetPin) bool {
		return o.IsProducing()
	})
}

// ProducedSchematics returns a list of unique schematics currently in production.
func (cp CharacterPlanet) ProducedSchematics() []*EveSchematic {
	schematics := make(map[int64]*EveSchematic)
	for pp := range cp.ActiveProducers() {
		if v, ok := pp.Schematic.Value(); ok {
			schematics[v.ID] = v
		}
	}
	return slices.Collect(maps.Values(schematics))
}

type PlanetPin struct {
	ID                   int64
	ExpiryTime           optional.Optional[time.Time]
	ExtractorProductType optional.Optional[*EveType]
	FactorySchematic     optional.Optional[*EveSchematic]
	InstallTime          optional.Optional[time.Time]
	LastCycleStart       optional.Optional[time.Time]
	Schematic            optional.Optional[*EveSchematic]
	Type                 *EveType
}

func (pp PlanetPin) IsExtracting() bool {
	return pp.Type.Group.ID == EveGroupExtractorControlUnits && !pp.ExtractorProductType.IsEmpty()
}

func (pp PlanetPin) IsProducing() bool {
	return pp.Type.Group.ID == EveGroupProcessors && !pp.Schematic.IsEmpty()
}

type CharacterShipAbility struct {
	Type   EntityShort
	Group  EntityShort
	CanFly bool
}

type CharacterWalletJournalEntry struct {
	Amount        optional.Optional[float64]
	Balance       optional.Optional[float64]
	CharacterID   int64
	ContextID     optional.Optional[int64]
	ContextIDType optional.Optional[string]
	Date          time.Time
	Description   string
	FirstParty    optional.Optional[*EveEntity]
	ID            int64
	Reason        optional.Optional[string]
	RefID         int64
	RefType       string
	SecondParty   optional.Optional[*EveEntity]
	Tax           optional.Optional[float64]
	TaxReceiver   optional.Optional[*EveEntity]
}

func (we CharacterWalletJournalEntry) RefTypeDisplay() string {
	return xstrings.Title(strings.ReplaceAll(we.RefType, "_", " "))
}

type CharacterWalletTransaction struct {
	CharacterID   int64
	Client        *EveEntity
	Date          time.Time
	ID            int64
	IsBuy         bool
	IsPersonal    bool
	JournalRefID  int64
	Location      *EveLocationShort
	Region        *EntityShort
	Quantity      int64
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
	return xstrings.Title(strings.ReplaceAll(string(x), "_", " "))
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
