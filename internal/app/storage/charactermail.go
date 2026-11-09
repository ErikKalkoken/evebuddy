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
	tx, err := st.dbRW.Begin()
	if err != nil {
		return 0, wrapErr(err)
	}
	defer tx.Rollback()
	id, err := createCharacterMail(ctx, st.qRW.WithTx(tx), arg)
	if err != nil {
		return 0, wrapErr(err)
	}
	if err := tx.Commit(); err != nil {
		return 0, wrapErr(err)
	}
	slog.Info("Created mail", "characterID", arg.CharacterID, "mailID", arg.MailID)
	return id, nil
}

// createCharacterMail creates a mail with its recipients and labels. Must be called within a transaction.
func createCharacterMail(ctx context.Context, q *queries.Queries, arg CreateCharacterMailParams) (int64, error) {
	if arg.CharacterID == 0 || arg.MailID == 0 || len(arg.RecipientIDs) == 0 {
		return 0, app.ErrInvalid
	}
	mail, err := q.CreateMail(ctx, queries.CreateMailParams{
		Body2:       optional.ToNullString(arg.Body),
		CharacterID: arg.CharacterID,
		FromID:      arg.FromID,
		MailID:      arg.MailID,
		Subject:     arg.Subject.ValueOrZero(),
		IsProcessed: arg.IsProcessed,
		IsRead:      arg.IsRead.ValueOrZero(),
		Timestamp:   arg.Timestamp.UTC(), // stored times must be UTC, because SQLite compares them as text
	})
	if err != nil {
		return 0, err
	}
	for id := range set.Of(arg.RecipientIDs...).All() {
		err := q.CreateMailRecipient(ctx, queries.CreateMailRecipientParams{
			MailID:      mail.ID,
			EveEntityID: id,
		})
		if err != nil {
			return 0, fmt.Errorf("create mail recipient: %w", err)
		}
	}
	if err := addCharacterMailLabels(ctx, q, arg.CharacterID, mail.ID, arg.LabelIDs); err != nil {
		return 0, err
	}
	return mail.ID, nil
}

// UpdateOrCreateCharacterMailParams is a mail header from ESI.
type UpdateOrCreateCharacterMailParams struct {
	FromID       int64
	IsRead       optional.Optional[bool]
	LabelIDs     []int64
	MailID       int64
	RecipientIDs []int64
	Subject      optional.Optional[string]
	Timestamp    time.Time
}

// UpdateOrCreateCharacterMails creates new mail messages of a character
// and updates read state and labels of stored messages.
// Existing mail messages missing in args are kept.
// Reports whether any mail message was created or updated.
func (st *Storage) UpdateOrCreateCharacterMails(ctx context.Context, characterID int64, args []UpdateOrCreateCharacterMailParams) (bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCharacterMails: character %d: %w", characterID, err)
	}
	if characterID == 0 {
		return false, wrapErr(app.ErrInvalid)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return false, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	var mailIDs set.Set[int64]
	for _, arg := range args {
		mailIDs.Add(arg.MailID)
	}
	storedMessages := make(map[int64]queries.ListCharacterMailStatesRow) // by mail ID
	storedLabelIDs := make(map[int64]set.Set[int64])                     // by mail PK
	for ids := range slices.Chunk(slices.Collect(mailIDs.All()), st.MaxIDsPerQuery) {
		stateRows, err := qtx.ListCharacterMailStates(ctx, queries.ListCharacterMailStatesParams{
			CharacterID: characterID,
			MailIds:     ids,
		})
		if err != nil {
			return false, wrapErr(err)
		}
		for _, r := range stateRows {
			storedMessages[r.MailID] = r
		}
		labelRows, err := qtx.ListCharacterMailLabelIDs(ctx, queries.ListCharacterMailLabelIDsParams{
			CharacterID: characterID,
			MailIds:     ids,
		})
		if err != nil {
			return false, wrapErr(err)
		}
		for _, r := range labelRows {
			labelIDs := storedLabelIDs[r.CharacterMailID]
			labelIDs.Add(r.LabelID)
			storedLabelIDs[r.CharacterMailID] = labelIDs
		}
	}
	var created, updated int
	for _, arg := range args {
		isRead := arg.IsRead.ValueOrZero()
		labelIDs := set.Of(arg.LabelIDs...)
		message, found := storedMessages[arg.MailID]
		if !found {
			id, err := createCharacterMail(ctx, qtx, CreateCharacterMailParams{
				CharacterID:  characterID,
				FromID:       arg.FromID,
				IsRead:       arg.IsRead,
				LabelIDs:     slices.Collect(labelIDs.All()),
				MailID:       arg.MailID,
				RecipientIDs: arg.RecipientIDs,
				Subject:      arg.Subject,
				Timestamp:    arg.Timestamp,
			})
			if err != nil {
				return false, wrapErr(fmt.Errorf("create mail %d: %w", arg.MailID, err))
			}
			storedMessages[arg.MailID] = queries.ListCharacterMailStatesRow{ID: id, MailID: arg.MailID, IsRead: isRead}
			storedLabelIDs[id] = labelIDs
			created++
			continue
		}
		var changed bool
		if isRead != message.IsRead {
			err := qtx.UpdateCharacterMailIsRead(ctx, queries.UpdateCharacterMailIsReadParams{
				ID:     message.ID,
				IsRead: isRead,
			})
			if err != nil {
				return false, wrapErr(fmt.Errorf("update read state of mail %d: %w", arg.MailID, err))
			}
			message.IsRead = isRead
			storedMessages[arg.MailID] = message
			changed = true
		}
		if !labelIDs.Equal(storedLabelIDs[message.ID]) {
			if err := qtx.DeleteMailCharacterMailLabels(ctx, message.ID); err != nil {
				return false, wrapErr(fmt.Errorf("delete labels of mail %d: %w", arg.MailID, err))
			}
			if err := addCharacterMailLabels(ctx, qtx, characterID, message.ID, slices.Collect(labelIDs.All())); err != nil {
				return false, wrapErr(fmt.Errorf("add labels to mail %d: %w", arg.MailID, err))
			}
			storedLabelIDs[message.ID] = labelIDs
			changed = true
		}
		if changed {
			updated++
		}
	}
	if err := tx.Commit(); err != nil {
		return false, wrapErr(err)
	}
	if created > 0 || updated > 0 {
		slog.Info("Stored mail", "characterID", characterID, "created", created, "updated", updated)
	}
	return created > 0 || updated > 0, nil
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

func (st *Storage) CountCharacterMailsUnread(ctx context.Context, id int64) (int, error) {
	count, err := st.qRO.CountCharacterMailsUnread(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("get mail unread count for character %d: %w", id, convertGetError(err))
	}
	return int(count), err
}

func (st *Storage) CountAllCharactersMailsUnread(ctx context.Context) (int, error) {
	count, err := st.qRO.CountAllCharactersMailsUnread(ctx)
	if err != nil {
		return 0, fmt.Errorf("get all mail unread count: %w", err)
	}
	return int(count), err
}

// CountAllCharactersMails returns the number of mail messages of all characters.
func (st *Storage) CountAllCharactersMails(ctx context.Context) (int, error) {
	count, err := st.qRO.CountAllCharactersMails(ctx)
	if err != nil {
		return 0, fmt.Errorf("get all mail count: %w", err)
	}
	return int(count), nil
}

// CountAllCharactersMailsWithoutBody returns the number of mail messages of all characters without a body.
func (st *Storage) CountAllCharactersMailsWithoutBody(ctx context.Context) (int, error) {
	count, err := st.qRO.CountAllCharactersMailsWithoutBody(ctx)
	if err != nil {
		return 0, fmt.Errorf("get all mail without body count: %w", err)
	}
	return int(count), nil
}

func (st *Storage) CountCharacterMails(ctx context.Context, characterID int64) (int, error) {
	count, err := st.qRO.CountCharacterMails(ctx, characterID)
	if err != nil {
		return 0, fmt.Errorf("get mail count for character %d: %w", characterID, convertGetError(err))
	}
	return int(count), err
}

// CountCharacterMailsWithoutBody returns the number of mail messages of a character without a body.
func (st *Storage) CountCharacterMailsWithoutBody(ctx context.Context, characterID int64) (int, error) {
	count, err := st.qRO.CountCharacterMailsWithoutBody(ctx, characterID)
	if err != nil {
		return 0, fmt.Errorf("CountCharacterMailsWithoutBody for character %d: %w", characterID, err)
	}
	return int(count), nil
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

func (st *Storage) CountAllCharactersMailsUnreadByLabel(ctx context.Context) (map[int64]int, error) {
	rows, err := st.qRO.CountAllCharactersMailsUnreadByLabel(ctx)
	if err != nil {
		return nil, fmt.Errorf("get mail label unread counts for all characters: %w", err)
	}
	result := make(map[int64]int)
	for _, r := range rows {
		result[r.LabelID] = int(r.UnreadCount2)
	}
	return result, nil
}

func (st *Storage) CountAllCharactersMailsUnreadByList(ctx context.Context) (map[int64]int, error) {
	rows, err := st.qRO.CountAllCharactersMailsUnreadByList(ctx)
	if err != nil {
		return nil, fmt.Errorf("get mail list unread counts for all characters: %w", err)
	}
	result := make(map[int64]int)
	for _, r := range rows {
		result[r.ListID] = int(r.UnreadCount2)
	}
	return result, nil
}

func (st *Storage) CountCharacterMailsUnreadByLabel(ctx context.Context, characterID int64) (map[int64]int, error) {
	rows, err := st.qRO.CountCharacterMailsUnreadByLabel(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("get mail label unread counts for character %d: %w", characterID, convertGetError(err))
	}
	result := make(map[int64]int)
	for _, r := range rows {
		result[int64(r.LabelID)] = int(r.UnreadCount2)
	}
	return result, nil
}

func (st *Storage) CountCharacterMailsUnreadByList(ctx context.Context, characterID int64) (map[int64]int, error) {
	rows, err := st.qRO.CountCharacterMailsUnreadByList(ctx, characterID)
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

// ListAllCharacterMailListsOrdered returns the mailing lists of all characters without duplicates.
func (st *Storage) ListAllCharacterMailListsOrdered(ctx context.Context) ([]*app.EveEntity, error) {
	ll, err := st.qRO.ListAllCharacterMailListsOrdered(ctx)
	if err != nil {
		return nil, fmt.Errorf("list mail lists for all characters: %w", err)
	}
	ee := make([]*app.EveEntity, len(ll))
	for i, l := range ll {
		ee[i] = eveEntityFromDBModel(l)
	}
	return ee, nil
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

// addCharacterMailLabels adds labels to a mail. Must be called within a transaction.
func addCharacterMailLabels(ctx context.Context, q *queries.Queries, characterID int64, mailPK int64, labelIDs []int64) error {
	for _, l := range labelIDs {
		// labels not stored yet are created with a generic name, which the next labels update replaces
		label, err := getOrCreateCharacterMailLabel(ctx, q, MailLabelParams{
			CharacterID: characterID,
			LabelID:     l,
			Name:        optional.New(fmt.Sprintf("Label %d", l)),
		})
		if err != nil {
			return fmt.Errorf("get or create mail label %d: %w", l, err)
		}
		if err := q.CreateMailCharacterMailLabel(ctx, queries.CreateMailCharacterMailLabelParams{
			CharacterMailID:      mailPK,
			CharacterMailLabelID: label.ID,
		}); err != nil {
			return fmt.Errorf("create mail label: %w", err)
		}
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
	ll := make([]*app.CharacterMailLabel, len(labels))
	for i, l := range labels {
		ll[i] = characterMailLabelFromDBModel(l)
	}
	rr := make([]*app.EveEntity, len(recipients))
	for i, r := range recipients {
		rr[i] = eveEntityFromDBModel(r)
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
