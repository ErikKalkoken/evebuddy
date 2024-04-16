package repository

import (
	"context"
	"database/sql"
	"errors"
	"example/evebuddy/internal/sqlc"
	"fmt"
)

// Special mail label IDs
const (
	LabelAll      = 1<<31 - 1
	LabelNone     = 0
	LabelInbox    = 1
	LabelSent     = 2
	LabelCorp     = 4
	LabelAlliance = 8
)

type MailLabel struct {
	ID          int64
	CharacterID int32
	Color       string
	LabelID     int32
	Name        string
	UnreadCount int
}

func mailLabelFromDBModel(l sqlc.MailLabel) MailLabel {
	return MailLabel{
		ID:          l.ID,
		CharacterID: int32(l.CharacterID),
		Color:       l.Color,
		LabelID:     int32(l.LabelID),
		Name:        l.Name,
		UnreadCount: int(l.UnreadCount),
	}
}

func (r *Repository) GetMailLabel(ctx context.Context, characterID, labelID int32) (MailLabel, error) {
	arg := sqlc.GetMailLabelParams{
		CharacterID: int64(characterID),
		LabelID:     int64(labelID),
	}
	l, err := r.q.GetMailLabel(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return MailLabel{}, fmt.Errorf("failed to get MailLabel %v: %w", arg, err)
	}
	l2 := mailLabelFromDBModel(l)
	return l2, nil
}

func (r *Repository) ListMailLabels(ctx context.Context, characterID int32) ([]MailLabel, error) {
	ll, err := r.q.ListMailLabels(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	ll2 := make([]MailLabel, len(ll))
	for i, l := range ll {
		ll2[i] = mailLabelFromDBModel(l)
	}
	return ll2, nil
}

type UpdateOrCreateMailLabelParams struct {
	CharacterID int32
	Color       string
	LabelID     int32
	Name        string
	UnreadCount int
}

func (r *Repository) UpdateOrCreateMailLabel(ctx context.Context, arg UpdateOrCreateMailLabelParams) (MailLabel, error) {
	label, err := func() (MailLabel, error) {
		var l sqlc.MailLabel
		tx, err := r.db.Begin()
		if err != nil {
			return MailLabel{}, err
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
			if !isSqlite3ErrConstraint(err) {
				return MailLabel{}, err
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
				return MailLabel{}, err
			}
		}
		if err := tx.Commit(); err != nil {
			return MailLabel{}, err
		}
		l2 := mailLabelFromDBModel(l)
		return l2, nil
	}()
	if err != nil {
		return label, fmt.Errorf("failed to update or create MailLabel %v: %w", arg, err)
	}
	return label, nil
}
