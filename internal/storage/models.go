package storage

import (
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
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

// A mail header belonging to a character
type MailHeader struct {
	CharacterID int32
	Character   Character
	FromID      int32
	From        EveEntity
	MailID      int32
	IsRead      bool
	Subject     string
	TimeStamp   time.Time
}

// An entity in Eve Online
type EveEntity struct {
	Category string
	ID       int32 `gorm:"primaryKey"`
	Name     string
}
