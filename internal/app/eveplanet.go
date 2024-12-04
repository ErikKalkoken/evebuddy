package app

import (
	"regexp"
)

var rePlanetType = regexp.MustCompile(`Planet \((\S*)\)`)

// EvePlanet is a planet in Eve Online.
type EvePlanet struct {
	ID          int32
	Name        string
	SolarSystem *EveSolarSystem
	Type        *EveType
}

func (ep EvePlanet) TypeDisplay() string {
	if ep.Type == nil {
		return ""
	}
	m := rePlanetType.FindStringSubmatch(ep.Type.Name)
	if m == nil || len(m) < 2 {
		return ""
	}
	return m[1]
}
