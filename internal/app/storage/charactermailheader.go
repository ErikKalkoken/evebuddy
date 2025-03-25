package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

// ListMailsForLabel returns a character's mails for a label in descending order by timestamp.
// Return mails for all labels, when labelID = 0
func (st *Storage) ListCharacterMailHeadersForLabelOrdered(ctx context.Context, characterID int32, labelID int32) ([]*app.CharacterMailHeader, error) {
	switch labelID {
	case app.MailLabelAll:
		rows, err := st.qRO.ListMailsOrdered(ctx, int64(characterID))
		if err != nil {
			return nil, fmt.Errorf("list mails for character %d: %w", characterID, err)
		}
		mm := make([]*app.CharacterMailHeader, len(rows))
		for i, r := range rows {
			mm[i] = characterMailHeaderFromDBModel(characterID, r.CharacterMail, r.EveEntity)
		}
		return mm, nil
	case app.MailLabelUnread:
		rows, err := st.qRO.ListMailsUnreadOrdered(ctx, int64(characterID))
		if err != nil {
			return nil, fmt.Errorf("list unread mails for character %d: %w", characterID, err)
		}
		mm := make([]*app.CharacterMailHeader, len(rows))
		for i, r := range rows {
			mm[i] = characterMailHeaderFromDBModel(characterID, r.CharacterMail, r.EveEntity)
		}
		return mm, nil
	case app.MailLabelNone:
		rows, err := st.qRO.ListMailsNoLabelOrdered(ctx, int64(characterID))
		if err != nil {
			return nil, fmt.Errorf("list mails wo labels for character %d: %w", characterID, err)
		}
		mm := make([]*app.CharacterMailHeader, len(rows))
		for i, r := range rows {
			mm[i] = characterMailHeaderFromDBModel(characterID, r.CharacterMail, r.EveEntity)
		}
		return mm, nil
	default:
		arg := queries.ListMailsForLabelOrderedParams{
			CharacterID: int64(characterID),
			LabelID:     int64(labelID),
		}
		rows, err := st.qRO.ListMailsForLabelOrdered(ctx, arg)
		if err != nil {
			return nil, fmt.Errorf("list mails for character %d and label %d: %w", characterID, labelID, err)
		}
		mm := make([]*app.CharacterMailHeader, len(rows))
		for i, r := range rows {
			mm[i] = characterMailHeaderFromDBModel(characterID, r.CharacterMail, r.EveEntity)
		}
		return mm, nil
	}
}

func (st *Storage) ListCharacterMailHeadersForListOrdered(ctx context.Context, characterID int32, listID int32) ([]*app.CharacterMailHeader, error) {
	arg := queries.ListMailsForListOrderedParams{
		CharacterID: int64(characterID),
		EveEntityID: int64(listID),
	}
	rows, err := st.qRO.ListMailsForListOrdered(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list mail ids for character %d and list %d: %w", characterID, listID, err)
	}
	mm := make([]*app.CharacterMailHeader, len(rows))
	for i, r := range rows {
		mm[i] = characterMailHeaderFromDBModel(characterID, r.CharacterMail, r.EveEntity)
	}
	return mm, nil
}

func (st *Storage) ListCharacterMailHeadersForUnprocessed(ctx context.Context, characterID int32, earliest time.Time) ([]*app.CharacterMailHeader, error) {
	arg := queries.ListMailsUnprocessedParams{
		CharacterID: int64(characterID),
		LabelID:     app.MailLabelSent,
		Timestamp:   earliest,
	}
	rows, err := st.qRO.ListMailsUnprocessed(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list unprocessed mails for character %d: %w", characterID, err)
	}
	mm := make([]*app.CharacterMailHeader, len(rows))
	for i, r := range rows {
		mm[i] = characterMailHeaderFromDBModel(characterID, r.CharacterMail, r.EveEntity)
	}
	return mm, nil
}

func characterMailHeaderFromDBModel(
	characterID int32, mail queries.CharacterMail, from queries.EveEntity) *app.CharacterMailHeader {
	m := &app.CharacterMailHeader{
		CharacterID: characterID,
		From:        eveEntityFromDBModel(from),
		ID:          mail.ID,
		IsRead:      mail.IsRead,
		MailID:      int32(mail.MailID),
		Subject:     mail.Subject,
		Timestamp:   mail.Timestamp,
	}
	return m
}
