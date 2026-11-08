package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveTypeDogmaEffectParams struct {
	DogmaEffectID int64
	EveTypeID     int64
	IsDefault     bool
}

func (st *Storage) CreateEveTypeDogmaEffect(ctx context.Context, arg CreateEveTypeDogmaEffectParams) error {
	return createEveTypeDogmaEffect(ctx, st.qRW, arg)
}

func createEveTypeDogmaEffect(ctx context.Context, q *queries.Queries, arg CreateEveTypeDogmaEffectParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("createEveTypeDogmaEffect: %+v: %w", arg, err)
	}
	if arg.DogmaEffectID == 0 || arg.EveTypeID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := q.CreateEveTypeDogmaEffect(ctx, queries.CreateEveTypeDogmaEffectParams{
		DogmaEffectID: arg.DogmaEffectID,
		EveTypeID:     arg.EveTypeID,
		IsDefault:     arg.IsDefault,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetEveTypeDogmaEffect(ctx context.Context, eveTypeID, dogmaAttributeID int64) (bool, error) {
	arg := queries.GetEveTypeDogmaEffectParams{
		DogmaEffectID: dogmaAttributeID,
		EveTypeID:     eveTypeID,
	}
	row, err := st.qRO.GetEveTypeDogmaEffect(ctx, arg)
	if err != nil {
		return false, fmt.Errorf("get EveTypeDogmaEffect for %+v: %w", arg, convertGetError(err))
	}
	return row.IsDefault, nil
}
