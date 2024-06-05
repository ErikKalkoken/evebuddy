package character

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/antihax/goesi/esi"

	igoesi "github.com/ErikKalkoken/evebuddy/internal/helper/goesi"
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
func (s *CharacterService) DeleteCharacterMail(ctx context.Context, characterID, mailID int32) error {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return err
	}
	ctx = igoesi.ContextWithESIToken(ctx, token.AccessToken)
	_, err = s.esiClient.ESI.MailApi.DeleteCharactersCharacterIdMailMailId(ctx, characterID, mailID, nil)
	if err != nil {
		return err
	}
	err = s.st.DeleteCharacterMail(ctx, characterID, mailID)
	if err != nil {
		return err
	}
	return nil
}

func (s *CharacterService) GetCharacterMail(ctx context.Context, characterID int32, mailID int32) (*model.CharacterMail, error) {
	o, err := s.st.GetCharacterMail(ctx, characterID, mailID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	return o, err
}

// GetMailUnreadCount returns the number of unread mail for a character.
func (s *CharacterService) GetCharacterMailCounts(ctx context.Context, characterID int32) (int, int, error) {
	total, err := s.st.GetCharacterMailCount(ctx, characterID)
	if err != nil {
		return 0, 0, err
	}
	unread, err := s.st.GetCharacterMailUnreadCount(ctx, characterID)
	if err != nil {
		return 0, 0, err
	}
	return total, unread, nil
}

// SendCharacterMail creates a new mail on ESI and stores it locally.
func (s *CharacterService) SendCharacterMail(ctx context.Context, characterID int32, subject string, recipients []*model.EveEntity, body string) (int32, error) {
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
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return 0, err
	}
	esiMail := esi.PostCharactersCharacterIdMailMail{
		Body:       body,
		Subject:    subject,
		Recipients: rr,
	}
	ctx = igoesi.ContextWithESIToken(ctx, token.AccessToken)
	mailID, _, err := s.esiClient.ESI.MailApi.PostCharactersCharacterIdMail(ctx, characterID, esiMail, nil)
	if err != nil {
		return 0, err
	}
	recipientIDs := make([]int32, len(rr))
	for i, r := range rr {
		recipientIDs[i] = r.RecipientId
	}
	ids := slices.Concat(recipientIDs, []int32{characterID})
	_, err = s.eu.AddMissingEveEntities(ctx, ids)
	if err != nil {
		return 0, err
	}
	arg1 := storage.MailLabelParams{
		CharacterID: characterID,
		LabelID:     model.MailLabelSent,
		Name:        "Sent",
	}
	_, err = s.st.GetOrCreateCharacterMailLabel(ctx, arg1) // make sure sent label exists
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
	_, err = s.st.CreateCharacterMail(ctx, arg2)
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

func (s *CharacterService) GetCharacterMailLabelUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error) {
	return s.st.GetCharacterMailLabelUnreadCounts(ctx, characterID)
}

func (s *CharacterService) GetCharacterMailListUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error) {
	return s.st.GetCharacterMailListUnreadCounts(ctx, characterID)
}

func (s *CharacterService) ListCharacterMailLists(ctx context.Context, characterID int32) ([]*model.EveEntity, error) {
	return s.st.ListCharacterMailListsOrdered(ctx, characterID)
}

// ListMailsForLabel returns a character's mails for a label in descending order by timestamp.
// Return mails for all labels, when labelID = 0
func (s *CharacterService) ListCharacterMailHeadersForLabelOrdered(ctx context.Context, characterID int32, labelID int32) ([]*model.CharacterMailHeader, error) {
	return s.st.ListCharacterMailHeadersForLabelOrdered(ctx, characterID, labelID)
}

func (s *CharacterService) ListCharacterMailHeadersForListOrdered(ctx context.Context, characterID int32, listID int32) ([]*model.CharacterMailHeader, error) {
	return s.st.ListCharacterMailHeadersForListOrdered(ctx, characterID, listID)
}

func (s *CharacterService) ListCharacterMailLabelsOrdered(ctx context.Context, characterID int32) ([]*model.CharacterMailLabel, error) {
	return s.st.ListCharacterMailLabelsOrdered(ctx, characterID)
}
