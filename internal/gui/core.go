package gui

import (
	"context"
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/api/sso"
	"example/esiapp/internal/helper/set"
	"example/esiapp/internal/model"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const maxMails = 1000

var httpClient = &http.Client{
	Timeout: time.Second * 30, // Timeout after 30 seconds
}

var scopes = []string{
	"esi-characters.read_contacts.v1",
	"esi-universe.read_structures.v1",
	"esi-mail.read_mail.v1",
}

// AddCharacter adds a new character via SSO authentication and returns the new token.
func AddCharacter(ctx context.Context) (*model.Token, error) {
	ssoToken, err := sso.Authenticate(ctx, httpClient, scopes)
	if err != nil {
		return nil, err
	}
	charID := ssoToken.CharacterID
	charEsi, err := esi.FetchCharacter(httpClient, charID)
	if err != nil {
		return nil, err
	}
	ids := []int32{charID, charEsi.CorporationID}
	if charEsi.AllianceID != 0 {
		ids = append(ids, charEsi.AllianceID)
	}
	if charEsi.FactionID != 0 {
		ids = append(ids, charEsi.FactionID)
	}
	err = addMissingEveEntities(ids)
	if err != nil {
		return nil, err
	}
	character := model.Character{
		ID:            charID,
		Name:          charEsi.Name,
		CorporationID: charEsi.CorporationID,
	}
	if err = character.Save(); err != nil {
		return nil, err
	}
	token := model.Token{
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

// TODO: Add ability to update existing mails
// UpdateMails fetches and stores new mails from ESI for a character.
func UpdateMails(characterID int32, status *statusBar) error {
	token, err := model.FetchToken(characterID)
	if err != nil {
		return err
	}
	s := fmt.Sprintf("Checking for new mail for %v", token.Character.Name)
	status.setText(s)
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

func updateMailLabels(token *model.Token) error {
	if err := ensureFreshToken(token); err != nil {
		return err
	}
	ll, err := esi.FetchMailLabels(httpClient, token.CharacterID, token.AccessToken)
	if err != nil {
		return err
	}
	labels := ll.Labels
	slog.Info("Received mail labels from ESI", "count", len(labels), "characterID", token.CharacterID)
	for _, o := range labels {
		l := model.MailLabel{
			CharacterID: token.CharacterID,
			LabelID:     o.LabelID,
			Color:       o.Color,
			Name:        o.Name,
			UnreadCount: o.UnreadCount,
		}
		if err := l.Save(); err != nil {
			slog.Error("Failed to update mail label", "labelID", o.LabelID, "characterID", token.CharacterID, "error", err)
		}
	}
	return nil
}

func updateMailLists(token *model.Token) error {
	if err := ensureFreshToken(token); err != nil {
		return err
	}
	lists, err := esi.FetchMailLists(httpClient, token.CharacterID, token.AccessToken)
	if err != nil {
		return err
	}
	for _, o := range lists {
		e := model.EveEntity{ID: o.ID, Name: o.Name, Category: model.EveEntityMailList}
		if err := e.Save(); err != nil {
			return err
		}
	}
	return nil
}

func fetchMailHeaders(token *model.Token) ([]esi.MailHeader, error) {
	if err := ensureFreshToken(token); err != nil {
		return nil, err
	}
	headers, err := esi.FetchMailHeaders(httpClient, token.CharacterID, token.AccessToken, maxMails)
	if err != nil {
		return nil, err
	}
	return headers, nil
}

func updateMails(token *model.Token, headers []esi.MailHeader, status *statusBar) error {
	existingIDs, missingIDs, err := determineMailIDs(token.CharacterID, headers)
	if err != nil {
		return err
	}
	newMailsCount := missingIDs.Size()
	if newMailsCount == 0 {
		s := "No new mail"
		status.setText(s)
		slog.Info(s, "characterID", token.CharacterID)
		return nil
	}

	if err := ensureFreshToken(token); err != nil {
		return err
	}

	var c atomic.Int32
	var wg sync.WaitGroup
	maxGoroutines := 20
	guard := make(chan struct{}, maxGoroutines)
	for _, header := range headers {
		if existingIDs.Has(header.ID) {
			continue
		}
		guard <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			fetchAndStoreMail(header, token, newMailsCount, &c, status)
			<-guard
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
	slog.Info(s)
	return nil
}

func fetchAndStoreMail(header esi.MailHeader, token *model.Token, newMailsCount int, c *atomic.Int32, status *statusBar) {
	entityIDs := set.New[int32]()
	entityIDs.Add(header.FromID)
	for _, r := range header.Recipients {
		entityIDs.Add(r.ID)
	}
	if err := addMissingEveEntities(entityIDs.ToSlice()); err != nil {
		slog.Error("Failed to process mail", "header", header, "error", err)
		return
	}
	m, err := esi.FetchMail(httpClient, token.CharacterID, header.ID, token.AccessToken)
	if err != nil {
		slog.Error("Failed to process mail", "header", header, "error", err)
		return
	}
	mail := model.Mail{
		Character: token.Character,
		MailID:    header.ID,
		Subject:   header.Subject,
		Body:      m.Body,
	}
	timestamp, err := time.Parse(time.RFC3339, header.Timestamp)
	if err != nil {
		slog.Error("Failed to parse timestamp for mail", "header", header, "error", err)
		return
	}
	mail.Timestamp = timestamp
	from, err := model.FetchEveEntity(header.FromID)
	if err != nil {
		slog.Error("Failed to parse \"from\" in mail", "header", header, "error", err)
		return
	}
	mail.From = *from

	rr := fetchMailRecipients(header)
	mail.Recipients = rr

	labels, err := model.FetchMailLabels(token.CharacterID, m.Labels)
	if err != nil {
		slog.Error("Failed to resolve mail labels", "header", header, "error", err)
		return
	}
	mail.Labels = labels

	mail.Create()
	slog.Info("Created new mail", "mailID", header.ID, "characterID", token.CharacterID)
	c.Add(1)
	current := c.Load()
	s := fmt.Sprintf("Fetched %d / %d new mails for %v", current, newMailsCount, token.Character.Name)
	status.setText(s)
}

func fetchMailRecipients(header esi.MailHeader) []model.EveEntity {
	var rr []model.EveEntity
	for _, r := range header.Recipients {
		o, err := model.FetchEveEntity(r.ID)
		if err != nil {
			slog.Error("Failed to resolve mail recipient", "header", header, "recipient", r, "error", err)
			continue
		} else {
			rr = append(rr, *o)
		}
	}
	return rr
}

func determineMailIDs(characterID int32, headers []esi.MailHeader) (*set.Set[int32], *set.Set[int32], error) {
	ids, err := model.FetchMailIDs(characterID)
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
func ensureFreshToken(token *model.Token) error {
	if !token.RemainsValid(time.Second * 60) {
		slog.Debug("Need to refresh token", "characterID", token.CharacterID)
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
		slog.Info("Token refreshed", "characterID", token.CharacterID)
	}
	return nil
}

func addMissingEveEntities(ids []int32) error {
	c, err := model.FetchEntityIDs()
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
		return fmt.Errorf("failed to resolve IDs: %v %v", err, ids)
	}

	for _, entity := range entities {
		e := model.EveEntity{
			ID:       entity.ID,
			Category: model.EveEntityCategory(entity.Category),
			Name:     entity.Name,
		}
		err := e.Save()
		if err != nil {
			return err
		}
	}

	slog.Debug("Added missing eve entities", "count", len(entities))
	return nil
}
