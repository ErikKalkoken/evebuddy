package logic

import (
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/helper/set"
	"example/esiapp/internal/model"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2/data/binding"
)

const maxMails = 1000

// DeleteMail deleted a mail both on ESI and in the database.
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

// SendMail created a new mail on ESI stores it locally.
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
	mailID, err := esi.SendMail(httpClient, characterID, token.AccessToken, m)
	if err != nil {
		return err
	}
	ids := []int32{characterID}
	for _, r := range recipients {
		ids = append(ids, r.ID)
	}
	_, err = AddMissingEveEntities(ids)
	if err != nil {
		return err
	}
	from, err := model.FetchEveEntityByID(token.CharacterID)
	if err != nil {
		return err
	}
	// FIXME: Ensure this still works when no labels have yet been loaded from ESI
	label, err := model.FetchMailLabel(token.CharacterID, model.LabelSent)
	if err != nil {
		return err
	}
	var rr []model.EveEntity
	for _, r := range recipients {
		e, err := model.FetchEveEntityByID(r.ID)
		if err != nil {
			return err
		}
		rr = append(rr, *e)
	}
	mail := model.Mail{
		Body:       body,
		Character:  token.Character,
		From:       *from,
		Labels:     []model.MailLabel{*label},
		MailID:     mailID,
		Recipients: rr,
		Subject:    subject,
		IsRead:     true,
		Timestamp:  time.Now(),
	}
	if err := mail.Create(); err != nil {
		return err
	}
	return nil
}

// FIXME: Delete obsolete labels and mail lists
// TODO: Add ability to update existing mails for is_read and labels

// FetchMail fetches and stores new mails from ESI for a character.
func FetchMail(characterID int32, status binding.String, headerData binding.IntList) error {
	token, err := model.FetchToken(characterID)
	if err != nil {
		return err
	}
	s := fmt.Sprintf("Checking for new mail for %v", token.Character.Name)
	status.Set(s)
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
	err = updateMails(token, headers, status, headerData)
	if err != nil {
		return err
	}
	return nil
}

func updateMailLabels(token *model.Token) error {
	if err := EnsureValidToken(token); err != nil {
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
	if err := EnsureValidToken(token); err != nil {
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
		m := model.MailList{Character: token.Character, EveEntity: e}
		if err := m.CreateIfNew(); err != nil {
			return err
		}
	}
	return nil
}

func fetchMailHeaders(token *model.Token) ([]esi.MailHeader, error) {
	if err := EnsureValidToken(token); err != nil {
		return nil, err
	}
	headers, err := esi.FetchMailHeaders(httpClient, token.CharacterID, token.AccessToken, maxMails)
	if err != nil {
		return nil, err
	}
	return headers, nil
}

func updateMails(token *model.Token, headers []esi.MailHeader, status binding.String, headerData binding.IntList) error {
	existingIDs, missingIDs, err := determineMailIDs(token.CharacterID, headers)
	if err != nil {
		return err
	}
	newMailsCount := missingIDs.Size()
	if newMailsCount == 0 {
		s := "No new mail"
		status.Set(s)
		slog.Info(s, "characterID", token.CharacterID)
		return nil
	}

	if err := EnsureValidToken(token); err != nil {
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
			fetchAndStoreMail(header, token, newMailsCount, &c, status, headerData)
			<-guard
		}()
	}
	wg.Wait()
	total := c.Load()
	if total == 0 {
		status.Set("")
		return nil
	}
	s := fmt.Sprintf("Stored %d new mails", total)
	status.Set(s)
	slog.Info(s)
	return nil
}

func fetchAndStoreMail(header esi.MailHeader, token *model.Token, newMailsCount int, c *atomic.Int32, status binding.String, headerData binding.IntList) {
	entityIDs := set.New[int32]()
	entityIDs.Add(header.FromID)
	for _, r := range header.Recipients {
		entityIDs.Add(r.ID)
	}
	_, err := AddMissingEveEntities(entityIDs.ToSlice())
	if err != nil {
		slog.Error("Failed to process mail", "header", header, "error", err)
		return
	}
	m, err := esi.FetchMail(httpClient, token.CharacterID, header.ID, token.AccessToken)
	if err != nil {
		slog.Error("Failed to process mail", "header", header, "error", err)
		return
	}
	mail := model.Mail{
		Body:      m.Body,
		Character: token.Character,
		MailID:    header.ID,
		Subject:   header.Subject,
		IsRead:    header.IsRead,
	}
	timestamp, err := time.Parse(time.RFC3339, header.Timestamp)
	if err != nil {
		slog.Error("Failed to parse timestamp for mail", "header", header, "error", err)
		return
	}
	mail.Timestamp = timestamp
	from, err := model.FetchEveEntityByID(header.FromID)
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
	status.Set(s)
}

func fetchMailRecipients(header esi.MailHeader) []model.EveEntity {
	var rr []model.EveEntity
	for _, r := range header.Recipients {
		o, err := model.FetchEveEntityByID(r.ID)
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

// UpdateMailRead updates an existing mail as read
func UpdateMailRead(m *model.Mail) error {
	token, err := FetchValidToken(m.CharacterID)
	if err != nil {
		return err
	}
	labelIDs := make([]int32, len(m.Labels))
	for i, l := range m.Labels {
		labelIDs[i] = l.LabelID
	}
	data := esi.MailUpdate{Read: true, Labels: labelIDs}
	if err := esi.UpdateMail(httpClient, m.CharacterID, m.MailID, data, token.AccessToken); err != nil {
		return err
	}
	m.IsRead = true
	if err := m.Save(); err != nil {
		return err
	}
	return nil

}
