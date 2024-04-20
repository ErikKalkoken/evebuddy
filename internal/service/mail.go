package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/antihax/goesi/esi"
	"github.com/antihax/goesi/optional"

	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
)

const (
	maxMails          = 1000
	maxHeadersPerPage = 50 // maximum header objects returned per page
)

var eveEntityCategory2MailRecipientType = map[model.EveEntityCategory]string{
	model.EveEntityAlliance:    "alliance",
	model.EveEntityCharacter:   "character",
	model.EveEntityCorporation: "corporation",
	model.EveEntityMailList:    "mailing_list",
}

// DeleteMail deletes a mail both on ESI and in the database.
func (s *Service) DeleteMail(characterID, mailID int32) error {
	ctx := context.Background()
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	_, err = s.esiClient.ESI.MailApi.DeleteCharactersCharacterIdMailMailId(ctx, characterID, mailID, nil)
	if err != nil {
		return err
	}
	err = s.r.DeleteMail(ctx, characterID, mailID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) GetMail(characterID int32, mailID int32) (model.Mail, error) {
	ctx := context.Background()
	return s.r.GetMail(ctx, characterID, mailID)
}

// SendMail created a new mail on ESI stores it locally.
func (s *Service) SendMail(characterID int32, subject string, recipients []model.EveEntity, body string) error {
	if subject == "" {
		return fmt.Errorf("missing subject")
	}
	if body == "" {
		return fmt.Errorf("missing body")
	}
	if len(recipients) == 0 {
		return fmt.Errorf("missing recipients")
	}
	rr, err := eveEntitiesToESIMailRecipients(recipients)
	if err != nil {
		return err
	}
	ctx := context.Background()
	token, err := s.getValidToken(ctx, characterID)
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
	arg1 := storage.MailLabelParams{
		CharacterID: characterID,
		LabelID:     model.MailLabelSent,
		Name:        "Sent",
	}
	_, err = s.r.GetOrCreateMailLabel(ctx, arg1) // make sure sent label exists
	if err != nil {
		return err
	}
	arg2 := storage.CreateMailParams{
		Body:         body,
		CharacterID:  characterID,
		FromID:       characterID,
		IsRead:       true,
		LabelIDs:     []int32{model.MailLabelSent},
		MailID:       mailID,
		RecipientIDs: recipientIDs,
		Subject:      subject,
		Timestamp:    time.Now(),
	}
	_, err = s.r.CreateMail(ctx, arg2)
	if err != nil {
		return err
	}
	return nil
}

func eveEntitiesToESIMailRecipients(ee []model.EveEntity) ([]esi.PostCharactersCharacterIdMailRecipient, error) {
	rr := make([]esi.PostCharactersCharacterIdMailRecipient, len(ee))
	for i, e := range ee {
		c, ok := eveEntityCategory2MailRecipientType[e.Category]
		if !ok {
			return rr, fmt.Errorf("failed to match EveEntity category to ESI mail recipient type: %v", e)
		}
		rr[i] = esi.PostCharactersCharacterIdMailRecipient{
			RecipientId:   e.ID,
			RecipientType: c,
		}
	}
	return rr, nil
}

// FIXME: Delete obsolete labels and mail lists
// TODO: Add ability to update existing mails for is_read and labels

// UpdateMails fetches and stores new mails from ESI for a character.
func (s *Service) UpdateMails(characterID int32) error {
	ctx := context.Background()
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return err
	}
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
	if err := s.updateMails(ctx, &token, headers); err != nil {
		return err
	}
	if err := s.DictionarySetTime(makeMailUpdateAtDictKey(characterID), time.Now()); err != nil {
		return err
	}
	return nil
}

func makeMailUpdateAtDictKey(characterID int32) string {
	return fmt.Sprintf("mail-updated-at-%d", characterID)
}

func (s *Service) MailUpdatedAt(characterID int32) (time.Time, error) {
	return s.DictionaryTime(makeMailUpdateAtDictKey(characterID))
}

func (s *Service) updateMailLabels(ctx context.Context, token *model.Token) error {
	if err := s.ensureValidToken(ctx, token); err != nil {
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
		arg := storage.MailLabelParams{
			CharacterID: token.CharacterID,
			Color:       o.Color,
			LabelID:     o.LabelId,
			Name:        o.Name,
			UnreadCount: int(o.UnreadCount),
		}
		_, err := s.r.UpdateOrCreateMailLabel(ctx, arg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) updateMailLists(ctx context.Context, token *model.Token) error {
	if err := s.ensureValidToken(ctx, token); err != nil {
		return err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	lists, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailLists(ctx, token.CharacterID, nil)
	if err != nil {
		return err
	}
	for _, o := range lists {
		_, err := s.r.UpdateOrCreateEveEntity(ctx, o.MailingListId, o.Name, model.EveEntityMailList)
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
func (s *Service) listMailHeaders(ctx context.Context, token *model.Token) ([]esi.GetCharactersCharacterIdMail200Ok, error) {
	if err := s.ensureValidToken(ctx, token); err != nil {
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

func (s *Service) updateMails(ctx context.Context, token *model.Token, headers []esi.GetCharactersCharacterIdMail200Ok) error {
	existingIDs, _, err := s.determineMailIDs(ctx, token.CharacterID, headers)
	if err != nil {
		return err
	}
	if err := s.ensureValidToken(ctx, token); err != nil {
		return err
	}
	ctx = contextWithToken(ctx, token.AccessToken)

	var c atomic.Int32
	var wg sync.WaitGroup
	maxGoroutines := 20
	guard := make(chan struct{}, maxGoroutines)
	for _, h := range headers {
		if existingIDs.Has(h.MailId) {
			continue
		}
		guard <- struct{}{}
		wg.Add(1)
		go func(mailID int32) {
			defer wg.Done()
			err := s.fetchAndStoreMail(ctx, token.CharacterID, mailID, &c)
			if err != nil {
				slog.Error("Failed to fetch new mail", "characterID", token.CharacterID, "mailID", mailID, "error", err)
			}
			<-guard
		}(h.MailId)
	}
	wg.Wait()
	total := c.Load()
	slog.Info("Received new mail", "count", total)
	return nil
}

func (s *Service) fetchAndStoreMail(ctx context.Context, characterID, mailID int32, c *atomic.Int32) error {
	m, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailMailId(ctx, characterID, mailID, nil)
	if err != nil {
		return err
	}
	entityIDs := set.New[int32]()
	entityIDs.Add(m.From)
	for _, r := range m.Recipients {
		entityIDs.Add(r.RecipientId)
	}
	_, err = s.addMissingEveEntities(ctx, entityIDs.ToSlice())
	if err != nil {
		return err
	}
	recipientIDs := make([]int32, len(m.Recipients))
	for i, r := range m.Recipients {
		recipientIDs[i] = r.RecipientId
	}
	arg := storage.CreateMailParams{
		Body:         m.Body,
		CharacterID:  characterID,
		FromID:       m.From,
		IsRead:       m.Read,
		LabelIDs:     m.Labels,
		MailID:       mailID,
		RecipientIDs: recipientIDs,
		Subject:      m.Subject,
		Timestamp:    m.Timestamp,
	}
	_, err = s.r.CreateMail(ctx, arg)
	if err != nil {
		return err
	}
	c.Add(1)
	return nil
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

func (s *Service) ListMailLists(characterID int32) ([]model.EveEntity, error) {
	ctx := context.Background()
	return s.r.ListMailListsOrdered(ctx, characterID)
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

func (s *Service) ListMailLabelsOrdered(characterID int32) ([]model.MailLabel, error) {
	ctx := context.Background()
	return s.r.ListMailLabelsOrdered(ctx, characterID)
}

// UpdateMailRead updates an existing mail as read
func (s *Service) UpdateMailRead(characterID, mailID int32) error {
	ctx := context.Background()
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	m, err := s.r.GetMail(ctx, characterID, mailID)
	if err != nil {
		return err
	}
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
