// Package icons contains Eve online icons as fyne resources.
package icons

import (
	"fyne.io/fyne/v2"
)

// Popular icon IDs
const (
	IDUndefined          int32 = 0
	IDCharisma           int32 = 1378
	IDIntelligence       int32 = 1380
	IDMemory             int32 = 1381
	IDPerception         int32 = 1382
	IDWillpower          int32 = 1379
	IDHeliumIsotopes     int32 = 2699
	IDHydrogenIsotopes   int32 = 2700
	IDNitrogenIsotopes   int32 = 2702
	IDOxygenIsotopes     int32 = 2701
	IDCloningCenter      int32 = 21596
	IDModuleJumpEnhancer int32 = 97
)

type iconName uint

const (
	NameTech1 iconName = iota
	NameTech2
	NameTech3
)

// GetResourceByIconID returns an icon resource for an icon ID and reports if it was found.
// When the icon was not found it will the icon for ID 0 as substitute.
func GetResourceByIconID(id int32) (*fyne.StaticResource, bool) {
	r, ok := id2fileMap[id]
	if !ok {
		return id2fileMap[IDUndefined], false
	}
	return r, true
}

func GetResourceByName(name iconName) (*fyne.StaticResource, bool) {
	switch name {
	case NameTech1:
		return resource7316241Png, true
	case NameTech2:
		return resource7316242Png, true
	case NameTech3:
		return resource7316242Png, true
	}
	return id2fileMap[IDUndefined], false
}
