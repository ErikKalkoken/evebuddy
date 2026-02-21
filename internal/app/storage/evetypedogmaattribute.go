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
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateEveTypeDogmaAttribute: %+v: %w", arg, err)
	}
	if arg.DogmaAttributeID == 0 || arg.EveTypeID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.CreateEveTypeDogmaAttribute(ctx, queries.CreateEveTypeDogmaAttributeParams{
		DogmaAttributeID: arg.DogmaAttributeID,
		EveTypeID:        arg.EveTypeID,
		Value:            arg.Value,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetEveTypeDogmaAttribute(ctx context.Context, typeID, dogmaAttributeID int64) (float64, error) {
	arg := queries.GetEveTypeDogmaAttributeParams{
		DogmaAttributeID: dogmaAttributeID,
		EveTypeID:        typeID,
	}
	wrapErr := func(err error) error {
		return fmt.Errorf("GetEveTypeDogmaAttribute: %+v: %w", arg, err)
	}
	if arg.DogmaAttributeID == 0 || arg.EveTypeID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	row, err := st.qRO.GetEveTypeDogmaAttribute(ctx, arg)
	if err != nil {
		return 0, wrapErr(convertGetError(err))
	}
	return row.Value, nil
}

func (st *Storage) ListEveTypeDogmaAttributesForType(ctx context.Context, typeID int64) ([]*app.EveTypeDogmaAttribute, error) {
	rows, err := st.qRO.ListEveTypeDogmaAttributesForType(ctx, typeID)
	if err != nil {
		return nil, fmt.Errorf("list dogma attributes for type %d: %w", typeID, err)
	}
	var oo []*app.EveTypeDogmaAttribute
	for _, r := range rows {
		oo = append(oo, &app.EveTypeDogmaAttribute{
			DogmaAttribute: eveDogmaAttributeFromDBModel(r.EveDogmaAttribute),
			Type:        eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory),
			Value:          r.Value,
		})
	}
	return oo, nil
}

func (st *Storage) ListEveTypeDogmaAttributesForSkills(ctx context.Context) ([]*app.EveTypeDogmaAttribute, error) {
	rows, err := st.qRO.ListEveTypeDogmaAttributesForSkills(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListEveTypeDogmaAttributesForSkills: %w", err)
	}
	var oo []*app.EveTypeDogmaAttribute
	for _, r := range rows {
		oo = append(oo, &app.EveTypeDogmaAttribute{
			DogmaAttribute: eveDogmaAttributeFromDBModel(r.EveDogmaAttribute),
			Type:        eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory),
			Value:          r.Value,
		})
	}
	return oo, nil
}
