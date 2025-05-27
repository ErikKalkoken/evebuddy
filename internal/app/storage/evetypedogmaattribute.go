package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveTypeDogmaAttributeParams struct {
	DogmaAttributeID int32
	EveTypeID        int32
	Value            float32
}

func (st *Storage) CreateEveTypeDogmaAttribute(ctx context.Context, arg CreateEveTypeDogmaAttributeParams) error {
	if arg.DogmaAttributeID == 0 || arg.EveTypeID == 0 {
		return fmt.Errorf("CreateEveTypeDogmaAttribute: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateEveTypeDogmaAttributeParams{
		DogmaAttributeID: int64(arg.DogmaAttributeID),
		EveTypeID:        int64(arg.EveTypeID),
		Value:            float64(arg.Value),
	}
	err := st.qRW.CreateEveTypeDogmaAttribute(ctx, arg2)
	if err != nil {
		return fmt.Errorf("CreateEveTypeDogmaAttribute: %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetEveTypeDogmaAttribute(ctx context.Context, eveTypeID, dogmaAttributeID int32) (float32, error) {
	arg := queries.GetEveTypeDogmaAttributeParams{
		DogmaAttributeID: int64(dogmaAttributeID),
		EveTypeID:        int64(eveTypeID),
	}
	row, err := st.qRO.GetEveTypeDogmaAttribute(ctx, arg)
	if err != nil {
		return 0, fmt.Errorf("get EveTypeDogmaAttribute for %v: %w", arg, convertGetError(err))
	}
	return float32(row.Value), nil
}

func (st *Storage) ListEveTypeDogmaAttributesForType(ctx context.Context, typeID int32) ([]*app.EveTypeDogmaAttribute, error) {
	rows, err := st.qRO.ListEveTypeDogmaAttributesForType(ctx, int64(typeID))
	if err != nil {
		return nil, fmt.Errorf("list dogma attributes for type %d: %w", typeID, err)
	}
	oo := make([]*app.EveTypeDogmaAttribute, len(rows))
	for i, r := range rows {
		o := &app.EveTypeDogmaAttribute{
			DogmaAttribute: eveDogmaAttributeFromDBModel(r.EveDogmaAttribute),
			EveType:        eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory),
			Value:          float32(r.Value),
		}
		oo[i] = o
	}
	return oo, nil
}
