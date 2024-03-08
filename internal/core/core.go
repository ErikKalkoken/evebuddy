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
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var httpClient http.Client

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
func UpdateMails(characterID int32) error {
	token, err := storage.FetchToken(characterID)
	if err != nil {
		return err
	}
	character := token.Character

	if err := ensureFreshToken(token); err != nil {
		return err
	}
	lists, err := esi.FetchMailLists(httpClient, token.CharacterID, token.AccessToken)
	if err != nil {
		return err
	}
	if err := updateMailLists(lists); err != nil {
		return err
	}

	if err := ensureFreshToken(token); err != nil {
		return err
	}
	l, err := esi.FetchMailLabels(httpClient, token.CharacterID, token.AccessToken)
	if err != nil {
		return err
	}
	labels := l.Labels
	log.Printf("Received %d mail labels from ESI for character %d", len(labels), token.CharacterID)
	if err := updateMailLabels(characterID, labels); err != nil {
		return err
	}

	if err := ensureFreshToken(token); err != nil {
		return err
	}
	headers, err := esi.FetchMailHeaders(httpClient, token.CharacterID, token.AccessToken)
	if err != nil {
		return err
	}
	log.Printf("Received %d mail headers from ESI for character %d", len(headers), token.CharacterID)

	ids, err := storage.FetchMailIDs(characterID)
	if err != nil {
		return err
	}
	existingIDs := helpers.NewSet(ids)

	if err := ensureFreshToken(token); err != nil {
		return err
	}

	var c atomic.Int32
	var wg sync.WaitGroup
	for _, header := range headers {
		if existingIDs.Has(header.ID) {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			entityIDs := helpers.NewSet([]int32{})
			entityIDs.Add(header.FromID)
			for _, r := range header.Recipients {
				entityIDs.Add(r.ID)
			}
			if err := addMissingEveEntities(entityIDs.ToSlice()); err != nil {
				log.Printf("Failed to process mail %d: %v", header.ID, err)
				return
			}

			m, err := esi.FetchMail(httpClient, token.CharacterID, header.ID, token.AccessToken)
			if err != nil {
				log.Printf("Failed to process mail %d: %v", header.ID, err)
				return
			}

			mail := storage.Mail{
				Character: character,
				MailID:    header.ID,
				Subject:   header.Subject,
				Body:      m.Body,
			}

			timestamp, err := time.Parse(time.RFC3339, header.Timestamp)
			if err != nil {
				log.Printf("Failed to parse timestamp for mail %d: %v", header.ID, err)
				return
			}
			mail.TimeStamp = timestamp

			from, err := storage.GetEveEntity(header.FromID)
			if err != nil {
				log.Printf("Failed to parse \"from\" mail %d: %v", header.FromID, err)
				return
			}
			mail.From = *from

			var rr []storage.EveEntity
			for _, r := range header.Recipients {
				o, err := storage.GetEveEntity(r.ID)
				if err != nil {
					log.Printf("Failed to resolve recipient %v for mail %d", r, header.ID)
					continue
				} else {
					rr = append(rr, *o)
				}
			}
			mail.Recipients = rr

			labels, err := storage.FetchMailLabels(characterID, m.Labels)
			if err != nil {
				log.Printf("Failed to resolve labels for mail %d: %v", header.ID, err)
			} else {
				mail.Labels = labels
			}

			mail.Save()
			log.Printf("Stored new mail %d for character %v", header.ID, token.CharacterID)
			c.Add(1)
		}()
	}
	wg.Wait()
	if total := c.Load(); total == 0 {
		log.Printf("No new mail")
	} else {
		log.Printf("Stored %d new mails", total)
	}
	return nil
}

func updateMailLabels(characterID int32, l []esi.MailLabel) error {
	for _, o := range l {
		e := storage.MailLabel{
			CharacterID: characterID,
			ID:          o.ID,
			Name:        o.Name,
			Color:       o.Color,
			UnreadCount: o.UnreadCount,
		}
		if err := e.Save(); err != nil {
			return err
		}
	}
	return nil
}

// updateMailLists stores mail lists
func updateMailLists(l []esi.MailList) error {
	for _, o := range l {
		e := storage.EveEntity{ID: o.ID, Name: o.Name, Category: "mail_list"}
		if err := e.Save(); err != nil {
			return err
		}
	}
	return nil
}

// ensureFreshToken will automatically try to refresh a token that is already or about to become invalid.
func ensureFreshToken(token *storage.Token) error {
	if !token.RemainsValid(time.Second * 60) {
		log.Printf("Need to refresh token: %v", token)
		rawToken, err := sso.RefreshToken(token.RefreshToken)
		if err != nil {
			return err
		}
		token.AccessToken = rawToken.AccessToken
		token.RefreshToken = rawToken.RefreshToken
		token.ExpiresAt = rawToken.ExpiresAt
		err = token.Save()
		if err != nil {
			return err
		}
		log.Printf("Refreshed token for %v", token.CharacterID)
	}
	return nil
}

// // TODO: Add error handling

// // fetchMailBodies fetches multiple mails from ESI concurrently and returns them.
// func fetchMailBodies(token *storage.Token, ids []int32) (map[int32]string, error) {
// 	var wg sync.WaitGroup
// 	var contexts = sync.Map{}

// 	for _, i := range ids {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			mail, err := esi.FetchMail(httpClient, token.CharacterID, i, token.AccessToken)
// 			if err != nil {
// 				log.Printf("Failed to fetch mail body ID %d for character ID %d: %v", i, token.CharacterID, err)
// 			} else {
// 				contexts.Store(i, mail.Body)
// 			}
// 		}()
// 	}
// 	wg.Wait()

// 	res := make(map[int32]string, len(ids))
// 	for _, i := range ids {
// 		v, ok := contexts.Load(i)
// 		if ok {
// 			res[i] = v.(string)
// 		}
// 	}
// 	return res, nil
// }

func addMissingEveEntities(ids []int32) error {
	c, err := storage.FetchEntityIDs()
	if err != nil {
		return err
	}
	current := helpers.NewSet(c)
	incoming := helpers.NewSet(ids)
	missing := incoming.Difference(current)

	if missing.Size() == 0 {
		return nil
	}

	entities, err := esi.ResolveEntityIDs(httpClient, missing.ToSlice())
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
