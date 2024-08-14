package app

// EvePlanet is a planet in Eve Online.
type EvePlanet struct {
	ID          int32
	Name        string
	SolarSystem *EveSolarSystem
	Type        *EveType
}
