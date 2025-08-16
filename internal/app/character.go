package app

import (
	"bytes"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/yuin/goldmark"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
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

type SendMailMode uint

const (
	SendMailNew SendMailMode = iota + 1
	SendMailReply
	SendMailReplyAll
	SendMailForward
)

// Special mail label IDs
const (
	MailLabelAll      = 1<<31 - 1
	MailLabelUnread   = 1<<31 - 2
	MailLabelNone     = 0
	MailLabelInbox    = 1
	MailLabelSent     = 2
	MailLabelCorp     = 4
	MailLabelAlliance = 8
)

// CharacterMailLabel is a mail label for an EVE mail belonging to a character.
type CharacterMailLabel struct {
	ID          int64
	CharacterID int32
	Color       string
	LabelID     int32
	Name        string
	UnreadCount int
}

// CharacterMail is an EVE mail belonging to a character.
type CharacterMail struct {
	Body        string
	CharacterID int32
	From        *EveEntity
	Labels      []*CharacterMailLabel
	IsProcessed bool
	IsRead      bool
	ID          int64
	MailID      int32
	Recipients  []*EveEntity
	Subject     string
	Timestamp   time.Time
}

// CharacterMailHeader is an EVE mail header belonging to a character.
type CharacterMailHeader struct {
	CharacterID int32
	From        *EveEntity
	IsRead      bool
	ID          int64
	MailID      int32
	Subject     string
	Timestamp   time.Time
}

// BodyPlain returns a mail's body as plain text.
func (cm CharacterMail) BodyPlain() string {
	return evehtml.ToPlain(cm.Body)
}

// String returns a mail's content as string.
func (cm CharacterMail) String() string {
	s := fmt.Sprintf("%s\n", cm.Subject) + cm.Header() + "\n\n" + cm.BodyPlain()
	return s
}

// Header returns a mail's header as string.
func (cm CharacterMail) Header() string {
	var names []string
	for _, n := range cm.Recipients {
		names = append(names, n.Name)
	}
	header := fmt.Sprintf(
		"From: %s\n"+
			"Sent: %s\n"+
			"To: %s",
		cm.From.Name,
		cm.Timestamp.Format(DateTimeFormat),
		strings.Join(names, ", "),
	)
	return header
}

// RecipientNames returns the names of the recipients.
func (cm CharacterMail) RecipientNames() []string {
	ss := make([]string, len(cm.Recipients))
	for i, r := range cm.Recipients {
		ss[i] = r.Name
	}
	return ss
}

func (cm CharacterMail) BodyToMarkdown() string {
	s, err := evehtml.ToMarkdown(cm.Body)
	if err != nil {
		slog.Error("Failed to convert mail body to markdown", "characterID", cm.CharacterID, "mailID", cm.MailID, "error", err)
		return ""
	}
	return s
}

type CharacterNotification struct {
	ID             int64
	Body           optional.Optional[string] // generated body text in markdown
	CharacterID    int32
	IsProcessed    bool
	IsRead         bool
	NotificationID int64
	RecipientName  string // TODO: Replace with EveEntity
	Sender         *EveEntity
	Text           string
	Timestamp      time.Time
	Title          optional.Optional[string] // generated title text in markdown
	Type           string                    // This is a string, so that it can handle unknown types
}

// TitleDisplay returns the rendered title when it exists or else the fake tile.
func (cn *CharacterNotification) TitleDisplay() string {
	if cn.Title.IsEmpty() {
		return cn.TitleFake()
	}
	return cn.Title.ValueOrZero()
}

// TitleFake returns a title for output made from the name of the type.
func (cn *CharacterNotification) TitleFake() string {
	var b strings.Builder
	var last rune
	for _, r := range cn.Type {
		if unicode.IsUpper(r) && unicode.IsLower(last) {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
		last = r
	}
	return b.String()
}

// Header returns the header of a notification.
func (cn *CharacterNotification) Header() string {
	s := fmt.Sprintf(
		"From: %s\n"+
			"Sent: %s",
		cn.Sender.Name,
		cn.Timestamp.Format(DateTimeFormat),
	)
	if cn.RecipientName != "" {
		s += fmt.Sprintf("\nTo: %s", cn.RecipientName)
	}
	return s
}

// String returns the content of a notification as string.
func (cn *CharacterNotification) String() string {
	s := cn.TitleDisplay() + "\n" + cn.Header()
	b, err := cn.BodyPlain()
	if err != nil {
		slog.Error("render notification to string", "id", cn.ID, "error", err)
		return s
	}
	s += "\n\n"
	if b.IsEmpty() {
		s += "(no body)"
	} else {
		s += b.ValueOrZero()
	}
	return s
}

// BodyPlain returns the body of a notification as plain text.
func (cn *CharacterNotification) BodyPlain() (optional.Optional[string], error) {
	var b optional.Optional[string]
	if cn.Body.IsEmpty() {
		return b, nil
	}
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(cn.Body.ValueOrZero()), &buf); err != nil {
		return b, fmt.Errorf("convert notification body: %w", err)
	}
	b.Set(evehtml.Strip(buf.String()))
	return b, nil
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

type NotificationGroup uint

const (
	GroupBills NotificationGroup = iota + 1
	GroupFactionWarfare
	GroupContacts
	GroupCorporate
	GroupInsurance
	GroupInsurgencies
	GroupMoonMining
	GroupMiscellaneous
	GroupOld
	GroupSovereignty
	GroupStructure
	GroupWar

	GroupUnknown
	GroupUnread
	GroupAll
)

func (c NotificationGroup) String() string {
	return group2Name[c]
}

var group2Name = map[NotificationGroup]string{
	GroupAll:            "All",
	GroupBills:          "Bills",
	GroupContacts:       "Contacts",
	GroupCorporate:      "Corporate",
	GroupFactionWarfare: "Faction Warfare",
	GroupInsurance:      "Insurance",
	GroupInsurgencies:   "Insurgencies",
	GroupMiscellaneous:  "Miscellaneous",
	GroupMoonMining:     "Moon Mining",
	GroupOld:            "Old",
	GroupSovereignty:    "Sovereignty",
	GroupStructure:      "Structure",
	GroupUnknown:        "Unknown",
	GroupUnread:         "Unread",
	GroupWar:            "War",
}

// NotificationGroups returns a slice of all normal groups in alphabetical order.
func NotificationGroups() []NotificationGroup {
	return []NotificationGroup{
		GroupBills,
		GroupFactionWarfare,
		GroupContacts,
		GroupCorporate,
		GroupInsurance,
		GroupInsurgencies,
		GroupMoonMining,
		GroupMiscellaneous,
		GroupOld,
		GroupSovereignty,
		GroupStructure,
		GroupWar,
	}
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
