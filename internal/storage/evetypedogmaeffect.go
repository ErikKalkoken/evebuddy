package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateEveTypeDogmaEffectParams struct {
	DogmaEffectID int32
	EveTypeID     int32
	IsDefault     bool
}

func (r *Storage) CreateEveTypeDogmaEffect(ctx context.Context, arg CreateEveTypeDogmaEffectParams) error {
	if arg.DogmaEffectID == 0 || arg.EveTypeID == 0 {
		return fmt.Errorf("invalid IDs for EveTypeDogmaEffect: %v", arg)
	}
	arg2 := queries.CreateEveTypeDogmaEffectParams{
		DogmaEffectID: int64(arg.DogmaEffectID),
		EveTypeID:     int64(arg.EveTypeID),
		IsDefault:     arg.IsDefault,
	}
	err := r.q.CreateEveTypeDogmaEffect(ctx, arg2)
	if err != nil {
		return fmt.Errorf("failed to create EveTypeDogmaEffect %v, %w", arg, err)
	}
	return nil
}

func (r *Storage) GetEveTypeDogmaEffect(ctx context.Context, eveTypeID, dogmaAttributeID int32) (bool, error) {
	arg := queries.GetEveTypeDogmaEffectParams{
		DogmaEffectID: int64(dogmaAttributeID),
		EveTypeID:     int64(eveTypeID),
	}
	row, err := r.q.GetEveTypeDogmaEffect(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return false, fmt.Errorf("failed to get EveTypeDogmaEffect for %v: %w", arg, err)
	}
	return row.IsDefault, nil
}
