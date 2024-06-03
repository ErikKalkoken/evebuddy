package icons

import (
	"fyne.io/fyne/v2"
)

// Get returns an icon resource for the given icon ID.
func Get(id int32) *fyne.StaticResource {
	return id2fileMap[int(id)]
}
