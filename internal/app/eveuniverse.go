package app

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type EveAlliance struct {
	CreatorCorporation  *EveEntity
	Creator             *EveEntity
	DateFounded         time.Time
	ExecutorCorporation optional.Optional[*EveEntity]
	Faction             optional.Optional[*EveEntity]
	ID                  int64
	Name                string
	Ticker              string
}

func (ea EveAlliance) ToEveEntity() *EveEntity {
	return &EveEntity{ID: ea.ID, Name: ea.Name, Category: EveEntityAlliance}
}

// eveBloodline2IconID maps a bloodline ID to an eveIcon ID.
// A missing mapping means there is no icon for the bloodline.
var eveBloodline2IconID = map[int64]int64{
	1:  1633,
	2:  1631,
	3:  1634,
	4:  1635,
	5:  1628,
	6:  1632,
	7:  1629,
	8:  1630,
	11: 3022,
	12: 3024,
	13: 3023,
	14: 3021,
	19: 21404,
}

// EveBloodline is a bloodline in EVE Online.
type EveBloodline struct {
	Charisma     optional.Optional[int64]
	Corporation  *EveEntity
	Description  string
	ID           int64
	Intelligence optional.Optional[int64]
	Memory       optional.Optional[int64]
	Name         string
	Perception   optional.Optional[int64]
	Race         *EveRace
	ShipTypeID   optional.Optional[int64]
	Willpower    optional.Optional[int64]
}

// Logo returns the logo for a bloodline and reports whether one exists.
func (eb *EveBloodline) Logo() (fyne.Resource, bool) {
	if eb == nil {
		return nil, false
	}
	iconID, ok := eveBloodline2IconID[eb.ID]
	if !ok {
		return nil, false
	}
	return eveicon.FromID(iconID)
}

// TODO: SecurityStatus not really optional. Change the DB field to nullable and remove workaround.

// EveCharacter is a character in EVE Online.
type EveCharacter struct {
	Alliance         optional.Optional[*EveEntity]
	Birthday         time.Time
	Bloodline        optional.Optional[*EntityShort] // optional, because added later
	Corporation      *EveEntity
	CorporationTitle optional.Optional[string]
	Description      optional.Optional[string]
	Faction          optional.Optional[*EveEntity]
	Gender           string
	ID               int64
	Name             string
	Race             *EveRace
	SecurityStatus   optional.Optional[float64]
}

func (ec EveCharacter) DescriptionPlain() string {
	return evehtml.ToPlain(ec.Description.ValueOrZero())
}

// EntityIDs returns the IDs of all entities for a character.
func (ec EveCharacter) EntityIDs() set.Set[int64] {
	s := set.Of(ec.ID, ec.Corporation.ID)
	if v, ok := ec.Alliance.Value(); ok {
		s.Add(v.ID)
	}
	if v, ok := ec.Faction.Value(); ok {
		s.Add(v.ID)
	}
	return s
}

// Equal reports whether two characters are equal.
// Two characters must have the same values in all fields to be equal.
func (ec EveCharacter) Equal(other *EveCharacter) bool {
	if other == nil {
		return false
	}
	return ec.ID == other.ID && ec.Hash() == other.Hash()
}

// Hash returns the hash for this character.
// It can be used to detect changes between instances.
func (ec EveCharacter) Hash() string {
	allianceID := optional.Map(ec.Alliance, 0, func(x *EveEntity) int64 {
		return x.ID
	})
	factionID := optional.Map(ec.Faction, 0, func(x *EveEntity) int64 {
		return x.ID
	})
	bloodlineID := optional.Map(ec.Bloodline, 0, func(x *EntityShort) int64 {
		return x.ID
	})
	xx := []any{
		allianceID,
		bloodlineID,
		ec.Birthday,
		ec.Corporation.ID,
		ec.CorporationTitle,
		ec.Description,
		factionID,
		ec.Gender,
		ec.ID,
		ec.Name,
		ec.Race.ID,
		math.Round(ec.SecurityStatus.ValueOrZero() * 100),
	}
	var s []string
	for _, x := range xx {
		s = append(s, fmt.Sprint(x))
	}
	h1 := md5.Sum([]byte(strings.Join(s, "-")))
	h2 := hex.EncodeToString(h1[:])
	return h2
}

func (ec EveCharacter) ToEveEntity() *EveEntity {
	return &EveEntity{ID: ec.ID, Name: ec.Name, Category: EveEntityCharacter}
}

// EveCorporation is a corporation in EVE Online.
type EveCorporation struct {
	Alliance    optional.Optional[*EveEntity]
	Ceo         optional.Optional[*EveEntity]
	Creator     optional.Optional[*EveEntity]
	DateFounded optional.Optional[time.Time]
	Description string
	Faction     optional.Optional[*EveEntity]
	HomeStation optional.Optional[*EveEntity]
	ID          int64
	MemberCount int64
	Name        string
	Shares      optional.Optional[int64]
	TaxRate     float64
	Ticker      string
	URL         optional.Optional[string]
	WarEligible optional.Optional[bool]
	Timestamp   time.Time
}

func (ec EveCorporation) DescriptionPlain() string {
	return evehtml.ToPlain(ec.Description)
}

func (ec EveCorporation) ToEveEntity() *EveEntity {
	return &EveEntity{ID: ec.ID, Name: ec.Name, Category: EveEntityCorporation}
}

// Equal reports whether two characters are equal.
// Two characters must have the same values in all fields to be equal.
func (ec EveCorporation) Equal(other *EveCorporation) bool {
	if other == nil {
		return false
	}
	return ec.ID == other.ID && ec.Hash() == other.Hash()
}

func (ec EveCorporation) Hash() string {
	allianceID := optional.Map(ec.Alliance, 0, func(x *EveEntity) int64 {
		return x.ID
	})
	ceoID := optional.Map(ec.Ceo, 0, func(x *EveEntity) int64 {
		return x.ID
	})
	creatorID := optional.Map(ec.Creator, 0, func(x *EveEntity) int64 {
		return x.ID
	})
	factionID := optional.Map(ec.Faction, 0, func(x *EveEntity) int64 {
		return x.ID
	})
	homeStationID := optional.Map(ec.HomeStation, 0, func(x *EveEntity) int64 {
		return x.ID
	})
	xx := []any{
		allianceID,
		ceoID,
		creatorID,
		ec.DateFounded.ValueOrZero().UTC(),
		ec.Description,
		factionID,
		homeStationID,
		ec.ID,
		ec.MemberCount,
		ec.Name,
		ec.Shares,
		math.Round(float64(ec.TaxRate) * 100),
		ec.Ticker,
		ec.URL,
		ec.WarEligible,
	}
	var s []string
	for _, x := range xx {
		s = append(s, fmt.Sprint(x))
	}
	h1 := md5.Sum([]byte(strings.Join(s, "-")))
	h2 := hex.EncodeToString(h1[:])
	return h2
}

// EveFaction is a faction in EVE Online.
type EveFaction struct {
	ID                 int64
	Corporation        optional.Optional[*EveEntity]
	Description        string
	IsUnique           bool
	MilitiaCorporation optional.Optional[*EveEntity]
	Name               string
	SizeFactor         float64
	SolarSystem        optional.Optional[*EveEntity]
	StationCount       int64
	StationSystemCount int64
}

// EveRace is a race in EVE Online.
type EveRace struct {
	Faction     optional.Optional[*EveEntity] // optional, because added later
	Description string
	Name        string
	ID          int64
}

// FactionID returns the faction ID of a race.
func (er EveRace) FactionID() (int64, bool) {
	m := map[int64]int64{
		1:   500001,
		2:   500002,
		4:   500003,
		8:   500004,
		16:  500005,
		135: 500026,
	}
	factionID, ok := m[er.ID]
	return factionID, ok
}

// EveSchematic is a schematic for planetary industry in EVE Online.
type EveSchematic struct {
	ID        int64
	CycleTime int
	Name      string
}

func (es EveSchematic) Icon() (fyne.Resource, bool) {
	return eveicon.FromSchematicID(es.ID)
}

type EveShipSkill struct {
	ID          int64
	Rank        uint
	ShipTypeID  int64
	SkillTypeID int64
	SkillName   string
	SkillLevel  uint
}

type MembershipHistoryItem struct {
	EndDate      time.Time
	Days         int
	IsDeleted    optional.Optional[bool]
	IsOldest     bool
	Organization *EveEntity
	RecordID     int64
	StartDate    time.Time
}

func (hi MembershipHistoryItem) OrganizationName() string {
	if hi.Organization != nil {
		return hi.Organization.Name
	}
	return "?"
}
