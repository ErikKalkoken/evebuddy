package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveTypeDogmaAttributeParams struct {
	DogmaAttributeID int64
	EveTypeID        int64
	Value            float64
}

func (st *Storage) CreateEveTypeDogmaAttribute(ctx context.Context, arg CreateEveTypeDogmaAttributeParams) error {
	if arg.DogmaAttributeID == 0 || arg.EveTypeID == 0 {
		return fmt.Errorf("CreateEveTypeDogmaAttribute: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateEveTypeDogmaAttributeParams{
		DogmaAttributeID: arg.DogmaAttributeID,
		EveTypeID:        arg.EveTypeID,
		Value:            arg.Value,
	}
	err := st.qRW.CreateEveTypeDogmaAttribute(ctx, arg2)
	if err != nil {
		return fmt.Errorf("CreateEveTypeDogmaAttribute: %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetEveTypeDogmaAttribute(ctx context.Context, eveTypeID, dogmaAttributeID int64) (float64, error) {
	arg := queries.GetEveTypeDogmaAttributeParams{
		DogmaAttributeID: dogmaAttributeID,
		EveTypeID:        eveTypeID,
	}
	row, err := st.qRO.GetEveTypeDogmaAttribute(ctx, arg)
	if err != nil {
		return 0, fmt.Errorf("get EveTypeDogmaAttribute for %+v: %w", arg, convertGetError(err))
	}
	return row.Value, nil
}

func (st *Storage) ListEveTypeDogmaAttributesForType(ctx context.Context, typeID int64) ([]*app.EveTypeDogmaAttribute, error) {
	rows, err := st.qRO.ListEveTypeDogmaAttributesForType(ctx,typeID)
	if err != nil {
		return nil, fmt.Errorf("list dogma attributes for type %d: %w", typeID, err)
	}
	oo := make([]*app.EveTypeDogmaAttribute, len(rows))
	for i, r := range rows {
		o := &app.EveTypeDogmaAttribute{
			DogmaAttribute: eveDogmaAttributeFromDBModel(r.EveDogmaAttribute),
			EveType:        eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory),
			Value:          r.Value,
		}
		oo[i] = o
	}
	return oo, nil
}
