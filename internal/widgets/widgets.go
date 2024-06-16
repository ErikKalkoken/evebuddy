package widgets

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

type InventoryTypeImageProvider interface {
	InventoryTypeBPO(int32, int) (fyne.Resource, error)
	InventoryTypeBPC(int32, int) (fyne.Resource, error)
	InventoryTypeIcon(int32, int) (fyne.Resource, error)
	InventoryTypeRender(int32, int) (fyne.Resource, error)
	InventoryTypeSKIN(int32, int) (fyne.Resource, error)
}

// refreshImageResourceAsync refreshes the resource of an image asynchronously.
// This prevents fyne to wait with rendering an image until a resource is fully loaded from a web server.
func refreshImageResourceAsync(image *canvas.Image, loader func() (fyne.Resource, error)) {
	go func(*canvas.Image) {
		r, err := loader()
		if err != nil {
			slog.Warn("Failed to fetch image resource", "err", err)
		} else {
			image.Resource = r
			image.Refresh()
		}
	}(image)
}
