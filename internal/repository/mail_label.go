package repository

import (
	"context"
	"example/evebuddy/internal/sqlc"
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

func (r *Repository) UpdateOrCreateMailLabel(ctx context.Context, characterID int32, labelID int32, name string, color string, unreadCount int) error {
	arg := sqlc.CreateMailLabelParams{
		CharacterID: int64(characterID),
		LabelID:     int64(labelID),
		Color:       color,
		Name:        name,
		UnreadCount: int64(unreadCount),
	}
	if err := r.q.CreateMailLabel(ctx, arg); err != nil {
		if !isSqlite3ErrConstraint(err) {
			return err
		}
		arg := sqlc.UpdateMailLabelParams{
			CharacterID: int64(characterID),
			LabelID:     int64(labelID),
			Color:       color,
			Name:        name,
			UnreadCount: int64(unreadCount),
		}
		if err := r.q.UpdateMailLabel(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}