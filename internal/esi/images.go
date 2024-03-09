package esi

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

const PlaceholderCharacterID = 1

// PortraitURL returns an image URL for a portrait of a character
func CharacterPortraitURL(charID int32, size int) fyne.URI {
	switch size {
	case 32, 64, 128, 256, 512, 1024:
		// valid size
	default:
		log.Fatalf("Invalid size %d", size)
	}
	s := fmt.Sprintf("https://images.evetech.net/characters/%d/portrait?size=%d", charID, size)
	u, err := storage.ParseURI(s)
	if err != nil {
		log.Fatal((err))
	}
	return u
}
