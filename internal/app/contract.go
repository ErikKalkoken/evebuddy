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

func (cs ContractStatus) IsCompleted() bool {
	return cs.consolidated() == contractConsolidatedHistory
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

func contractHasIssue(status ContractStatus, expired time.Time) bool {
	statusIssue := status.consolidated() == contractConsolidatedHasIssue
	expiredButStillActive := (contractIsExpired(expired) && status.IsActive())
	return statusIssue || expiredButStillActive
}

func contractIsExpired(expired time.Time) bool {
	return expired.Before(time.Now())
}
func contractNameDisplay(ct ContractType, start1, end1 *EntityShort[int32], volume float64, items []string) string {
	if ct == ContractTypeCourier {
		var startName, endName string
		if start1 != nil {
			startName = start1.Name
		} else {
			startName = "?"
		}
		if end1 != nil {
			endName = end1.Name
		} else {
			endName = "?"
		}
		return fmt.Sprintf(
			"%s >> %s (%.0f m3)",
			startName,
			endName,
			volume,
		)
	}
	if len(items) > 1 {
		return "[Multiple Items]"
	}
	if len(items) == 1 {
		return items[0]
	}
	return "?"
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
