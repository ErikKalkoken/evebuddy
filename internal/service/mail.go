package service

import (
	"context"
	"database/sql"
	"errors"
	"example/evebuddy/internal/helper/set"
	islices "example/evebuddy/internal/helper/slices"
	"example/evebuddy/internal/repository"
	"example/evebuddy/internal/repository/sqlc"

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
	token, err := s.GetValidToken(m.CharacterID)
	if err != nil {
		return err
	}
	ctx := token.NewContext()
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
	token, err := s.GetValidToken(characterID)
	if err != nil {
		return err
	}
	esiMail := esi.PostCharactersCharacterIdMailMail{
		Body:       body,
		Subject:    subject,
		Recipients: rr,
	}
	ctx := token.NewContext()
	mailID, _, err := s.esiClient.ESI.MailApi.PostCharactersCharacterIdMail(ctx, characterID, esiMail, nil)
	if err != nil {
		return err
	}
	ids := []int32{characterID}
	for _, r := range rr {
		ids = append(ids, r.RecipientId)
	}
	_, err = s.addMissingEveEntities(ids)
	if err != nil {
		return err
	}
	from, err := s.r.GetEveEntity(ctx, token.CharacterID)
	if err != nil {
		return err
	}
	// FIXME: Ensure this still works when no labels have yet been loaded from ESI
	mailLabelParams := sqlc.GetMailLabelParams{CharacterID: int64(token.CharacterID), LabelID: LabelSent}
	label, err := s.r.GetMailLabel(ctx, mailLabelParams)
	if err != nil {
		return err
	}
	var ee []sqlc.EveEntity
	for _, r := range rr {
		e, err := s.r.GetEveEntity(ctx, r.RecipientId)
		if err != nil {
			return err
		}
		ee = append(ee, e)
	}
	mailParams := sqlc.CreateMailParams{
		Body:        body,
		CharacterID: int64(token.CharacterID),
		FromID:      from.ID,
		MailID:      int64(mailID),
		Subject:     subject,
		IsRead:      true,
		Timestamp:   time.Now(),
	}
	mail, err := s.r.CreateMail(ctx, mailParams)
	if err != nil {
		return err
	}
	mailMailLabelParams := sqlc.CreateMailMailLabelParams{
		MailID: mail.ID, MailLabelID: label.ID,
	}
	if err := s.r.CreateMailMailLabel(ctx, mailMailLabelParams); err != nil {
		return err
	}
	for _, e := range ee {
		arg := sqlc.CreateMailRecipientParams{MailID: mail.ID, EveEntityID: e.ID}
		err := s.r.CreateMailRecipient(ctx, arg)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateMailRead updates an existing mail as read
func (s *Service) UpdateMailRead(m *repository.Mail) error {
	token, err := s.GetValidToken(m.CharacterID)
	if err != nil {
		return err
	}
	ctx := token.NewContext()
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
	arg := sqlc.GetMailParams{
		CharacterID: int64(characterID),
		MailID:      int64(mailID),
	}
	row, err := s.r.GetMail(ctx, arg)
	if err != nil {
		return repository.Mail{}, err
	}
	ll, err := s.r.GetMailLabels(ctx, row.repository.Mail.ID)
	if err != nil {
		return repository.Mail{}, err
	}
	rr, err := s.r.GetMailRecipients(ctx, row.repository.Mail.ID)
	if err != nil {
		return repository.Mail{}, err
	}
	return mailFromDBModel(row.repository.Mail, row.EveEntity, ll, rr), nil
}

// FIXME: Delete obsolete labels and mail lists
// TODO: Add ability to update existing mails for is_read and labels

// FetchMail fetches and stores new mails from ESI for a character.
func (s *Service) FetchMail(characterID int32, status binding.String) error {
	ctx := context.Background()
	token, err := s.GetValidToken(characterID)
	if err != nil {
		return err
	}
	row, err := s.r.GetCharacter(ctx, int64(characterID))
	if err != nil {
		return err
	}
	character := characterFromDBModel(row.Character, row.EveEntity, row.EveEntity_2, row.EveEntity_3)
	status.Set(fmt.Sprintf("Checking for new mail for %v", character.Name))
	if err := s.updateMailLists(token); err != nil {
		return err
	}
	if err := s.updateMailLabels(token); err != nil {
		return err
	}
	headers, err := s.listMailHeaders(token)
	if err != nil {
		return err
	}
	err = s.updateMails(token, headers, status)
	if err != nil {
		return err
	}
	character.MailUpdatedAt = time.Now()
	if err := s.r.UpdateCharacter(ctx, character.ToDBUpdateParams()); err != nil {
		return err
	}
	return nil
}

func (s *Service) updateMailLabels(token *Token) error {
	if err := s.EnsureValid(token); err != nil {
		return err
	}
	ctx := token.NewContext()
	ll, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailLabels(ctx, token.CharacterID, nil)
	if err != nil {
		return err
	}
	labels := ll.Labels
	slog.Info("Received mail labels from ESI", "count", len(labels), "characterID", token.CharacterID)
	for _, o := range labels {
		arg := sqlc.CreateMailLabelParams{
			CharacterID: int64(token.CharacterID),
			LabelID:     int64(o.LabelId),
			Color:       o.Color,
			Name:        o.Name,
			UnreadCount: int64(o.UnreadCount),
		}
		if err := s.r.CreateMailLabel(ctx, arg); err != nil {
			if !isSqlite3ErrConstraint(err) {
				return err
			}
			arg := sqlc.UpdateMailLabelParams{
				CharacterID: int64(token.CharacterID),
				LabelID:     int64(o.LabelId),
				Color:       o.Color,
				Name:        o.Name,
				UnreadCount: int64(o.UnreadCount),
			}
			if err := s.r.UpdateMailLabel(ctx, arg); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) updateMailLists(token *Token) error {
	if err := s.EnsureValid(token); err != nil {
		return err
	}
	ctx := token.NewContext()
	lists, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailLists(ctx, token.CharacterID, nil)
	if err != nil {
		return err
	}
	for _, o := range lists {
		arg1 := sqlc.UpdateEveEntityParams{
			ID:       int64(o.MailingListId),
			Name:     o.Name,
			Category: sqlc.EveEntityMailList,
		}
		if err := s.r.UpdateEveEntity(ctx, arg1); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				arg := sqlc.CreateEveEntityParams{
					ID:       int64(o.MailingListId),
					Name:     o.Name,
					Category: sqlc.EveEntityMailList,
				}
				_, err := s.r.CreateEveEntity(ctx, arg)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
		arg2 := sqlc.CreateMailListParams{CharacterID: int64(token.CharacterID), EveEntityID: arg1.ID}
		if err := s.r.CreateMailList(ctx, arg2); err != nil {
			return err
		}
	}
	return nil
}

// listMailHeaders fetched mail headers from ESI with paging and returns them.
func (s *Service) listMailHeaders(token *Token) ([]esi.GetCharactersCharacterIdMail200Ok, error) {
	if err := s.EnsureValid(token); err != nil {
		return nil, err
	}
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
		objs, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMail(token.NewContext(), token.CharacterID, opts)
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

func (s *Service) updateMails(token *Token, headers []esi.GetCharactersCharacterIdMail200Ok, status binding.String) error {
	existingIDs, missingIDs, err := s.determineMailIDs(token.CharacterID, headers)
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

	if err := s.EnsureValid(token); err != nil {
		return err
	}
	ctx := token.NewContext()
	characterRow, err := s.r.GetCharacter(ctx, int64(token.CharacterID))
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
			s.fetchAndStoreMail(ctx, header, token, newMailsCount, &c, status, characterRow.Character.Name)
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

func (s *Service) fetchAndStoreMail(ctx context.Context, header esi.GetCharactersCharacterIdMail200Ok, token *Token, newMailsCount int, c *atomic.Int32, status binding.String, characterName string) {
	entityIDs := set.New[int32]()
	entityIDs.Add(header.From)
	for _, r := range header.Recipients {
		entityIDs.Add(r.RecipientId)
	}
	_, err := s.addMissingEveEntities(entityIDs.ToSlice())
	if err != nil {
		slog.Error("Failed to process mail", "header", header, "error", err)
		return
	}
	m, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailMailId(ctx, token.CharacterID, header.MailId, nil)
	if err != nil {
		slog.Error("Failed to process mail", "header", header, "error", err)
		return
	}
	from, err := s.r.GetEveEntity(ctx, int64(header.From))
	if err != nil {
		slog.Error("Failed to parse \"from\" in mail", "header", header, "error", err)
		return
	}
	mailArg := sqlc.CreateMailParams{
		Body:        m.Body,
		CharacterID: int64(token.CharacterID),
		FromID:      from.ID,
		MailID:      int64(header.MailId),
		Subject:     header.Subject,
		IsRead:      header.IsRead,
		Timestamp:   header.Timestamp,
	}
	mail, err := s.r.CreateMail(ctx, mailArg)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	if err := s.fetchAndStoreMailRecipients(ctx, mail.ID, header); err != nil {
		slog.Error(err.Error())
		return
	}

	if err := s.fetchAndStoreMailLabels(ctx, mail, m.Labels); err != nil {
		slog.Error(err.Error())
		return
	}

	slog.Info("Created new mail", "mailID", header.MailId, "characterID", token.CharacterID)
	c.Add(1)
	current := c.Load()
	status.Set(fmt.Sprintf("Fetched %d / %d new mails for %v", current, newMailsCount, characterName))
}

func (s *Service) fetchAndStoreMailRecipients(ctx context.Context, mailID int64, header esi.GetCharactersCharacterIdMail200Ok) error {
	for _, r := range header.Recipients {
		entity, err := s.r.GetEveEntity(ctx, int64(r.RecipientId))
		if err != nil {
			return fmt.Errorf("failed to resolve mail recipient %v: %s", r, err)
		}
		arg := sqlc.CreateMailRecipientParams{
			MailID:      mailID,
			EveEntityID: entity.ID,
		}
		if err := s.r.CreateMailRecipient(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) fetchAndStoreMailLabels(ctx context.Context, mail sqlc.repository.Mail, labelIDs []int32) error {
	labelIDs2 := islices.ConvertNumeric[int32, int64](labelIDs)
	labelArg := sqlc.ListMailLabelsByIDsParams{
		CharacterID: mail.CharacterID,
		Ids:         labelIDs2,
	}
	labels, err := s.r.ListMailLabelsByIDs(ctx, labelArg)
	if err != nil {
		return fmt.Errorf("failed to resolve mail labels %v: %s", labelIDs, err)
	}
	for _, label := range labels {
		arg := sqlc.CreateMailMailLabelParams{
			MailLabelID: label.ID, MailID: mail.ID,
		}
		if err := s.r.CreateMailMailLabel(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) determineMailIDs(characterID int32, headers []esi.GetCharactersCharacterIdMail200Ok) (*set.Set[int32], *set.Set[int32], error) {
	ids, err := s.r.ListMailIDs(context.Background(), int64(characterID))
	if err != nil {
		return nil, nil, err
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	existingIDs := set.NewFromSlice(ids2)
	incomingIDs := set.New[int32]()
	for _, h := range headers {
		incomingIDs.Add(h.MailId)
	}
	missingIDs := incomingIDs.Difference(existingIDs)
	return existingIDs, missingIDs, nil
}

func (s *Service) GetMailLabelUnreadCounts(characterID int32) (map[int32]int, error) {
	rows, err := s.r.GetMailLabelUnreadCounts(context.Background(), int64(characterID))
	if err != nil {
		return nil, err
	}
	result := make(map[int32]int)
	for _, r := range rows {
		result[int32(r.LabelID)] = int(r.UnreadCount2)
	}
	return result, nil
}

func (s *Service) GetMailListUnreadCounts(characterID int32) (map[int32]int, error) {
	rows, err := s.r.GetMailListUnreadCounts(context.Background(), int64(characterID))
	if err != nil {
		return nil, err
	}
	result := make(map[int32]int)
	for _, r := range rows {
		result[int32(r.ListID)] = int(r.UnreadCount2)
	}
	return result, nil
}

func (s *Service) ListMailLists(characterID int32) ([]EveEntity, error) {
	ll, err := s.r.ListMailLists(context.Background(), int64(characterID))
	if err != nil {
		return nil, err
	}
	ee := make([]EveEntity, len(ll))
	for i, l := range ll {
		ee[i] = eveEntityFromDBModel(l)
	}
	return ee, nil
}

// ListMailsForLabel returns a character's mails for a label in descending order by timestamp.
// Return mails for all labels, when labelID = 0
func (s *Service) ListMailIDsForLabelOrdered(characterID int32, labelID int32) ([]int32, error) {
	ctx := context.Background()
	switch labelID {
	case LabelAll:
		ids, err := s.r.ListMailIDsOrdered(ctx, int64(characterID))
		if err != nil {
			return nil, err
		}
		ids2 := islices.ConvertNumeric[int64, int32](ids)
		return ids2, nil
	case LabelNone:
		ids, err := s.r.ListMailIDsNoLabelOrdered(ctx, int64(characterID))
		if err != nil {
			return nil, err
		}
		ids2 := islices.ConvertNumeric[int64, int32](ids)
		return ids2, nil
	default:
		arg := sqlc.ListMailIDsForLabelOrderedParams{
			CharacterID: int64(characterID),
			LabelID:     int64(labelID),
		}
		ids, err := s.r.ListMailIDsForLabelOrdered(ctx, arg)
		if err != nil {
			return nil, err
		}
		ids2 := islices.ConvertNumeric[int64, int32](ids)
		return ids2, nil
	}
}

func (s *Service) ListMailIDsForListOrdered(characterID int32, listID int32) ([]int32, error) {
	arg := sqlc.ListMailIDsForListParams{
		CharacterID: int64(characterID),
		EveEntityID: int64(listID),
	}
	ids, err := s.r.ListMailIDsForList(context.Background(), arg)
	if err != nil {
		return nil, err
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}
