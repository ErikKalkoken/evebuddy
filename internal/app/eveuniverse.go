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
	ExecutorCorporation *EveEntity
	Faction             *EveEntity
	ID                  int32
	Name                string
	Ticker              string
}

func (ea EveAlliance) EveEntity() *EveEntity {
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

func (ec EveCharacter) DescriptionPlain() string {
	return evehtml.ToPlain(ec.Description)
}

// EntityIDs returns the IDs of all entities for a character.
func (ec EveCharacter) EntityIDs() set.Set[int32] {
	s := set.Of(ec.ID, ec.Corporation.ID)
	if ec.HasAlliance() {
		s.Add(ec.Alliance.ID)
	}
	if ec.HasFaction() {
		s.Add(ec.Faction.ID)
	}
	return s
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

// Equal reports whether two characters are equal.
// Two characters must have the same values in all fields to be equal.
func (ec EveCharacter) Equal(other *EveCharacter) bool {
	return ec.ID == other.ID && ec.Hash() == other.Hash()
}

// Hash returns the hash for this character.
// It can be used to detect changes between instances.
func (ec EveCharacter) Hash() string {
	var corporationID, raceID int32
	if ec.Corporation != nil {
		corporationID = ec.Corporation.ID
	}
	if ec.Race != nil {
		raceID = ec.Race.ID
	}
	xx := []any{
		ec.AllianceID(),
		ec.Birthday,
		corporationID,
		ec.Description,
		ec.FactionID(),
		ec.Gender,
		ec.ID,
		ec.Name,
		raceID,
		math.Round(ec.SecurityStatus * 100),
		ec.Title,
	}
	s := make([]string, 0)
	for _, x := range xx {
		s = append(s, fmt.Sprint(x))
	}
	h1 := md5.Sum([]byte(strings.Join(s, "-")))
	h2 := hex.EncodeToString(h1[:])
	return h2
}

func (ec EveCharacter) RaceDescription() string {
	if ec.Race == nil {
		return ""
	}
	return ec.Race.Description
}

func (ec EveCharacter) EveEntity() *EveEntity {
	return &EveEntity{ID: ec.ID, Name: ec.Name, Category: EveEntityCharacter}
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

func (ec EveCorporation) CeoID() int32 {
	if ec.Ceo == nil {
		return 0
	}
	return ec.Ceo.ID
}

func (ec EveCorporation) CreatorID() int32 {
	if ec.Creator == nil {
		return 0
	}
	return ec.Creator.ID
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

func (ec EveCorporation) EveEntity() *EveEntity {
	return &EveEntity{ID: ec.ID, Name: ec.Name, Category: EveEntityCorporation}
}

// Equal reports whether two characters are equal.
// Two characters must have the same values in all fields to be equal.
func (ec EveCorporation) Equal(other *EveCorporation) bool {
	return ec.ID == other.ID && ec.Hash() == other.Hash()
}

func (ec EveCorporation) Hash() string {
	xx := []any{
		ec.AllianceID(),
		ec.CeoID(),
		ec.CreatorID(),
		ec.DateFounded.ValueOrZero().UTC(),
		ec.Description,
		ec.FactionID(),
		ec.HomeStationID(),
		ec.ID,
		ec.MemberCount,
		ec.Name,
		ec.Shares,
		math.Round(float64(ec.TaxRate) * 100),
		ec.Ticker,
		ec.URL,
		ec.WarEligible,
	}
	s := make([]string, 0)
	for _, x := range xx {
		s = append(s, fmt.Sprint(x))
	}
	h1 := md5.Sum([]byte(strings.Join(s, "-")))
	h2 := hex.EncodeToString(h1[:])
	return h2
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
