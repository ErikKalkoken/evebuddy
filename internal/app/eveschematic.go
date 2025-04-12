package app

import (
	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
)

// EveSchematic is a schematic for planetary industry in Eve Online.
type EveSchematic struct {
	ID        int32
	CycleTime int
	Name      string
}

func (es EveSchematic) Icon() (fyne.Resource, bool) {
	return eveicon.FromSchematicID(es.ID)
}
