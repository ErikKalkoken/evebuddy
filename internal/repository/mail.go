package repository

import (
	"context"
	"database/sql"
	"errors"
	islices "example/evebuddy/internal/helper/slices"
	"example/evebuddy/internal/sqlc"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
)

var bodyPolicy = bluemonday.StrictPolicy()

// An Eve mail belonging to a character
type Mail struct {
	Body        string
	CharacterID int32
	From        EveEntity
	Labels      []MailLabel
	IsRead      bool
	ID          int64
	MailID      int32
	Recipients  []EveEntity
	Subject     string
	Timestamp   time.Time
}

func mailFromDBModel(mail sqlc.Mail, from sqlc.EveEntity, labels []sqlc.MailLabel, recipients []sqlc.EveEntity) Mail {
	if mail.CharacterID == 0 {
		panic("missing character ID")
	}
	var ll []MailLabel
	for _, l := range labels {
		ll = append(ll, mailLabelFromDBModel(l))
	}
	var rr []EveEntity
	for _, r := range recipients {
		rr = append(rr, eveEntityFromDBModel(r))
	}
	m := Mail{
		Body:        mail.Body,
		CharacterID: int32(mail.CharacterID),
		From:        eveEntityFromDBModel(from),
		IsRead:      mail.IsRead,
		ID:          mail.ID,
		Labels:      ll,
		MailID:      int32(mail.MailID),
		Recipients:  rr,
		Subject:     mail.Subject,
		Timestamp:   mail.Timestamp,
	}
	return m
}

// BodyPlain returns a mail's body as plain text.
func (m *Mail) BodyPlain() string {
	t := strings.ReplaceAll(m.Body, "<br>", "\n")
	b := html.UnescapeString(bodyPolicy.Sanitize(t))
	return b
}

// BodyForward returns a mail's body for a mail forward or reply
func (m *Mail) ToString(format string) string {
	s := "\n---\n"
	s += m.MakeHeaderText(format)
	s += "\n\n"
	s += m.BodyPlain()
	return s
}

func (m *Mail) MakeHeaderText(format string) string {
	var names []string
	for _, n := range m.Recipients {
		names = append(names, n.Name)
	}
	header := fmt.Sprintf(
		"From: %s\nSent: %s\nTo: %s",
		m.From.Name,
		m.Timestamp.Format(format),
		strings.Join(names, ", "),
	)
	return header
}

func (m *Mail) RecipientNames() []string {
	ss := make([]string, len(m.Recipients))
	for i, r := range m.Recipients {
		ss[i] = r.Name
	}
	return ss
}

func (r *Repository) CreateMail(ctx context.Context, characterID, mailID, fromID int32, subject string, recipientIDs []int32, body string, labelIDs []int32) (int64, error) {
	characterID2 := int64(characterID)
	from, err := r.GetEveEntity(ctx, fromID)
	if err != nil {
		return 0, err
	}
	mailParams := sqlc.CreateMailParams{
		Body:        body,
		CharacterID: characterID2,
		FromID:      int64(from.ID),
		MailID:      int64(mailID),
		Subject:     subject,
		IsRead:      true,
		Timestamp:   time.Now(),
	}
	mail, err := r.q.CreateMail(ctx, mailParams)
	if err != nil {
		return 0, err
	}
	for _, id := range recipientIDs {
		arg := sqlc.CreateMailRecipientParams{MailID: mail.ID, EveEntityID: int64(id)}
		err := r.q.CreateMailRecipient(ctx, arg)
		if err != nil {
			return 0, err
		}
	}
	if len(labelIDs) > 0 {
		// FIXME: Ensure this still works when no labels have yet been loaded from ESI
		mailLabelParams := sqlc.GetMailLabelParams{CharacterID: characterID2, LabelID: LabelSent}
		label, err := r.q.GetMailLabel(ctx, mailLabelParams)
		if err != nil {
			return 0, err
		}
		mailMailLabelParams := sqlc.CreateMailMailLabelParams{
			MailID: mail.ID, MailLabelID: label.ID,
		}
		if err := r.q.CreateMailMailLabel(ctx, mailMailLabelParams); err != nil {
			return 0, err
		}
	}
	slog.Info("Created new mail", "characterID", characterID, "mailID", mailID)
	return mail.ID, nil
}

// from, err := s.r.GetEveEntity(ctx, header.From)
// 	if err != nil {
// 		slog.Error("Failed to parse \"from\" in mail", "header", header, "error", err)
// 		return
// 	}
// 	mailArg := sqlc.CreateMailParams{
// 		Body:        m.Body,
// 		CharacterID: int64(token.CharacterID),
// 		FromID:      from.ID,
// 		MailID:      int64(header.MailId),
// 		Subject:     header.Subject,
// 		IsRead:      header.IsRead,
// 		Timestamp:   header.Timestamp,
// 	}
// 	mail, err := s.r.CreateMail(ctx, mailArg)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return
// 	}
// 	if err := s.fetchAndStoreMailRecipients(ctx, mail.ID, header); err != nil {
// 		slog.Error(err.Error())
// 		return
// 	}

// 	if err := s.r.AddMailLabelsToMail(ctx, mail.ID, m.Labels); err != nil {
// 		slog.Error(err.Error())
// 		return
// 	}

func (r *Repository) GetMail(ctx context.Context, characterID, mailID int32) (Mail, error) {
	arg := sqlc.GetMailParams{
		CharacterID: int64(characterID),
		MailID:      int64(mailID),
	}
	row, err := r.q.GetMail(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Mail{}, ErrNotFound
		}
		return Mail{}, err
	}
	ll, err := r.q.GetMailLabels(ctx, row.Mail.ID)
	if err != nil {
		return Mail{}, err
	}
	rr, err := r.q.GetMailRecipients(ctx, row.Mail.ID)
	if err != nil {
		return Mail{}, err
	}
	mail := mailFromDBModel(row.Mail, row.EveEntity, ll, rr)
	return mail, nil
}

func (r *Repository) DeleteMail(ctx context.Context, mailID int64) error {
	return r.q.DeleteMail(ctx, mailID)
}

func (r *Repository) ListMailIDs(ctx context.Context, characterID int32) ([]int32, error) {
	ids, err := r.q.ListMailIDs(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}

func (r *Repository) GetMailLabelUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error) {
	rows, err := r.q.GetMailLabelUnreadCounts(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	result := make(map[int32]int)
	for _, r := range rows {
		result[int32(r.LabelID)] = int(r.UnreadCount2)
	}
	return result, nil
}

func (r *Repository) GetMailListUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error) {
	rows, err := r.q.GetMailListUnreadCounts(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	result := make(map[int32]int)
	for _, r := range rows {
		result[int32(r.ListID)] = int(r.UnreadCount2)
	}
	return result, nil
}

func (r *Repository) ListMailLists(ctx context.Context, characterID int32) ([]EveEntity, error) {
	ll, err := r.q.ListMailLists(ctx, int64(characterID))
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
func (r *Repository) ListMailIDsForLabelOrdered(ctx context.Context, characterID int32, labelID int32) ([]int32, error) {
	switch labelID {
	case LabelAll:
		ids, err := r.q.ListMailIDsOrdered(ctx, int64(characterID))
		if err != nil {
			return nil, err
		}
		ids2 := islices.ConvertNumeric[int64, int32](ids)
		return ids2, nil
	case LabelNone:
		ids, err := r.q.ListMailIDsNoLabelOrdered(ctx, int64(characterID))
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
		ids, err := r.q.ListMailIDsForLabelOrdered(ctx, arg)
		if err != nil {
			return nil, err
		}
		ids2 := islices.ConvertNumeric[int64, int32](ids)
		return ids2, nil
	}
}

func (r *Repository) ListMailIDsForListOrdered(ctx context.Context, characterID int32, listID int32) ([]int32, error) {
	arg := sqlc.ListMailIDsForListOrderedParams{
		CharacterID: int64(characterID),
		EveEntityID: int64(listID),
	}
	ids, err := r.q.ListMailIDsForListOrdered(ctx, arg)
	if err != nil {
		return nil, err
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}

func (r *Repository) AddMailLabelsToMail(ctx context.Context, characterID int32, mailID int64, labelIDs []int32) error {
	labelIDs2 := islices.ConvertNumeric[int32, int64](labelIDs)
	labelArg := sqlc.ListMailLabelsByIDsParams{
		CharacterID: int64(characterID),
		Ids:         labelIDs2,
	}
	labels, err := r.q.ListMailLabelsByIDs(ctx, labelArg)
	if err != nil {
		return fmt.Errorf("failed to resolve mail labels %v: %s", labelIDs, err)
	}
	for _, label := range labels {
		arg := sqlc.CreateMailMailLabelParams{
			MailLabelID: label.ID, MailID: mailID,
		}
		if err := r.q.CreateMailMailLabel(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) UpdateMailSetRead(ctx context.Context, id int64) error {
	return r.q.UpdateMailSetRead(ctx, id)
}