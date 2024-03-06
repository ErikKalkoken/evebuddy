// Package core contains the main business logic.
// This package will access all other internal packages.
package core

import (
	"example/esiapp/internal/esi"
	"example/esiapp/internal/helpers"
	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"
	"fmt"
	"log"
	"sync"
	"time"
)

// AddCharacter adds a new character via SSO authentication and returns the new token.
func AddCharacter() (*storage.Token, error) {
	scopes := []string{
		"esi-characters.read_contacts.v1",
		"esi-universe.read_structures.v1",
		"esi-mail.read_mail.v1",
	}
	ssoToken, err := sso.Authenticate(scopes)
	if err != nil {
		return nil, err
	}
	character := storage.Character{
		ID:   ssoToken.CharacterID,
		Name: ssoToken.CharacterName,
	}
	if err = character.Save(); err != nil {
		return nil, err
	}
	token := storage.Token{
		AccessToken:  ssoToken.AccessToken,
		Character:    character,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		TokenType:    ssoToken.TokenType,
	}
	if err = token.Save(); err != nil {
		return nil, err
	}
	return &token, nil
}

// UpdateMails fetches and stores new mails from ESI for a character.
func UpdateMails(characterId int32) error {
	token, err := fetchValidToken(characterId)
	if err != nil {
		return err
	}
	character := token.Character
	headers, err := esi.FetchMailHeaders(token.CharacterID, token.AccessToken)
	if err != nil {
		return err
	}

	incomingIDs := helpers.NewSet([]int32{})
	entityIDs := helpers.NewSet([]int32{})
	for _, header := range headers {
		entityIDs.Add(header.FromID)
		incomingIDs.Add(header.ID)
	}

	addMissingEveEntities(entityIDs.ToSlice())

	ids, err := storage.FetchMailIDs(characterId)
	if err != nil {
		return err
	}
	existingIds := helpers.NewSet(ids)
	missingIDs := incomingIDs.Difference(existingIds)

	bodies, err := fetchMailBodies(token, missingIDs.ToSlice())
	if err != nil {
		return err
	}

	createdCount := 0
	for _, header := range headers {
		if existingIds.Has(header.ID) {
			continue
		}
		mail := storage.Mail{
			Character: character,
			MailID:    header.ID,
			Subject:   header.Subject,
		}

		timestamp, err := time.Parse(time.RFC3339, header.Timestamp)
		if err != nil {
			log.Printf("Failed to parse timestamp for mail %d: %v", header.ID, err)
			continue
		}
		mail.TimeStamp = timestamp

		from, err := storage.GetEveEntity(header.FromID)
		if err != nil {
			log.Printf("Failed to parse \"from\" mail %d: %v", header.FromID, err)
			continue
		}
		mail.From = *from

		body, ok := bodies[header.ID]
		if !ok {
			log.Printf("No body found for mail %d", header.ID)
			continue
		}
		mail.Body = body

		mail.Save()
		createdCount++
	}
	log.Printf("Stored %d new mails", createdCount)

	return nil
}

func fetchValidToken(characterId int32) (*storage.Token, error) {
	token, err := storage.FetchToken(characterId)
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

// TODO: Add error handling

// fetchMailBodies fetches multiple mails from ESI concurrently and returns them.
func fetchMailBodies(token *storage.Token, ids []int32) (map[int32]string, error) {
	var wg sync.WaitGroup
	var contexts = sync.Map{}

	for _, i := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mail, err := esi.FetchMail(token.CharacterID, i, token.AccessToken)
			if err != nil {
				log.Printf("Error when fetching mail bodies: %v", err)
			}
			contexts.Store(i, mail.Body)
		}()
	}
	wg.Wait()

	res := make(map[int32]string, len(ids))
	for _, i := range ids {
		v, ok := contexts.Load(i)
		if ok {
			res[i] = v.(string)
		}
	}
	return res, nil
}

func addMissingEveEntities(ids []int32) error {
	c, err := storage.FetchEntityIDs()
	if err != nil {
		return err
	}
	current := helpers.NewSet(c)
	incoming := helpers.NewSet(ids)
	missing := incoming.Difference(current)

	if missing.Size() == 0 {
		log.Println("No missing eve entities")
		return nil
	}

	entities, err := esi.ResolveEntityIDs(missing.ToSlice())
	if err != nil {
		return fmt.Errorf("failed to resolve IDs: %v", err)
	}

	for _, entity := range entities {
		e := storage.EveEntity{ID: entity.ID, Category: entity.Category, Name: entity.Name}
		err := e.Save()
		if err != nil {
			return err
		}
	}

	log.Printf("Added %d missing eve entities", len(entities))
	return nil
}
