package app

import (
	"math"
	"time"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type EveAlliance struct {
	CreatorCorporation  *EveEntity
	Creator             *EveEntity
	DateFounded         time.Time
	ExecutorCorporation *EveEntity
	Faction             *EveEntity
	ID                  int32
	Name                string
	Ticker              string
}

func (ea EveAlliance) ToEveEntity() *EveEntity {
	return &EveEntity{ID: ea.ID, Name: ea.Name, Category: EveEntityAlliance}
}

// TODO: Add Bloodline (e.g. to show in character description)

// EveCharacter is a character in Eve Online.
type EveCharacter struct {
	Alliance       *EveEntity
	Birthday       time.Time
	Corporation    *EveEntity
	Description    string
	Faction        *EveEntity
	Gender         string
	ID             int32
	Name           string
	Race           *EveRace
	SecurityStatus float64
	Title          string
}

func (ec EveCharacter) AllianceID() int32 {
	if !ec.HasAlliance() {
		return 0
	}
	return ec.Alliance.ID
}

func (ec EveCharacter) AllianceName() string {
	if !ec.HasAlliance() {
		return ""
	}
	return ec.Alliance.Name
}

func (ec EveCharacter) FactionID() int32 {
	if !ec.HasFaction() {
		return 0
	}
	return ec.Faction.ID
}

func (ec EveCharacter) FactionName() string {
	if !ec.HasFaction() {
		return ""
	}
	return ec.Faction.Name
}

// HasAlliance reports whether the character is member of an alliance.
func (ec EveCharacter) HasAlliance() bool {
	return ec.Alliance != nil
}

// HasFaction reports whether the character is member of a faction.
func (ec EveCharacter) HasFaction() bool {
	return ec.Faction != nil
}

func (ec EveCharacter) DescriptionPlain() string {
	return evehtml.ToPlain(ec.Description)
}

func (ec EveCharacter) RaceDescription() string {
	if ec.Race == nil {
		return ""
	}
	return ec.Race.Description
}

func (ec EveCharacter) ToEveEntity() *EveEntity {
	return &EveEntity{ID: ec.ID, Name: ec.Name, Category: EveEntityCharacter}
}

// IsIdentical reports whether two characters are identical.
// Two characters must have the same values in all fields to be identical.
func (ec EveCharacter) IsIdentical(other EveCharacter) bool {
	return ec.ID == other.ID &&
		ec.AllianceID() == other.AllianceID() &&
		ec.Birthday.Equal(other.Birthday) &&
		ec.Corporation.ID == other.Corporation.ID &&
		ec.Description == other.Description &&
		ec.FactionID() == other.FactionID() &&
		ec.Gender == other.Gender &&
		ec.Name == other.Name &&
		ec.Race.ID == other.Race.ID &&
		math.Abs(ec.SecurityStatus-other.SecurityStatus) < 0.01 &&
		ec.Title == other.Title
}

// EveCorporation is a corporation in Eve Online.
type EveCorporation struct {
	Alliance    *EveEntity
	Ceo         *EveEntity
	Creator     *EveEntity
	DateFounded optional.Optional[time.Time]
	Description string
	Faction     *EveEntity
	HomeStation *EveEntity
	ID          int32
	MemberCount int
	Name        string
	Shares      optional.Optional[int]
	TaxRate     float32
	Ticker      string
	URL         string
	WarEligible bool
	Timestamp   time.Time
}

func (ec EveCorporation) AllianceID() int32 {
	if !ec.HasAlliance() {
		return 0
	}
	return ec.Alliance.ID
}

func (ec EveCorporation) FactionID() int32 {
	if !ec.HasFaction() {
		return 0
	}
	return ec.Faction.ID
}

func (ec EveCorporation) HasAlliance() bool {
	return ec.Alliance != nil
}

func (ec EveCorporation) HasFaction() bool {
	return ec.Faction != nil
}

func (ec EveCorporation) HomeStationID() int32 {
	if ec.HomeStation == nil {
		return 0
	}
	return ec.HomeStation.ID
}
func (ec EveCorporation) DescriptionPlain() string {
	return evehtml.ToPlain(ec.Description)
}

func (ec EveCorporation) ToEveEntity() *EveEntity {
	return &EveEntity{ID: ec.ID, Name: ec.Name, Category: EveEntityCorporation}
}

// IsIdentical reports whether two characters are identical.
// Two characters must have the same values in all fields to be identical.
func (ec EveCorporation) IsIdentical(other EveCorporation) bool {
	return ec.ID == other.ID &&
		ec.AllianceID() == other.AllianceID() &&
		ec.Ceo.ID == other.Ceo.ID &&
		ec.Creator.ID == other.Creator.ID &&
		ec.DateFounded.ValueOrZero().Equal(other.DateFounded.ValueOrZero()) &&
		ec.Description == other.Description &&
		ec.FactionID() == other.FactionID() &&
		ec.HomeStationID() == other.HomeStationID() &&
		ec.MemberCount == other.MemberCount &&
		ec.Name == other.Name &&
		ec.Shares == other.Shares &&
		math.Abs(float64(ec.TaxRate-other.TaxRate)) < 0.01
}

// TODO: Add race alliance

// EveRace is a race in Eve Online.
type EveRace struct {
	Description string
	Name        string
	ID          int32
}

// FactionID returns the faction ID of a race.
func (er EveRace) FactionID() (int32, bool) {
	m := map[int32]int32{
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

// EveSchematic is a schematic for planetary industry in Eve Online.
type EveSchematic struct {
	ID        int32
	CycleTime int
	Name      string
}

func (es EveSchematic) Icon() (fyne.Resource, bool) {
	return eveicon.FromSchematicID(es.ID)
}

type EveShipSkill struct {
	ID          int64
	Rank        uint
	ShipTypeID  int32
	SkillTypeID int32
	SkillName   string
	SkillLevel  uint
}

type MembershipHistoryItem struct {
	EndDate      time.Time
	Days         int
	IsDeleted    bool
	IsOldest     bool
	Organization *EveEntity
	RecordID     int
	StartDate    time.Time
}

func (hi MembershipHistoryItem) OrganizationName() string {
	if hi.Organization != nil {
		return hi.Organization.Name
	}
	return "?"
}
