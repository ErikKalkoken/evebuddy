package service

import (
	"context"
	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/repository"

	"fmt"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/antihax/goesi/esi"
	"github.com/antihax/goesi/optional"
)

const (
	maxMails          = 1000
	maxHeadersPerPage = 50 // maximum header objects returned per page
)

// DeleteMail deletes a mail both on ESI and in the database.
func (s *Service) DeleteMail(m *repository.Mail) error {
	ctx := context.Background()
	token, err := s.GetValidToken(ctx, m.CharacterID)
	if err != nil {
		return err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	_, err = s.esiClient.ESI.MailApi.DeleteCharactersCharacterIdMailMailId(ctx, m.CharacterID, m.MailID, nil)
	if err != nil {
		return err
	}
	err = s.r.DeleteMail(ctx, m.ID)
	if err != nil {
		return err
	}
	return nil
}

// SendMail created a new mail on ESI stores it locally.
func (s *Service) SendMail(characterID int32, subject string, recipients *Recipients, body string) error {
	if subject == "" {
		return fmt.Errorf("missing subject")
	}
	if body == "" {
		return fmt.Errorf("missing body")
	}
	rr, err := recipients.ToMailRecipients(s)
	if err != nil {
		return err
	}
	if len(rr) == 0 {
		return fmt.Errorf("missing recipients")
	}
	ctx := context.Background()
	token, err := s.GetValidToken(ctx, characterID)
	if err != nil {
		return err
	}
	esiMail := esi.PostCharactersCharacterIdMailMail{
		Body:       body,
		Subject:    subject,
		Recipients: rr,
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	mailID, _, err := s.esiClient.ESI.MailApi.PostCharactersCharacterIdMail(ctx, characterID, esiMail, nil)
	if err != nil {
		return err
	}
	recipientIDs := make([]int32, len(rr))
	for i, r := range rr {
		recipientIDs[i] = r.RecipientId
	}
	ids := slices.Concat(recipientIDs, []int32{characterID})
	_, err = s.addMissingEveEntities(ctx, ids)
	if err != nil {
		return err
	}
	arg := repository.CreateMailParams{
		Body:         body,
		CharacterID:  characterID,
		FromID:       characterID,
		IsRead:       true,
		LabelIDs:     []int32{repository.LabelSent},
		MailID:       mailID,
		RecipientIDs: recipientIDs,
		Subject:      subject,
		Timestamp:    time.Now(),
	}
	_, err = s.r.CreateMail(ctx, arg)
	if err != nil {
		return err
	}
	return nil
}

// UpdateMailRead updates an existing mail as read
func (s *Service) UpdateMailRead(m *repository.Mail) error {
	ctx := context.Background()
	token, err := s.GetValidToken(ctx, m.CharacterID)
	if err != nil {
		return err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	labelIDs := make([]int32, len(m.Labels))
	for i, l := range m.Labels {
		labelIDs[i] = l.LabelID
	}
	contents := esi.PutCharactersCharacterIdMailMailIdContents{Read: true, Labels: labelIDs}
	_, err = s.esiClient.ESI.MailApi.PutCharactersCharacterIdMailMailId(ctx, m.CharacterID, contents, m.MailID, nil)
	if err != nil {
		return err
	}
	m.IsRead = true
	if err := s.r.UpdateMailSetRead(ctx, m.ID); err != nil {
		return err
	}
	return nil

}

func (s *Service) GetMailFromDB(characterID int32, mailID int32) (repository.Mail, error) {
	ctx := context.Background()
	return s.r.GetMail(ctx, characterID, mailID)
}

// FIXME: Delete obsolete labels and mail lists
// TODO: Add ability to update existing mails for is_read and labels

// FetchMail fetches and stores new mails from ESI for a character.
func (s *Service) FetchMail(characterID int32, status binding.String) error {
	ctx := context.Background()
	token, err := s.GetValidToken(ctx, characterID)
	if err != nil {
		return err
	}
	character, err := s.r.GetCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	status.Set(fmt.Sprintf("Checking for new mail for %v", character.Name))
	if err := s.updateMailLists(ctx, &token); err != nil {
		return err
	}
	if err := s.updateMailLabels(ctx, &token); err != nil {
		return err
	}
	headers, err := s.listMailHeaders(ctx, &token)
	if err != nil {
		return err
	}
	err = s.updateMails(ctx, &token, headers, status)
	if err != nil {
		return err
	}
	character.MailUpdatedAt = time.Now()
	if err := s.r.UpdateOrCreateCharacter(ctx, &character); err != nil {
		return err
	}
	return nil
}

func (s *Service) updateMailLabels(ctx context.Context, token *repository.Token) error {
	if err := s.EnsureValidToken(ctx, token); err != nil {
		return err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	ll, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailLabels(ctx, token.CharacterID, nil)
	if err != nil {
		return err
	}
	labels := ll.Labels
	slog.Info("Received mail labels from ESI", "count", len(labels), "characterID", token.CharacterID)
	for _, o := range labels {
		err := s.r.UpdateOrCreateMailLabel(ctx, token.CharacterID, o.LabelId, o.Name, o.Color, int(o.UnreadCount))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) updateMailLists(ctx context.Context, token *repository.Token) error {
	if err := s.EnsureValidToken(ctx, token); err != nil {
		return err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	lists, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailLists(ctx, token.CharacterID, nil)
	if err != nil {
		return err
	}
	for _, o := range lists {
		_, err := s.r.UpdateOrCreateEveEntity(ctx, o.MailingListId, o.Name, repository.EveEntityMailList)
		if err != nil {
			return err
		}
		if err := s.r.CreateMailList(ctx, token.CharacterID, o.MailingListId); err != nil {
			return err
		}
	}
	return nil
}

// listMailHeaders fetched mail headers from ESI with paging and returns them.
func (s *Service) listMailHeaders(ctx context.Context, token *repository.Token) ([]esi.GetCharactersCharacterIdMail200Ok, error) {
	if err := s.EnsureValidToken(ctx, token); err != nil {
		return nil, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	var mm []esi.GetCharactersCharacterIdMail200Ok
	lastMailID := int32(0)
	for {
		var opts *esi.GetCharactersCharacterIdMailOpts
		if lastMailID > 0 {
			l := optional.NewInt32(lastMailID)
			opts = &esi.GetCharactersCharacterIdMailOpts{LastMailId: l}
		} else {
			opts = nil
		}
		objs, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMail(ctx, token.CharacterID, opts)
		if err != nil {
			return nil, err
		}
		mm = append(mm, objs...)
		isLimitExceeded := (maxMails != 0 && len(mm)+maxHeadersPerPage > maxMails)
		if len(objs) < maxHeadersPerPage || isLimitExceeded {
			break
		}
		ids := make([]int32, 0)
		for _, o := range objs {
			ids = append(ids, o.MailId)
		}
		lastMailID = slices.Min(ids)
	}
	slog.Info("Received mail headers", "characterID", token.CharacterID, "count", len(mm))
	return mm, nil
}

func (s *Service) updateMails(ctx context.Context, token *repository.Token, headers []esi.GetCharactersCharacterIdMail200Ok, status binding.String) error {
	existingIDs, missingIDs, err := s.determineMailIDs(ctx, token.CharacterID, headers)
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

	if err := s.EnsureValidToken(ctx, token); err != nil {
		return err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	character, err := s.r.GetCharacter(ctx, token.CharacterID)
	if err != nil {
		return err
	}

	var c atomic.Int32
	var wg sync.WaitGroup
	maxGoroutines := 20
	guard := make(chan struct{}, maxGoroutines)
	for _, header := range headers {
		if existingIDs.Has(header.MailId) {
			continue
		}
		guard <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.fetchAndStoreMail(ctx, header, token, newMailsCount, &c, status, character.Name)
			<-guard
		}()
	}
	wg.Wait()
	total := c.Load()
	if total == 0 {
		status.Set("")
		return nil
	}
	t := fmt.Sprintf("Stored %d new mails", total)
	status.Set(t)
	slog.Info(t)
	return nil
}

func (s *Service) fetchAndStoreMail(ctx context.Context, header esi.GetCharactersCharacterIdMail200Ok, token *repository.Token, newMailsCount int, c *atomic.Int32, status binding.String, characterName string) {
	err := func() error {
		entityIDs := set.New[int32]()
		entityIDs.Add(header.From)
		for _, r := range header.Recipients {
			entityIDs.Add(r.RecipientId)
		}
		_, err := s.addMissingEveEntities(ctx, entityIDs.ToSlice())
		if err != nil {
			return err
		}
		m, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailMailId(ctx, token.CharacterID, header.MailId, nil)
		if err != nil {
			return err
		}
		recipientIDs := make([]int32, len(m.Recipients))
		for i, r := range m.Recipients {
			recipientIDs[i] = r.RecipientId
		}
		arg := repository.CreateMailParams{
			Body:         m.Body,
			CharacterID:  token.CharacterID,
			FromID:       header.From,
			IsRead:       false,
			LabelIDs:     header.Labels,
			MailID:       header.MailId,
			RecipientIDs: recipientIDs,
			Subject:      m.Subject,
			Timestamp:    m.Timestamp,
		}
		mailID, err := s.r.CreateMail(ctx, arg)
		if err != nil {
			return err
		}
		if err := s.r.AddMailLabelsToMail(ctx, token.CharacterID, mailID, m.Labels); err != nil {
			return err
		}
		c.Add(1)
		current := c.Load()
		if err := status.Set(fmt.Sprintf("Fetched %d / %d new mails for %v", current, newMailsCount, characterName)); err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		slog.Error("Failed to process mail", "header", header, "error", err)
	}
}

func (s *Service) determineMailIDs(ctx context.Context, characterID int32, headers []esi.GetCharactersCharacterIdMail200Ok) (*set.Set[int32], *set.Set[int32], error) {
	ids, err := s.r.ListMailIDs(ctx, characterID)
	if err != nil {
		return nil, nil, err
	}
	existingIDs := set.NewFromSlice(ids)
	incomingIDs := set.New[int32]()
	for _, h := range headers {
		incomingIDs.Add(h.MailId)
	}
	missingIDs := incomingIDs.Difference(existingIDs)
	return existingIDs, missingIDs, nil
}

func (s *Service) GetMailLabelUnreadCounts(characterID int32) (map[int32]int, error) {
	ctx := context.Background()
	return s.r.GetMailLabelUnreadCounts(ctx, characterID)
}

func (s *Service) GetMailListUnreadCounts(characterID int32) (map[int32]int, error) {
	ctx := context.Background()
	return s.r.GetMailListUnreadCounts(ctx, characterID)
}

func (s *Service) ListMailLists(characterID int32) ([]repository.EveEntity, error) {
	ctx := context.Background()
	return s.r.ListMailLists(ctx, characterID)
}

// ListMailsForLabel returns a character's mails for a label in descending order by timestamp.
// Return mails for all labels, when labelID = 0
func (s *Service) ListMailIDsForLabelOrdered(characterID int32, labelID int32) ([]int32, error) {
	ctx := context.Background()
	return s.r.ListMailIDsForLabelOrdered(ctx, characterID, labelID)
}

func (s *Service) ListMailIDsForListOrdered(characterID int32, listID int32) ([]int32, error) {
	ctx := context.Background()
	return s.r.ListMailIDsForListOrdered(ctx, characterID, listID)
}

func (s *Service) ListMailLabels(characterID int32) ([]repository.MailLabel, error) {
	ctx := context.Background()
	return s.r.ListMailLabels(ctx, characterID)
}
