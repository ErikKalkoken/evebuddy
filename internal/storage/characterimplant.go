package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateCharacterImplantParams struct {
	CharacterID int32
	EveTypeID   int32
}

func (r *Storage) CreateCharacterImplant(ctx context.Context, arg CreateCharacterImplantParams) error {
	return createCharacterImplant(ctx, r.q, arg)
}

func (r *Storage) GetCharacterImplant(ctx context.Context, characterID int32, eveTypeID int32) (*model.CharacterImplant, error) {
	arg := queries.GetCharacterImplantParams{
		CharacterID:      int64(characterID),
		DogmaAttributeID: model.EveDogmaAttributeIDImplantSlot,
		EveTypeID:        int64(eveTypeID),
	}
	row, err := r.q.GetCharacterImplant(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get CharacterSkill for character %d: %w", characterID, err)
	}
	t2 := characterImplantFromDBModel(
		row.CharacterImplant,
		row.EveType,
		row.EveGroup,
		row.EveCategory, row.SlotNum)
	return t2, nil
}

func (r *Storage) ListCharacterImplants(ctx context.Context, characterID int32) ([]*model.CharacterImplant, error) {
	arg := queries.ListCharacterImplantsParams{
		DogmaAttributeID: model.EveDogmaAttributeIDImplantSlot,
		CharacterID:      int64(characterID),
	}
	rows, err := r.q.ListCharacterImplants(ctx, arg)
	if err != nil {
		return nil, err
	}
	ii2 := make([]*model.CharacterImplant, len(rows))
	for i, row := range rows {
		ii2[i] = characterImplantFromDBModel(
			row.CharacterImplant,
			row.EveType,
			row.EveGroup,
			row.EveCategory,
			row.SlotNum)
	}
	return ii2, nil
}

func (r *Storage) ReplaceCharacterImplants(ctx context.Context, characterID int32, args []CreateCharacterImplantParams) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := r.q.WithTx(tx)
	if err := qtx.DeleteCharacterImplants(ctx, int64(characterID)); err != nil {
		return err
	}
	for _, arg := range args {
		err := createCharacterImplant(ctx, qtx, arg)
		if err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func createCharacterImplant(ctx context.Context, q *queries.Queries, arg CreateCharacterImplantParams) error {
	if arg.CharacterID == 0 || arg.EveTypeID == 0 {
		return fmt.Errorf("IDs must not be zero %v", arg)
	}
	arg2 := queries.CreateCharacterImplantParams{
		CharacterID: int64(arg.CharacterID),
		EveTypeID:   int64(arg.EveTypeID),
	}
	err := q.CreateCharacterImplant(ctx, arg2)
	if err != nil {
		return fmt.Errorf("failed to create character implant %v, %w", arg, err)
	}
	return nil
}

func characterImplantFromDBModel(
	o queries.CharacterImplant,
	t queries.EveType,
	g queries.EveGroup,
	c queries.EveCategory,
	s sql.NullFloat64,
) *model.CharacterImplant {
	if o.CharacterID == 0 {
		panic("missing character ID")
	}
	o2 := &model.CharacterImplant{
		CharacterID: int32(o.CharacterID),
		EveType:     eveTypeFromDBModel(t, g, c),
		ID:          o.ID,
	}
	if s.Valid {
		o2.SlotNum = int(s.Float64)
	}
	return o2
}
