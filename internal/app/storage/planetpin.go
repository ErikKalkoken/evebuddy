package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreatePlanetPinParams struct {
	CharacterPlanetID      int64
	ExpiryTime             optional.Optional[time.Time]
	ExtractorProductTypeID optional.Optional[int64]
	FactorySchematicID     optional.Optional[int64]
	InstallTime            optional.Optional[time.Time]
	LastCycleStart         optional.Optional[time.Time]
	PinID                  int64
	SchematicID            optional.Optional[int64]
	TypeID                 int64
}

func (st *Storage) CreatePlanetPin(ctx context.Context, arg CreatePlanetPinParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreatePlanetPin: %+v: %w", arg, err)
	}
	if arg.CharacterPlanetID == 0 || arg.PinID == 0 || arg.TypeID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.CreatePlanetPin(ctx, queries.CreatePlanetPinParams{
		CharacterPlanetID:      arg.CharacterPlanetID,
		ExpiryTime:             optional.ToNullTime(arg.ExpiryTime),
		ExtractorProductTypeID: optional.ToNullInt64(arg.ExtractorProductTypeID),
		FactorySchemaID:        optional.ToNullInt64(arg.FactorySchematicID),
		InstallTime:            optional.ToNullTime(arg.InstallTime),
		LastCycleStart:         optional.ToNullTime(arg.LastCycleStart),
		PinID:                  arg.PinID,
		SchematicID:            optional.ToNullInt64(arg.SchematicID),
		TypeID:                 arg.TypeID,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeletePlanetPins(ctx context.Context, characterPlanetID int64) error {
	if err := st.qRW.DeletePlanetPins(ctx, characterPlanetID); err != nil {
		return fmt.Errorf("delete planet pins for %d: %w", characterPlanetID, err)
	}
	return nil
}

func (st *Storage) GetPlanetPin(ctx context.Context, characterPlanetID, pinID int64) (*app.PlanetPin, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetPlanetPin: %d %d: %w", characterPlanetID, pinID, err)
	}
	if characterPlanetID == 0 || pinID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetPlanetPin(ctx, queries.GetPlanetPinParams{
		CharacterPlanetID: characterPlanetID,
		PinID:             pinID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	return st.planetPinFromDBModel(ctx, r)
}

func (st *Storage) ListPlanetPins(ctx context.Context, characterPlanetID int64) ([]*app.PlanetPin, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListPlanetPins: %d: %w", characterPlanetID, err)
	}
	if characterPlanetID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListPlanetPins(ctx, characterPlanetID)
	if err != nil {
		return nil, wrapErr(err)
	}
	var oo []*app.PlanetPin
	for _, r := range rows {
		o, err := st.planetPinFromDBModel(ctx, queries.GetPlanetPinRow(r))
		if err != nil {
			return nil, wrapErr(err)
		}
		oo = append(oo, o)
	}
	return oo, nil
}

func (st *Storage) planetPinFromDBModel(ctx context.Context, r queries.GetPlanetPinRow) (*app.PlanetPin, error) {
	o := &app.PlanetPin{
		ID:             r.PlanetPin.PinID,
		ExpiryTime:     optional.FromNullTime(r.PlanetPin.ExpiryTime),
		InstallTime:    optional.FromNullTime(r.PlanetPin.InstallTime),
		LastCycleStart: optional.FromNullTime(r.PlanetPin.LastCycleStart),
		Type:           eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory),
	}
	if r.SchematicName.Valid {
		o.Schematic.Set(eveSchematicFromDBModel(queries.EveSchematic{
			ID:        r.PlanetPin.SchematicID.Int64,
			Name:      r.SchematicName.String,
			CycleTime: r.SchematicCycle.Int64,
		}))
	}
	if r.FactorySchematicName.Valid {
		o.FactorySchematic.Set(eveSchematicFromDBModel(queries.EveSchematic{
			ID:        r.PlanetPin.FactorySchemaID.Int64,
			Name:      r.FactorySchematicName.String,
			CycleTime: r.FactorySchematicCycle.Int64,
		}))
	}
	if r.PlanetPin.ExtractorProductTypeID.Valid {
		et, err := st.GetEveType(ctx, r.PlanetPin.ExtractorProductTypeID.Int64)
		if err != nil {
			return nil, err
		}
		o.ExtractorProductType.Set(et)
	}
	return o, nil
}
