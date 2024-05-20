package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	islices "github.com/ErikKalkoken/evebuddy/internal/helper/slices"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateCharacterMailParams struct {
	Body         string
	CharacterID  int32
	FromID       int32
	LabelIDs     []int32
	IsRead       bool
	MailID       int32
	RecipientIDs []int32
	Subject      string
	Timestamp    time.Time
}

func (r *Storage) CreateCharacterMail(ctx context.Context, arg CreateCharacterMailParams) (int64, error) {
	id, err := func() (int64, error) {
		if len(arg.RecipientIDs) == 0 {
			return 0, errors.New("can not create mail without recipients")
		}
		characterID2 := int64(arg.CharacterID)
		from, err := r.GetEveEntity(ctx, arg.FromID)
		if err != nil {
			return 0, err
		}
		mailParams := queries.CreateMailParams{
			Body:        arg.Body,
			CharacterID: characterID2,
			FromID:      int64(from.ID),
			MailID:      int64(arg.MailID),
			Subject:     arg.Subject,
			IsRead:      arg.IsRead,
			Timestamp:   arg.Timestamp,
		}
		mail, err := r.q.CreateMail(ctx, mailParams)
		if err != nil {
			return 0, err
		}
		for _, id := range arg.RecipientIDs {
			arg := queries.CreateMailRecipientParams{MailID: mail.ID, EveEntityID: int64(id)}
			err := r.q.CreateMailRecipient(ctx, arg)
			if err != nil {
				return 0, err
			}
		}
		if err := r.updateCharacterMailLabels(ctx, arg.CharacterID, mail.ID, arg.LabelIDs); err != nil {
			return 0, err
		}
		slog.Info("Created new mail", "characterID", arg.CharacterID, "mailID", arg.MailID)
		return mail.ID, nil
	}()
	if err != nil {
		return 0, fmt.Errorf("failed to create mail for character %d and mail ID %d: %w", arg.CharacterID, arg.MailID, err)
	}
	return id, err
}

func (r *Storage) updateCharacterMailLabels(ctx context.Context, characterID int32, mailPK int64, labelIDs []int32) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := r.q.WithTx(tx)
	if err := qtx.DeleteMailCharacterMailLabels(ctx, mailPK); err != nil {
		return err
	}
	for _, l := range labelIDs {
		arg := queries.GetCharacterMailLabelParams{
			CharacterID: int64(characterID),
			LabelID:     int64(l),
		}
		label, err := qtx.GetCharacterMailLabel(ctx, arg)
		if err != nil {
			return err
		}
		mailMailLabelParams := queries.CreateMailCharacterMailLabelParams{
			CharacterMailID: mailPK, CharacterMailLabelID: label.ID,
		}
		if err := qtx.CreateMailCharacterMailLabel(ctx, mailMailLabelParams); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *Storage) GetCharacterMail(ctx context.Context, characterID, mailID int32) (*model.CharacterMail, error) {
	mail, err := func() (*model.CharacterMail, error) {
		arg := queries.GetMailParams{
			CharacterID: int64(characterID),
			MailID:      int64(mailID),
		}
		row, err := r.q.GetMail(ctx, arg)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				err = ErrNotFound
			}
			return nil, err
		}
		ll, err := r.q.GetCharacterMailLabels(ctx, row.CharacterMail.ID)
		if err != nil {
			return nil, err
		}
		rr, err := r.q.GetMailRecipients(ctx, row.CharacterMail.ID)
		if err != nil {
			return nil, err
		}
		mail := characterMailFromDBModel(row.CharacterMail, row.EveEntity, ll, rr)
		return mail, nil
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to get mail for character %d with mail ID %d: %w", characterID, mailID, err)
	}
	return mail, nil
}

func (r *Storage) GetCharacterMailUnreadCount(ctx context.Context, characterID int32) (int, error) {
	count, err := r.q.GetMailUnreadCount(ctx, int64(characterID))
	return int(count), err
}

func (r *Storage) GetCharacterMailCount(ctx context.Context, characterID int32) (int, error) {
	count, err := r.q.GetMailCount(ctx, int64(characterID))
	return int(count), err
}

func (r *Storage) DeleteCharacterMail(ctx context.Context, characterID, mailID int32) error {
	arg := queries.DeleteMailParams{
		CharacterID: int64(characterID),
		MailID:      int64(mailID),
	}
	err := r.q.DeleteMail(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to delete mail for character %d with ID%d: %w", characterID, mailID, err)
	}
	return nil
}

func (r *Storage) ListCharacterMailIDs(ctx context.Context, characterID int32) ([]int32, error) {
	ids, err := r.q.ListMailIDs(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("failed to list mail IDs for character %d: %w", characterID, err)
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}

func (r *Storage) GetCharacterMailLabelUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error) {
	rows, err := r.q.GetCharacterMailLabelUnreadCounts(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("failed to get mail label unread counts for character %d: %w", characterID, err)
	}
	result := make(map[int32]int)
	for _, r := range rows {
		result[int32(r.LabelID)] = int(r.UnreadCount2)
	}
	return result, nil
}

func (r *Storage) GetCharacterMailListUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error) {
	rows, err := r.q.GetCharacterMailListUnreadCounts(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("failed to get mail list unread counts for character %d: %w", characterID, err)
	}
	result := make(map[int32]int)
	for _, r := range rows {
		result[int32(r.ListID)] = int(r.UnreadCount2)
	}
	return result, nil
}

func (r *Storage) ListCharacterMailListsOrdered(ctx context.Context, characterID int32) ([]*model.EveEntity, error) {
	ll, err := r.q.ListCharacterMailListsOrdered(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("failed to list mail lists for character %d: %w", characterID, err)
	}
	ee := make([]*model.EveEntity, len(ll))
	for i, l := range ll {
		ee[i] = eveEntityFromDBModel(l)
	}
	return ee, nil
}

// ListMailsForLabel returns a character's mails for a label in descending order by timestamp.
// Return mails for all labels, when labelID = 0
func (r *Storage) ListCharacterMailIDsForLabelOrdered(ctx context.Context, characterID int32, labelID int32) ([]int32, error) {
	switch labelID {
	case model.MailLabelAll:
		ids, err := r.q.ListMailIDsOrdered(ctx, int64(characterID))
		if err != nil {
			return nil, fmt.Errorf("failed to list mail IDs for character %d and label %d: %w", characterID, labelID, err)
		}
		ids2 := islices.ConvertNumeric[int64, int32](ids)
		return ids2, nil
	case model.MailLabelNone:
		ids, err := r.q.ListMailIDsNoLabelOrdered(ctx, int64(characterID))
		if err != nil {
			return nil, err
		}
		ids2 := islices.ConvertNumeric[int64, int32](ids)
		return ids2, nil
	default:
		arg := queries.ListMailIDsForLabelOrderedParams{
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

func (r *Storage) ListCharacterMailIDsForListOrdered(ctx context.Context, characterID int32, listID int32) ([]int32, error) {
	arg := queries.ListMailIDsForListOrderedParams{
		CharacterID: int64(characterID),
		EveEntityID: int64(listID),
	}
	ids, err := r.q.ListMailIDsForListOrdered(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list mail IDs for character %d and list %d: %w", characterID, listID, err)
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}

func (r *Storage) UpdateCharacterMail(ctx context.Context, characterID int32, mailPK int64, isRead bool, labelIDs []int32) error {
	arg := queries.UpdateMailParams{
		ID:     mailPK,
		IsRead: isRead,
	}
	if err := r.q.UpdateMail(ctx, arg); err != nil {
		return fmt.Errorf("failed to update mail PK %d for character %d: %w", mailPK, characterID, err)
	}
	if err := r.updateCharacterMailLabels(ctx, characterID, mailPK, labelIDs); err != nil {
		return fmt.Errorf("failed to update labels for mail PK %d and character %d: %w", mailPK, characterID, err)
	}
	return nil
}

func characterMailFromDBModel(
	mail queries.CharacterMail,
	from queries.EveEntity,
	labels []queries.CharacterMailLabel,
	recipients []queries.EveEntity,
) *model.CharacterMail {
	if mail.CharacterID == 0 {
		panic("missing character ID")
	}
	var ll []*model.CharacterMailLabel
	for _, l := range labels {
		ll = append(ll, characterMailLabelFromDBModel(l))
	}
	var rr []*model.EveEntity
	for _, r := range recipients {
		rr = append(rr, eveEntityFromDBModel(r))
	}
	m := model.CharacterMail{
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
	return &m
}
