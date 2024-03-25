package models

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

// FetchToken returns the token for a character
func FetchToken(characterID int32) (*Token, error) {
	row := db.QueryRow(
		`SELECT *
		FROM tokens
		JOIN characters ON characters.id = tokens.character_id
		WHERE tokens.character_id = ?;`,
		characterID,
	)
	var o Token
	err := row.Scan(
		&o.AccessToken,
		&o.CharacterID,
		&o.ExpiresAt,
		&o.RefreshToken,
		&o.TokenType,
		&o.Character.ID,
		&o.Character.Name,
	)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
