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
)

type CreateCharacterNotificationParams struct {
	Body           optional.Optional[string]
	CharacterID    int32
	IsRead         bool
	NotificationID int64
	SenderID       int32
	Text           string
	Timestamp      time.Time
	Title          optional.Optional[string]
	Type           string
}

func (st *Storage) GetCharacterNotification(ctx context.Context, characterID int32, notificationID int64) (*app.CharacterNotification, error) {
	arg := queries.GetCharacterNotificationParams{
		CharacterID:    int64(characterID),
		NotificationID: notificationID,
	}
	row, err := st.q.GetCharacterNotification(ctx, arg)
	if err != nil {
		return nil, err
	}
	return characterNotificationFromDBModel(row.CharacterNotification, row.EveEntity, row.NotificationType), err
}

func (st *Storage) ListCharacterNotificationIDs(ctx context.Context, characterID int32) ([]int64, error) {
	return st.q.ListCharacterNotificationIDs(ctx, int64(characterID))
}

func (st *Storage) ListCharacterNotificationsTypes(ctx context.Context, characterID int32, types []string) ([]*app.CharacterNotification, error) {
	arg := queries.ListCharacterNotificationsTypesParams{
		CharacterID: int64(characterID),
		Names:       types,
	}
	rows, err := st.q.ListCharacterNotificationsTypes(ctx, arg)
	if err != nil {
		return nil, err
	}
	ee := make([]*app.CharacterNotification, len(rows))
	for i, row := range rows {
		ee[i] = characterNotificationFromDBModel(row.CharacterNotification, row.EveEntity, row.NotificationType)
	}
	return ee, nil
}

func (st *Storage) ListCharacterNotificationsUnread(ctx context.Context, characterID int32) ([]*app.CharacterNotification, error) {
	rows, err := st.q.ListCharacterNotificationsUnread(ctx, int64(characterID))
	if err != nil {
		return nil, err
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

func (st *Storage) CreateCharacterNotification(ctx context.Context, arg CreateCharacterNotificationParams) error {
	if arg.NotificationID == 0 {
		return fmt.Errorf("notification ID can not be zero, Character %d", arg.CharacterID)
	}
	typeID, err := st.GetOrCreateNotificationType(ctx, arg.Type)
	if err != nil {
		return err
	}
	arg2 := queries.CreateCharacterNotificationParams{
		CharacterID:    int64(arg.CharacterID),
		IsRead:         arg.IsRead,
		NotificationID: arg.NotificationID,
		SenderID:       int64(arg.SenderID),
		Text:           arg.Text,
		Timestamp:      arg.Timestamp,
		TypeID:         typeID,
	}
	return st.q.CreateCharacterNotification(ctx, arg2)
}

type UpdateCharacterNotificationParams struct {
	ID          int64
	Body        optional.Optional[string]
	CharacterID int32
	IsRead      bool
	Title       optional.Optional[string]
}

func (st *Storage) UpdateCharacterNotification(ctx context.Context, arg UpdateCharacterNotificationParams) error {
	arg2 := queries.UpdateCharacterNotificationParams{
		ID:     arg.ID,
		Body:   optional.ToNullString(arg.Body),
		IsRead: arg.IsRead,
		Title:  optional.ToNullString(arg.Title),
	}
	if err := st.q.UpdateCharacterNotification(ctx, arg2); err != nil {
		return fmt.Errorf("failed to update notification PK %d for character %d: %w", arg.ID, arg.CharacterID, err)
	}
	return nil
}

func (st *Storage) GetOrCreateNotificationType(ctx context.Context, name string) (int64, error) {
	tx, err := st.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	qtx := st.q.WithTx(tx)
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
}

func (st *Storage) CalcCharacterNotificationUnreadCounts(ctx context.Context, characterID int32) (map[string]int, error) {
	rows, err := st.q.CalcCharacterNotificationUnreadCounts(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	x := make(map[string]int)
	for _, r := range rows {
		x[r.Name] = int(r.Sum.Float64)
	}
	return x, nil
}