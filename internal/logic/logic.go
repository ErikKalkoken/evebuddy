// Package logic contains the app's business logic
package logic

import (
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/model"
	"log/slog"
	"net/http"
	"slices"
	"time"
)

var httpClient = &http.Client{
	Timeout: time.Second * 30, // Timeout after 30 seconds
}

func SendMail(characterID int32, subject string, recipients []esi.MailRecipient, body string) error {
	token, err := FetchValidToken(characterID)
	if err != nil {
		return err
	}
	m := esi.MailSend{
		Body:       body,
		Subject:    subject,
		Recipients: recipients,
	}
	_, err = esi.SendMail(httpClient, characterID, token.AccessToken, m)
	if err != nil {
		return err
	}
	return nil
}

func AddEveEntitiesFromESISearch(characterID int32, search string) ([]int32, error) {
	token, err := FetchValidToken(characterID)
	if err != nil {
		return nil, err
	}
	categories := []esi.SearchCategory{
		esi.SearchCategoryCorporation,
		esi.SearchCategoryCharacter,
		esi.SearchCategoryAlliance,
	}
	r, err := esi.Search(httpClient, characterID, search, categories, token.AccessToken)
	if err != nil {
		return nil, err
	}
	ids := slices.Concat(r.Alliance, r.Character, r.Corporation)
	missingIDs, err := AddMissingEveEntities(ids)
	if err != nil {
		slog.Error("Failed to fetch missing IDs", "error", err)
		return nil, err
	}
	return missingIDs, nil
}

func DeleteMail(m *model.Mail) error {
	token, err := FetchValidToken(m.CharacterID)
	if err != nil {
		return err
	}
	if err := esi.DeleteMail(httpClient, m.CharacterID, m.MailID, token.AccessToken); err != nil {
		return err
	}
	_, err = m.Delete()
	if err != nil {
		return err
	}
	return nil
}
