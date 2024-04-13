package logic

import "example/evebuddy/internal/model"

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
	ID          uint64
	CharacterID int32
	Color       string
	LabelID     int32
	Name        string
	UnreadCount int32
}

func mailLabelFromDBModel(l model.MailLabel) MailLabel {
	return MailLabel{
		ID:          l.ID,
		CharacterID: l.CharacterID,
		Color:       l.Color,
		LabelID:     l.LabelID,
		Name:        l.Name,
		UnreadCount: l.UnreadCount,
	}
}

func ListMailLabels(characterID int32) ([]MailLabel, error) {
	ll, err := model.ListMailLabels(characterID)
	if err != nil {
		return nil, err
	}
	ll2 := make([]MailLabel, len(ll))
	for i, l := range ll {
		ll2[i] = mailLabelFromDBModel(l)
	}
	return ll2, nil
}
