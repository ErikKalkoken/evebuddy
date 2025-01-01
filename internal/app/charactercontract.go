package app

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type CharacterContractAvailability uint

const (
	ContractAvailabilityUnknown CharacterContractAvailability = iota
	ContractAvailabilityAlliance
	ContractAvailabilityCorporation
	ContractAvailabilityPersonal
	ContractAvailabilityPublic
)

type CharacterContractStatus uint

const (
	ContractStatusUnknown CharacterContractStatus = iota
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

type CharacterContractType uint

const (
	ContractTypeUnknown CharacterContractType = iota
	ContractTypeAuction
	ContractTypeCourier
	ContractTypeItemExchange
	ContractTypeLoan
)

var cct2String = map[CharacterContractType]string{
	ContractTypeAuction:      "auction",
	ContractTypeCourier:      "courier",
	ContractTypeItemExchange: "item exchange",
	ContractTypeLoan:         "loan",
}

func (cct CharacterContractType) String() string {
	s, ok := cct2String[cct]
	if !ok {
		return "unknown"
	}
	return s
}

type CharacterContract struct {
	ID                int64
	Acceptor          *EveEntity
	Assignee          *EveEntity
	Availability      CharacterContractAvailability
	Buyout            float64
	CharacterID       int32
	Collateral        float64
	ContractID        int32
	DateAccepted      optional.Optional[time.Time]
	DateCompleted     optional.Optional[time.Time]
	DateExpired       time.Time
	DateIssued        time.Time
	DaysToComplete    int32
	EndLocationID     optional.Optional[int64]
	ForCorporation    bool
	IssuerCorporation *EveEntity
	Issuer            *EveEntity
	Price             float64
	Reward            float64
	StartLocationID   optional.Optional[int64]
	Status            CharacterContractStatus
	Title             string
	Type              CharacterContractType
	Volume            float64
}

func (cc CharacterContract) TypeDisplay() string {
	c := cases.Title(language.English)
	return c.String(cc.Type.String())
}
