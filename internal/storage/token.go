package storage

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

type Token struct {
	AccessToken   string
	CharacterID   int32
	CharacterName string
	ExpiresAt     time.Time
	RefreshToken  string
	TokenType     string
}

func (t Token) String() string {
	return t.CharacterName
}

type MyURL struct {
	url.URL
}

func (u MyURL) Authority() string {
	return ""
}

func (u MyURL) Extension() string {
	return ""
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
	_, err := db.Exec(
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
	err := db.QueryRow(s, characterId).Scan(
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

func FindAllToken() ([]Token, error) {
	s := `
		SELECT access_token, character_id, character_name, refresh_token, token_type
		FROM tokens;
	`
	rows, err := db.Query(s)
	if err != nil {
		return nil, err
	}
	var tokens []Token
	for rows.Next() {
		var t Token
		err = rows.Scan(
			&t.AccessToken,
			&t.CharacterID,
			&t.CharacterName,
			&t.RefreshToken,
			&t.TokenType,
		)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	rows.Close() //good habit to close

	return tokens, nil
}
