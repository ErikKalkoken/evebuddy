package app

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type Corporation struct {
	ID          int32
	Corporation *EveCorporation
}

type CorporationIndustryJob struct {
	Activity            IndustryActivity
	BlueprintID         int64
	BlueprintLocationID int64 // can be a corp hanger or container. not supported.
	BlueprintType       *EntityShort[int32]
	CorporationID       int32
	CompletedCharacter  optional.Optional[*EveEntity]
	CompletedDate       optional.Optional[time.Time]
	Cost                optional.Optional[float64]
	Duration            int
	EndDate             time.Time
	FacilityID          int64 // can be a corp hanger or container. not supported.
	Installer           *EveEntity
	JobID               int32
	LicensedRuns        optional.Optional[int]
	OutputLocationID    int64 // can be a corp hanger or container. not supported.
	PauseDate           optional.Optional[time.Time]
	Probability         optional.Optional[float32]
	ProductType         optional.Optional[*EntityShort[int32]]
	Runs                int
	StartDate           time.Time
	Location            *EveLocationShort
	Status              IndustryJobStatus
	SuccessfulRuns      optional.Optional[int32]
}
