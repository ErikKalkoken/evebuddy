// Package model contains the entity objects, which are used across the app.
package model

import (
	"time"
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

type MyCharacterShort struct {
	ID              int32
	Name            string
	CorporationName string
}
