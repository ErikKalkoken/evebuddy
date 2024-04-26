package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage/queries"
)

func mailLabelFromDBModel(l queries.MailLabel) *model.MailLabel {
	return &model.MailLabel{
		ID:            l.ID,
		MyCharacterID: int32(l.MyCharacterID),
		Color:         l.Color,
		LabelID:       int32(l.LabelID),
		Name:          l.Name,
		UnreadCount:   int(l.UnreadCount),
	}
}

func (r *Storage) DeleteObsoleteMailLabels(ctx context.Context, characterID int32) error {
	arg := queries.DeleteObsoleteMailLabelsParams{
		MyCharacterID:   int64(characterID),
		MyCharacterID_2: int64(characterID),
	}
	if err := r.q.DeleteObsoleteMailLabels(ctx, arg); err != nil {
		return fmt.Errorf("failed to delete obsolete mail labels for character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) GetMailLabel(ctx context.Context, characterID, labelID int32) (*model.MailLabel, error) {
	arg := queries.GetMailLabelParams{
		MyCharacterID: int64(characterID),
		LabelID:       int64(labelID),
	}
	l, err := r.q.GetMailLabel(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get mail label for character %d with label %d: %w", arg.MyCharacterID, arg.LabelID, err)
	}
	l2 := mailLabelFromDBModel(l)
	return l2, nil
}

type MailLabelParams struct {
	MyCharacterID int32
	Color         string
	LabelID       int32
	Name          string
	UnreadCount   int
}

func (r *Storage) GetOrCreateMailLabel(ctx context.Context, arg MailLabelParams) (*model.MailLabel, error) {
	label, err := func() (*model.MailLabel, error) {
		var l queries.MailLabel
		tx, err := r.db.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		qtx := r.q.WithTx(tx)
		arg2 := queries.GetMailLabelParams{
			MyCharacterID: int64(arg.MyCharacterID),
			LabelID:       int64(arg.LabelID),
		}
		l, err = qtx.GetMailLabel(ctx, arg2)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
			arg3 := queries.CreateMailLabelParams{
				MyCharacterID: int64(arg.MyCharacterID),
				LabelID:       int64(arg.LabelID),
				Color:         arg.Color,
				Name:          arg.Name,
				UnreadCount:   int64(arg.UnreadCount),
			}
			l, err = qtx.CreateMailLabel(ctx, arg3)
			if err != nil {
				return nil, err
			}
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return mailLabelFromDBModel(l), nil
	}()
	if err != nil {
		return label, fmt.Errorf("failed to get or create mail label for character %d and label %d: %w", arg.MyCharacterID, arg.LabelID, err)
	}
	return label, nil
}

func (r *Storage) ListMailLabelsOrdered(ctx context.Context, characterID int32) ([]*model.MailLabel, error) {
	ll, err := r.q.ListMailLabelsOrdered(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("failed to list mail label IDs for character %d: %w", characterID, err)
	}
	ll2 := make([]*model.MailLabel, len(ll))
	for i, l := range ll {
		ll2[i] = mailLabelFromDBModel(l)
	}
	return ll2, nil
}

func (r *Storage) UpdateOrCreateMailLabel(ctx context.Context, arg MailLabelParams) (*model.MailLabel, error) {
	arg1 := queries.UpdateOrCreateMailLabelParams{
		MyCharacterID: int64(arg.MyCharacterID),
		LabelID:       int64(arg.LabelID),
		Color:         arg.Color,
		Name:          arg.Name,
		UnreadCount:   int64(arg.UnreadCount),
	}
	l, err := r.q.UpdateOrCreateMailLabel(ctx, arg1)
	if err != nil {
		return nil, fmt.Errorf("failed to update or create mail label for character %d with label %d: %w", arg.MyCharacterID, arg.LabelID, err)
	}
	label := mailLabelFromDBModel(l)
	return label, nil
}
