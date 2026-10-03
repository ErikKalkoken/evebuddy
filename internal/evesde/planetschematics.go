package evesde

import (
	"slices"
	"time"
)

// PlanetSchematic is the recipe of a planetary industry schematic.
type PlanetSchematic struct {
	ID             int64
	CycleTime      time.Duration
	Inputs         []PlanetSchematicInput // sorted by type ID
	OutputTypeID   int64
	OutputQuantity int64
}

// PlanetSchematicInput is an input commodity of a planetary industry schematic.
type PlanetSchematicInput struct {
	TypeID   int64
	Quantity int64
}

type planetSchematic struct {
	cycleTime      int64
	inputs         []PlanetSchematicInput
	outputTypeID   int64
	outputQuantity int64
}

// PlanetSchematicByID returns the recipe for a planetary industry schematic
// and reports whether it was found.
func PlanetSchematicByID(id int64) (PlanetSchematic, bool) {
	s, ok := planetSchematics[id]
	if !ok {
		return PlanetSchematic{}, false
	}
	o := PlanetSchematic{
		ID:             id,
		CycleTime:      time.Duration(s.cycleTime) * time.Second,
		Inputs:         slices.Clone(s.inputs),
		OutputTypeID:   s.outputTypeID,
		OutputQuantity: s.outputQuantity,
	}
	return o, true
}
