package app

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

type ContractAvailability uint

const (
	ContractAvailabilityUndefined ContractAvailability = iota
	ContractAvailabilityAlliance
	ContractAvailabilityCorporation
	ContractAvailabilityPrivate
	ContractAvailabilityPublic
)

func (cca ContractAvailability) String() string {
	var m = map[ContractAvailability]string{
		ContractAvailabilityAlliance:    "alliance",
		ContractAvailabilityCorporation: "corporation",
		ContractAvailabilityPrivate:     "private",
		ContractAvailabilityPublic:      "public",
	}
	s, ok := m[cca]
	if !ok {
		return "?"
	}
	return s
}

func (cca ContractAvailability) Display() string {
	return xstrings.Title(cca.String())
}

// ContractCategory groups contracts like the game client does.
type ContractCategory uint

const (
	ContractCategoryUndefined ContractCategory = iota
	ContractCategoryOutstanding
	ContractCategoryInProgress
	ContractCategoryRequiresAttention
	ContractCategoryFinished
	ContractCategoryOther
)

// ContractCategories returns the categories in UI order.
func ContractCategories() []ContractCategory {
	return []ContractCategory{
		ContractCategoryOutstanding,
		ContractCategoryInProgress,
		ContractCategoryRequiresAttention,
		ContractCategoryFinished,
		ContractCategoryOther,
	}
}

// ContractCategoriesOpen defines which categories are considered open.
var ContractCategoriesOpen = set.Of(ContractCategoryOutstanding, ContractCategoryInProgress, ContractCategoryRequiresAttention)

func (cc ContractCategory) IsOpen() bool {
	return ContractCategoriesOpen.Contains(cc)
}

func (cc ContractCategory) Display() string {
	switch cc {
	case ContractCategoryOutstanding:
		return "Outstanding"
	case ContractCategoryInProgress:
		return "In progress"
	case ContractCategoryRequiresAttention:
		return "Requires attention"
	case ContractCategoryFinished:
		return "Finished"
	case ContractCategoryOther:
		return "Other"
	}
	return "?"
}

func (cc ContractCategory) Color() fyne.ThemeColorName {
	switch cc {
	case ContractCategoryOutstanding:
		return theme.ColorNameWarning
	case ContractCategoryRequiresAttention:
		return theme.ColorNameError
	case ContractCategoryFinished:
		return theme.ColorNameSuccess
	}
	return theme.ColorNameForeground
}

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

// ContractStatusActive defines which status is considered active.
var ContractStatusActive = set.Of(ContractStatusOutstanding, ContractStatusInProgress)

// IsFinished reports whether a contract has reached a final state that is kept after it drops out of ESI.
// Reversed counts as finished here, although the UI shows it under ContractCategoryOther.
func (cs ContractStatus) IsFinished() bool {
	switch cs {
	case
		ContractStatusFinished,
		ContractStatusFinishedContractor,
		ContractStatusFinishedIssuer,
		ContractStatusReversed:
		return true
	}
	return false
}

func (cs ContractStatus) Display() string {
	return xstrings.Title(cs.String())
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

func (cct ContractType) Display() string {
	return xstrings.Title(cct.String())
}

func (cct ContractType) String() string {
	s, ok := cct2String[cct]
	if !ok {
		return "?"
	}
	return s
}

// TODO: Consolidate "Contract" into base struct

type CharacterContract struct {
	ID                int64
	Acceptor          optional.Optional[*EveEntity]
	Assignee          optional.Optional[*EveEntity]
	Availability      ContractAvailability
	Buyout            optional.Optional[float64]
	CharacterID       int64
	Collateral        optional.Optional[float64]
	ContractID        int64
	DateAccepted      optional.Optional[time.Time]
	DateCompleted     optional.Optional[time.Time]
	DateExpired       time.Time
	DateIssued        time.Time
	DaysToComplete    optional.Optional[int64]
	EndLocation       optional.Optional[*EveLocationShort]
	EndSolarSystem    optional.Optional[*EntityShort] // TODO: Convert to EveEntity
	ForCorporation    bool
	Issuer            *EveEntity
	IssuerCorporation *EveEntity
	Items             []string
	Price             optional.Optional[float64]
	Reward            optional.Optional[float64]
	StartLocation     optional.Optional[*EveLocationShort]
	StartSolarSystem  optional.Optional[*EntityShort] // TODO: Convert to EveEntity
	Status            ContractStatus
	StatusNotified    ContractStatus
	Title             optional.Optional[string]
	Type              ContractType
	UpdatedAt         time.Time
	Volume            optional.Optional[float64]
}

func (cs CharacterContract) IssuerEffective() *EveEntity {
	if cs.ForCorporation {
		return cs.IssuerCorporation
	}
	return cs.Issuer
}

func (cs CharacterContract) NameDisplay() string {
	return contractNameDisplay(cs.Type, cs.StartSolarSystem, cs.EndSolarSystem, cs.Volume, cs.Items)
}

type CharacterContractBid struct {
	ContractID int64
	Amount     float64
	BidID      int64
	Bidder     *EveEntity
	DateBid    time.Time
}

type CharacterContractItem struct {
	ContractID  int64
	IsIncluded  bool
	IsSingleton bool
	Quantity    int64
	RawQuantity optional.Optional[int64]
	RecordID    int64
	Type        *EveType
}

type CorporationContract struct {
	ID                int64
	Acceptor          optional.Optional[*EveEntity]
	Assignee          optional.Optional[*EveEntity]
	Availability      ContractAvailability
	Buyout            optional.Optional[float64]
	CorporationID     int64
	Collateral        optional.Optional[float64]
	ContractID        int64
	DateAccepted      optional.Optional[time.Time]
	DateCompleted     optional.Optional[time.Time]
	DateExpired       time.Time
	DateIssued        time.Time
	DaysToComplete    optional.Optional[int64]
	EndLocation       optional.Optional[*EveLocationShort]
	EndSolarSystem    optional.Optional[*EntityShort]
	ForCorporation    bool
	Issuer            *EveEntity
	IssuerCorporation *EveEntity
	Items             []string
	Price             optional.Optional[float64]
	Reward            optional.Optional[float64]
	StartLocation     optional.Optional[*EveLocationShort]
	StartSolarSystem  optional.Optional[*EntityShort]
	Status            ContractStatus
	StatusNotified    ContractStatus
	Title             optional.Optional[string]
	Type              ContractType
	UpdatedAt         time.Time
	Volume            optional.Optional[float64]
}

func (cs CorporationContract) IssuerEffective() *EveEntity {
	if cs.ForCorporation {
		return cs.IssuerCorporation
	}
	return cs.Issuer
}

func (cs CorporationContract) NameDisplay() string {
	return contractNameDisplay(cs.Type, cs.StartSolarSystem, cs.EndSolarSystem, cs.Volume, cs.Items)
}

type CorporationContractBid struct {
	ContractID int64
	Amount     float64
	BidID      int64
	Bidder     *EveEntity
	DateBid    time.Time
}

type CorporationContractItem struct {
	ContractID  int64
	IsIncluded  bool
	IsSingleton bool
	Quantity    int64
	RawQuantity optional.Optional[int64]
	RecordID    int64
	Type        *EveType
}

// ContractCategoryAt returns the category of a contract at the given time.
// A contract only requires attention when one of the user's characters must act:
// the issuer of an expired contract or the acceptor of an overdue one.
// Corporation roles (e.g. Contract Manager, Director) are not checked.
func ContractCategoryAt(status ContractStatus, dateExpired time.Time, dateAccepted optional.Optional[time.Time], daysToComplete optional.Optional[int64], issuerIsMine, acceptorIsMine bool, now time.Time) ContractCategory {
	switch status {
	case ContractStatusOutstanding:
		if !ContractIsExpiredAt(dateExpired, now) {
			return ContractCategoryOutstanding
		}
		if issuerIsMine {
			return ContractCategoryRequiresAttention
		}
		return ContractCategoryOther
	case ContractStatusInProgress:
		if ContractIsOverdueAt(dateAccepted, daysToComplete, now) && acceptorIsMine {
			return ContractCategoryRequiresAttention
		}
		return ContractCategoryInProgress
	case
		ContractStatusFinished,
		ContractStatusFinishedContractor,
		ContractStatusFinishedIssuer:
		return ContractCategoryFinished
	}
	return ContractCategoryOther
}

// ContractIsExpiredAt reports whether a contract has expired at the given time.
func ContractIsExpiredAt(dateExpired, now time.Time) bool {
	return dateExpired.Before(now)
}

// ContractDeliveryDeadline returns when an accepted contract must be completed.
// It reports false when the contract has no such deadline.
func ContractDeliveryDeadline(dateAccepted optional.Optional[time.Time], daysToComplete optional.Optional[int64]) (time.Time, bool) {
	accepted, ok1 := dateAccepted.Value()
	days, ok2 := daysToComplete.Value()
	if !ok1 || !ok2 {
		return time.Time{}, false
	}
	return accepted.AddDate(0, 0, int(days)), true
}

// ContractIsOverdueAt reports whether an accepted contract has passed its deadline at the given time.
func ContractIsOverdueAt(dateAccepted optional.Optional[time.Time], daysToComplete optional.Optional[int64], now time.Time) bool {
	deadline, ok := ContractDeliveryDeadline(dateAccepted, daysToComplete)
	return ok && deadline.Before(now)
}
func contractNameDisplay(ct ContractType, start, end optional.Optional[*EntityShort], volume optional.Optional[float64], items []string) string {
	if ct == ContractTypeCourier {
		startName := optional.Map(start, "?", func(v *EntityShort) string {
			return v.Name
		})
		endName := optional.Map(end, "?", func(v *EntityShort) string {
			return v.Name
		})
		s := fmt.Sprintf("%s >> %s", startName, endName)
		if v, ok := volume.Value(); ok {
			s += fmt.Sprintf(" (%.0f m3)", v)
		}
		return s
	}
	if len(items) > 1 {
		return "[Multiple Items]"
	}
	if len(items) == 1 {
		if items[0] == "" {
			return "[Single Item]"
		}
		return items[0]
	}
	return "[Empty]"
}

// CharacterContractSlots represents counts of contract slots for a character.
type CharacterContractSlots struct {
	CharacterID     int64
	CharacterName   string
	CorporationID   int64
	CorporationName string
	Free            int
	IsCorporation   bool
	Total           int
	Used            int
}
