package app

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// IndustryJobType represents the type of industry jobs and defines which slots are utilized.
type IndustryJobType uint

const (
	UndefinedJob IndustryJobType = iota
	ManufacturingJob
	ScienceJob
	ReactionJob
)

var industryJobType2Activity = map[IndustryJobType]set.Set[IndustryActivity]{
	ManufacturingJob: set.Of(Manufacturing),
	ScienceJob:       set.Of(TimeEfficiencyResearch, MaterialEfficiencyResearch, Copying, Invention),
	ReactionJob:      set.Of(Reactions),
}

// Activities returns the industry activities that belong to a job type.
func (jt IndustryJobType) Activities() set.Set[IndustryActivity] {
	return industryJobType2Activity[jt]
}

func (jt IndustryJobType) String() string {
	m := map[IndustryJobType]string{
		ManufacturingJob: "manufacturing job",
		ScienceJob:       "science job",
		ReactionJob:      "reactions",
	}
	return m[jt]
}

func (jt IndustryJobType) Display() string {
	titler := cases.Title(language.English)
	return titler.String(jt.String())
}

// CharacterIndustrySlots represents counts of industry slots for a character.
type CharacterIndustrySlots struct {
	Busy          int
	CharacterID   int32
	CharacterName string
	Free          int
	Ready         int
	Total         int
	Type          IndustryJobType
}

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

func (a IndustryActivity) JobType() IndustryJobType {
	for k, v := range industryJobType2Activity {
		if v.Contains(a) {
			return k
		}
	}
	return UndefinedJob
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
	BlueprintLocation  *EveLocationShort
	BlueprintType      *EntityShort[int32]
	CharacterID        int32
	CompletedCharacter optional.Optional[*EveEntity]
	CompletedDate      optional.Optional[time.Time]
	Cost               optional.Optional[float64]
	Duration           int
	EndDate            time.Time
	Facility           *EveLocationShort
	Installer          *EveEntity
	JobID              int32
	LicensedRuns       optional.Optional[int]
	OutputLocation     *EveLocationShort
	PauseDate          optional.Optional[time.Time]
	Probability        optional.Optional[float32]
	ProductType        optional.Optional[*EntityShort[int32]]
	Runs               int
	StartDate          time.Time
	Station            *EveLocationShort
	Status             IndustryJobStatus
	SuccessfulRuns     optional.Optional[int32]
}

type IndustryJobActivityCount struct {
	InstallerID int32
	Activity    IndustryActivity
	Status      IndustryJobStatus
	Count       int
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
