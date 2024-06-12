package model

import (
	"math"
)

type SolarSystemSecurityType uint

const (
	NullSec SolarSystemSecurityType = iota
	LowSec
	HighSec
	SuperHighSec
)

// EveSolarSystem is a solar system in Eve Online.
type EveSolarSystem struct {
	Constellation  *EveConstellation
	ID             int32
	Name           string
	SecurityStatus float32
}

func (es EveSolarSystem) SecurityType() SolarSystemSecurityType {
	switch v := math.Round(float64(es.SecurityStatus)*10) / 10; {
	case v >= 0.9:
		return SuperHighSec
	case v >= 0.5:
		return HighSec
	case v > 0:
		return LowSec
	}
	return NullSec
}
