package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) DeleteObsoleteCharacterMailLabels(ctx context.Context, characterID int32) error {
	arg := queries.DeleteObsoleteCharacterMailLabelsParams{
		CharacterID:   int64(characterID),
		CharacterID_2: int64(characterID),
	}
	if err := st.qRW.DeleteObsoleteCharacterMailLabels(ctx, arg); err != nil {
		return fmt.Errorf("delete obsolete mail labels for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) GetCharacterMailLabel(ctx context.Context, characterID, labelID int32) (*app.CharacterMailLabel, error) {
	arg := queries.GetCharacterMailLabelParams{
		CharacterID: int64(characterID),
		LabelID:     int64(labelID),
	}
	l, err := st.qRO.GetCharacterMailLabel(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf(
			"get mail label for character %d with label %d: %w",
			arg.CharacterID,
			arg.LabelID,
			convertGetError(err),
		)
	}
	l2 := characterMailLabelFromDBModel(l)
	return l2, nil
}

type MailLabelParams struct {
	CharacterID int32
	Color       string
	LabelID     int32
	Name        string
	UnreadCount int
}

func (st *Storage) GetOrCreateCharacterMailLabel(ctx context.Context, arg MailLabelParams) (*app.CharacterMailLabel, error) {
	label, err := func() (*app.CharacterMailLabel, error) {
		var l queries.CharacterMailLabel
		tx, err := st.dbRW.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		qtx := st.qRW.WithTx(tx)
		arg2 := queries.GetCharacterMailLabelParams{
			CharacterID: int64(arg.CharacterID),
			LabelID:     int64(arg.LabelID),
		}
		l, err = qtx.GetCharacterMailLabel(ctx, arg2)
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		} else if err != nil {
			arg3 := queries.CreateCharacterMailLabelParams{
				CharacterID: int64(arg.CharacterID),
				LabelID:     int64(arg.LabelID),
				Color:       arg.Color,
				Name:        arg.Name,
				UnreadCount: int64(arg.UnreadCount),
			}
			l, err = qtx.CreateCharacterMailLabel(ctx, arg3)
			if err != nil {
				return nil, err
			}
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return characterMailLabelFromDBModel(l), nil
	}()
	if err != nil {
		return label, fmt.Errorf("get or create mail label for character %d and label %d: %w", arg.CharacterID, arg.LabelID, err)
	}
	return label, nil
}

func (st *Storage) ListCharacterMailLabelsOrdered(ctx context.Context, characterID int32) ([]*app.CharacterMailLabel, error) {
	ll, err := st.qRO.ListCharacterMailLabelsOrdered(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list mail label IDs for character %d: %w", characterID, err)
	}
	ll2 := make([]*app.CharacterMailLabel, len(ll))
	for i, l := range ll {
		ll2[i] = characterMailLabelFromDBModel(l)
	}
	return ll2, nil
}

func (st *Storage) UpdateOrCreateCharacterMailLabel(ctx context.Context, arg MailLabelParams) (*app.CharacterMailLabel, error) {
	arg1 := queries.UpdateOrCreateCharacterMailLabelParams{
		CharacterID: int64(arg.CharacterID),
		LabelID:     int64(arg.LabelID),
		Color:       arg.Color,
		Name:        arg.Name,
		UnreadCount: int64(arg.UnreadCount),
	}
	l, err := st.qRW.UpdateOrCreateCharacterMailLabel(ctx, arg1)
	if err != nil {
		return nil, fmt.Errorf("update or create mail label for character %d with label %d: %w", arg.CharacterID, arg.LabelID, err)
	}
	label := characterMailLabelFromDBModel(l)
	return label, nil
}

func characterMailLabelFromDBModel(l queries.CharacterMailLabel) *app.CharacterMailLabel {
	return &app.CharacterMailLabel{
		ID:          l.ID,
		CharacterID: int32(l.CharacterID),
		Color:       l.Color,
		LabelID:     int32(l.LabelID),
		Name:        l.Name,
		UnreadCount: int(l.UnreadCount),
	}
}
