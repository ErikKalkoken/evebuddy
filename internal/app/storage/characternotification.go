package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func (st *Storage) CreateCharacterNotification(ctx context.Context, arg CreateCharacterNotificationParams) error {
	if !arg.isValid() {
		return fmt.Errorf("CreateCharacterNotification: %+v: %w", arg, app.ErrInvalid)
	}
	typeID, err := st.GetOrCreateNotificationType(ctx, arg.Type)
	if err != nil {
		return err
	}
	arg2 := queries.CreateCharacterNotificationParams{
		Body:           optional.ToNullString(arg.Body),
		CharacterID:    int64(arg.CharacterID),
		IsRead:         arg.IsRead,
		IsProcessed:    arg.IsProcessed,
		NotificationID: arg.NotificationID,
		SenderID:       int64(arg.SenderID),
		Text:           arg.Text,
		Timestamp:      arg.Timestamp,
		Title:          optional.ToNullString(arg.Title),
		TypeID:         typeID,
	}
	if err := st.qRW.CreateCharacterNotification(ctx, arg2); err != nil {
		arg.Body.Clear()
		arg.Text = ""
		return fmt.Errorf("create character notification %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetCharacterNotification(ctx context.Context, characterID int32, notificationID int64) (*app.CharacterNotification, error) {
	arg := queries.GetCharacterNotificationParams{
		CharacterID:    int64(characterID),
		NotificationID: notificationID,
	}
	row, err := st.qRO.GetCharacterNotification(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get character notification %+v: %w", arg, err)
	}
	return characterNotificationFromDBModel(row.CharacterNotification, row.EveEntity, row.NotificationType), err
}

func (st *Storage) ListCharacterNotificationIDs(ctx context.Context, characterID int32) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterNotificationIDs(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list character notification ids for character %d: %w", characterID, err)
	}
	return set.NewFromSlice(ids), nil
}

func (st *Storage) ListCharacterNotificationsTypes(ctx context.Context, characterID int32, types []string) ([]*app.CharacterNotification, error) {
	arg := queries.ListCharacterNotificationsTypesParams{
		CharacterID: int64(characterID),
		Names:       types,
	}
	rows, err := st.qRO.ListCharacterNotificationsTypes(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list notification types %+v: %w", arg, err)
	}
	ee := make([]*app.CharacterNotification, len(rows))
	for i, row := range rows {
		ee[i] = characterNotificationFromDBModel(row.CharacterNotification, row.EveEntity, row.NotificationType)
	}
	return ee, nil
}

func (st *Storage) ListCharacterNotificationsAll(ctx context.Context, characterID int32) ([]*app.CharacterNotification, error) {
	rows, err := st.qRO.ListCharacterNotificationsAll(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list all notifications for character %d: %w", characterID, err)
	}
	ee := make([]*app.CharacterNotification, len(rows))
	for i, r := range rows {
		ee[i] = characterNotificationFromDBModel(r.CharacterNotification, r.EveEntity, r.NotificationType)
	}
	return ee, nil
}

func (st *Storage) ListCharacterNotificationsUnread(ctx context.Context, characterID int32) ([]*app.CharacterNotification, error) {
	rows, err := st.qRO.ListCharacterNotificationsUnread(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list unread notification for character %d: %w", characterID, err)
	}
	ee := make([]*app.CharacterNotification, len(rows))
	for i, row := range rows {
		ee[i] = characterNotificationFromDBModel(row.CharacterNotification, row.EveEntity, row.NotificationType)
	}
	return ee, nil
}

func (st *Storage) ListCharacterNotificationsUnprocessed(ctx context.Context, characterID int32, earliest time.Time) ([]*app.CharacterNotification, error) {
	arg := queries.ListCharacterNotificationsUnprocessedParams{
		CharacterID: int64(characterID),
		Timestamp:   earliest,
	}
	rows, err := st.qRO.ListCharacterNotificationsUnprocessed(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list unprocessed notifications %+v: %w", arg, err)
	}
	ee := make([]*app.CharacterNotification, len(rows))
	for i, row := range rows {
		ee[i] = characterNotificationFromDBModel(row.CharacterNotification, row.EveEntity, row.NotificationType)
	}
	return ee, nil
}

func characterNotificationFromDBModel(o queries.CharacterNotification, sender queries.EveEntity, type_ queries.NotificationType) *app.CharacterNotification {
	o2 := &app.CharacterNotification{
		ID:             o.ID,
		Body:           optional.FromNullString(o.Body),
		CharacterID:    int32(o.CharacterID),
		IsProcessed:    o.IsProcessed,
		IsRead:         o.IsRead,
		NotificationID: o.NotificationID,
		Sender:         eveEntityFromDBModel(sender),
		Text:           o.Text,
		Timestamp:      o.Timestamp,
		Title:          optional.FromNullString(o.Title),
		Type:           type_.Name,
	}
	return o2
}

type CreateCharacterNotificationParams struct {
	Body           optional.Optional[string]
	CharacterID    int32
	IsRead         bool
	IsProcessed    bool
	NotificationID int64
	SenderID       int32
	Text           string
	Timestamp      time.Time
	Title          optional.Optional[string]
	Type           string
}

func (arg CreateCharacterNotificationParams) isValid() bool {
	return arg.CharacterID != 0 && arg.NotificationID != 0 && arg.SenderID != 0
}

type UpdateCharacterNotificationParams struct {
	ID     int64
	Body   optional.Optional[string]
	IsRead bool
	Title  optional.Optional[string]
}

func (st *Storage) UpdateCharacterNotification(ctx context.Context, arg UpdateCharacterNotificationParams) error {
	if arg.ID == 0 {
		return fmt.Errorf("UpdateCharacterNotification: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.UpdateCharacterNotificationParams{
		ID:     arg.ID,
		Body:   optional.ToNullString(arg.Body),
		IsRead: arg.IsRead,
		Title:  optional.ToNullString(arg.Title),
	}
	if err := st.qRW.UpdateCharacterNotification(ctx, arg2); err != nil {
		return fmt.Errorf("update character notification %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetOrCreateNotificationType(ctx context.Context, name string) (int64, error) {
	id, err := func() (int64, error) {
		tx, err := st.dbRW.Begin()
		if err != nil {
			return 0, err
		}
		defer tx.Rollback()
		qtx := st.qRW.WithTx(tx)
		id, err := qtx.GetNotificationTypeID(ctx, name)
		if errors.Is(err, sql.ErrNoRows) {
			id, err = qtx.CreateNotificationType(ctx, name)
		}
		if err != nil {
			return 0, err
		}
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return id, nil
	}()
	if err != nil {
		return 0, fmt.Errorf("get or create notification type %s: %w", name, err)
	}
	return id, nil
}

func (st *Storage) CountCharacterNotificationUnreads(ctx context.Context, characterID int32) (map[string]int, error) {
	rows, err := st.qRO.CalcCharacterNotificationUnreadCounts(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("count unread notifications for character %d: %w", characterID, err)
	}
	x := make(map[string]int)
	for _, r := range rows {
		x[r.Name] = int(r.Sum.Float64)
	}
	return x, nil
}

func (st *Storage) UpdateCharacterNotificationSetProcessed(ctx context.Context, id int64) error {
	if err := st.qRW.UpdateCharacterNotificationSetProcessed(ctx, id); err != nil {
		return fmt.Errorf("update notification set processed for id %d: %w", id, err)
	}
	return nil
}
