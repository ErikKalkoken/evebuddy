package characterservice

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// DeleteMail deletes a mail both on ESI and in the database.
func (s *CharacterService) DeleteMail(ctx context.Context, characterID, mailID int32) error {
	token, err := s.GetValidCharacterTokenWithScopes(ctx, characterID, app.SectionCharacterMailHeaders.Scopes())
	if err != nil {
		return err
	}
	ctx = xgoesi.NewContextWithAuth(ctx, token.CharacterID, token.AccessToken)
	ctx = xgoesi.NewContextWithOperationID(ctx, "DeleteCharactersCharacterIdMailMailId")
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
	return s.st.GetAllCharactersMailUnreadCount(ctx)
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

func (s *CharacterService) NotifyMails(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error {
	_, err, _ := s.sfg.Do(fmt.Sprintf("NotifyMails-%d", characterID), func() (any, error) {
		mm, err := s.st.ListCharacterMailHeadersForUnprocessed(ctx, characterID, earliest)
		if err != nil {
			return nil, err
		}
		characterName, err := s.getCharacterName(ctx, characterID)
		if err != nil {
			return nil, err
		}
		for _, m := range mm {
			if m.Timestamp.Before(earliest) {
				continue
			}
			title := fmt.Sprintf("%s: New Mail from %s", characterName, m.From.Name)
			content := m.Subject
			notify(title, content)
			if err := s.st.UpdateCharacterMailSetProcessed(ctx, m.ID); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("NotifyMails for character %d: %w", characterID, err)
	}
	return nil
}

func (s *CharacterService) ListMailLists(ctx context.Context, characterID int32) ([]*app.EveEntity, error) {
	return s.st.ListCharacterMailListsOrdered(ctx, characterID)
}

// ListMailHeadersForLabelOrdered returns a character's mails for a label in descending order by timestamp.
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
	token, err := s.GetValidCharacterTokenWithScopes(ctx, characterID, app.SectionCharacterMailHeaders.Scopes())
	if err != nil {
		return 0, err
	}
	ctx = xgoesi.NewContextWithAuth(ctx, token.CharacterID, token.AccessToken)
	ctx = xgoesi.NewContextWithOperationID(ctx, "PostCharactersCharacterIdMail")
	mailID, _, err := s.esiClient.ESI.MailApi.PostCharactersCharacterIdMail(ctx, characterID, esi.PostCharactersCharacterIdMailMail{
		Body:       body,
		Subject:    subject,
		Recipients: rr,
	}, nil,
	)
	if err != nil {
		return 0, err
	}
	recipientIDs := make([]int32, len(rr))
	for i, r := range rr {
		recipientIDs[i] = r.RecipientId
	}
	ids := set.Union(set.Of(recipientIDs...), set.Of(characterID))
	_, err = s.eus.AddMissingEntities(ctx, ids)
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
		Body:         optional.New(body),
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

// UpdateMailRead updates an existing mail as read
func (s *CharacterService) UpdateMailRead(ctx context.Context, characterID, mailID int32, isRead bool) error {
	_, err, _ := s.sfg.Do(fmt.Sprintf("UpdateMailRead-%d-%d", characterID, mailID), func() (any, error) {
		token, err := s.GetValidCharacterTokenWithScopes(ctx, characterID, app.SectionCharacterMailHeaders.Scopes())
		if err != nil {
			return nil, err
		}
		ctx = xgoesi.NewContextWithAuth(ctx, token.CharacterID, token.AccessToken)
		ctx = xgoesi.NewContextWithOperationID(ctx, "PutCharactersCharacterIdMailMailId")
		m, err := s.st.GetCharacterMail(ctx, characterID, mailID)
		if err != nil {
			return nil, err
		}
		contents := esi.PutCharactersCharacterIdMailMailIdContents{
			Labels: m.LabelIDs(),
			Read:   isRead,
		}
		_, err = s.esiClient.ESI.MailApi.PutCharactersCharacterIdMailMailId(ctx, m.CharacterID, contents, m.MailID, nil)
		if err != nil {
			return nil, err
		}
		if err := s.st.UpdateCharacterMailSetIsRead(ctx, characterID, m.ID, isRead); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		return err
	}
	return nil
}

// UpdateMailBodyESI updates the body of a mail from ESI.
func (s *CharacterService) UpdateMailBodyESI(ctx context.Context, characterID int32, mailID int32) (string, error) {
	b, err := s.updateMailBodyESI(ctx, characterID, mailID)
	if err != nil {
		return "", err
	}
	slog.Info("Mail body updated", "characterID", characterID, "mailID", mailID)
	return b, err
}

func (s *CharacterService) updateMailBodyESI(ctx context.Context, characterID int32, mailID int32) (string, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("UpdateMailBodyESI-%d-%d", characterID, mailID), func() (any, error) {
		token, err := s.GetValidCharacterTokenWithScopes(ctx, characterID, app.SectionCharacterMailHeaders.Scopes())
		if err != nil {
			return "", err
		}
		ctx = xgoesi.NewContextWithAuth(ctx, token.CharacterID, token.AccessToken)
		ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdMailMailId")
		mail, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailMailId(ctx, characterID, mailID, nil)
		if err != nil {
			return "", err
		}
		err = s.st.UpdateCharacterMailSetBody(ctx, characterID, mailID, optional.New(mail.Body))
		if err != nil {
			return "", err
		}
		return mail.Body, nil
	})
	if err != nil {
		return "", err
	}
	return x.(string), nil
}

// DownloadMissingMailBodies downloads missing mail bodies for a character
// and reports whether the function was aborted.
// Only one instance per character will run at a time and additional calls will be aborted.
func (s *CharacterService) DownloadMissingMailBodies(ctx context.Context, characterID int32) (bool, error) {
	_, err, aborted := s.sig.Do(fmt.Sprintf("DownloadMissingMailBodies-%d", characterID), func() (any, error) {
		ids, err := s.st.ListCharacterMailsWithoutBody(ctx, characterID)
		if err != nil {
			return nil, err
		}
		if ids.Size() == 0 {
			return nil, nil
		}
		ids2 := slices.SortedFunc(ids.All(), func(a, b int32) int {
			return cmp.Compare(b, a)
		})
		slog.Info("Started downloading mail bodies", "characterID", characterID, "count", len(ids2))
		remaining := len(ids2)
		for _, mailID := range ids2 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				// needed for non-blocking
			}
			if _, err := s.updateMailBodyESI(ctx, characterID, mailID); err != nil {
				return nil, err
			}
			remaining--
			slog.Info("Mail body downloaded", "characterID", characterID, "mailID", mailID, "remaining", remaining)
		}
		slog.Info("Finished downloading mail bodies", "characterID", characterID, "count", len(ids2))
		return nil, nil
	})
	if err != nil {
		return false, err
	}
	return aborted, nil
}

// TODO: Refactor into one DB call

func (s *CharacterService) DownloadedBodiesPercentage(ctx context.Context, characterID int32) (total int, missing int, err error) {
	m2, err := s.st.ListCharacterMailsWithoutBody(ctx, characterID)
	if err != nil {
		return 0, 0, err
	}
	t2, err := s.st.ListCharacterMailIDs(ctx, characterID)
	if err != nil {
		return 0, 0, err
	}
	return t2.Size(), m2.Size(), nil
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

// updateMailLabelsESI updates the mail labels for a character from ESI
// and reports whether it has changed.
func (s *CharacterService) updateMailLabelsESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterMailLabels {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdMailLabels")
			ll, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailLabels(ctx, characterID, nil)
			if err != nil {
				return ll, err
			}
			slog.Debug("Received mail labels from ESI", "characterID", characterID, "count", len(ll.Labels))
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

// updateMailListsESI updates the mailing lists for a character from ESI
// and reports whether it has changed.
func (s *CharacterService) updateMailListsESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterMailLists {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdMailLists")
			lists, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailLists(ctx, characterID, nil)
			if err != nil {
				return nil, err
			}
			return lists, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			lists := data.([]esi.GetCharactersCharacterIdMailLists200Ok)
			for _, o := range lists {
				_, err := s.st.UpdateOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
					ID:       o.MailingListId,
					Name:     o.Name,
					Category: app.EveEntityMailList,
				})
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

// updateMailHeadersESI updates the mail headers for a character from ESI
// and reports whether they have changed.
func (s *CharacterService) updateMailHeadersESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterMailHeaders {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			headers, err := s.fetchMailHeadersESI(ctx, characterID, arg.MaxMails)
			if err != nil {
				return false, err
			}
			slog.Debug("Received mail headers from ESI", "characterID", characterID, "count", len(headers))
			return headers, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			headers := data.([]esi.GetCharactersCharacterIdMail200Ok)
			existingIDs, err := s.st.ListCharacterMailIDs(ctx, characterID)
			if err != nil {
				return err
			}
			newHeaders := make([]esi.GetCharactersCharacterIdMail200Ok, 0)
			existingHeaders := make([]esi.GetCharactersCharacterIdMail200Ok, 0)
			for _, h := range headers {
				if existingIDs.Contains(h.MailId) {
					existingHeaders = append(existingHeaders, h)
				} else {
					newHeaders = append(newHeaders, h)
				}
			}
			if len(newHeaders) > 0 {
				if err := s.addNewMailsESI(ctx, characterID, newHeaders); err != nil {
					return err
				}
			}
			if len(existingHeaders) > 0 {
				if err := s.updateExistingMail(ctx, characterID, existingHeaders); err != nil {
					return err
				}
			}
			// TODO: Delete obsolete mail labels and list
			// if err := s.st.DeleteObsoleteCharacterMailLabels(ctx, characterID); err != nil {
			// 	return err
			// }
			// if err := s.st.DeleteObsoleteCharacterMailLists(ctx, characterID); err != nil {
			// 	return err
			// }
			return nil
		})
}

// fetchMailHeadersESI fetches and returns a slice of mail headers for a character from ESI.
// The headers are guaranteed to be in descending order by mailID.
// It will at most return (maxMail + page size) headers.
func (s *CharacterService) fetchMailHeadersESI(ctx context.Context, characterID int32, maxMails int) ([]esi.GetCharactersCharacterIdMail200Ok, error) {
	const maxMailHeadersPerPage = 50 // maximum header objects returned per page
	mails := make([]esi.GetCharactersCharacterIdMail200Ok, 0)
	var lastMailID int32
	ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdMail")
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
		mails = slices.Concat(mails, oo)
		isLimitExceeded := (maxMails != 0 && len(mails)+maxMailHeadersPerPage > maxMails)
		if len(oo) < maxMailHeadersPerPage || isLimitExceeded {
			break
		}
		lastMailID = slices.Min(xslices.Map(oo, func(x esi.GetCharactersCharacterIdMail200Ok) int32 {
			return x.MailId
		}))
	}
	slices.SortFunc(mails, func(a, b esi.GetCharactersCharacterIdMail200Ok) int {
		return cmp.Compare(b.MailId, a.MailId) // descending order
	})
	slog.Debug("Received mail headers", "characterID", characterID, "count", len(mails))
	return mails, nil
}

func (s *CharacterService) addNewMailsESI(ctx context.Context, characterID int32, headers []esi.GetCharactersCharacterIdMail200Ok) error {
	var entityIDs set.Set[int32]
	for _, m := range headers {
		entityIDs.Add(m.From)
		for _, r := range m.Recipients {
			entityIDs.Add(r.RecipientId)
		}
	}
	_, err := s.eus.AddMissingEntities(ctx, entityIDs)
	if err != nil {
		return err
	}
	for _, h := range headers {
		recipientIDs := xslices.Map(h.Recipients, func(x esi.GetCharactersCharacterIdMailRecipient) int32 {
			return x.RecipientId
		})
		_, err := s.st.CreateCharacterMail(ctx, storage.CreateCharacterMailParams{
			CharacterID:  characterID,
			FromID:       h.From,
			IsRead:       h.IsRead,
			LabelIDs:     h.Labels,
			MailID:       h.MailId,
			RecipientIDs: recipientIDs,
			Subject:      h.Subject,
			Timestamp:    h.Timestamp,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *CharacterService) updateExistingMail(ctx context.Context, characterID int32, headers []esi.GetCharactersCharacterIdMail200Ok) error {
	var updated int
	for _, h := range headers {
		m, err := s.st.GetCharacterMail(ctx, characterID, h.MailId)
		if err != nil {
			return err
		}
		if m.IsRead != h.IsRead {
			err := s.st.UpdateCharacterMailSetIsRead(ctx, characterID, m.ID, h.IsRead)
			if err != nil {
				return err
			}
			updated++
		}
		if !set.Of(h.Labels...).Equal(set.Of(m.LabelIDs()...)) {
			err := s.st.UpdateCharacterMailSetLabels(ctx, characterID, m.ID, h.Labels)
			if err != nil {
				return err
			}
			updated++
		}
	}
	if updated > 0 {
		slog.Info("Updated mail", "characterID", characterID, "count", updated)
	}
	return nil
}
