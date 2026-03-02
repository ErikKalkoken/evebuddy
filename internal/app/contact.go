package app

import (
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type StandingCategory uint

const (
	TerribleStanding StandingCategory = iota + 1
	BadStanding
	NeutralStanding
	GoodStanding
	ExcellentStanding
)

func (sc StandingCategory) String() string {
	switch sc {
	case TerribleStanding:
		return "terrible"
	case BadStanding:
		return "bad"
	case NeutralStanding:
		return "neutral"
	case GoodStanding:
		return "good"
	case ExcellentStanding:
		return "excellent"
	}
	return ""
}

func NewStandingCategory(v float64) StandingCategory {
	switch {
	case v < -5:
		return TerribleStanding
	case v < 0:
		return BadStanding
	case v == 0:
		return NeutralStanding
	case v <= 5:
		return GoodStanding
	case v > 5:
		return ExcellentStanding
	}
	panic(fmt.Sprintf("not reachable: %v", v))
}

type CharacterContact struct {
	CharacterID int64
	Contact     *EveEntity
	IsBlocked   optional.Optional[bool]
	IsWatched   optional.Optional[bool]
	Standing    float64
}
