package storage

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"gorm.io/gorm"
)

// An Eve Online character.
type Character struct {
	gorm.Model
	ID   int32 `gorm:"primaryKey"`
	Name string
}

// Save updates or creates a character.
func (c *Character) Save() error {
	if err := db.Save(c).Error; err != nil {
		return err
	}
	return nil
}

// PortraitURL returns an image URL for a portrait of a character
func (c *Character) PortraitURL(size int) fyne.URI {
	switch size {
	case 32, 64, 128, 256, 512, 1024:
		// valid size
	default:
		log.Fatalf("Invalid size %d", size)
	}
	s := fmt.Sprintf("https://images.evetech.net/characters/%d/portrait?size=%d", c.ID, size)
	u, err := storage.ParseURI(s)
	if err != nil {
		log.Fatal((err))
	}
	log.Println(u)
	return u
}

// FetchFirstCharacter returns a random character.
func FetchFirstCharacter() (*Character, error) {
	var obj Character
	if err := db.First(&obj).Error; err != nil {
		return nil, err
	}
	return &obj, nil
}

func FetchCharacter(characterID int32) (*Character, error) {
	var obj Character
	if err := db.First(&obj, characterID).Error; err != nil {
		return nil, err
	}
	return &obj, nil
}

// FetchAllCharacters returns all characters.
func FetchAllCharacters() ([]Character, error) {
	var objs []Character
	err := db.Order("name").Find(&objs).Error
	if err != nil {
		return nil, err
	}
	return objs, nil
}
