package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func (st *Storage) DeleteObsoleteCharacterMailLabels(ctx context.Context, characterID int64) error {
	arg := queries.DeleteObsoleteCharacterMailLabelsParams{
		CharacterID:   characterID,
		CharacterID_2: characterID,
	}
	if err := st.qRW.DeleteObsoleteCharacterMailLabels(ctx, arg); err != nil {
		return fmt.Errorf("delete obsolete mail labels for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) GetCharacterMailLabel(ctx context.Context, characterID, labelID int64) (*app.CharacterMailLabel, error) {
	arg := queries.GetCharacterMailLabelParams{
		CharacterID: characterID,
		LabelID:     labelID,
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
	CharacterID int64
	Color       optional.Optional[string]
	LabelID     int64
	Name        optional.Optional[string]
	UnreadCount optional.Optional[int64]
}

func (st *Storage) GetOrCreateCharacterMailLabel(ctx context.Context, arg MailLabelParams) (*app.CharacterMailLabel, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("get or create mail label for character %d and label %d: %w", arg.CharacterID, arg.LabelID, err)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return nil, wrapErr(err)
	}
	defer tx.Rollback()
	l, err := getOrCreateCharacterMailLabel(ctx, st.qRW.WithTx(tx), arg)
	if err != nil {
		return nil, wrapErr(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, wrapErr(err)
	}
	return characterMailLabelFromDBModel(l), nil
}

// getOrCreateCharacterMailLabel returns a mail label and creates it when it does not exist.
// Must be called within a transaction.
func getOrCreateCharacterMailLabel(ctx context.Context, q *queries.Queries, arg MailLabelParams) (queries.CharacterMailLabel, error) {
	if arg.CharacterID == 0 || arg.LabelID == 0 {
		return queries.CharacterMailLabel{}, app.ErrInvalid
	}
	l, err := q.GetCharacterMailLabel(ctx, queries.GetCharacterMailLabelParams{
		CharacterID: arg.CharacterID,
		LabelID:     arg.LabelID,
	})
	if err == nil {
		return l, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return queries.CharacterMailLabel{}, err
	}
	return q.CreateCharacterMailLabel(ctx, queries.CreateCharacterMailLabelParams{
		CharacterID: arg.CharacterID,
		LabelID:     arg.LabelID,
		Color:       arg.Color.ValueOrZero(),
		Name:        arg.Name.ValueOrZero(),
		UnreadCount: arg.UnreadCount.ValueOrZero(),
	})
}

func (st *Storage) ListCharacterMailLabelsOrdered(ctx context.Context, characterID int64) ([]*app.CharacterMailLabel, error) {
	ll, err := st.qRO.ListCharacterMailLabelsOrdered(ctx, characterID)
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
	l, err := st.qRW.UpdateOrCreateCharacterMailLabel(ctx, queries.UpdateOrCreateCharacterMailLabelParams{
		CharacterID: arg.CharacterID,
		LabelID:     arg.LabelID,
		Color:       arg.Color.ValueOrZero(),
		Name:        arg.Name.ValueOrZero(),
		UnreadCount: arg.UnreadCount.ValueOrZero(),
	})
	if err != nil {
		return nil, fmt.Errorf("update or create mail label for character %d with label %d: %w", arg.CharacterID, arg.LabelID, err)
	}
	label := characterMailLabelFromDBModel(l)
	return label, nil
}

func characterMailLabelFromDBModel(l queries.CharacterMailLabel) *app.CharacterMailLabel {
	return &app.CharacterMailLabel{
		ID:          l.ID,
		CharacterID: l.CharacterID,
		Color:       optional.FromZeroValue(l.Color),
		LabelID:     l.LabelID,
		Name:        optional.FromZeroValue(l.Name),
		UnreadCount: optional.FromZeroValue(l.UnreadCount),
	}
}
