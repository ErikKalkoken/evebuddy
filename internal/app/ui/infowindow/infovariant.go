package infowindow

import (
	"maps"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type InfoVariant uint

const (
	NotSupported InfoVariant = iota
	Alliance
	Character
	Constellation
	Corporation
	InventoryType
	Location
	Region
	SolarSystem
)

var eveEntityCategory2InfoVariant = map[app.EveEntityCategory]InfoVariant{
	app.EveEntityAlliance:      Alliance,
	app.EveEntityCharacter:     Character,
	app.EveEntityConstellation: Constellation,
	app.EveEntityCorporation:   Corporation,
	app.EveEntityRegion:        Region,
	app.EveEntitySolarSystem:   SolarSystem,
	app.EveEntityStation:       Location,
	app.EveEntityInventoryType: InventoryType,
}

func EveEntity2InfoVariant(ee *app.EveEntity) InfoVariant {
	v, ok := eveEntityCategory2InfoVariant[ee.Category]
	if !ok {
		return NotSupported
	}
	return v

}

func SupportedEveEntities() set.Set[app.EveEntityCategory] {
	return set.Collect(maps.Keys(eveEntityCategory2InfoVariant))

}
