package service

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/antihax/goesi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

var eveEntityCategory2MailRecipientType = map[model.EveEntityCategory]string{
	model.EveEntityAlliance:    "alliance",
	model.EveEntityCharacter:   "character",
	model.EveEntityCorporation: "corporation",
	model.EveEntityMailList:    "mailing_list",
}

// DeleteCharacterMail deletes a mail both on ESI and in the database.
func (s *Service) DeleteCharacterMail(characterID, mailID int32) error {
	ctx := context.Background()
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	_, err = s.esiClient.ESI.MailApi.DeleteCharactersCharacterIdMailMailId(ctx, characterID, mailID, nil)
	if err != nil {
		return err
	}
	err = s.r.DeleteCharacterMail(ctx, characterID, mailID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) GetCharacterMail(characterID int32, mailID int32) (*model.CharacterMail, error) {
	ctx := context.Background()
	return s.r.GetCharacterMail(ctx, characterID, mailID)
}

// GetMailUnreadCount returns the number of unread mail for a character.
func (s *Service) GetCharacterMailCounts(characterID int32) (int, int, error) {
	ctx := context.Background()
	total, err := s.r.GetCharacterMailCount(ctx, characterID)
	if err != nil {
		return 0, 0, err
	}
	unread, err := s.r.GetCharacterMailUnreadCount(ctx, characterID)
	if err != nil {
		return 0, 0, err
	}
	return total, unread, nil
}

// SendCharacterMail creates a new mail on ESI and stores it locally.
func (s *Service) SendCharacterMail(characterID int32, subject string, recipients []*model.EveEntity, body string) (int32, error) {
	if subject == "" {
		return 0, fmt.Errorf("missing subject")
	}
	if body == "" {
		return 0, fmt.Errorf("missing body")
	}
	if len(recipients) == 0 {
		return 0, fmt.Errorf("missing recipients")
	}
	rr, err := eveEntitiesToESIMailRecipients(recipients)
	if err != nil {
		return 0, err
	}
	ctx := context.Background()
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return 0, err
	}
	esiMail := esi.PostCharactersCharacterIdMailMail{
		Body:       body,
		Subject:    subject,
		Recipients: rr,
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	mailID, _, err := s.esiClient.ESI.MailApi.PostCharactersCharacterIdMail(ctx, characterID, esiMail, nil)
	if err != nil {
		return 0, err
	}
	recipientIDs := make([]int32, len(rr))
	for i, r := range rr {
		recipientIDs[i] = r.RecipientId
	}
	ids := slices.Concat(recipientIDs, []int32{characterID})
	_, err = s.AddMissingEveEntities(ctx, ids)
	if err != nil {
		return 0, err
	}
	arg1 := storage.MailLabelParams{
		CharacterID: characterID,
		LabelID:     model.MailLabelSent,
		Name:        "Sent",
	}
	_, err = s.r.GetOrCreateCharacterMailLabel(ctx, arg1) // make sure sent label exists
	if err != nil {
		return 0, err
	}
	arg2 := storage.CreateCharacterMailParams{
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
	_, err = s.r.CreateCharacterMail(ctx, arg2)
	if err != nil {
		return 0, err
	}
	return mailID, nil
}

func eveEntitiesToESIMailRecipients(ee []*model.EveEntity) ([]esi.PostCharactersCharacterIdMailRecipient, error) {
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

func (s *Service) GetCharacterMailLabelUnreadCounts(characterID int32) (map[int32]int, error) {
	ctx := context.Background()
	return s.r.GetCharacterMailLabelUnreadCounts(ctx, characterID)
}

func (s *Service) GetCharacterMailListUnreadCounts(characterID int32) (map[int32]int, error) {
	ctx := context.Background()
	return s.r.GetCharacterMailListUnreadCounts(ctx, characterID)
}

func (s *Service) ListCharacterMailLists(characterID int32) ([]*model.EveEntity, error) {
	ctx := context.Background()
	return s.r.ListCharacterMailListsOrdered(ctx, characterID)
}

// ListMailsForLabel returns a character's mails for a label in descending order by timestamp.
// Return mails for all labels, when labelID = 0
func (s *Service) ListCharacterMailIDsForLabelOrdered(characterID int32, labelID int32) ([]int32, error) {
	ctx := context.Background()
	return s.r.ListCharacterMailIDsForLabelOrdered(ctx, characterID, labelID)
}

func (s *Service) ListCharacterMailIDsForListOrdered(characterID int32, listID int32) ([]int32, error) {
	ctx := context.Background()
	return s.r.ListCharacterMailIDsForListOrdered(ctx, characterID, listID)
}

func (s *Service) ListCharacterMailLabelsOrdered(characterID int32) ([]*model.CharacterMailLabel, error) {
	ctx := context.Background()
	return s.r.ListCharacterMailLabelsOrdered(ctx, characterID)
}