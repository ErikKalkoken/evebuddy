package storage

import (
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

// A SSO token belonging to a character
type Token struct {
	AccessToken  string
	CharacterID  int32
	Character    Character
	ExpiresAt    time.Time
	RefreshToken string
	TokenType    string
}

func (t *Token) Save() error {
	if err := db.Where("character_id = ?", t.CharacterID).Save(t).Error; err != nil {
		return err
	}
	return nil
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

func FirstToken() (*Token, error) {
	var token Token
	if err := db.Joins("Character").First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func FindToken(characterId int32) (*Token, error) {
	var token Token
	err := db.Joins("Character").First(&token, "character_id = ?", characterId).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}
