package storage

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type UpdateOrCreateCharacterContactLabelParams struct {
	CharacterID int64
	LabelID     int64
	Name        string
}

func (st *Storage) UpdateOrCreateCharacterContactLabel(ctx context.Context, arg UpdateOrCreateCharacterContactLabelParams) error {
	err := st.qRW.UpdateOrCreateCharacterContactLabel(ctx, queries.UpdateOrCreateCharacterContactLabelParams{
		CharacterID: arg.CharacterID,
		LabelID:     arg.LabelID,
		Name:        arg.Name,
	})
	if err != nil {
		return fmt.Errorf("UpdateOrCreateCharacterContactLabel: %v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetCharacterContactLabel(ctx context.Context, characterID, labelID int64) (*app.CharacterContactLabel, error) {
	r, err := st.qRO.GetCharacterContactLabel(ctx, queries.GetCharacterContactLabelParams{
		CharacterID: characterID,
		LabelID:     labelID,
	})
	if err != nil {
		return nil, fmt.Errorf("GetCharacterContactLabel: %d %d: %w", characterID, labelID, convertGetError(err))
	}
	return characterContactLabelFromDBModel(r), nil
}

func (st *Storage) ListCharacterContactLabels(ctx context.Context, characterID int64) ([]*app.CharacterContactLabel, error) {
	rows, err := st.qRO.ListCharacterContactLabels(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("ListCharacterContactLabels for character %d: %w", characterID, err)
	}
	var oo []*app.CharacterContactLabel
	for _, r := range rows {
		oo = append(oo, characterContactLabelFromDBModel(r))
	}
	return oo, nil
}

func (st *Storage) ListCharacterContactLabelIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	rows, err := st.qRO.ListCharacterContactLabelIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("ListCharacterContactLabels for character %d: %w", characterID, err)
	}
	x := set.Collect(slices.Values(rows))
	return x, nil
}

func (st *Storage) DeleteCharacterContactLabels(ctx context.Context, characterID int64, labelIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCharacterContactLabels for character %d and contact IDs: %v: %w", characterID, labelIDs, err)
	}
	if characterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if labelIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.DeleteCharacterContactLabels(ctx, queries.DeleteCharacterContactLabelsParams{
		CharacterID: characterID,
		LabelIds:    slices.Collect(labelIDs.All()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Character labels deleted", "characterID", characterID, "labels", labelIDs)
	return nil
}

func characterContactLabelFromDBModel(r queries.CharacterContactLabel) *app.CharacterContactLabel {
	o2 := &app.CharacterContactLabel{
		CharacterID: r.CharacterID,
		LabelID:     r.LabelID,
		Name:        r.Name,
	}
	return o2
}
