package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"example/evebuddy/internal/model"
	"example/evebuddy/internal/sqlc"

	"github.com/mattn/go-sqlite3"
)

func mailLabelFromDBModel(l sqlc.MailLabel) model.MailLabel {
	return model.MailLabel{
		ID:          l.ID,
		CharacterID: int32(l.CharacterID),
		Color:       l.Color,
		LabelID:     int32(l.LabelID),
		Name:        l.Name,
		UnreadCount: int(l.UnreadCount),
	}
}

func (r *Storage) GetMailLabel(ctx context.Context, characterID, labelID int32) (model.MailLabel, error) {
	arg := sqlc.GetMailLabelParams{
		CharacterID: int64(characterID),
		LabelID:     int64(labelID),
	}
	l, err := r.q.GetMailLabel(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return model.MailLabel{}, fmt.Errorf("failed to get mail label for character %d with label %d: %w", arg.CharacterID, arg.LabelID, err)
	}
	l2 := mailLabelFromDBModel(l)
	return l2, nil
}

type MailLabelParams struct {
	CharacterID int32
	Color       string
	LabelID     int32
	Name        string
	UnreadCount int
}

func (r *Storage) GetOrCreateMailLabel(ctx context.Context, arg MailLabelParams) (model.MailLabel, error) {
	label, err := func() (model.MailLabel, error) {
		var l sqlc.MailLabel
		tx, err := r.db.Begin()
		if err != nil {
			return model.MailLabel{}, err
		}
		defer tx.Rollback()
		qtx := r.q.WithTx(tx)
		arg2 := sqlc.GetMailLabelParams{
			CharacterID: int64(arg.CharacterID),
			LabelID:     int64(arg.LabelID),
		}
		l, err = qtx.GetMailLabel(ctx, arg2)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return model.MailLabel{}, err
			}
			arg3 := sqlc.CreateMailLabelParams{
				CharacterID: int64(arg.CharacterID),
				LabelID:     int64(arg.LabelID),
				Color:       arg.Color,
				Name:        arg.Name,
				UnreadCount: int64(arg.UnreadCount),
			}
			l, err = qtx.CreateMailLabel(ctx, arg3)
			if err != nil {
				return model.MailLabel{}, err
			}
		}
		if err := tx.Commit(); err != nil {
			return model.MailLabel{}, err
		}
		return mailLabelFromDBModel(l), nil
	}()
	if err != nil {
		return label, fmt.Errorf("failed to get or create mail label for character %d and label %d: %w", arg.CharacterID, arg.LabelID, err)
	}
	return label, nil
}

func (r *Storage) ListMailLabelsOrdered(ctx context.Context, characterID int32) ([]model.MailLabel, error) {
	ll, err := r.q.ListMailLabelsOrdered(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("failed to list mail label IDs for character %d: %w", characterID, err)
	}
	ll2 := make([]model.MailLabel, len(ll))
	for i, l := range ll {
		ll2[i] = mailLabelFromDBModel(l)
	}
	return ll2, nil
}

func (r *Storage) UpdateOrCreateMailLabel(ctx context.Context, arg MailLabelParams) (model.MailLabel, error) {
	label, err := func() (model.MailLabel, error) {
		var l sqlc.MailLabel
		tx, err := r.db.Begin()
		if err != nil {
			return model.MailLabel{}, err
		}
		defer tx.Rollback()
		qtx := r.q.WithTx(tx)
		arg1 := sqlc.CreateMailLabelParams{
			CharacterID: int64(arg.CharacterID),
			LabelID:     int64(arg.LabelID),
			Color:       arg.Color,
			Name:        arg.Name,
			UnreadCount: int64(arg.UnreadCount),
		}
		l, err = qtx.CreateMailLabel(ctx, arg1)
		if err != nil {
			sqlErr, ok := err.(sqlite3.Error)
			if !ok || sqlErr.ExtendedCode != sqlite3.ErrConstraintUnique {
				return model.MailLabel{}, err
			}
			arg2 := sqlc.UpdateMailLabelParams{
				CharacterID: int64(arg.CharacterID),
				LabelID:     int64(arg.LabelID),
				Color:       arg.Color,
				Name:        arg.Name,
				UnreadCount: int64(arg.UnreadCount),
			}
			l, err = qtx.UpdateMailLabel(ctx, arg2)
			if err != nil {
				return model.MailLabel{}, err
			}
		}
		if err := tx.Commit(); err != nil {
			return model.MailLabel{}, err
		}
		l2 := mailLabelFromDBModel(l)
		return l2, nil
	}()
	if err != nil {
		return model.MailLabel{}, fmt.Errorf("failed to update or create mail label for character %d with label %d: %w", arg.CharacterID, arg.LabelID, err)
	}
	return label, nil
}
