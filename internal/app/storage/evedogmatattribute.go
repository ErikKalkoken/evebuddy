package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveDogmaAttributeParams struct {
	ID           int32
	DefaultValue float32
	Description  string
	DisplayName  string
	IconID       int32
	Name         string
	IsHighGood   bool
	IsPublished  bool
	IsStackable  bool
	UnitID       app.EveUnitID
}

func (st *Storage) CreateEveDogmaAttribute(ctx context.Context, arg CreateEveDogmaAttributeParams) (*app.EveDogmaAttribute, error) {
	if arg.ID == 0 {
		return nil, fmt.Errorf("CreateEveDogmaAttribute: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateEveDogmaAttributeParams{
		ID:           int64(arg.ID),
		DefaultValue: float64(arg.DefaultValue),
		Description:  arg.Description,
		DisplayName:  arg.DisplayName,
		IconID:       int64(arg.IconID),
		Name:         arg.Name,
		IsHighGood:   arg.IsHighGood,
		IsPublished:  arg.IsPublished,
		IsStackable:  arg.IsStackable,
		UnitID:       int64(arg.UnitID),
	}
	o, err := st.qRW.CreateEveDogmaAttribute(ctx, arg2)
	if err != nil {
		return nil, fmt.Errorf("CreateEveDogmaAttribute: %+v: %w", arg, err)
	}
	return eveDogmaAttributeFromDBModel(o), nil
}

func (st *Storage) GetEveDogmaAttribute(ctx context.Context, id int32) (*app.EveDogmaAttribute, error) {
	c, err := st.qRO.GetEveDogmaAttribute(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = app.ErrNotFound
		}
		return nil, fmt.Errorf("get EveDogmaAttribute for id %d: %w", id, err)
	}
	return eveDogmaAttributeFromDBModel(c), nil
}

func eveDogmaAttributeFromDBModel(eda queries.EveDogmaAttribute) *app.EveDogmaAttribute {
	return &app.EveDogmaAttribute{
		ID:           int32(eda.ID),
		DefaultValue: float32(eda.DefaultValue),
		Description:  eda.Description,
		DisplayName:  eda.DisplayName,
		IconID:       int32(eda.IconID),
		Name:         eda.Name,
		IsHighGood:   eda.IsHighGood,
		IsPublished:  eda.IsPublished,
		IsStackable:  eda.IsStackable,
		Unit:         app.EveUnitID(eda.UnitID),
	}
}
