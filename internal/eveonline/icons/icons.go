// Package icons contains Eve online icons as fyne resources.
package icons

import (
	"fyne.io/fyne/v2"
)

// Popular icon IDs
const (
	Charisma      = 1378
	CloningCenter = 21596
	Intelligence  = 1380
	Memory        = 1381
	Perception    = 1382
	Undefined     = 0
	Willpower     = 1379
)

// GetResource returns an icon resource for an icon ID and reports if it was found.
// When the icon was not found it will the icon for ID 0 as substitute.
func GetResource(id int32) (*fyne.StaticResource, bool) {
	r, ok := id2fileMap[id]
	if !ok {
		return id2fileMap[0], false
	}
	return r, true
}