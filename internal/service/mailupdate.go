package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/antihax/goesi/esi"
	"github.com/antihax/goesi/optional"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

const (
	maxMails          = 1000
	maxHeadersPerPage = 50 // maximum header objects returned per page
)

// TODO: Add ability to delete obsolete mail labels

// updateMailLabelsESI updates the skillqueue for a character from ESI
// and reports wether it has changed.
func (s *Service) updateMailLabelsESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	ll, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailLabels(ctx, token.CharacterID, nil)
	if err != nil {
		return false, err
	}
	slog.Info("Received mail labels from ESI", "count", len(ll.Labels), "characterID", token.CharacterID)
	changed, err := s.hasSectionChanged(ctx, characterID, model.UpdateSectionMailLabels, ll)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	labels := ll.Labels
	for _, o := range labels {
		arg := storage.MailLabelParams{
			MyCharacterID: token.CharacterID,
			Color:         o.Color,
			LabelID:       o.LabelId,
			Name:          o.Name,
			UnreadCount:   int(o.UnreadCount),
		}
		_, err := s.r.UpdateOrCreateMailLabel(ctx, arg)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

// updateMailListsESI updates the skillqueue for a character from ESI
// and reports wether it has changed.
func (s *Service) updateMailListsESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	lists, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailLists(ctx, token.CharacterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.hasSectionChanged(ctx, characterID, model.UpdateSectionMailLists, lists)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	for _, o := range lists {
		_, err := s.r.UpdateOrCreateEveEntity(ctx, o.MailingListId, o.Name, model.EveEntityMailList)
		if err != nil {
			return false, err
		}
		if err := s.r.CreateMailList(ctx, token.CharacterID, o.MailingListId); err != nil {
			return false, err
		}
	}
	return true, nil
}

// updateMailESI updates the skillqueue for a character from ESI
// and reports wether it has changed.
func (s *Service) updateMailESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	headers, err := s.fetchMailHeadersESI(ctx, token)
	if err != nil {
		return false, err
	}
	changed, err := s.hasSectionChanged(ctx, characterID, model.UpdateSectionMails, headers)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	newHeaders, existingHeaders, err := s.determineNewMail(ctx, token.CharacterID, headers)
	if err != nil {
		return false, err
	}
	if len(newHeaders) > 0 {
		if err := s.resolveMailEntities(ctx, newHeaders); err != nil {
			return false, err
		}
		if err := s.addNewMailsESI(ctx, token, newHeaders); err != nil {
			return false, err
		}
	}
	if len(existingHeaders) > 0 {
		if err := s.updateExistingMail(ctx, characterID, existingHeaders); err != nil {
			return false, err
		}
	}
	if err := s.r.DeleteObsoleteMailLabels(ctx, characterID); err != nil {
		return false, err
	}
	if err := s.r.DeleteObsoleteMailLists(ctx, characterID); err != nil {
		return false, err
	}
	return true, nil
}

// fetchMailHeadersESI fetched mail headers from ESI with paging and returns them.
func (s *Service) fetchMailHeadersESI(ctx context.Context, token *model.CharacterToken) ([]esi.GetCharactersCharacterIdMail200Ok, error) {
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

func (s *Service) determineNewMail(ctx context.Context, characterID int32, mm []esi.GetCharactersCharacterIdMail200Ok) ([]esi.GetCharactersCharacterIdMail200Ok, []esi.GetCharactersCharacterIdMail200Ok, error) {
	newMail := make([]esi.GetCharactersCharacterIdMail200Ok, 0, len(mm))
	existingMail := make([]esi.GetCharactersCharacterIdMail200Ok, 0, len(mm))
	existingIDs, _, err := s.determineMailIDs(ctx, characterID, mm)
	if err != nil {
		return newMail, existingMail, err
	}
	for _, h := range mm {
		if existingIDs.Has(h.MailId) {
			existingMail = append(existingMail, h)
		} else {
			newMail = append(newMail, h)
		}
	}
	return newMail, existingMail, nil
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
	_, err := s.AddMissingEveEntities(ctx, entityIDs.ToSlice())
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) addNewMailsESI(ctx context.Context, token *model.CharacterToken, headers []esi.GetCharactersCharacterIdMail200Ok) error {
	if err := s.ensureValidToken(ctx, token); err != nil {
		return err
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
		return err
	}
	slog.Info("Received new mail from ESI", "characterID", token.CharacterID, "count", count)
	return nil
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
		Body:          m.Body,
		MyCharacterID: characterID,
		FromID:        m.From,
		IsRead:        m.Read,
		LabelIDs:      m.Labels,
		MailID:        mailID,
		RecipientIDs:  recipientIDs,
		Subject:       m.Subject,
		Timestamp:     m.Timestamp,
	}
	_, err = s.r.CreateMail(ctx, arg)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) updateExistingMail(ctx context.Context, characterID int32, headers []esi.GetCharactersCharacterIdMail200Ok) error {
	for _, h := range headers {
		m, err := s.r.GetMail(ctx, characterID, h.MailId)
		if err != nil {
			return err
		}
		if m.IsRead != h.IsRead {
			err := s.r.UpdateMail(ctx, characterID, m.ID, h.IsRead, h.Labels)
			if err != nil {
				return err
			}
		}
	}
	return nil
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
	if err := s.r.UpdateMail(ctx, characterID, m.ID, m.IsRead, labelIDs); err != nil {
		return err
	}
	return nil

}
