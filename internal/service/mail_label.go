package service

import (
	"context"
	"example/evebuddy/internal/repository"
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

func mailLabelFromDBModel(l repository.MailLabel) MailLabel {
	return MailLabel{
		ID:          l.ID,
		CharacterID: int32(l.CharacterID),
		Color:       l.Color,
		LabelID:     int32(l.LabelID),
		Name:        l.Name,
		UnreadCount: int(l.UnreadCount),
	}
}

func (s *Service) ListMailLabels(characterID int32) ([]MailLabel, error) {
	ll, err := s.queries.ListMailLabels(context.Background(), int64(characterID))
	if err != nil {
		return nil, err
	}
	ll2 := make([]MailLabel, len(ll))
	for i, l := range ll {
		ll2[i] = mailLabelFromDBModel(l)
	}
	return ll2, nil
}
