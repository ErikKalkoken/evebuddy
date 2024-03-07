package storage

import (
	"time"

	"gorm.io/gorm"
)

// A SSO token belonging to a character.
type Token struct {
	gorm.Model
	AccessToken  string
	CharacterID  int32
	Character    Character
	ExpiresAt    time.Time
	RefreshToken string
	TokenType    string
}

// Save updates or creates a token.
func (t *Token) Save() error {
	if err := db.Where("character_id = ?", t.CharacterID).Save(t).Error; err != nil {
		return err
	}
	return nil
}

// RemainsValid reports wether a token remains valid within a duration
func (t *Token) RemainsValid(d time.Duration) bool {
	return t.ExpiresAt.After(time.Now().Add(d))
}

// FetchToken returns the token for a character
func FetchToken(characterId int32) (*Token, error) {
	var token Token
	err := db.Joins("Character").First(&token, "character_id = ?", characterId).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}
