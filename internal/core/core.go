package core

import (
	"example/esiapp/internal/esi"
	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"
	"log"
)

func FetchMail(characterId int32) error {
	token, err := fetchValidToken(characterId)
	if err != nil {
		return err
	}
	character := token.Character
	headers, err := esi.FetchMailHeaders(token.CharacterID, token.AccessToken)
	if err != nil {
		return err
	}

	for _, header := range headers {
		mail := storage.MailHeader{
			Character: character,
			MailID:    header.ID,
			Subject:   header.Subject,
		}
		mail.Save()
	}
	return nil
}

func fetchValidToken(characterId int32) (*storage.Token, error) {
	token, err := storage.FindToken(characterId)
	if err != nil {
		return nil, err
	}
	log.Printf("Current token: %v", token)
	if !token.IsValid() {
		rawToken, err := sso.RefreshToken(token.RefreshToken)
		if err != nil {
			return nil, err
		}
		token.AccessToken = rawToken.AccessToken
		token.RefreshToken = rawToken.RefreshToken
		token.ExpiresAt = rawToken.ExpiresAt
		err = token.Save()
		if err != nil {
			return nil, err
		}
		log.Printf("Refreshed token for %v", characterId)
	}
	return token, nil
}
