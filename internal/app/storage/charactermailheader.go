package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

// ListCharacterMailHeadersForLabelOrdered returns a character's mails for a label
// in descending order by timestamp.
// Return mails for all labels, when labelID = 0
func (st *Storage) ListCharacterMailHeadersForLabelOrdered(ctx context.Context, characterID int64, labelID int64) ([]*app.CharacterMailHeader, error) {
	switch labelID {
	case app.MailLabelAll:
		rows, err := st.qRO.ListMailsOrdered(ctx,characterID)
		if err != nil {
			return nil, fmt.Errorf("list mails for character %d: %w", characterID, err)
		}
		mm := make([]*app.CharacterMailHeader, len(rows))
		for i, r := range rows {
			mm[i] = characterMailHeaderFromDBModel(characterID, r.CharacterMail, r.EveEntity)
		}
		return mm, nil
	case app.MailLabelNone:
		rows, err := st.qRO.ListMailsNoLabelOrdered(ctx,characterID)
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
			CharacterID:characterID,
			LabelID:    labelID,
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

func (st *Storage) ListCharacterMailHeadersForListOrdered(ctx context.Context, characterID int64, listID int64) ([]*app.CharacterMailHeader, error) {
	arg := queries.ListMailsForListOrderedParams{
		CharacterID:characterID,
		EveEntityID:listID,
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

// ListAllCharacterMailHeadersForLabelOrdered returns the mails of all characters for a label
// in descending order by timestamp.
func (st *Storage) ListAllCharacterMailHeadersForLabelOrdered(ctx context.Context, labelID int64) ([]*app.CharacterMailHeader, error) {
	if labelID == app.MailLabelAll {
		rows, err := st.qRO.ListAllMailsOrdered(ctx)
		if err != nil {
			return nil, fmt.Errorf("list mails for all characters: %w", err)
		}
		mm := make([]*app.CharacterMailHeader, len(rows))
		for i, r := range rows {
			mm[i] = characterMailHeaderFromDBModel(r.CharacterMail.CharacterID, r.CharacterMail, r.EveEntity)
		}
		return mm, nil
	}
	rows, err := st.qRO.ListAllMailsForLabelOrdered(ctx, labelID)
	if err != nil {
		return nil, fmt.Errorf("list mails for all characters and label %d: %w", labelID, err)
	}
	mm := make([]*app.CharacterMailHeader, len(rows))
	for i, r := range rows {
		mm[i] = characterMailHeaderFromDBModel(r.CharacterMail.CharacterID, r.CharacterMail, r.EveEntity)
	}
	return mm, nil
}

// ListAllCharacterMailHeadersForListOrdered returns the mails of all characters for a mailing list
// in descending order by timestamp.
func (st *Storage) ListAllCharacterMailHeadersForListOrdered(ctx context.Context, listID int64) ([]*app.CharacterMailHeader, error) {
	rows, err := st.qRO.ListAllMailsForListOrdered(ctx, listID)
	if err != nil {
		return nil, fmt.Errorf("list mails for all characters and list %d: %w", listID, err)
	}
	mm := make([]*app.CharacterMailHeader, len(rows))
	for i, r := range rows {
		mm[i] = characterMailHeaderFromDBModel(r.CharacterMail.CharacterID, r.CharacterMail, r.EveEntity)
	}
	return mm, nil
}

func (st *Storage) ListCharacterMailHeadersForUnprocessed(ctx context.Context, characterID int64, earliest time.Time) ([]*app.CharacterMailHeader, error) {
	arg := queries.ListMailsUnprocessedParams{
		CharacterID:characterID,
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
	characterID int64, mail queries.CharacterMail, from queries.EveEntity) *app.CharacterMailHeader {
	m := &app.CharacterMailHeader{
		CharacterID: characterID,
		From:        eveEntityFromDBModel(from),
		ID:          mail.ID,
		IsRead:      mail.IsRead,
		MailID:     mail.MailID,
		Subject:     mail.Subject,
		Timestamp:   mail.Timestamp,
	}
	return m
}
