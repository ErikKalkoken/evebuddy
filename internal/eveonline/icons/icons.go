// Package icons contains Eve online icons as fyne resources.
package icons

import (
	"fyne.io/fyne/v2"
)

// Popular icon IDs
const (
	HeliumIsotopesID int32 = 2699
)

// Enum for defining named icons
type iconName uint

const (
	Undefined iconName = iota
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
	Tech1
	Tech2
	Tech3
)

var namedIcons = map[iconName]*fyne.StaticResource{
	Undefined:        resource76415Png,
	CloningCenter:    resource127641Png,
	Charisma:         resource22321Png,
	Intelligence:     resource22323Png,
	Memory:           resource22324Png,
	Perception:       resource22325Png,
	Willpower:        resource22322Png,
	HeliumIsotopes:   resource516413Png,
	HydrogenIsotopes: resource516414Png,
	NitrogenIsotopes: resource516416Png,
	OxygenIsotopes:   resource516415Png,
	Tech1:            resource7316241Png,
	Tech2:            resource7316242Png,
	Tech3:            resource7316242Png,
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
func GetResourceByName(name iconName) *fyne.StaticResource {
	return namedIcons[name]
}
