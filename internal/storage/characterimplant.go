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
	if arg.CharacterID == 0 || arg.EveTypeID == 0 {
		return fmt.Errorf("IDs must not be zero %v", arg)
	}
	arg2 := queries.CreateCharacterImplantParams{
		CharacterID: int64(arg.CharacterID),
		EveTypeID:   int64(arg.EveTypeID),
	}
	err := r.q.CreateCharacterImplant(ctx, arg2)
	if err != nil {
		return fmt.Errorf("failed to create character implant %v, %w", arg, err)
	}
	return nil
}

func (r *Storage) DeleteCharacterImplants(ctx context.Context, characterID int32) error {
	if err := r.q.DeleteCharacterImplants(ctx, int64(characterID)); err != nil {
		return fmt.Errorf("failed to delete implants for character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) GetCharacterImplant(ctx context.Context, characterID int32, eveTypeID int32) (*model.CharacterImplant, error) {
	arg := queries.GetCharacterImplantParams{
		CharacterID: int64(characterID),
		EveTypeID:   int64(eveTypeID),
	}
	row, err := r.q.GetCharacterImplant(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get CharacterSkill for character %d: %w", characterID, err)
	}
	t2 := characterImplantFromDBModel(row.CharacterImplant, row.EveType, row.EveGroup, row.EveCategory)
	return t2, nil
}

func characterImplantFromDBModel(o queries.CharacterImplant, t queries.EveType, g queries.EveGroup, c queries.EveCategory) *model.CharacterImplant {
	if o.CharacterID == 0 {
		panic("missing character ID")
	}
	return &model.CharacterImplant{
		CharacterID: int32(o.CharacterID),
		EveType:     eveTypeFromDBModel(t, g, c),
		ID:          o.ID,
	}
}
