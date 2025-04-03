package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/antihax/goesi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

// DeleteMail deletes a mail both on ESI and in the database.
func (s *CharacterService) DeleteMail(ctx context.Context, characterID, mailID int32) error {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	_, err = s.esiClient.ESI.MailApi.DeleteCharactersCharacterIdMailMailId(ctx, characterID, mailID, nil)
	if err != nil {
		return err
	}
	err = s.st.DeleteCharacterMail(ctx, characterID, mailID)
	if err != nil {
		return err
	}
	slog.Info("Mail deleted", "characterID", characterID, "mailID", mailID)
	return nil
}

func (s *CharacterService) GetMail(ctx context.Context, characterID int32, mailID int32) (*app.CharacterMail, error) {
	return s.st.GetCharacterMail(ctx, characterID, mailID)
}

func (s *CharacterService) GetAllMailUnreadCount(ctx context.Context) (int, error) {
	return s.st.GetAllCharacterMailUnreadCount(ctx)
}

// GetMailCounts returns the number of unread mail for a character.
func (s *CharacterService) GetMailCounts(ctx context.Context, characterID int32) (int, int, error) {
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

func (s *CharacterService) GetMailLabelUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error) {
	return s.st.GetCharacterMailLabelUnreadCounts(ctx, characterID)
}

func (s *CharacterService) GetMailListUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error) {
	return s.st.GetCharacterMailListUnreadCounts(ctx, characterID)
}

func (cs *CharacterService) NotifyMails(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error {
	mm, err := cs.st.ListCharacterMailHeadersForUnprocessed(ctx, characterID, earliest)
	if err != nil {
		return err
	}
	characterName, err := cs.getCharacterName(ctx, characterID)
	if err != nil {
		return err
	}
	for _, m := range mm {
		if m.Timestamp.Before(earliest) {
			continue
		}
		title := fmt.Sprintf("%s: New Mail from %s", characterName, m.From.Name)
		content := m.Subject
		notify(title, content)
		if err := cs.st.UpdateCharacterMailSetProcessed(ctx, m.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *CharacterService) ListMailLists(ctx context.Context, characterID int32) ([]*app.EveEntity, error) {
	return s.st.ListCharacterMailListsOrdered(ctx, characterID)
}

// ListMailsForLabel returns a character's mails for a label in descending order by timestamp.
func (s *CharacterService) ListMailHeadersForLabelOrdered(ctx context.Context, characterID int32, labelID int32) ([]*app.CharacterMailHeader, error) {
	return s.st.ListCharacterMailHeadersForLabelOrdered(ctx, characterID, labelID)
}

func (s *CharacterService) ListMailHeadersForListOrdered(ctx context.Context, characterID int32, listID int32) ([]*app.CharacterMailHeader, error) {
	return s.st.ListCharacterMailHeadersForListOrdered(ctx, characterID, listID)
}

func (s *CharacterService) ListMailLabelsOrdered(ctx context.Context, characterID int32) ([]*app.CharacterMailLabel, error) {
	return s.st.ListCharacterMailLabelsOrdered(ctx, characterID)
}

// SendMail creates a new mail on ESI and stores it locally.
func (s *CharacterService) SendMail(ctx context.Context, characterID int32, subject string, recipients []*app.EveEntity, body string) (int32, error) {
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
	_, err = s.EveUniverseService.AddMissingEntities(ctx, ids)
	if err != nil {
		return 0, err
	}
	arg1 := storage.MailLabelParams{
		CharacterID: characterID,
		LabelID:     app.MailLabelSent,
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
		LabelIDs:     []int32{app.MailLabelSent},
		MailID:       mailID,
		RecipientIDs: recipientIDs,
		Subject:      subject,
		Timestamp:    time.Now(),
	}
	_, err = s.st.CreateCharacterMail(ctx, arg2)
	if err != nil {
		return 0, err
	}
	slog.Info("Mail sent", "characterID", characterID, "mailID", mailID)
	return mailID, nil
}

var eveEntityCategory2MailRecipientType = map[app.EveEntityCategory]string{
	app.EveEntityAlliance:    "alliance",
	app.EveEntityCharacter:   "character",
	app.EveEntityCorporation: "corporation",
	app.EveEntityMailList:    "mailing_list",
}

func eveEntitiesToESIMailRecipients(ee []*app.EveEntity) ([]esi.PostCharactersCharacterIdMailRecipient, error) {
	rr := make([]esi.PostCharactersCharacterIdMailRecipient, len(ee))
	for i, e := range ee {
		c, ok := eveEntityCategory2MailRecipientType[e.Category]
		if !ok {
			return rr, fmt.Errorf("match EveEntity category to ESI mail recipient type: %v", e)
		}
		rr[i] = esi.PostCharactersCharacterIdMailRecipient{
			RecipientId:   e.ID,
			RecipientType: c,
		}
	}
	return rr, nil
}
