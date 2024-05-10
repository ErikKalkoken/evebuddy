// Package model contains the entity objects, which are used across the app.
package model

import (
	"time"

	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/eveonline/images"
)

// An owned character in Eve Online.
type MyCharacter struct {
	Character     *EveCharacter
	ID            int32
	LastLoginAt   time.Time
	Location      *EveSolarSystem
	Ship          *EveType
	SkillPoints   int
	WalletBalance float64
}

// PortraitURL returns an image URL for a portrait of a character
func (c *MyCharacter) PortraitURL(size int) (fyne.URI, error) {
	return images.CharacterPortraitURL(int32(c.ID), size)
}

type MyCharacterShort struct {
	ID              int32
	Name            string
	CorporationName string
}
