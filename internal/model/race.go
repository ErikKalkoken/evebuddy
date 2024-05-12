package model

type EveRace struct {
	Description string
	Name        string
	ID          int32
}

func (r *EveRace) FactionID() (int32, bool) {
	m := map[int32]int32{
		1: 500001,
		2: 500002,
		4: 500003,
		8: 500004,
	}
	factionID, ok := m[r.ID]
	return factionID, ok
}
