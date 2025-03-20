package infowindow

import (
	"maps"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type infoVariant uint

const (
	infoNotSupported infoVariant = iota
	infoAlliance
	infoCharacter
	infoConstellation
	infoCorporation
	infoInventoryType
	infoLocation
	infoRegion
	infoSolarSystem
)

var eveEntityCategory2InfoVariant = map[app.EveEntityCategory]infoVariant{
	app.EveEntityAlliance:      infoAlliance,
	app.EveEntityCharacter:     infoCharacter,
	app.EveEntityConstellation: infoConstellation,
	app.EveEntityCorporation:   infoCorporation,
	app.EveEntityRegion:        infoRegion,
	app.EveEntitySolarSystem:   infoSolarSystem,
	app.EveEntityStation:       infoLocation,
	app.EveEntityInventoryType: infoInventoryType,
}

func eveEntity2InfoVariant(ee *app.EveEntity) infoVariant {
	v, ok := eveEntityCategory2InfoVariant[ee.Category]
	if !ok {
		return infoNotSupported
	}
	return v

}

func SupportedEveEntities() set.Set[app.EveEntityCategory] {
	return set.Collect(maps.Keys(eveEntityCategory2InfoVariant))

}
