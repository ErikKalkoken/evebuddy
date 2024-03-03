package storage

import (
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

type Token struct {
	AccessToken   string
	CharacterID   int32 `gorm:"primaryKey"`
	CharacterName string
	ExpiresAt     time.Time
	RefreshToken  string
	TokenType     string
}

func (t Token) String() string {
	return t.CharacterName
}

func (t *Token) IconUrl(size int) fyne.URI {
	s := fmt.Sprintf("https://images.evetech.net/characters/%d/portrait?size=%d", t.CharacterID, size)
	u, err := storage.ParseURI(s)
	if err != nil {
		log.Fatal((err))
	}
	log.Println(u)
	return u
}
