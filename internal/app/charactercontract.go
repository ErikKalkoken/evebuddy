package app

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CharacterContractAvailability uint

const (
	ContractAvailabilityUnknown CharacterContractAvailability = iota
	ContractAvailabilityPublic
	ContractAvailabilityPersonal
	ContractAvailabilityCorporation
	ContractAvailabilityAlliance
)

type CharacterContractStatus uint

const (
	ContractStatusUnknown CharacterContractStatus = iota
	ContractStatusOutstanding
	ContractStatusInProgress
	ContractStatusFinishedIssuer
	ContractStatusFinishedContractor
	ContractStatusFinished
	ContractStatusCancelled
	ContractStatusRejected
	ContractStatusFailed
	ContractStatusDeleted
	ContractStatusReversed
)

type CharacterContractType uint

const (
	ContractTypeUnknown CharacterContractType = iota
	ContractTypeItemExchange
	ContractTypeAuction
	ContractTypeCourier
	ContractTypeLoan
)

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
