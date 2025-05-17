package app

import (
	"bytes"
	"context"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/yuin/goldmark"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// An Eve Online character owners by the user.
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
		return VariantBPO
	}
	return VariantRegular
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

type ContractAvailability uint

const (
	ContractAvailabilityUndefined ContractAvailability = iota
	ContractAvailabilityAlliance
	ContractAvailabilityCorporation
	ContractAvailabilityPersonal
	ContractAvailabilityPublic
)

func (cca ContractAvailability) String() string {
	var m = map[ContractAvailability]string{
		ContractAvailabilityAlliance:    "alliance",
		ContractAvailabilityCorporation: "corporation",
		ContractAvailabilityPersonal:    "private",
		ContractAvailabilityPublic:      "public",
	}
	s, ok := m[cca]
	if !ok {
		return "?"
	}
	return s
}

// contractConsolidatedStatus represents a consolidated status of a contract based on the original contract.
type contractConsolidatedStatus uint

const (
	contractConcolidatedUndefined contractConsolidatedStatus = iota
	contractConsolidatedOustanding
	contractConsolidatedInProgress
	contractConsolidatedHasIssue
	contractConsolidatedHistory
)

// ContractStatus represents the original status of a contract.
type ContractStatus uint

const (
	ContractStatusUndefined ContractStatus = iota
	ContractStatusCancelled
	ContractStatusDeleted
	ContractStatusFailed
	ContractStatusFinished
	ContractStatusFinishedContractor
	ContractStatusFinishedIssuer
	ContractStatusInProgress
	ContractStatusOutstanding
	ContractStatusRejected
	ContractStatusReversed
)

var cs2String = map[ContractStatus]string{
	ContractStatusCancelled:          "cancelled",
	ContractStatusDeleted:            "deleted",
	ContractStatusFailed:             "failed",
	ContractStatusFinished:           "finished",
	ContractStatusFinishedContractor: "finished contractor",
	ContractStatusFinishedIssuer:     "finished issuer",
	ContractStatusInProgress:         "in progress",
	ContractStatusOutstanding:        "outstanding",
	ContractStatusRejected:           "rejected",
	ContractStatusReversed:           "reversed",
}

func (cs ContractStatus) String() string {
	s, ok := cs2String[cs]
	if !ok {
		return "?"
	}
	return s
}

func (cs ContractStatus) Display() string {
	caser := cases.Title(language.English)
	return caser.String(cs.String())
}

func (cs ContractStatus) DisplayRichText() []widget.RichTextSegment {
	var color fyne.ThemeColorName
	switch cs.consolidated() {
	case contractConsolidatedOustanding:
		color = theme.ColorNamePrimary
	case contractConsolidatedInProgress:
		color = theme.ColorNameWarning
	case contractConsolidatedHistory:
		color = theme.ColorNameSuccess
	case contractConsolidatedHasIssue:
		color = theme.ColorNameError
	default:
		color = theme.ColorNameForeground
	}
	return iwidget.NewRichTextSegmentFromText(cs.Display(), widget.RichTextStyle{
		ColorName: color,
	})
}

func (cs ContractStatus) consolidated() contractConsolidatedStatus {
	switch cs {
	case ContractStatusOutstanding:
		return contractConsolidatedOustanding
	case ContractStatusInProgress:
		return contractConsolidatedInProgress
	case
		ContractStatusDeleted,
		ContractStatusCancelled,
		ContractStatusFinished,
		ContractStatusFinishedContractor,
		ContractStatusFinishedIssuer:
		return contractConsolidatedHistory
	case
		ContractStatusFailed,
		ContractStatusRejected:
		return contractConsolidatedHasIssue
	}
	return contractConcolidatedUndefined
}

type ContractType uint

const (
	ContractTypeUndefined ContractType = iota
	ContractTypeAuction
	ContractTypeCourier
	ContractTypeItemExchange
	ContractTypeLoan
	ContractTypeUnknown
)

var cct2String = map[ContractType]string{
	ContractTypeAuction:      "auction",
	ContractTypeCourier:      "courier",
	ContractTypeItemExchange: "item exchange",
	ContractTypeLoan:         "loan",
	ContractTypeUnknown:      "unknown",
}

func (cct ContractType) String() string {
	s, ok := cct2String[cct]
	if !ok {
		return "?"
	}
	return s
}

type CharacterContract struct {
	ID                int64
	Acceptor          *EveEntity
	Assignee          *EveEntity
	Availability      ContractAvailability
	Buyout            float64
	CharacterID       int32
	Collateral        float64
	ContractID        int32
	DateAccepted      optional.Optional[time.Time]
	DateCompleted     optional.Optional[time.Time]
	DateExpired       time.Time
	DateIssued        time.Time
	DaysToComplete    int32
	EndLocation       *EveLocationShort
	EndSolarSystem    *EntityShort[int32]
	ForCorporation    bool
	Issuer            *EveEntity
	IssuerCorporation *EveEntity
	Items             []string
	Price             float64
	Reward            float64
	StartLocation     *EveLocationShort
	StartSolarSystem  *EntityShort[int32]
	Status            ContractStatus
	StatusNotified    ContractStatus
	Title             string
	Type              ContractType
	UpdatedAt         time.Time
	Volume            float64
}

func (cc CharacterContract) AssigneeName() string {
	if cc.Assignee == nil {
		return ""
	}
	return cc.Assignee.Name
}

func (cc CharacterContract) AvailabilityDisplay() string {
	titler := cases.Title(language.English)
	s := titler.String(cc.Availability.String())
	return s
}

func (cc CharacterContract) ContractorDisplay() string {
	if cc.Acceptor == nil {
		return "(None)"
	}
	return cc.Acceptor.Name
}

func (cc CharacterContract) HasIssue() bool {
	return cc.Status.consolidated() == contractConsolidatedHasIssue || (cc.IsExpired() && cc.IsActive())
}

func (cc CharacterContract) IsExpired() bool {
	return cc.DateExpired.Before(time.Now())
}

func (cc CharacterContract) IsActive() bool {
	switch cc.Status.consolidated() {
	case contractConsolidatedInProgress, contractConsolidatedOustanding:
		return true
	}
	return false
}

func (cc CharacterContract) IsCompleted() bool {
	return cc.Status.consolidated() == contractConsolidatedHistory
}

func (cc CharacterContract) IssuerEffective() *EveEntity {
	if cc.ForCorporation {
		return cc.IssuerCorporation
	}
	return cc.Issuer
}

func (cc CharacterContract) NameDisplay() string {
	if cc.Type == ContractTypeCourier {
		var start, end string
		if cc.StartSolarSystem != nil {
			start = cc.StartSolarSystem.Name
		} else {
			start = "?"
		}
		if cc.EndSolarSystem != nil {
			end = cc.EndSolarSystem.Name
		} else {
			end = "?"
		}
		return fmt.Sprintf(
			"%s >> %s (%.0f m3)",
			start,
			end,
			cc.Volume,
		)
	}
	if len(cc.Items) > 1 {
		return "[Multiple Items]"
	}
	if len(cc.Items) == 1 {
		return cc.Items[0]
	}
	return "?"
}

func (cc CharacterContract) StatusDisplay() string {
	return cc.Status.Display()
}

func (cc CharacterContract) StatusDisplayRichText() []widget.RichTextSegment {
	return cc.Status.DisplayRichText()
}

func (cc CharacterContract) TitleDisplay() string {
	if cc.Title == "" {
		return "(None)"
	}
	return cc.Title
}

func (cc CharacterContract) TypeDisplay() string {
	caser := cases.Title(language.English)
	return caser.String(cc.Type.String())
}

type CharacterContractBid struct {
	ContractID int64
	Amount     float32
	BidID      int32
	Bidder     *EveEntity
	DateBid    time.Time
}

type CharacterContractItem struct {
	ContractID  int64
	IsIncluded  bool
	IsSingleton bool
	Quantity    int
	RawQuantity int
	RecordID    int64
	Type        *EveType
}

type CharacterImplant struct {
	CharacterID int32
	EveType     *EveType
	ID          int64
	SlotNum     int // 0 = unknown
}

// A Role is the in-game role of a character in a corporatipn.
type Role uint

const (
	RoleAccountant Role = iota + 1
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

func CorporationRoles() iter.Seq[Role] {
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

// IndustryActivity represents the activity type of an industry job.
// See also: https://github.com/esi/esi-issues/issues/894
type IndustryActivity int32

const (
	None                       IndustryActivity = 0
	Manufacturing              IndustryActivity = 1
	TimeEfficiencyResearch     IndustryActivity = 3
	MaterialEfficiencyResearch IndustryActivity = 4
	Copying                    IndustryActivity = 5
	Invention                  IndustryActivity = 8
	Reactions                  IndustryActivity = 11
)

func (a IndustryActivity) String() string {
	m := map[IndustryActivity]string{
		None:                       "none",
		Manufacturing:              "manufacturing",
		TimeEfficiencyResearch:     "time efficiency research",
		MaterialEfficiencyResearch: "material efficiency research",
		Copying:                    "copying",
		Invention:                  "invention",
		Reactions:                  "reactions",
	}
	s, ok := m[a]
	if !ok {
		return "?"
	}
	return s
}

func (a IndustryActivity) Display() string {
	titler := cases.Title(language.English)
	return titler.String(a.String())
}

type IndustryJobStatus uint

const (
	JobUndefined IndustryJobStatus = iota
	JobActive
	JobCancelled
	JobDelivered
	JobPaused
	JobReady
	JobReverted
)

func (s IndustryJobStatus) String() string {
	m := map[IndustryJobStatus]string{
		JobUndefined: "undefined",
		JobActive:    "in progress",
		JobCancelled: "cancelled",
		JobDelivered: "delivered",
		JobPaused:    "halted",
		JobReady:     "ready",
		JobReverted:  "reverted",
	}
	x, ok := m[s]
	if !ok {
		return "?"
	}
	return x
}

func (s IndustryJobStatus) Display() string {
	titler := cases.Title(language.English)
	return titler.String(s.String())
}

func (s IndustryJobStatus) Color() fyne.ThemeColorName {
	m := map[IndustryJobStatus]fyne.ThemeColorName{
		JobActive:    theme.ColorNamePrimary,
		JobCancelled: theme.ColorNameError,
		JobPaused:    theme.ColorNameWarning,
		JobReady:     theme.ColorNameSuccess,
	}
	c, ok := m[s]
	if ok {
		return c
	}
	return theme.ColorNameForeground
}

type CharacterIndustryJob struct {
	Activity           IndustryActivity
	BlueprintID        int64
	BlueprintLocation  *EveLocationShort
	BlueprintType      *EntityShort[int32]
	CharacterID        int32
	CompletedCharacter optional.Optional[*EveEntity]
	CompletedDate      optional.Optional[time.Time]
	Cost               optional.Optional[float64]
	Duration           int
	EndDate            time.Time
	Facility           *EveLocationShort
	Installer          *EveEntity
	JobID              int32
	LicensedRuns       optional.Optional[int]
	OutputLocation     *EveLocationShort
	PauseDate          optional.Optional[time.Time]
	Probability        optional.Optional[float32]
	ProductType        optional.Optional[*EntityShort[int32]]
	Runs               int
	StartDate          time.Time
	Station            *EveLocationShort
	Status             IndustryJobStatus
	SuccessfulRuns     optional.Optional[int32]
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

func (j CharacterJumpClone2) CharacterName() string {
	if j.Character == nil {
		return ""
	}
	return j.Character.Name
}

func (j CharacterJumpClone2) LocationName() string {
	if j.Location == nil {
		return ""
	}
	return j.Location.DisplayName()
}

func (j CharacterJumpClone2) SolarSystemName() string {
	if j.Location == nil || j.Location.SolarSystem == nil {
		return ""
	}
	return j.Location.SolarSystem.Name
}

func (j CharacterJumpClone2) RegionName() string {
	if j.Location == nil || j.Location.SolarSystem == nil {
		return ""
	}
	return j.Location.SolarSystem.Constellation.Region.Name
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

// A mail label for an Eve mail belonging to a character.
type CharacterMailLabel struct {
	ID          int64
	CharacterID int32
	Color       string
	LabelID     int32
	Name        string
	UnreadCount int
}

// An Eve mail belonging to a character.
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

// An Eve mail header belonging to a character.
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
	return evehtml.ToMarkdown(cm.Body)
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
		iwidget.NewRichTextSegmentFromText("  "+cp.EvePlanet.Name),
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

func (cp CharacterPlanet) Extracting() string {
	extractions := strings.Join(cp.ExtractedTypeNames(), ", ")
	if extractions == "" {
		extractions = "-"
	}
	return extractions
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

func (cp CharacterPlanet) Producing() string {
	productions := strings.Join(cp.ProducedSchematicNames(), ", ")
	if productions == "" {
		productions = "-"
	}
	return productions
}

func (cp CharacterPlanet) DueRichText() []widget.RichTextSegment {
	if cp.IsExpired() {
		return iwidget.NewRichTextSegmentFromText("OFFLINE", widget.RichTextStyle{ColorName: theme.ColorNameError})
	}
	due := cp.ExtractionsExpiryTime()
	if due.IsZero() {
		return iwidget.NewRichTextSegmentFromText("-")
	}
	return iwidget.NewRichTextSegmentFromText(due.Format(DateTimeFormat))
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

type CharacterSkill struct {
	ActiveSkillLevel   int
	CharacterID        int32
	EveType            *EveType
	ID                 int64
	SkillPointsInSkill int
	TrainedSkillLevel  int
}

func SkillDisplayName[N int | int32 | int64 | uint | uint32 | uint64](name string, level N) string {
	return fmt.Sprintf("%s %s", name, ihumanize.RomanLetter(level))
}

type ListCharacterSkillGroupProgress struct {
	GroupID   int32
	GroupName string
	Total     float64
	Trained   float64
}

type ListSkillProgress struct {
	ActiveSkillLevel  int
	TrainedSkillLevel int
	TypeID            int32
	TypeDescription   string
	TypeName          string
}

type CharacterShipSkill struct {
	ActiveSkillLevel  optional.Optional[int]
	ID                int64
	CharacterID       int32
	Rank              uint
	ShipTypeID        int32
	SkillTypeID       int32
	SkillName         string
	SkillLevel        uint
	TrainedSkillLevel optional.Optional[int]
}

type CharacterServiceSkillqueue interface {
	ListSkillqueueItems(context.Context, int32) ([]*CharacterSkillqueueItem, error)
}

// CharacterSkillqueue represents the skillqueue of a character.
type CharacterSkillqueue struct {
	mu          sync.RWMutex
	characterID int32
	items       []*CharacterSkillqueueItem
}

func NewCharacterSkillqueue() *CharacterSkillqueue {
	sq := &CharacterSkillqueue{items: make([]*CharacterSkillqueueItem, 0)}
	return sq
}

func (sq *CharacterSkillqueue) CharacterID() int32 {
	return sq.characterID
}

func (sq *CharacterSkillqueue) Current() *CharacterSkillqueueItem {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	for _, item := range sq.items {
		if item.IsActive() {
			return item
		}
	}
	return nil
}

func (sq *CharacterSkillqueue) Completion() optional.Optional[float64] {
	c := sq.Current()
	if c == nil {
		return optional.Optional[float64]{}
	}
	return optional.From(c.CompletionP())
}

func (sq *CharacterSkillqueue) IsActive() bool {
	c := sq.Current()
	if c == nil {
		return false
	}
	return sq.Remaining().ValueOrZero() > 0
}

func (sq *CharacterSkillqueue) Item(id int) *CharacterSkillqueueItem {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	if id < 0 || id >= len(sq.items) {
		return nil
	}
	return sq.items[id]
}

func (sq *CharacterSkillqueue) Size() int {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	return len(sq.items)
}

func (sq *CharacterSkillqueue) Remaining() optional.Optional[time.Duration] {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	var r optional.Optional[time.Duration]
	for _, item := range sq.items {
		r = optional.From(r.ValueOrZero() + item.Remaining().ValueOrZero())
	}
	return r
}

func (sq *CharacterSkillqueue) Update(cs CharacterServiceSkillqueue, characterID int32) error {
	var items []*CharacterSkillqueueItem
	if characterID == 0 {
		items = []*CharacterSkillqueueItem{}
	} else {
		var err error
		items, err = cs.ListSkillqueueItems(context.Background(), characterID)
		if err != nil {
			return err
		}
	}
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.items = items
	sq.characterID = characterID
	return nil
}

type CharacterSkillqueueItem struct {
	CharacterID      int32
	GroupName        string
	FinishDate       time.Time
	FinishedLevel    int
	LevelEndSP       int
	LevelStartSP     int
	ID               int64
	QueuePosition    int
	StartDate        time.Time
	SkillName        string
	SkillDescription string
	TrainingStartSP  int
}

func (qi CharacterSkillqueueItem) String() string {
	return fmt.Sprintf("%s %s", qi.SkillName, ihumanize.RomanLetter(qi.FinishedLevel))
}

func (qi CharacterSkillqueueItem) IsActive() bool {
	now := time.Now()
	return !qi.StartDate.IsZero() && qi.StartDate.Before(now) && qi.FinishDate.After(now)
}

func (qi CharacterSkillqueueItem) IsCompleted() bool {
	return qi.CompletionP() == 1
}

func (qi CharacterSkillqueueItem) CompletionP() float64 {
	d := qi.Duration()
	if d.IsEmpty() {
		return 0
	}
	duration := d.ValueOrZero()
	now := time.Now()
	if qi.FinishDate.Before(now) {
		return 1
	}
	if qi.StartDate.After(now) {
		return 0
	}
	if duration == 0 {
		return 0
	}
	remaining := qi.FinishDate.Sub(now)
	c := remaining.Seconds() / duration.Seconds()
	base := float64(qi.LevelEndSP-qi.TrainingStartSP) / float64(qi.LevelEndSP-qi.LevelStartSP)
	return 1 - (c * base)
}

func (qi CharacterSkillqueueItem) Duration() optional.Optional[time.Duration] {
	if qi.StartDate.IsZero() || qi.FinishDate.IsZero() {
		return optional.Optional[time.Duration]{}
	}
	return optional.From(qi.FinishDate.Sub(qi.StartDate))
}

func (qi CharacterSkillqueueItem) Remaining() optional.Optional[time.Duration] {
	if qi.StartDate.IsZero() || qi.FinishDate.IsZero() {
		return optional.Optional[time.Duration]{}
	}
	remainingP := 1 - qi.CompletionP()
	d := qi.Duration()
	return optional.From(time.Duration(float64(d.ValueOrZero()) * remainingP))
}

// A SSO token belonging to a character in Eve Online.
type CharacterToken struct {
	AccessToken  string
	CharacterID  int32
	ExpiresAt    time.Time
	ID           int64
	RefreshToken string
	Scopes       []string
	TokenType    string
}

// RemainsValid reports whether a token remains valid within a duration.
func (ct CharacterToken) RemainsValid(d time.Duration) bool {
	return ct.ExpiresAt.After(time.Now().Add(d))
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

type CharacterWalletTransaction struct {
	CharacterID   int32
	Client        *EveEntity
	Date          time.Time
	EveType       *EntityShort[int32]
	ID            int64
	IsBuy         bool
	IsPersonal    bool
	JournalRefID  int64
	Location      *EveLocationShort
	Quantity      int32
	TransactionID int64
	UnitPrice     float64
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

// Groups returns a slice of all normal groups in alphabetical order.
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
