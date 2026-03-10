// Package ui provides globals for UI packages.
package ui

import "github.com/ErikKalkoken/evebuddy/internal/app"

type InfoWindow interface {
	Show(o *app.EveEntity)
	ShowLocation(id int64)
	ShowRace(id int64)
	ShowType(typeID, characterID int64)
}
