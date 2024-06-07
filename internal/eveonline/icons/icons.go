// Package icons contains Eve online icons as fyne resources.
package icons

import (
	"fyne.io/fyne/v2"
)

// An icon name
type Name uint

// Named icons
const (
	Undefined Name = iota
	CloningCenter
	Charisma
	Intelligence
	Memory
	Perception
	Willpower
	HeliumIsotopes
	HydrogenIsotopes
	NitrogenIsotopes
	OxygenIsotopes
	Structure
	Tech1
	Tech2
	Tech3
	Tech4
	Storyline
	Faction
	Deadspace
	Officer
	Caldari
	Minmatar
	Gallente
	Amarr
)

var namedIcons = map[Name]*fyne.StaticResource{
	Undefined:     resource76415Png,
	CloningCenter: resource127641Png,
	Charisma:      resource22321Png,
	Intelligence:  resource22323Png,
	Memory:        resource22324Png,
	Perception:    resource22325Png,
	Willpower:     resource22322Png,
	Structure:     resource2649Png,
	Tech1:         resource7316241Png,
	Tech2:         resource7316242Png,
	Tech3:         resource7316243Png,
	Tech4:         resource7316244Png,
	Storyline:     resource7316245Png,
	Faction:       resource7316246Png,
	Deadspace:     resource7316247Png,
	Officer:       resource7316248Png,
	Caldari:       resource881281Png,
	Minmatar:      resource881282Png,
	Gallente:      resource881283Png,
	Amarr:         resource881284Png,
}

// GetResourceByIconID returns an Eve Online icon by icon ID and reports if it was found.
// When the icon was not found it will return the undefined icon as substitute.
func GetResourceByIconID(id int32) (*fyne.StaticResource, bool) {
	r, ok := id2fileMap[id]
	if !ok {
		return namedIcons[Undefined], false
	}
	return r, true
}

// GetResourceByName returns an Eve Online icon by name and reports if it was found.
// When the icon was not found it will return the undefined icon as substitute.
func GetResourceByName(name Name) *fyne.StaticResource {
	return namedIcons[name]
}
