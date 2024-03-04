package core

import (
	"example/esiapp/internal/esi"
	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"
	"log"

	"gorm.io/gorm"
)

func FetchMail(db *gorm.DB, characterId int32) error {
	token, err := fetchValidToken(db, characterId)
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
		db.Where("character_id = ? AND mail_id = ?", characterId, header.ID).Save(&mail)
	}
	return nil
}

func fetchValidToken(db *gorm.DB, characterId int32) (*storage.Token, error) {
	var token storage.Token
	err := db.Preload("Character").First(&token, "character_id = ?", characterId).Error
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
		err = db.Where("character_id = ?", characterId).Save(&token).Error
		if err != nil {
			return nil, err
		}
		log.Printf("Refreshed token for %v", characterId)
	}
	return &token, nil
}
