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
	oo := make([]*app.CharacterContactLabel, len(rows))
	for i, r := range rows {
		oo[i] = characterContactLabelFromDBModel(r)
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

// ReplaceCharacterContactLabels replaces the contact labels of a character with labels.
// labels maps label IDs to names. Labels still used by contacts are not deleted.
// Reports whether anything changed.
func (st *Storage) ReplaceCharacterContactLabels(ctx context.Context, characterID int64, labels map[int64]string) (bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceCharacterContactLabels: character %d: %w", characterID, err)
	}
	if _, ok := labels[0]; characterID == 0 || ok {
		return false, wrapErr(app.ErrInvalid)
	}
	current, used, err := st.listCharacterContactLabelStates(ctx, characterID)
	if err != nil {
		return false, wrapErr(err)
	}
	var removed []int64
	for id := range current {
		// labels still used by contacts are kept, since contacts and labels can be out of sync
		if _, ok := labels[id]; !ok && !used.Contains(id) {
			removed = append(removed, id)
		}
	}
	changed := make(map[int64]string)
	for id, name := range labels {
		if n, ok := current[id]; !ok || n != name {
			changed[id] = name
		}
	}
	if len(removed) == 0 && len(changed) == 0 {
		return false, nil
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return false, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	for ids := range slices.Chunk(removed, st.MaxIDsPerQuery) {
		err := qtx.DeleteCharacterContactLabels(ctx, queries.DeleteCharacterContactLabelsParams{
			CharacterID: characterID,
			LabelIds:    ids,
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("delete %d labels: %w", len(ids), err))
		}
	}
	for id, name := range changed {
		err := qtx.UpdateOrCreateCharacterContactLabel(ctx, queries.UpdateOrCreateCharacterContactLabelParams{
			CharacterID: characterID,
			LabelID:     id,
			Name:        name,
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("update or create label %d: %w", id, err))
		}
	}
	if err := tx.Commit(); err != nil {
		return false, wrapErr(err)
	}
	slog.Info("Contact labels replaced", "characterID", characterID, "changed", len(changed), "removed", len(removed))
	return true, nil
}

// listCharacterContactLabelStates returns the names of a character's contact labels by label ID
// and the IDs of labels used by the character's contacts.
func (st *Storage) listCharacterContactLabelStates(ctx context.Context, characterID int64) (map[int64]string, set.Set[int64], error) {
	tx, err := st.dbRO.Begin()
	if err != nil {
		return nil, set.Set[int64]{}, err
	}
	defer tx.Rollback()
	qtx := st.qRO.WithTx(tx)
	rows, err := qtx.ListCharacterContactLabels(ctx, characterID)
	if err != nil {
		return nil, set.Set[int64]{}, err
	}
	mapRows, err := qtx.ListCharacterContactContactLabelIdsForCharacter(ctx, characterID)
	if err != nil {
		return nil, set.Set[int64]{}, err
	}
	if err := tx.Commit(); err != nil {
		return nil, set.Set[int64]{}, err
	}
	names := make(map[int64]string, len(rows))
	for _, r := range rows {
		names[r.LabelID] = r.Name
	}
	var used set.Set[int64]
	for _, r := range mapRows {
		used.Add(r.LabelID)
	}
	return names, used, nil
}

func characterContactLabelFromDBModel(r queries.CharacterContactLabel) *app.CharacterContactLabel {
	o2 := &app.CharacterContactLabel{
		CharacterID: r.CharacterID,
		LabelID:     r.LabelID,
		Name:        r.Name,
	}
	return o2
}
