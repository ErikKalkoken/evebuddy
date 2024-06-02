package model

// EveRace is a race in Eve Online.
type EveRace struct {
	Description string
	Name        string
	ID          int32
}

// FactionID returns the faction ID of a race.
func (er EveRace) FactionID() (int32, bool) {
	m := map[int32]int32{
		1: 500001,
		2: 500002,
		4: 500003,
		8: 500004,
	}
	factionID, ok := m[er.ID]
	return factionID, ok
}
