package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateEveTypeDogmaAttributeParams struct {
	DogmaAttributeID int32
	EveTypeID        int32
	Value            float32
}

func (r *Storage) CreateEveTypeDogmaAttribute(ctx context.Context, arg CreateEveTypeDogmaAttributeParams) error {
	if arg.DogmaAttributeID == 0 || arg.EveTypeID == 0 {
		return fmt.Errorf("invalid IDs for EveTypeDogmaAttribute: %v", arg)
	}
	arg2 := queries.CreateEveTypeDogmaAttributeParams{
		DogmaAttributeID: int64(arg.DogmaAttributeID),
		EveTypeID:        int64(arg.EveTypeID),
		Value:            float64(arg.Value),
	}
	err := r.q.CreateEveTypeDogmaAttribute(ctx, arg2)
	if err != nil {
		return fmt.Errorf("failed to create EveTypeDogmaAttribute %v, %w", arg, err)
	}
	return nil
}

func (r *Storage) GetEveTypeDogmaAttribute(ctx context.Context, dogmaAttributeID, eveTypeID int32) (float32, error) {
	arg := queries.GetEveTypeDogmaAttributeParams{
		DogmaAttributeID: int64(dogmaAttributeID),
		EveTypeID:        int64(eveTypeID),
	}
	row, err := r.q.GetEveTypeDogmaAttribute(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return 0, fmt.Errorf("failed to get EveTypeDogmaAttribute for %v: %w", arg, err)
	}
	return float32(row.Value), nil
}
