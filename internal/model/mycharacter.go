// Package model contains the entity objects, which are used across the app.
package model

import (
	"time"
)

// An Eve Online character owners by the user.
type MyCharacter struct {
	Character     *EveCharacter
	ID            int32
	LastLoginAt   time.Time
	Location      *EveSolarSystem
	Ship          *EveType
	SkillPoints   int
	WalletBalance float64
}

// A shortened version of MyCharacter.
type MyCharacterShort struct {
	ID              int32
	Name            string
	CorporationName string
}
