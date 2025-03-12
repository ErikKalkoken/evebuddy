// Package model contains the entity objects, which are used across the app.
package app

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
)

// TODO: Add Bloodline (e.g. to show in character description)

// An Eve Online character.
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

func (ec EveCharacter) AllianceName() string {
	if !ec.HasAlliance() {
		return ""
	}
	return ec.Alliance.Name
}

func (ec EveCharacter) FactionName() string {
	if !ec.HasFaction() {
		return ""
	}
	return ec.Faction.Name
}

// HasAlliance reports wether the character is member of an alliance.
func (ec EveCharacter) HasAlliance() bool {
	return ec.Alliance != nil
}

// HasFaction reports wether the character is member of a faction.
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
