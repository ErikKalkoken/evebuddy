package app

import "time"

type CharacterPlanet struct {
	CharacterID  int32
	EvePlanet    *EvePlanet
	LastUpdate   time.Time
	NumPins      int
	UpgradeLevel int
}
