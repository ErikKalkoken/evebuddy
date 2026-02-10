package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateEveDogmaAttributeParams struct {
	ID           int64
	DefaultValue optional.Optional[float64]
	Description  optional.Optional[string]
	DisplayName  optional.Optional[string]
	IconID       optional.Optional[int64]
	Name         optional.Optional[string]
	IsHighGood   optional.Optional[bool]
	IsPublished  optional.Optional[bool]
	IsStackable  optional.Optional[bool]
	UnitID       app.EveUnitID
}

func (st *Storage) CreateEveDogmaAttribute(ctx context.Context, arg CreateEveDogmaAttributeParams) (*app.EveDogmaAttribute, error) {
	if arg.ID == 0 {
		return nil, fmt.Errorf("CreateEveDogmaAttribute: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateEveDogmaAttributeParams{
		ID:           arg.ID,
		DefaultValue: arg.DefaultValue.ValueOrZero(),
		Description:  arg.Description.ValueOrZero(),
		DisplayName:  arg.DisplayName.ValueOrZero(),
		IconID:       arg.IconID.ValueOrZero(),
		Name:         arg.Name.ValueOrZero(),
		IsHighGood:   arg.IsHighGood.ValueOrZero(),
		IsPublished:  arg.IsPublished.ValueOrZero(),
		IsStackable:  arg.IsStackable.ValueOrZero(),
		UnitID:       int64(arg.UnitID),
	}
	o, err := st.qRW.CreateEveDogmaAttribute(ctx, arg2)
	if err != nil {
		return nil, fmt.Errorf("CreateEveDogmaAttribute: %+v: %w", arg, err)
	}
	return eveDogmaAttributeFromDBModel(o), nil
}

func (st *Storage) GetEveDogmaAttribute(ctx context.Context, id int64) (*app.EveDogmaAttribute, error) {
	c, err := st.qRO.GetEveDogmaAttribute(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get EveDogmaAttribute for id %d: %w", id, convertGetError(err))
	}
	return eveDogmaAttributeFromDBModel(c), nil
}

func eveDogmaAttributeFromDBModel(eda queries.EveDogmaAttribute) *app.EveDogmaAttribute {
	return &app.EveDogmaAttribute{
		ID:           eda.ID,
		DefaultValue: optional.FromZeroValue(eda.DefaultValue),
		Description:  optional.FromZeroValue(eda.Description),
		DisplayName:  optional.FromZeroValue(eda.DisplayName),
		IconID:       optional.FromZeroValue(eda.IconID),
		Name:         optional.FromZeroValue(eda.Name),
		IsHighGood:   optional.New(eda.IsHighGood),
		IsPublished:  optional.New(eda.IsPublished),
		IsStackable:  optional.New(eda.IsStackable),
		Unit:         app.EveUnitID(eda.UnitID),
	}
}
