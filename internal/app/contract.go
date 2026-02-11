package app

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
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

// contractConsolidatedStatus represents a consolidated status of a contract based on the original contract.
type contractConsolidatedStatus uint

const (
	contractConsolidatedUndefined contractConsolidatedStatus = iota
	contractConsolidatedOutstanding
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

func (cs ContractStatus) IsActive() bool {
	switch cs.consolidated() {
	case contractConsolidatedInProgress, contractConsolidatedOutstanding:
		return true
	}
	return false
}

func (cs ContractStatus) IsHistory() bool {
	return cs.consolidated() == contractConsolidatedHistory
}

func (cs ContractStatus) IsFinished() bool {
	switch cs {
	case
		ContractStatusFinished,
		ContractStatusFinishedContractor,
		ContractStatusFinishedIssuer:
		return true
	}
	return false
}

func (cs ContractStatus) Display() string {
	return xstrings.Title(cs.String())
}

func (cs ContractStatus) DisplayRichText() []widget.RichTextSegment {
	var color fyne.ThemeColorName
	switch cs.consolidated() {
	case contractConsolidatedOutstanding:
		color = theme.ColorNameWarning
	case contractConsolidatedInProgress:
		color = theme.ColorNameForeground
	case contractConsolidatedHistory:
		color = theme.ColorNameSuccess
	case contractConsolidatedHasIssue:
		color = theme.ColorNameError
	default:
		color = theme.ColorNameForeground
	}
	return iwidget.RichTextSegmentsFromText(cs.Display(), widget.RichTextStyle{
		ColorName: color,
	})
}

func (cs ContractStatus) consolidated() contractConsolidatedStatus {
	switch cs {
	case ContractStatusOutstanding:
		return contractConsolidatedOutstanding
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
	return contractConsolidatedUndefined
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
	EndSolarSystem    optional.Optional[*EntityShort[int64]]
	ForCorporation    bool
	Issuer            *EveEntity
	IssuerCorporation *EveEntity
	Items             []string
	Price             optional.Optional[float64]
	Reward            optional.Optional[float64]
	StartLocation     optional.Optional[*EveLocationShort]
	StartSolarSystem  optional.Optional[*EntityShort[int64]]
	Status            ContractStatus
	StatusNotified    ContractStatus
	Title             optional.Optional[string]
	Type              ContractType
	UpdatedAt         time.Time
	Volume            optional.Optional[float64]
}

func (cs CharacterContract) HasIssue() bool {
	return contractHasIssue(cs.Status, cs.DateExpired)
}

func (cs CharacterContract) IsExpired() bool {
	return contractIsExpired(cs.DateExpired)
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
	Amount     float32
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
	EndSolarSystem    optional.Optional[*EntityShort[int64]]
	ForCorporation    bool
	Issuer            *EveEntity
	IssuerCorporation *EveEntity
	Items             []string
	Price             optional.Optional[float64]
	Reward            optional.Optional[float64]
	StartLocation     optional.Optional[*EveLocationShort]
	StartSolarSystem  optional.Optional[*EntityShort[int64]]
	Status            ContractStatus
	StatusNotified    ContractStatus
	Title             optional.Optional[string]
	Type              ContractType
	UpdatedAt         time.Time
	Volume            optional.Optional[float64]
}

func (cs CorporationContract) HasIssue() bool {
	return contractHasIssue(cs.Status, cs.DateExpired)
}

func (cs CorporationContract) IsExpired() bool {
	return contractIsExpired(cs.DateExpired)
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

func contractHasIssue(status ContractStatus, expired time.Time) bool {
	statusIssue := status.consolidated() == contractConsolidatedHasIssue
	expiredButStillActive := (contractIsExpired(expired) && status.IsActive())
	return statusIssue || expiredButStillActive
}

func contractIsExpired(expired time.Time) bool {
	return expired.Before(time.Now())
}
func contractNameDisplay(ct ContractType, start, end optional.Optional[*EntityShort[int64]], volume optional.Optional[float64], items []string) string {
	if ct == ContractTypeCourier {
		startName := optional.MapOrFallback(start, "?", func(v *EntityShort[int64]) string {
			return v.Name
		})
		endName := optional.MapOrFallback(end, "?", func(v *EntityShort[int64]) string {
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
		return items[0]
	}
	return "?"
}
