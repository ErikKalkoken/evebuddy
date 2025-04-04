package app

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// An Eve Online character owners by the user.
type Character struct {
	AssetValue        optional.Optional[float64]
	EveCharacter      *EveCharacter
	Home              *EveLocation
	ID                int32
	IsTrainingWatched bool
	LastCloneJumpAt   optional.Optional[time.Time]
	LastLoginAt       optional.Optional[time.Time]
	Location          *EveLocation
	Ship              *EveType
	TotalSP           optional.Optional[int]
	UnallocatedSP     optional.Optional[int]
	WalletBalance     optional.Optional[float64]
	// Calculated fields
	NextCloneJump optional.Optional[time.Time] // zero time == now
}

// A shortened version of Character.
type CharacterShort struct {
	ID   int32
	Name string
}
