package core

import (
	"example/esiapp/internal/esi"
	"example/esiapp/internal/helpers"
	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2/data/binding"
)

const maxMails = 1000

// UpdateMails fetches and stores new mails from ESI for a character.
func UpdateMails(characterID int32, statusLabelText binding.String) error {
	status := newStatusLabel(statusLabelText)
	token, err := storage.FetchToken(characterID)
	if err != nil {
		return err
	}
	character := token.Character

	status.SetText("Checking for new mail for %v", character.Name)
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
	headers, err := esi.FetchMailHeaders(httpClient, token.CharacterID, token.AccessToken, maxMails)
	if err != nil {
		return err
	}
	log.Printf("Received %d mail headers from ESI for character %d", len(headers), token.CharacterID)

	ids, err := storage.FetchMailIDs(characterID)
	if err != nil {
		return err
	}
	existingIDs := helpers.NewSet(ids)
	incomingIDs := helpers.NewSet([]int32{})
	for _, h := range headers {
		incomingIDs.Add(h.ID)
	}
	missingIDs := incomingIDs.Difference(existingIDs)
	newMailsCount := missingIDs.Size()
	if newMailsCount == 0 {
		s := "No new mail"
		status.SetText(s)
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
			current := c.Load()
			status.SetText("Fetched %d / %d new mails for %v", current, newMailsCount, token.Character.Name)
		}()
	}
	wg.Wait()
	total := c.Load()
	if total == 0 {
		status.Clear()
		return nil
	}
	s := fmt.Sprintf("Stored %d new mails", total)
	status.SetText(s)
	log.Print(s)
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
