package gui

import (
	"context"
	"example/esiapp/internal/esi"
	"example/esiapp/internal/set"
	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const maxMails = 1000

var httpClient http.Client

var scopes = []string{
	"esi-characters.read_contacts.v1",
	"esi-universe.read_structures.v1",
	"esi-mail.read_mail.v1",
}

// AddCharacter adds a new character via SSO authentication and returns the new token.
func AddCharacter(ctx context.Context) (*storage.Token, error) {
	ssoToken, err := sso.Authenticate(ctx, httpClient, scopes)
	if err != nil {
		return nil, err
	}
	if ssoToken == nil {
		return nil, fmt.Errorf("auth process canceled prematurely")
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
func UpdateMails(characterID int32, status *statusBar) error {
	token, err := storage.FetchToken(characterID)
	if err != nil {
		return err
	}
	status.setText("Checking for new mail for %v", token.Character.Name)
	if err := updateMailLists(token); err != nil {
		return err
	}
	if err := updateMailLabels(token); err != nil {
		return err
	}
	headers, err := fetchMailHeaders(token)
	if err != nil {
		return err
	}
	err = updateMails(token, headers, status)
	if err != nil {
		return err
	}
	return nil
}

func updateMailLabels(token *storage.Token) error {
	if err := ensureFreshToken(token); err != nil {
		return err
	}
	ll, err := esi.FetchMailLabels(httpClient, token.CharacterID, token.AccessToken)
	if err != nil {
		return err
	}
	labels := ll.Labels
	log.Printf("Received %d mail labels from ESI for character %d", len(labels), token.CharacterID)
	for _, o := range labels {
		_, err := storage.UpdateOrCreateMailLabel(token.CharacterID, o.LabelID, o.Color, o.Name, o.UnreadCount)
		if err != nil {
			log.Printf("Error when trying to update mail label: %v", err)
		}
	}
	return nil
}

func updateMailLists(token *storage.Token) error {
	if err := ensureFreshToken(token); err != nil {
		return err
	}
	lists, err := esi.FetchMailLists(httpClient, token.CharacterID, token.AccessToken)
	if err != nil {
		return err
	}
	for _, o := range lists {
		e := storage.EveEntity{ID: o.ID, Name: o.Name, Category: "mail_list"}
		if err := e.Save(); err != nil {
			return err
		}
	}
	return nil
}

func fetchMailHeaders(token *storage.Token) ([]esi.MailHeader, error) {
	if err := ensureFreshToken(token); err != nil {
		return nil, err
	}
	headers, err := esi.FetchMailHeaders(httpClient, token.CharacterID, token.AccessToken, maxMails)
	if err != nil {
		return nil, err
	}
	log.Printf("Received %d mail headers from ESI for character %d", len(headers), token.CharacterID)
	return headers, nil
}

func updateMails(token *storage.Token, headers []esi.MailHeader, status *statusBar) error {
	existingIDs, missingIDs, err := determineMailIDs(token.CharacterID, headers)
	if err != nil {
		return err
	}
	newMailsCount := missingIDs.Size()
	if newMailsCount == 0 {
		s := "No new mail"
		status.setText(s)
		log.Print(s)
		return nil
	}

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
			entityIDs := set.New[int32]()
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
				Character: token.Character,
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

			labels, err := storage.FetchMailLabels(token.CharacterID, m.Labels)
			if err != nil {
				log.Printf("Failed to resolve labels for mail %d: %v", header.ID, err)
			} else {
				mail.Labels = labels
			}

			mail.Save()
			log.Printf("Stored new mail %d for character %v", header.ID, token.CharacterID)
			c.Add(1)
			current := c.Load()
			status.setText("Fetched %d / %d new mails for %v", current, newMailsCount, token.Character.Name)
		}()
	}
	wg.Wait()
	total := c.Load()
	if total == 0 {
		status.clear()
		return nil
	}
	s := fmt.Sprintf("Stored %d new mails", total)
	status.setText(s)
	log.Print(s)
	return nil
}

func determineMailIDs(characterID int32, headers []esi.MailHeader) (*set.Set[int32], *set.Set[int32], error) {
	ids, err := storage.FetchMailIDs(characterID)
	if err != nil {
		return nil, nil, err
	}
	existingIDs := set.NewFromSlice(ids)
	incomingIDs := set.New[int32]()
	for _, h := range headers {
		incomingIDs.Add(h.ID)
	}
	missingIDs := incomingIDs.Difference(existingIDs)
	return existingIDs, missingIDs, nil
}

// ensureFreshToken will automatically try to refresh a token that is already or about to become invalid.
func ensureFreshToken(token *storage.Token) error {
	if !token.RemainsValid(time.Second * 60) {
		log.Printf("Need to refresh token: %v", token)
		rawToken, err := sso.RefreshToken(httpClient, token.RefreshToken)
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

func addMissingEveEntities(ids []int32) error {
	c, err := storage.FetchEntityIDs()
	if err != nil {
		return err
	}
	current := set.NewFromSlice(c)
	incoming := set.NewFromSlice(ids)
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
