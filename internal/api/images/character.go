// Package images provides image URLs for the Eve Online image server.
package images

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

const PlaceholderCharacterID = 1
const baseURL = "https://images.evetech.net"

// PortraitURL returns an image URL for a portrait of a character
func CharacterPortraitURL(charID int32, size int) (fyne.URI, error) {
	switch size {
	case 32, 64, 128, 256, 512, 1024:
		// valid size
	default:
		return nil, fmt.Errorf("invalid size %d", size)
	}
	s := fmt.Sprintf("%s/characters/%d/portrait?size=%d", baseURL, charID, size)
	u, err := storage.ParseURI(s)
	if err != nil {
		return nil, err
	}
	return u, nil
}
