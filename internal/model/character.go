// Package model contains the entity objects, which are used across the app.
package model

import (
	"example/evebuddy/internal/api/images"
	"time"

	"fyne.io/fyne/v2"
)

// An Eve Online character.
type Character struct {
	Alliance       EveEntity
	Birthday       time.Time
	Corporation    EveEntity
	Description    string
	Faction        EveEntity
	Gender         string
	ID             int32
	MailUpdatedAt  time.Time
	Name           string
	SecurityStatus float64
	SkillPoints    int
	SolarSystem    EveEntity
	WalletBalance  float64
}

// HasAlliance reports wether the character is member of an alliance.
func (c *Character) HasAlliance() bool {
	return c.Alliance.ID != 0
}

// HasFaction reports wether the character is member of a faction.
func (c *Character) HasFaction() bool {
	return c.Faction.ID != 0
}

// PortraitURL returns an image URL for a portrait of a character
func (c *Character) PortraitURL(size int) (fyne.URI, error) {
	return images.CharacterPortraitURL(int32(c.ID), size)
}
