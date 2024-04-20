package model

import (
	"example/evebuddy/internal/api/images"
	"fmt"

	"fyne.io/fyne/v2"
)

type Race struct {
	Description string
	Name        string
	ID          int32
}

func (r *Race) FactionID() (int32, bool) {
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
func (r *Race) IconURL(size int) (fyne.URI, error) {
	factionID, ok := r.FactionID()
	if !ok {
		return nil, fmt.Errorf("can not match faction for race: %d", r.ID)
	}
	return images.FactionLogoURL(factionID, size)
}
