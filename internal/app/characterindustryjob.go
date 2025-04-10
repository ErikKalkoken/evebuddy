package app

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// IndustryActivity represents the activity type of an industry job.
// See also: https://github.com/esi/esi-issues/issues/894
type IndustryActivity int32

const (
	None                       IndustryActivity = 0
	Manufacturing              IndustryActivity = 1
	TimeEfficiencyResearch     IndustryActivity = 3
	MaterialEfficiencyResearch IndustryActivity = 4
	Copying                    IndustryActivity = 5
	Invention                  IndustryActivity = 8
	Reactions                  IndustryActivity = 11
)

func (a IndustryActivity) String() string {
	m := map[IndustryActivity]string{
		None:                       "none",
		Manufacturing:              "manufacturing",
		TimeEfficiencyResearch:     "time efficiency research",
		MaterialEfficiencyResearch: "material efficiency research",
		Copying:                    "copying",
		Invention:                  "invention",
		Reactions:                  "reactions",
	}
	s, ok := m[a]
	if !ok {
		return "?"
	}
	return s
}

func (a IndustryActivity) Display() string {
	titler := cases.Title(language.English)
	return titler.String(a.String())
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
		JobActive:    "in progress",
		JobCancelled: "cancelled",
		JobDelivered: "delivered",
		JobPaused:    "halted",
		JobReady:     "ready",
		JobReverted:  "reverted",
	}
	x, ok := m[s]
	if !ok {
		return "?"
	}
	return x
}

func (s IndustryJobStatus) Display() string {
	titler := cases.Title(language.English)
	return titler.String(s.String())
}

func (s IndustryJobStatus) Color() fyne.ThemeColorName {
	m := map[IndustryJobStatus]fyne.ThemeColorName{
		JobActive:    theme.ColorNamePrimary,
		JobCancelled: theme.ColorNameError,
		JobPaused:    theme.ColorNameWarning,
		JobReady:     theme.ColorNameSuccess,
	}
	c, ok := m[s]
	if ok {
		return c
	}
	return theme.ColorNameForeground
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
	Facility           *EntityShort[int64]
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

// StatusCorrected returns a corrected status.
func (j CharacterIndustryJob) StatusCorrected() IndustryJobStatus {
	if j.Status == JobActive && j.EndDate.Before(time.Now()) {
		// Workaroud for known bug: https://github.com/esi/esi-issues/issues/752
		return JobReady
	}
	return j.Status
}

func (j CharacterIndustryJob) IsActive() bool {
	switch s := j.StatusCorrected(); s {
	case JobActive, JobReady, JobPaused:
		return true
	}
	return false
}
