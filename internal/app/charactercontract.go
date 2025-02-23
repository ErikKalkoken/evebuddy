package app

import (
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ContractAvailability uint

const (
	ContractAvailabilityUndefined ContractAvailability = iota
	ContractAvailabilityAlliance
	ContractAvailabilityCorporation
	ContractAvailabilityPersonal
	ContractAvailabilityPublic
)

var cca2String = map[ContractAvailability]string{
	ContractAvailabilityAlliance:    "alliance",
	ContractAvailabilityCorporation: "corporation",
	ContractAvailabilityPersonal:    "private",
	ContractAvailabilityPublic:      "public",
}

func (cca ContractAvailability) String() string {
	s, ok := cca2String[cca]
	if !ok {
		return "?"
	}
	return s
}

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
	EndLocation       *EntityShort[int64]
	EndSolarSystem    *EntityShort[int32]
	ForCorporation    bool
	Issuer            *EveEntity
	IssuerCorporation *EveEntity
	Items             []string
	Price             float64
	Reward            float64
	StartLocation     *EntityShort[int64]
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
	if cc.Assignee != nil && cc.Availability != ContractAvailabilityPublic && cc.Availability != ContractAvailabilityUndefined {
		s += " - " + cc.Assignee.Name
	}
	return s
}

func (cc CharacterContract) ContractorDisplay() string {
	if cc.Acceptor == nil {
		return "(None)"
	}
	return cc.Acceptor.Name
}

func (cc CharacterContract) NameDisplay() string {
	if cc.Type == ContractTypeCourier {
		return fmt.Sprintf("%s >> %s (%.0f m3)", cc.StartSolarSystem.Name, cc.EndSolarSystem.Name, cc.Volume)
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

func (cc CharacterContract) DateExpiredEffective() time.Time {
	if cc.DateAccepted.IsEmpty() {
		return cc.DateExpired
	}
	return cc.DateAccepted.ValueOrZero().Add(time.Duration(cc.DaysToComplete) * time.Hour * 24)
}

func (cc CharacterContract) IsExpired() bool {
	return cc.DateExpiredEffective().Before(time.Now())
}
