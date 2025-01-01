package app

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CharacterContractAvailability string

const (
	AvailabilityPublic      CharacterContractAvailability = "PU"
	AvailabilityPersonal    CharacterContractAvailability = "PR"
	AvailabilityCorporation CharacterContractAvailability = "CR"
	AvailabilityAlliance    CharacterContractAvailability = "AL"
)

type CharacterContractStatus string

const (
	StatusOutstanding        CharacterContractStatus = "OS"
	StatusInProgress         CharacterContractStatus = "IP"
	StatusFinishedIssuer     CharacterContractStatus = "FI"
	StatusFinishedContractor CharacterContractStatus = "FC"
	StatusFinished           CharacterContractStatus = "FN"
	StatusCancelled          CharacterContractStatus = "CC"
	StatusRejected           CharacterContractStatus = "RJ"
	StatusFailed             CharacterContractStatus = "FL"
	StatusDeleted            CharacterContractStatus = "DL"
	StatusReversed           CharacterContractStatus = "RV"
)

type CharacterContractType string

const (
	TypeUnknown      CharacterContractType = "UN"
	TypeItemExchange CharacterContractType = "IE"
	TypeAuction      CharacterContractType = "AT"
	TypeCourier      CharacterContractType = "CR"
	TypeLoan         CharacterContractType = "LN"
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
