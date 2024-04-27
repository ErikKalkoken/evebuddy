// Package model contains the entity objects, which are used across the app.
package model

import (
	"example/evebuddy/internal/api/images"
	"time"

	"fyne.io/fyne/v2"
)

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

func (c *EveCharacter) AllianceName() string {
	if !c.HasAlliance() {
		return ""
	}
	return c.Alliance.Name
}

func (c *EveCharacter) FactionName() string {
	if !c.HasFaction() {
		return ""
	}
	return c.Faction.Name
}

// HasAlliance reports wether the character is member of an alliance.
func (c *EveCharacter) HasAlliance() bool {
	return c.Alliance != nil
}

// HasFaction reports wether the character is member of a faction.
func (c *EveCharacter) HasFaction() bool {
	return c.Faction != nil
}

// PortraitURL returns an image URL for a portrait of a character
func (c *EveCharacter) PortraitURL(size int) (fyne.URI, error) {
	return images.CharacterPortraitURL(int32(c.ID), size)
}
