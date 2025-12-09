package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type CreateCharacterMailParams struct {
	Body         string
	CharacterID  int32
	FromID       int32
	LabelIDs     []int32
	IsProcessed  bool
	IsRead       bool
	MailID       int32
	RecipientIDs []int32
	Subject      string
	Timestamp    time.Time
}

func (st *Storage) CreateCharacterMail(ctx context.Context, arg CreateCharacterMailParams) (int64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharacterMail: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.MailID == 0 || len(arg.RecipientIDs) == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	characterID2 := int64(arg.CharacterID)
	from, err := st.GetEveEntity(ctx, arg.FromID)
	if err != nil {
		return 0, wrapErr(err)
	}
	mailParams := queries.CreateMailParams{
		Body:        arg.Body,
		CharacterID: characterID2,
		FromID:      int64(from.ID),
		MailID:      int64(arg.MailID),
		Subject:     arg.Subject,
		IsProcessed: arg.IsProcessed,
		IsRead:      arg.IsRead,
		Timestamp:   arg.Timestamp,
	}
	mail, err := st.qRW.CreateMail(ctx, mailParams)
	if err != nil {
		return 0, wrapErr(err)
	}
	for _, id := range arg.RecipientIDs {
		arg := queries.CreateMailRecipientParams{MailID: mail.ID, EveEntityID: int64(id)}
		err := st.qRW.CreateMailRecipient(ctx, arg)
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

func (st *Storage) GetCharacterMail(ctx context.Context, characterID, mailID int32) (*app.CharacterMail, error) {
	mail, err := func() (*app.CharacterMail, error) {
		arg := queries.GetMailParams{
			CharacterID: int64(characterID),
			MailID:      int64(mailID),
		}
		row, err := st.qRO.GetMail(ctx, arg)
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
		return nil, fmt.Errorf("get mail for character %d with mail ID %d: %w", characterID, mailID, err)
	}
	return mail, nil
}

func (st *Storage) GetCharacterMailUnreadCount(ctx context.Context, id int32) (int, error) {
	count, err := st.qRO.GetMailUnreadCount(ctx, int64(id))
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

func (st *Storage) GetCharacterMailCount(ctx context.Context, characterID int32) (int, error) {
	count, err := st.qRO.GetMailCount(ctx, int64(characterID))
	if err != nil {
		return 0, fmt.Errorf("get mail count for character %d: %w", characterID, convertGetError(err))
	}
	return int(count), err
}

func (st *Storage) DeleteCharacterMail(ctx context.Context, characterID, mailID int32) error {
	arg := queries.DeleteMailParams{
		CharacterID: int64(characterID),
		MailID:      int64(mailID),
	}
	err := st.qRW.DeleteMail(ctx, arg)
	if err != nil {
		return fmt.Errorf("delete mail for character %d with ID%d: %w", characterID, mailID, err)
	}
	return nil
}

func (st *Storage) GetCharacterMailLabelUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error) {
	rows, err := st.qRO.GetCharacterMailLabelUnreadCounts(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("get mail label unread counts for character %d: %w", characterID, convertGetError(err))
	}
	result := make(map[int32]int)
	for _, r := range rows {
		result[int32(r.LabelID)] = int(r.UnreadCount2)
	}
	return result, nil
}

func (st *Storage) GetCharacterMailListUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error) {
	rows, err := st.qRO.GetCharacterMailListUnreadCounts(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("get mail list unread counts for character %d: %w", characterID, convertGetError(err))
	}
	result := make(map[int32]int)
	for _, r := range rows {
		result[int32(r.ListID)] = int(r.UnreadCount2)
	}
	return result, nil
}

func (st *Storage) ListCharacterMailIDs(ctx context.Context, characterID int32) (set.Set[int32], error) {
	ids, err := st.qRO.ListMailIDs(ctx, int64(characterID))
	if err != nil {
		return set.Set[int32]{}, fmt.Errorf("list mail IDs for character %d: %w", characterID, err)
	}
	return set.Of(convertNumericSlice[int32](ids)...), nil
}

func (st *Storage) ListCharacterMailListsOrdered(ctx context.Context, characterID int32) ([]*app.EveEntity, error) {
	ll, err := st.qRO.ListCharacterMailListsOrdered(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list mail lists for character %d: %w", characterID, err)
	}
	ee := make([]*app.EveEntity, len(ll))
	for i, l := range ll {
		ee[i] = eveEntityFromDBModel(l)
	}
	return ee, nil
}

func (st *Storage) UpdateCharacterMailSetBody(ctx context.Context, characterID int32, mailID int32, body string) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterMailBody: %d %d: %w", characterID, mailID, err)
	}
	if characterID == 0 || mailID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateCharacterMailSetBody(ctx, queries.UpdateCharacterMailSetBodyParams{
		CharacterID: int64(characterID),
		MailID:      int64(mailID),
		Body:        body,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) UpdateCharacterMailSetIsRead(ctx context.Context, characterID int32, mailPK int64, isRead bool) error {
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

func (st *Storage) UpdateCharacterMailSetLabels(ctx context.Context, characterID int32, mailPK int64, labelIDs []int32) error {
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
		arg := queries.GetCharacterMailLabelParams{
			CharacterID: int64(characterID),
			LabelID:     int64(l),
		}
		label, err := qtx.GetCharacterMailLabel(ctx, arg)
		if err != nil {
			return wrapErr(fmt.Errorf("get mail label: %w", err))
		}
		mailMailLabelParams := queries.CreateMailCharacterMailLabelParams{
			CharacterMailID: mailPK, CharacterMailLabelID: label.ID,
		}
		if err := qtx.CreateMailCharacterMailLabel(ctx, mailMailLabelParams); err != nil {
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
		Body:        mail.Body,
		CharacterID: int32(mail.CharacterID),
		From:        eveEntityFromDBModel(from),
		IsProcessed: mail.IsProcessed,
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

func (st *Storage) UpdateCharacterMailSetProcessed(ctx context.Context, id int64) error {
	if err := st.qRW.UpdateCharacterMailSetProcessed(ctx, id); err != nil {
		return fmt.Errorf("set mail PK %d as processed: %w", id, err)
	}
	return nil
}
