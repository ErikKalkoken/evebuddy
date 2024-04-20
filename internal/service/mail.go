package service

import (
	"context"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
	"fmt"
	"slices"
	"time"

	"github.com/antihax/goesi/esi"
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
