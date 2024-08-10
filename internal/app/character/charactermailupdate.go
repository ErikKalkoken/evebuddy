package character

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

const (
	// maxMails              = 1000
	maxMailHeadersPerPage = 50 // maximum header objects returned per page
)

// TODO: Add ability to delete obsolete mail labels

// updateCharacterMailLabelsESI updates the skillqueue for a character from ESI
// and reports wether it has changed.
func (s *CharacterService) updateCharacterMailLabelsESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionMailLabels {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			ll, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailLabels(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			slog.Info("Received mail labels from ESI", "characterID", characterID, "count", len(ll.Labels))
			return ll, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			ll := data.(esi.GetCharactersCharacterIdMailLabelsOk)
			labels := ll.Labels
			for _, o := range labels {
				arg := storage.MailLabelParams{
					CharacterID: characterID,
					Color:       o.Color,
					LabelID:     o.LabelId,
					Name:        o.Name,
					UnreadCount: int(o.UnreadCount),
				}
				_, err := s.st.UpdateOrCreateCharacterMailLabel(ctx, arg)
				if err != nil {
					return err
				}
			}
			return nil
		})
}

// updateCharacterMailListsESI updates the skillqueue for a character from ESI
// and reports wether it has changed.
func (s *CharacterService) updateCharacterMailListsESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionMailLists {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			lists, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailLists(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return lists, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			lists := data.([]esi.GetCharactersCharacterIdMailLists200Ok)
			for _, o := range lists {
				_, err := s.st.UpdateOrCreateEveEntity(ctx, o.MailingListId, o.Name, app.EveEntityMailList)
				if err != nil {
					return err
				}
				if err := s.st.CreateCharacterMailList(ctx, characterID, o.MailingListId); err != nil {
					return err
				}
			}
			return nil
		})
}

// updateCharacterMailsESI updates the skillqueue for a character from ESI
// and reports wether it has changed.
func (s *CharacterService) updateCharacterMailsESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionMails {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			headers, err := s.fetchMailHeadersESI(ctx, characterID)
			if err != nil {
				return false, err
			}
			return headers, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			headers := data.([]esi.GetCharactersCharacterIdMail200Ok)
			newHeaders, existingHeaders, err := s.determineNewMail(ctx, characterID, headers)
			if err != nil {
				return err
			}
			if len(newHeaders) > 0 {
				if err := s.resolveMailEntities(ctx, newHeaders); err != nil {
					return err
				}
				if err := s.addNewMailsESI(ctx, characterID, newHeaders); err != nil {
					return err
				}
			}
			if len(existingHeaders) > 0 {
				if err := s.updateExistingMail(ctx, characterID, existingHeaders); err != nil {
					return err
				}
			}
			// if err := s.st.DeleteObsoleteCharacterMailLabels(ctx, characterID); err != nil {
			// 	return err
			// }
			// if err := s.st.DeleteObsoleteCharacterMailLists(ctx, characterID); err != nil {
			// 	return err
			// }
			return nil
		})
}

// fetchMailHeadersESI fetched mail headers from ESI with paging and returns them.
func (s *CharacterService) fetchMailHeadersESI(ctx context.Context, characterID int32) ([]esi.GetCharactersCharacterIdMail200Ok, error) {
	var oo2 []esi.GetCharactersCharacterIdMail200Ok
	lastMailID := int32(0)
	maxMails, err := s.DictionaryService.IntWithFallback(app.SettingMaxMails, app.SettingMaxMailsDefault)
	if err != nil {
		return nil, err
	}
	for {
		var opts *esi.GetCharactersCharacterIdMailOpts
		if lastMailID > 0 {
			opts = &esi.GetCharactersCharacterIdMailOpts{LastMailId: esioptional.NewInt32(lastMailID)}
		} else {
			opts = nil
		}
		oo, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMail(ctx, characterID, opts)
		if err != nil {
			return nil, err
		}
		oo2 = slices.Concat(oo2, oo)
		isLimitExceeded := (maxMails != 0 && len(oo2)+maxMailHeadersPerPage > maxMails)
		if len(oo) < maxMailHeadersPerPage || isLimitExceeded {
			break
		}
		ids := make([]int32, len(oo))
		for i, o := range oo {
			ids[i] = o.MailId
		}
		lastMailID = slices.Min(ids)
	}
	slog.Info("Received mail headers", "characterID", characterID, "count", len(oo2))
	return oo2, nil
}

func (s *CharacterService) determineNewMail(ctx context.Context, characterID int32, mm []esi.GetCharactersCharacterIdMail200Ok) ([]esi.GetCharactersCharacterIdMail200Ok, []esi.GetCharactersCharacterIdMail200Ok, error) {
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

func (s *CharacterService) determineMailIDs(ctx context.Context, characterID int32, headers []esi.GetCharactersCharacterIdMail200Ok) (*set.Set[int32], *set.Set[int32], error) {
	ids, err := s.st.ListCharacterMailIDs(ctx, characterID)
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

func (s *CharacterService) resolveMailEntities(ctx context.Context, mm []esi.GetCharactersCharacterIdMail200Ok) error {
	entityIDs := set.New[int32]()
	for _, m := range mm {
		entityIDs.Add(m.From)
		for _, r := range m.Recipients {
			entityIDs.Add(r.RecipientId)
		}
	}
	_, err := s.EveUniverseService.AddMissingEveEntities(ctx, entityIDs.ToSlice())
	if err != nil {
		return err
	}
	return nil
}

func (s *CharacterService) addNewMailsESI(ctx context.Context, characterID int32, headers []esi.GetCharactersCharacterIdMail200Ok) error {
	count := 0
	g := new(errgroup.Group)
	g.SetLimit(20)
	for _, h := range headers {
		count++
		mailID := h.MailId
		g.Go(func() error {
			err := s.fetchAndStoreMail(ctx, characterID, mailID)
			if err != nil {
				return fmt.Errorf("failed to fetch mail %d: %w", mailID, err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	slog.Info("Received new mail from ESI", "characterID", characterID, "count", count)
	return nil
}

func (s *CharacterService) fetchAndStoreMail(ctx context.Context, characterID, mailID int32) error {
	m, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailMailId(ctx, characterID, mailID, nil)
	if err != nil {
		return err
	}
	recipientIDs := make([]int32, len(m.Recipients))
	for i, r := range m.Recipients {
		recipientIDs[i] = r.RecipientId
	}
	arg := storage.CreateCharacterMailParams{
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
	_, err = s.st.CreateCharacterMail(ctx, arg)
	if err != nil {
		return err
	}
	return nil
}

func (s *CharacterService) updateExistingMail(ctx context.Context, characterID int32, headers []esi.GetCharactersCharacterIdMail200Ok) error {
	for _, h := range headers {
		m, err := s.st.GetCharacterMail(ctx, characterID, h.MailId)
		if err != nil {
			return err
		}
		if m.IsRead != h.IsRead {
			err := s.st.UpdateCharacterMail(ctx, characterID, m.ID, h.IsRead, h.Labels)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateMailRead updates an existing mail as read
func (s *CharacterService) UpdateMailRead(ctx context.Context, characterID, mailID int32) error {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	m, err := s.st.GetCharacterMail(ctx, characterID, mailID)
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
	if err := s.st.UpdateCharacterMail(ctx, characterID, m.ID, m.IsRead, labelIDs); err != nil {
		return err
	}
	return nil

}
