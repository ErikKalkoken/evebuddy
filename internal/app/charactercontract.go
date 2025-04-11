package app

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

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
	contractUndefiend contractConsolidatedStatus = iota
	contractOustanding
	contractInProgress
	contractHasIssue
	contractCompleted
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

var ccs2String = map[ContractStatus]string{
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

func (ccs ContractStatus) String() string {
	s, ok := ccs2String[ccs]
	if !ok {
		return "?"
	}
	return s
}

func (cc ContractStatus) consolidated() contractConsolidatedStatus {
	switch cc {
	case ContractStatusOutstanding:
		return contractOustanding
	case ContractStatusInProgress:
		return contractInProgress
	case
		ContractStatusFinished,
		ContractStatusFinishedContractor,
		ContractStatusFinishedIssuer:
		return contractCompleted
	case
		ContractStatusCancelled,
		ContractStatusDeleted,
		ContractStatusFailed,
		ContractStatusRejected:
		return contractHasIssue
	}
	return contractUndefiend
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
	return cc.Status.consolidated() == contractHasIssue
}

func (cc CharacterContract) IsExpired() bool {
	return cc.DateExpired.Before(time.Now())
}

func (cc CharacterContract) IsActive() bool {
	return cc.Status.consolidated() == contractInProgress || cc.Status.consolidated() == contractOustanding
}

func (cc CharacterContract) IsCompleted() bool {
	return cc.Status.consolidated() == contractCompleted
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
	caser := cases.Title(language.English)
	return caser.String(cc.Status.String())
}

func (cc CharacterContract) StatusDisplayRichText() []widget.RichTextSegment {
	var color fyne.ThemeColorName
	switch cc.Status.consolidated() {
	case contractOustanding:
		color = theme.ColorNamePrimary
	case contractInProgress:
		color = theme.ColorNameWarning
	case contractCompleted:
		color = theme.ColorNameSuccess
	case contractHasIssue:
		color = theme.ColorNameError
	default:
		color = theme.ColorNameForeground
	}
	return iwidget.NewRichTextSegmentFromText(cc.StatusDisplay(), widget.RichTextStyle{
		ColorName: color,
	})
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
