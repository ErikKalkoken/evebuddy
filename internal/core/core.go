// Package core contains the main logic and accesses all the other internal packages
package core

import (
	"example/esiapp/internal/esi"
	"example/esiapp/internal/helpers"
	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"
	"log"
	"time"
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

	ids, err := storage.FetchMailIDs(characterId)
	if err != nil {
		return err
	}
	existingIds := helpers.NewSetFromSlice(ids)

	createdCount := 0
	newEntityIds := helpers.NewSet[int32]()
	for _, header := range headers {
		if existingIds.Has(header.ID) {
			continue
		}
		from, created, err := storage.GetOrCreateEveEntity(header.FromID)
		if created {
			newEntityIds.Add(from.ID)
		}
		if err != nil {
			log.Printf("Failed to parse \"from\" mail %d: %v", header.FromID, err)
			continue
		}
		mail := storage.MailHeader{
			Character: character,
			From:      *from,
			MailID:    header.ID,
			Subject:   header.Subject,
		}
		timestamp, err := time.Parse(time.RFC3339, header.Timestamp)
		if err == nil {
			mail.TimeStamp = timestamp
		} else {
			log.Printf("Failed to parse timestamp for mail %d: %v", header.ID, err)
		}
		mail.Save()
		createdCount++
	}
	log.Printf("Stored %d new mails", createdCount)

	// if len(newEntityIds) > 0 {
	// 	entities, err := esi.ResolveEntityIDs(newEntityIds)
	// 	if err != nil {
	// 		log.Printf("Failed to resolve IDs: %v", err)
	// 	} else {

	// 	}
	// }

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
