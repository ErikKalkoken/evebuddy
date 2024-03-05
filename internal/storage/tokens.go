package storage

import (
	"time"
)

// A SSO token belonging to a character.
type Token struct {
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

// IsValid reports wether a token is still valid.
func (t *Token) IsValid() bool {
	return t.ExpiresAt.After(time.Now())
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
