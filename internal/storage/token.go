package storage

import (
	"log"
	"time"
)

type Token struct {
	AccessToken   string
	CharacterID   int32
	CharacterName string
	ExpiresAt     time.Time
	RefreshToken  string
	TokenType     string
}

func (t *Token) Store() error {
	s := `
		INSERT OR REPLACE INTO tokens(
			access_token,
			character_id,
			character_name,
			expires_at,
			refresh_token,
			token_type
		)
		VALUES(?, ?, ?, ?, ?, ?);
	`
	_, err := DB.Exec(
		s,
		t.AccessToken,
		t.CharacterID,
		t.CharacterName,
		t.ExpiresAt.Unix(),
		t.RefreshToken,
		t.TokenType,
	)
	if err != nil {
		log.Printf("Stored token for %s", t.CharacterName)
	}
	return err
}

func FindToken(characterId int32) (*Token, error) {
	s := `
		SELECT access_token, character_id, character_name, refresh_token, token_type
		FROM tokens WHERE character_id = ?;
	`
	var t Token
	err := DB.QueryRow(s, characterId).Scan(
		&t.AccessToken,
		&t.CharacterID,
		&t.CharacterName,
		&t.RefreshToken,
		&t.TokenType,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
