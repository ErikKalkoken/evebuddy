package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveTypeDogmaEffectParams struct {
	DogmaEffectID int32
	EveTypeID     int32
	IsDefault     bool
}

func (st *Storage) CreateEveTypeDogmaEffect(ctx context.Context, arg CreateEveTypeDogmaEffectParams) error {
	if arg.DogmaEffectID == 0 || arg.EveTypeID == 0 {
		return fmt.Errorf("CreateEveTypeDogmaEffect: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateEveTypeDogmaEffectParams{
		DogmaEffectID: int64(arg.DogmaEffectID),
		EveTypeID:     int64(arg.EveTypeID),
		IsDefault:     arg.IsDefault,
	}
	err := st.qRW.CreateEveTypeDogmaEffect(ctx, arg2)
	if err != nil {
		return fmt.Errorf("CreateEveTypeDogmaEffect: %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetEveTypeDogmaEffect(ctx context.Context, eveTypeID, dogmaAttributeID int32) (bool, error) {
	arg := queries.GetEveTypeDogmaEffectParams{
		DogmaEffectID: int64(dogmaAttributeID),
		EveTypeID:     int64(eveTypeID),
	}
	row, err := st.qRO.GetEveTypeDogmaEffect(ctx, arg)
	if err != nil {
		return false, fmt.Errorf("get EveTypeDogmaEffect for %v: %w", arg, convertGetError(err))
	}
	return row.IsDefault, nil
}
