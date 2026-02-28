package characterservice

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/fnt-eve/goesi-openapi/esi"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xsingleflight"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// DeleteMail deletes a mail both on ESI and in the database.
func (s *CharacterService) DeleteMail(ctx context.Context, characterID, mailID int64) error {
	ts, err := s.TokenSource(ctx, characterID, app.SectionCharacterMailHeaders.Scopes())
	if err != nil {
		return err
	}
	ctx = xgoesi.NewContextWithAuth(ctx, characterID, ts)
	ctx = xgoesi.NewContextWithOperationID(ctx, "DeleteCharactersCharacterIdMailMailId")
	_, err = s.esiClient.MailAPI.DeleteCharactersCharacterIdMailMailId(ctx, characterID, mailID).Execute()
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

func (s *CharacterService) GetMail(ctx context.Context, characterID int64, mailID int64) (*app.CharacterMail, error) {
	return s.st.GetCharacterMail(ctx, characterID, mailID)
}

func (s *CharacterService) GetAllMailUnreadCount(ctx context.Context) (int, error) {
	return s.st.GetAllCharactersMailUnreadCount(ctx)
}

// GetMailCounts returns the number of unread mail for a character.
func (s *CharacterService) GetMailCounts(ctx context.Context, characterID int64) (int, int, error) {
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

func (s *CharacterService) GetMailLabelUnreadCounts(ctx context.Context, characterID int64) (map[int64]int, error) {
	return s.st.GetCharacterMailLabelUnreadCounts(ctx, characterID)
}

func (s *CharacterService) GetMailListUnreadCounts(ctx context.Context, characterID int64) (map[int64]int, error) {
	return s.st.GetCharacterMailListUnreadCounts(ctx, characterID)
}

func (s *CharacterService) NotifyMails(ctx context.Context, characterID int64, earliest time.Time, notify func(title, content string)) error {
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

func (s *CharacterService) ListMailLists(ctx context.Context, characterID int64) ([]*app.EveEntity, error) {
	return s.st.ListCharacterMailListsOrdered(ctx, characterID)
}

// ListMailHeadersForLabelOrdered returns a character's mails for a label in descending order by timestamp.
func (s *CharacterService) ListMailHeadersForLabelOrdered(ctx context.Context, characterID int64, labelID int64) ([]*app.CharacterMailHeader, error) {
	return s.st.ListCharacterMailHeadersForLabelOrdered(ctx, characterID, labelID)
}

func (s *CharacterService) ListMailHeadersForListOrdered(ctx context.Context, characterID int64, listID int64) ([]*app.CharacterMailHeader, error) {
	return s.st.ListCharacterMailHeadersForListOrdered(ctx, characterID, listID)
}

func (s *CharacterService) ListMailLabelsOrdered(ctx context.Context, characterID int64) ([]*app.CharacterMailLabel, error) {
	return s.st.ListCharacterMailLabelsOrdered(ctx, characterID)
}

// SendMail creates a new mail on ESI and stores it locally.
func (s *CharacterService) SendMail(ctx context.Context, characterID int64, subject string, recipients []*app.EveEntity, body string) (int64, error) {
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
	ts, err := s.TokenSource(ctx, characterID, app.SectionCharacterMailHeaders.Scopes())
	if err != nil {
		return 0, err
	}
	ctx = xgoesi.NewContextWithAuth(ctx, characterID, ts)
	ctx = xgoesi.NewContextWithOperationID(ctx, "PostCharactersCharacterIdMail")
	request := esi.PostCharactersCharacterIdMailRequest{
		Body:       body,
		Subject:    subject,
		Recipients: rr,
	}
	mailID, _, err := s.esiClient.MailAPI.PostCharactersCharacterIdMail(ctx, characterID).PostCharactersCharacterIdMailRequest(request).Execute()
	if err != nil {
		return 0, err
	}
	recipientIDs := make([]int64, len(rr))
	for i, r := range rr {
		recipientIDs[i] = r.RecipientId
	}
	ids := set.Union(set.Of(recipientIDs...), set.Of(characterID))
	_, err = s.eus.AddMissingEntities(ctx, ids)
	if err != nil {
		return 0, err
	}
	_, err = s.st.GetOrCreateCharacterMailLabel(ctx, storage.MailLabelParams{
		CharacterID: characterID,
		LabelID:     app.MailLabelSent,
		Name:        optional.New("Sent"),
	}) // make sure sent label exists
	if err != nil {
		return 0, err
	}
	_, err = s.st.CreateCharacterMail(ctx, storage.CreateCharacterMailParams{
		Body:         optional.New(body),
		CharacterID:  characterID,
		FromID:       characterID,
		IsRead:       optional.New(true),
		LabelIDs:     []int64{app.MailLabelSent},
		MailID:       mailID,
		RecipientIDs: recipientIDs,
		Subject:      optional.New(subject),
		Timestamp:    time.Now(),
	})
	if err != nil {
		return 0, err
	}
	slog.Info("Mail sent", "characterID", characterID, "mailID", mailID)
	return mailID, nil
}

// UpdateMailRead updates an existing mail as read
func (s *CharacterService) UpdateMailRead(ctx context.Context, characterID, mailID int64, isRead bool) error {
	_, err, _ := s.sfg.Do(fmt.Sprintf("UpdateMailRead-%d-%d", characterID, mailID), func() (any, error) {
		ts, err := s.TokenSource(ctx, characterID, app.SectionCharacterMailHeaders.Scopes())
		if err != nil {
			return nil, err
		}
		ctx = xgoesi.NewContextWithAuth(ctx, characterID, ts)
		ctx = xgoesi.NewContextWithOperationID(ctx, "PutCharactersCharacterIdMailMailId")
		m, err := s.st.GetCharacterMail(ctx, characterID, mailID)
		if err != nil {
			return nil, err
		}
		req := esi.PutCharactersCharacterIdMailMailIdRequest{
			Labels: m.LabelIDs(),
		}
		if isRead {
			req.Read = &isRead
		}
		_, err = s.esiClient.MailAPI.PutCharactersCharacterIdMailMailId(ctx, m.CharacterID, m.MailID).PutCharactersCharacterIdMailMailIdRequest(req).Execute()
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
func (s *CharacterService) UpdateMailBodyESI(ctx context.Context, characterID int64, mailID int64) (string, error) {
	b, err := s.updateMailBodyESI(ctx, characterID, mailID)
	if err != nil {
		return "", err
	}
	slog.Info("Mail body updated", "characterID", characterID, "mailID", mailID)
	return b, err
}

func (s *CharacterService) updateMailBodyESI(ctx context.Context, characterID int64, mailID int64) (string, error) {
	v, err, _ := xsingleflight.Do(&s.sfg, fmt.Sprintf("UpdateMailBodyESI-%d-%d", characterID, mailID), func() (string, error) {
		ts, err := s.TokenSource(ctx, characterID, app.SectionCharacterMailHeaders.Scopes())
		if err != nil {
			return "", err
		}
		ctx = xgoesi.NewContextWithAuth(ctx, characterID, ts)
		ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdMailMailId")
		mail, _, err := s.esiClient.MailAPI.GetCharactersCharacterIdMailMailId(ctx, characterID, mailID).Execute()
		if err != nil {
			return "", err
		}
		body := optional.FromPtr(mail.Body)
		err = s.st.UpdateCharacterMailSetBody(ctx, characterID, mailID, body)
		if err != nil {
			return "", err
		}
		return body.ValueOrZero(), nil
	})
	if err != nil {
		return "", err
	}
	return v, nil
}

// DownloadMissingMailBodies downloads missing mail bodies for a character
// and reports whether the function was aborted.
// Only one instance per character will run at a time and additional calls will be aborted.
func (s *CharacterService) DownloadMissingMailBodies(ctx context.Context, characterID int64) (bool, error) {
	_, err, aborted := s.sig.Do(fmt.Sprintf("DownloadMissingMailBodies-%d", characterID), func() (any, error) {
		ids, err := s.st.ListCharacterMailsWithoutBody(ctx, characterID)
		if err != nil {
			return nil, err
		}
		if ids.Size() == 0 {
			return nil, nil
		}
		ids2 := slices.SortedFunc(ids.All(), func(a, b int64) int {
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

func (s *CharacterService) DownloadedBodiesPercentage(ctx context.Context, characterID int64) (total int, missing int, err error) {
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

func eveEntitiesToESIMailRecipients(ee []*app.EveEntity) ([]esi.PostCharactersCharacterIdMailRequestRecipientsInner, error) {
	rr := make([]esi.PostCharactersCharacterIdMailRequestRecipientsInner, len(ee))
	for i, e := range ee {
		c, ok := eveEntityCategory2MailRecipientType[e.Category]
		if !ok {
			return rr, fmt.Errorf("match EveEntity category to ESI mail recipient type: %v", e)
		}
		rr[i] = esi.PostCharactersCharacterIdMailRequestRecipientsInner{
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
		ctx, arg, false,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdMailLabels")
			ll, _, err := s.esiClient.MailAPI.GetCharactersCharacterIdMailLabels(ctx, characterID).Execute()
			if err != nil {
				return ll, err
			}
			slog.Debug("Received mail labels from ESI", "characterID", characterID, "count", len(ll.Labels))
			return ll, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			ll := data.(*esi.CharactersCharacterIdMailLabelsGet)
			labels := ll.Labels
			for _, o := range labels {
				if o.LabelId == nil {
					continue
				}
				_, err := s.st.UpdateOrCreateCharacterMailLabel(ctx, storage.MailLabelParams{
					CharacterID: characterID,
					Color:       optional.FromPtr(o.Color),
					LabelID:     *o.LabelId,
					Name:        optional.FromPtr(o.Name),
					UnreadCount: optional.FromPtr(o.UnreadCount),
				})
				if err != nil {
					return false, err
				}
			}
			return true, nil
		})
}

// updateMailListsESI updates the mailing lists for a character from ESI
// and reports whether it has changed.
func (s *CharacterService) updateMailListsESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterMailLists {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, false,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdMailLists")
			lists, _, err := s.esiClient.MailAPI.GetCharactersCharacterIdMailLists(ctx, characterID).Execute()
			if err != nil {
				return nil, err
			}
			return lists, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			lists := data.([]esi.CharactersCharacterIdMailListsGetInner)
			for _, o := range lists {
				_, err := s.st.UpdateOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
					ID:       o.MailingListId,
					Name:     o.Name,
					Category: app.EveEntityMailList,
				})
				if err != nil {
					return false, err
				}
				if err := s.st.CreateCharacterMailList(ctx, characterID, o.MailingListId); err != nil {
					return false, err
				}
			}
			return true, nil
		})
}

// updateMailHeadersESI updates the mail headers for a character from ESI
// and reports whether they have changed.
func (s *CharacterService) updateMailHeadersESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterMailHeaders {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, false,
		func(ctx context.Context, characterID int64) (any, error) {
			mail, err := s.fetchMailHeadersESI(ctx, characterID, arg.MaxMails)
			if err != nil {
				return false, err
			}
			slog.Debug("Received mail headers from ESI", "characterID", characterID, "count", len(mail))
			return mail, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			mail := data.([]storage.CreateCharacterMailParams)
			existingIDs, err := s.st.ListCharacterMailIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			newMail := make([]storage.CreateCharacterMailParams, 0)
			existingMail := make([]storage.CreateCharacterMailParams, 0)
			for _, m := range mail {
				if existingIDs.Contains(m.MailID) {
					existingMail = append(existingMail, m)
				} else {
					newMail = append(newMail, m)
				}
			}
			if len(newMail) > 0 {
				if err := s.addNewMailsESI(ctx, newMail); err != nil {
					return false, err
				}
			}
			if len(existingMail) > 0 {
				if err := s.updateExistingMail(ctx, characterID, existingMail); err != nil {
					return false, err
				}
			}
			// TODO: Delete obsolete mail labels and list
			// if err := s.st.DeleteObsoleteCharacterMailLabels(ctx, characterID); err != nil {
			// 	return err
			// }
			// if err := s.st.DeleteObsoleteCharacterMailLists(ctx, characterID); err != nil {
			// 	return err
			// }
			return false, nil
		})
}

// fetchMailHeadersESI fetches and returns a slice of mail headers for a character from ESI.
// The headers are guaranteed to be in descending order by mailID.
// It will at most return (maxMail + page size) headers.
func (s *CharacterService) fetchMailHeadersESI(ctx context.Context, characterID int64, maxMails int) ([]storage.CreateCharacterMailParams, error) {
	const maxMailHeadersPerPage = 50 // maximum header objects returned per page
	headers := make([]esi.CharactersCharacterIdMailGetInner, 0)
	var lastMailID int64
	ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdMail")
	for {
		var oo []esi.CharactersCharacterIdMailGetInner
		var err error
		if lastMailID > 0 {
			oo, _, err = s.esiClient.MailAPI.GetCharactersCharacterIdMail(ctx, characterID).LastMailId(lastMailID).Execute()
		} else {
			oo, _, err = s.esiClient.MailAPI.GetCharactersCharacterIdMail(ctx, characterID).Execute()
		}
		if err != nil {
			return nil, err
		}
		headers = slices.Concat(headers, oo)
		isLimitExceeded := (maxMails != 0 && len(headers)+maxMailHeadersPerPage > maxMails)
		if len(oo) < maxMailHeadersPerPage || isLimitExceeded {
			break
		}
		lastMailID = slices.Min(xslices.Map(oo, func(x esi.CharactersCharacterIdMailGetInner) int64 {
			if x.MailId == nil {
				return 0
			}
			return *x.MailId
		}))
	}
	mail := make([]storage.CreateCharacterMailParams, 0)
	for _, h := range headers {
		if h.MailId == nil || h.From == nil || h.Timestamp == nil {
			continue // ignore mails which are missing important fields
		}
		recipientIDs := xslices.Map(h.Recipients, func(x esi.PostCharactersCharacterIdMailRequestRecipientsInner) int64 {
			return x.RecipientId
		})
		mail = append(mail, storage.CreateCharacterMailParams{
			CharacterID:  characterID,
			FromID:       *h.From,
			IsRead:       optional.FromPtr(h.IsRead),
			LabelIDs:     h.Labels,
			MailID:       *h.MailId,
			RecipientIDs: recipientIDs,
			Subject:      optional.FromPtr(h.Subject),
			Timestamp:    *h.Timestamp,
		})
	}
	slices.SortFunc(mail, func(a, b storage.CreateCharacterMailParams) int {
		return cmp.Compare(b.MailID, a.MailID) // descending order
	})
	slog.Debug("Received mail headers", "characterID", characterID, "count", len(mail))
	return mail, nil
}

func (s *CharacterService) addNewMailsESI(ctx context.Context, mail []storage.CreateCharacterMailParams) error {
	var entityIDs set.Set[int64]
	for _, m := range mail {
		entityIDs.Add(m.FromID)
		entityIDs.Add(m.RecipientIDs...)
	}
	_, err := s.eus.AddMissingEntities(ctx, entityIDs)
	if err != nil {
		return err
	}
	for _, m := range mail {
		_, err := s.st.CreateCharacterMail(ctx, m)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *CharacterService) updateExistingMail(ctx context.Context, characterID int64, mail []storage.CreateCharacterMailParams) error {
	var updated int
	for _, m1 := range mail {
		m2, err := s.st.GetCharacterMail(ctx, m1.CharacterID, m1.MailID)
		if err != nil {
			return err
		}
		if m2.IsRead != m1.IsRead {
			err := s.st.UpdateCharacterMailSetIsRead(ctx, m2.CharacterID, m2.ID, m1.IsRead.ValueOrZero())
			if err != nil {
				return err
			}
			updated++
		}
		if !set.Of(m1.LabelIDs...).Equal(set.Of(m2.LabelIDs()...)) {
			err := s.st.UpdateCharacterMailSetLabels(ctx, m2.CharacterID, m2.ID, m1.LabelIDs)
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
