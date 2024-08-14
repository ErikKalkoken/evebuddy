package app

// EveMoon is a moon in Eve Online.
type EveMoon struct {
	ID          int32
	Name        string
	SolarSystem *EveSolarSystem
}
