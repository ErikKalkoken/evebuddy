// Package model contains the entity objects, which are used across the app.
package model

import "database/sql"

// An Eve Online character owners by the user.
type MyCharacter struct {
	Character     *EveCharacter
	ID            int32
	LastLoginAt   sql.NullTime
	Location      *EveSolarSystem
	Ship          *EveType
	SkillPoints   sql.NullInt64
	WalletBalance sql.NullFloat64
}

// A shortened version of MyCharacter.
type MyCharacterShort struct {
	ID              int32
	Name            string
	CorporationName string
}
