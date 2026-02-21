package storage

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateCharacterMailParams struct {
	Body         optional.Optional[string]
	CharacterID  int64
	FromID       int64
	LabelIDs     []int64
	IsProcessed  bool
	IsRead       optional.Optional[bool]
	MailID       int64
	RecipientIDs []int64
	Subject      optional.Optional[string]
	Timestamp    time.Time
}

func (st *Storage) CreateCharacterMail(ctx context.Context, arg CreateCharacterMailParams) (int64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharacterMail: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.MailID == 0 || len(arg.RecipientIDs) == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	characterID2 := arg.CharacterID
	from, err := st.GetEveEntity(ctx, arg.FromID)
	if err != nil {
		return 0, wrapErr(err)
	}
	mail, err := st.qRW.CreateMail(ctx, queries.CreateMailParams{
		Body2:       optional.ToNullString(arg.Body),
		CharacterID: characterID2,
		FromID:      from.ID,
		MailID:      arg.MailID,
		Subject:     arg.Subject.ValueOrZero(),
		IsProcessed: arg.IsProcessed,
		IsRead:      arg.IsRead.ValueOrZero(),
		Timestamp:   arg.Timestamp,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	for _, id := range arg.RecipientIDs {
		err := st.qRW.CreateMailRecipient(ctx, queries.CreateMailRecipientParams{
			MailID:      mail.ID,
			EveEntityID: id,
		})
		if err != nil {
			return 0, wrapErr(fmt.Errorf("create mail recipient: %w", err))
		}
	}
	if err := st.UpdateCharacterMailSetLabels(ctx, arg.CharacterID, mail.ID, arg.LabelIDs); err != nil {
		return 0, wrapErr(err)
	}
	slog.Info("Created mail", "characterID", arg.CharacterID, "mailID", arg.MailID)
	return mail.ID, nil

}

func (st *Storage) GetCharacterMail(ctx context.Context, characterID, mailID int64) (*app.CharacterMail, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterMail: character %d, mail ID %d: %w", characterID, mailID, err)
	}
	if characterID == 0 || mailID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	mail, err := func() (*app.CharacterMail, error) {
		row, err := st.qRO.GetMail(ctx, queries.GetMailParams{
			CharacterID: characterID,
			MailID:      mailID,
		})
		if err != nil {
			return nil, convertGetError(err)
		}
		ll, err := st.qRO.GetCharacterMailLabels(ctx, row.CharacterMail.ID)
		if err != nil {
			return nil, convertGetError(err)
		}
		rr, err := st.qRO.GetMailRecipients(ctx, row.CharacterMail.ID)
		if err != nil {
			return nil, convertGetError(err)
		}
		mail := characterMailFromDBModel(row.CharacterMail, row.EveEntity, ll, rr)
		return mail, nil
	}()
	if err != nil {
		return nil, wrapErr(err)
	}
	return mail, nil
}

func (st *Storage) GetCharacterMailUnreadCount(ctx context.Context, id int64) (int, error) {
	count, err := st.qRO.GetMailUnreadCount(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("get mail unread count for character %d: %w", id, convertGetError(err))
	}
	return int(count), err
}

func (st *Storage) GetAllCharactersMailUnreadCount(ctx context.Context) (int, error) {
	count, err := st.qRO.GetAllMailUnreadCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("get all mail unread count: %w", err)
	}
	return int(count), err
}

func (st *Storage) GetCharacterMailCount(ctx context.Context, characterID int64) (int, error) {
	count, err := st.qRO.GetMailCount(ctx, characterID)
	if err != nil {
		return 0, fmt.Errorf("get mail count for character %d: %w", characterID, convertGetError(err))
	}
	return int(count), err
}

func (st *Storage) DeleteCharacterMail(ctx context.Context, characterID, mailID int64) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCharacterMail: character %d, mail ID %d: %w", characterID, mailID, err)
	}
	if characterID == 0 || mailID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.DeleteMail(ctx, queries.DeleteMailParams{
		CharacterID: characterID,
		MailID:      mailID,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetCharacterMailLabelUnreadCounts(ctx context.Context, characterID int64) (map[int64]int, error) {
	rows, err := st.qRO.GetCharacterMailLabelUnreadCounts(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("get mail label unread counts for character %d: %w", characterID, convertGetError(err))
	}
	result := make(map[int64]int)
	for _, r := range rows {
		result[int64(r.LabelID)] = int(r.UnreadCount2)
	}
	return result, nil
}

func (st *Storage) GetCharacterMailListUnreadCounts(ctx context.Context, characterID int64) (map[int64]int, error) {
	rows, err := st.qRO.GetCharacterMailListUnreadCounts(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("get mail list unread counts for character %d: %w", characterID, convertGetError(err))
	}
	result := make(map[int64]int)
	for _, r := range rows {
		result[int64(r.ListID)] = int(r.UnreadCount2)
	}
	return result, nil
}

func (st *Storage) ListCharacterMailIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListMailIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list mail IDs for character %d: %w", characterID, err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) ListCharacterMailsWithoutBody(ctx context.Context, characterID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListMailsWithoutBody(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list mail IDs for character %d: %w", characterID, err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) ListCharacterMailListsOrdered(ctx context.Context, characterID int64) ([]*app.EveEntity, error) {
	ll, err := st.qRO.ListCharacterMailListsOrdered(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("list mail lists for character %d: %w", characterID, err)
	}
	ee := make([]*app.EveEntity, len(ll))
	for i, l := range ll {
		ee[i] = eveEntityFromDBModel(l)
	}
	return ee, nil
}

func (st *Storage) UpdateCharacterMailSetBody(ctx context.Context, characterID int64, mailID int64, body optional.Optional[string]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterMailBody: %d %d: %w", characterID, mailID, err)
	}
	if characterID == 0 || mailID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateCharacterMailSetBody(ctx, queries.UpdateCharacterMailSetBodyParams{
		CharacterID: characterID,
		MailID:      mailID,
		Body2:       optional.ToNullString(body),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) UpdateCharacterMailSetIsRead(ctx context.Context, characterID int64, mailPK int64, isRead bool) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterMailIsRead: %d %d: %w", characterID, mailPK, err)
	}
	if characterID == 0 || mailPK == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateCharacterMailIsRead(ctx, queries.UpdateCharacterMailIsReadParams{
		ID:     mailPK,
		IsRead: isRead,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) UpdateCharacterMailSetLabels(ctx context.Context, characterID int64, mailPK int64, labelIDs []int64) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterMailLabels: %d %d: %w", characterID, mailPK, err)
	}
	if characterID == 0 || mailPK == 0 {
		return wrapErr(app.ErrInvalid)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	if err := qtx.DeleteMailCharacterMailLabels(ctx, mailPK); err != nil {
		return wrapErr(fmt.Errorf("delete mail labels: %w", err))
	}
	for _, l := range labelIDs {
		label, err := qtx.GetCharacterMailLabel(ctx, queries.GetCharacterMailLabelParams{
			CharacterID: characterID,
			LabelID:     l,
		})
		if err != nil {
			return wrapErr(fmt.Errorf("get mail label: %w", err))
		}
		if err := qtx.CreateMailCharacterMailLabel(ctx, queries.CreateMailCharacterMailLabelParams{
			CharacterMailID:      mailPK,
			CharacterMailLabelID: label.ID,
		}); err != nil {
			return wrapErr(fmt.Errorf("create mail label: %w", err))
		}
	}
	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	return nil
}

func characterMailFromDBModel(
	mail queries.CharacterMail,
	from queries.EveEntity,
	labels []queries.CharacterMailLabel,
	recipients []queries.EveEntity,
) *app.CharacterMail {
	if mail.CharacterID == 0 {
		panic("missing character ID")
	}
	var ll []*app.CharacterMailLabel
	for _, l := range labels {
		ll = append(ll, characterMailLabelFromDBModel(l))
	}
	var rr []*app.EveEntity
	for _, r := range recipients {
		rr = append(rr, eveEntityFromDBModel(r))
	}
	m := app.CharacterMail{
		Body:        optional.FromNullString(mail.Body2),
		CharacterID: mail.CharacterID,
		From:        eveEntityFromDBModel(from),
		IsProcessed: mail.IsProcessed,
		IsRead:      optional.FromZeroValue(mail.IsRead),
		ID:          mail.ID,
		Labels:      ll,
		MailID:      mail.MailID,
		Recipients:  rr,
		Subject:     optional.FromZeroValue(mail.Subject),
		Timestamp:   mail.Timestamp,
	}
	return &m
}

func (st *Storage) UpdateCharacterMailSetProcessed(ctx context.Context, id int64) error {
	if err := st.qRW.UpdateCharacterMailSetProcessed(ctx, id); err != nil {
		return fmt.Errorf("set mail PK %d as processed: %w", id, err)
	}
	return nil
}
