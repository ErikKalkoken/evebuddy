package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/queries"
)

// ListMailsForLabel returns a character's mails for a label in descending order by timestamp.
// Return mails for all labels, when labelID = 0
func (st *Storage) ListCharacterMailHeadersForLabelOrdered(ctx context.Context, characterID int32, labelID int32) ([]*app.CharacterMailHeader, error) {
	switch labelID {
	case app.MailLabelAll:
		rows, err := st.q.ListMailsOrdered(ctx, int64(characterID))
		if err != nil {
			return nil, fmt.Errorf("failed to list mails for character %d: %w", characterID, err)
		}
		mm := make([]*app.CharacterMailHeader, len(rows))
		for i, r := range rows {
			mm[i] = characterMailHeaderFromDBModel(characterID, r.FromName, r.IsRead, r.MailID, r.Subject, r.Timestamp)
		}
		return mm, nil
	case app.MailLabelNone:
		rows, err := st.q.ListMailsNoLabelOrdered(ctx, int64(characterID))
		if err != nil {
			return nil, fmt.Errorf("failed to list mails for character %d: %w", characterID, err)
		}
		mm := make([]*app.CharacterMailHeader, len(rows))
		for i, r := range rows {
			mm[i] = characterMailHeaderFromDBModel(characterID, r.FromName, r.IsRead, r.MailID, r.Subject, r.Timestamp)
		}
		return mm, nil
	case app.MailLabelSent:
		arg := queries.ListMailsForSentOrderedParams{
			CharacterID: int64(characterID),
			LabelID:     int64(app.MailLabelSent),
		}
		rows, err := st.q.ListMailsForSentOrdered(ctx, arg)
		if err != nil {
			return nil, fmt.Errorf("failed to list mails for character %d: %w", characterID, err)
		}
		mm := make([]*app.CharacterMailHeader, len(rows))
		for i, r := range rows {
			mm[i] = characterMailHeaderFromDBModel(characterID, r.FromName, r.IsRead, r.MailID, r.Subject, r.Timestamp)
		}
		return mm, nil
	default:
		arg := queries.ListMailsForLabelOrderedParams{
			CharacterID: int64(characterID),
			LabelID:     int64(labelID),
		}
		rows, err := st.q.ListMailsForLabelOrdered(ctx, arg)
		if err != nil {
			return nil, err
		}
		mm := make([]*app.CharacterMailHeader, len(rows))
		for i, r := range rows {
			mm[i] = characterMailHeaderFromDBModel(characterID, r.FromName, r.IsRead, r.MailID, r.Subject, r.Timestamp)
		}
		return mm, nil
	}
}

func (st *Storage) ListCharacterMailHeadersForListOrdered(ctx context.Context, characterID int32, listID int32) ([]*app.CharacterMailHeader, error) {
	arg := queries.ListMailsForListOrderedParams{
		CharacterID: int64(characterID),
		EveEntityID: int64(listID),
	}
	rows, err := st.q.ListMailsForListOrdered(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list mail IDs for character %d and list %d: %w", characterID, listID, err)
	}
	mm := make([]*app.CharacterMailHeader, len(rows))
	for i, r := range rows {
		mm[i] = characterMailHeaderFromDBModel(characterID, r.FromName, r.IsRead, r.MailID, r.Subject, r.Timestamp)
	}
	return mm, nil
}

func characterMailHeaderFromDBModel(
	characterID int32,
	from string,
	isRead bool,
	mailID int64,
	subject string,
	timestamp time.Time,
) *app.CharacterMailHeader {
	m := &app.CharacterMailHeader{
		CharacterID: characterID,
		From:        from,
		IsRead:      isRead,
		MailID:      int32(mailID),
		Subject:     subject,
		Timestamp:   timestamp,
	}
	return m
}
