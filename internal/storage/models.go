package storage

import (
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"gorm.io/gorm"
)

// An Eve Online character
type Character struct {
	ID   int32 `gorm:"primaryKey"`
	Name string
}

// A SSO token belonging to a character
type Token struct {
	AccessToken  string
	CharacterID  int32
	Character    Character
	ExpiresAt    time.Time
	RefreshToken string
	TokenType    string
}

func (t *Token) IsValid() bool {
	return t.ExpiresAt.After(time.Now())
}

func (t *Token) IconUrl(size int) fyne.URI {
	switch size {
	case 32, 64, 128, 256, 512, 1024:
		// valid size
	default:
		log.Fatalf("Invalid size %d", size)
	}
	s := fmt.Sprintf("https://images.evetech.net/characters/%d/portrait?size=%d", t.CharacterID, size)
	u, err := storage.ParseURI(s)
	if err != nil {
		log.Fatal((err))
	}
	log.Println(u)
	return u
}

type Contact struct {
	gorm.Model
	ID       uint
	Name     string
	Type     string
	Standing float32
}
