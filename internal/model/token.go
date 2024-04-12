package model

import (
	"fmt"
	"time"
)

// A SSO token belonging to a character.
type Token struct {
	AccessToken  string `db:"access_token"`
	CharacterID  int32  `db:"character_id"`
	Character    Character
	ExpiresAt    time.Time `db:"expires_at"`
	RefreshToken string    `db:"refresh_token"`
	TokenType    string    `db:"token_type"`
}

// Save updates or creates a token.
func (t *Token) Save() error {
	if t.Character.ID != 0 {
		t.CharacterID = t.Character.ID
	}
	if t.Character.ID == 0 {
		return fmt.Errorf("can not save token without character")
	}
	t.CharacterID = t.Character.ID
	_, err := db.NamedExec(`
		INSERT INTO tokens (
			access_token,
			character_id,
			expires_at,
			refresh_token,
			token_type
		)
		VALUES (
			:access_token,
			:character_id,
			:expires_at,
			:refresh_token,
			:token_type
		)
		ON CONFLICT (character_id) DO
		UPDATE SET
			access_token=:access_token,
			expires_at=:expires_at,
			refresh_token=:refresh_token,
			token_type=:token_type;`,
		*t,
	)
	if err != nil {
		return err
	}
	return nil
}

// RemainsValid reports wether a token remains valid within a duration
func (t *Token) RemainsValid(d time.Duration) bool {
	return t.ExpiresAt.After(time.Now().Add(d))
}

func (t *Token) GetCharacter() error {
	c, err := GetCharacter(t.CharacterID)
	if err != nil {
		return err
	}
	t.Character = c
	return nil
}

// GetToken returns the token for a character
func GetToken(characterID int32) (Token, error) {
	var t Token
	err := db.Get(
		&t,
		`SELECT tokens.*
		FROM tokens
		JOIN characters ON characters.id = tokens.character_id
		WHERE tokens.character_id = ?;`,
		characterID,
	)
	if err != nil {
		return t, err
	}
	return t, nil
}
