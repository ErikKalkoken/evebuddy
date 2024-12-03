package app

import "time"

type CharacterPlanet struct {
	ID           int64
	CharacterID  int32
	EvePlanet    *EvePlanet
	LastUpdate   time.Time
	NumPins      int
	UpgradeLevel int
}
