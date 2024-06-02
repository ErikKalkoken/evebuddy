package model

import (
	"database/sql"
)

// An Eve Online character owners by the user.
type Character struct {
	EveCharacter  *EveCharacter
	Home          *Location
	ID            int32
	LastLoginAt   sql.NullTime
	Location      *Location
	Ship          *EveType
	TotalSP       sql.NullInt64
	UnallocatedSP sql.NullInt64
	WalletBalance sql.NullFloat64
}

// A shortened version of Character.
type CharacterShort struct {
	ID   int32
	Name string
}
