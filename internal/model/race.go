package model

import (
	"fmt"

	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/api/images"
)

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

// IconURL returns the URL for an icon image of a race.
func (r *EveRace) IconURL(size int) (fyne.URI, error) {
	factionID, ok := r.FactionID()
	if !ok {
		return nil, fmt.Errorf("can not match faction for race: %d", r.ID)
	}
	return images.FactionLogoURL(factionID, size)
}
