package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/antihax/goesi/esi"
	"github.com/antihax/goesi/optional"
	"golang.org/x/sync/errgroup"

	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
)

const (
	maxMails          = 1000
	maxHeadersPerPage = 50 // maximum header objects returned per page
)

// FIXME: Delete obsolete labels and mail lists
// TODO: Add ability to update existing mails for is_read and labels

// UpdateMails fetches and stores new mails from ESI for a character.
func (s *Service) UpdateMails(characterID int32) (int, error) {
	ctx := context.Background()
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return 0, err
	}
	if err := s.updateMailLists(ctx, &token); err != nil {
		return 0, err
	}
	if err := s.updateMailLabels(ctx, &token); err != nil {
		return 0, err
	}
	headers, err := s.listMailHeaders(ctx, &token)
	if err != nil {
		return 0, err
	}
	if err := s.resolveMailEntities(ctx, headers); err != nil {
		return 0, err
	}
	headers, err = s.determineNewMail(ctx, token.CharacterID, headers)
	if err != nil {
		return 0, err
	}
	count := 0
	if len(headers) > 0 {
		count, err = s.updateMails(ctx, &token, headers)
		if err != nil {
			return 0, err
		}
	}
	if err := s.DictionarySetTime(makeMailUpdateAtDictKey(characterID), time.Now()); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) MailUpdatedAt(characterID int32) (time.Time, error) {
	return s.DictionaryTime(makeMailUpdateAtDictKey(characterID))
}

func makeMailUpdateAtDictKey(characterID int32) string {
	return fmt.Sprintf("mail-updated-at-%d", characterID)
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

func (s *Service) determineNewMail(ctx context.Context, characterID int32, mm []esi.GetCharactersCharacterIdMail200Ok) ([]esi.GetCharactersCharacterIdMail200Ok, error) {
	newMail := make([]esi.GetCharactersCharacterIdMail200Ok, 0, len(mm))
	existingIDs, _, err := s.determineMailIDs(ctx, characterID, mm)
	if err != nil {
		return newMail, err
	}
	for _, h := range mm {
		if existingIDs.Has(h.MailId) {
			continue
		}
		newMail = append(newMail, h)
	}
	return newMail, nil
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

func (s *Service) resolveMailEntities(ctx context.Context, mm []esi.GetCharactersCharacterIdMail200Ok) error {
	entityIDs := set.New[int32]()
	for _, m := range mm {
		entityIDs.Add(m.From)
		for _, r := range m.Recipients {
			entityIDs.Add(r.RecipientId)
		}
	}
	_, err := s.addMissingEveEntities(ctx, entityIDs.ToSlice())
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) updateMails(ctx context.Context, token *model.Token, headers []esi.GetCharactersCharacterIdMail200Ok) (int, error) {
	if err := s.ensureValidToken(ctx, token); err != nil {
		return 0, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)

	count := 0
	g := new(errgroup.Group)
	g.SetLimit(20)
	for _, h := range headers {
		count++
		mailID := h.MailId
		g.Go(func() error {
			err := s.fetchAndStoreMail(ctx, token.CharacterID, mailID)
			if err != nil {
				return fmt.Errorf("failed to fetch mail %d: %w", mailID, err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return 0, err
	}
	slog.Info("Received new mail", "characterID", token.CharacterID, "count", count)
	return count, nil
}

func (s *Service) fetchAndStoreMail(ctx context.Context, characterID, mailID int32) error {
	m, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailMailId(ctx, characterID, mailID, nil)
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
	return nil
}
