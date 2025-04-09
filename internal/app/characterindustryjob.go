package app

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type IndustryActivity uint

const (
	Undefined                  IndustryActivity = 0
	Manufacturing              IndustryActivity = 1
	TimeEfficiencyResearch     IndustryActivity = 3
	MaterialEfficiencyResearch IndustryActivity = 4
	Copying                    IndustryActivity = 5
	Invention                  IndustryActivity = 8
	Reactions                  IndustryActivity = 11
)

func (ia IndustryActivity) String() string {
	m := map[IndustryActivity]string{
		Undefined:                  "undefined",
		Manufacturing:              "manufacturing",
		TimeEfficiencyResearch:     "time efficiency research",
		MaterialEfficiencyResearch: "material efficiency research",
		Copying:                    "copying",
		Invention:                  "invention",
		Reactions:                  "reactions",
	}
	s, ok := m[ia]
	if !ok {
		return "?"
	}
	return s
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
		JobActive:    "active",
		JobCancelled: "cancelled",
		JobDelivered: "delivered",
		JobPaused:    "paused",
		JobReady:     "ready",
		JobReverted:  "reverted",
	}
	x, ok := m[s]
	if !ok {
		return "?"
	}
	return x
}

type CharacterIndustryJob struct {
	Activity           IndustryActivity
	BlueprintID        int64
	BlueprintLocation  *EntityShort[int64]
	BlueprintType      *EntityShort[int32]
	CharacterID        int32
	CompletedCharacter optional.Optional[*EveEntity]
	CompletedDate      optional.Optional[time.Time]
	Cost               optional.Optional[float64]
	Duration           int
	EndDate            time.Time
	FacilityID         int64
	Installer          *EveEntity
	JobID              int32
	LicensedRuns       optional.Optional[int]
	OutputLocation     *EntityShort[int64]
	PauseDate          optional.Optional[time.Time]
	Probability        optional.Optional[float32]
	ProductType        optional.Optional[*EntityShort[int32]]
	Runs               int
	StartDate          time.Time
	Station            *EntityShort[int64]
	Status             IndustryJobStatus
	SuccessfulRuns     optional.Optional[int32]
}
