package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreatePlanetPinParams struct {
	CharacterPlanetID      int64
	ExtractorProductTypeID optional.Optional[int32]
	FactorySchemaID        optional.Optional[int32]
	ExpiryTime             time.Time
	InstallTime            time.Time
	LastCycleStart         time.Time
	PinID                  int64
	SchematicID            optional.Optional[int32]
	TypeID                 int32
}

func (st *Storage) CreatePlanetPin(ctx context.Context, arg CreatePlanetPinParams) error {
	if arg.CharacterPlanetID == 0 {
		return fmt.Errorf("create PlanetPin - invalid IDs %+v", arg)
	}
	arg2 := queries.CreatePlanetPinParams{
		CharacterPlanetID:      arg.CharacterPlanetID,
		ExtractorProductTypeID: optional.ToNullInt64(arg.ExtractorProductTypeID),
		FactorySchemaID:        optional.ToNullInt64(arg.FactorySchemaID),
		SchematicID:            optional.ToNullInt64(arg.SchematicID),
		TypeID:                 int64(arg.TypeID),
		ExpiryTime:             NewNullTime(arg.ExpiryTime),
		InstallTime:            NewNullTime(arg.InstallTime),
		LastCycleStart:         NewNullTime(arg.LastCycleStart),
		PinID:                  arg.PinID,
	}
	if err := st.q.CreatePlanetPin(ctx, arg2); err != nil {
		return fmt.Errorf("create PlanetPin %v, %w", arg, err)
	}
	return nil
}

func (st *Storage) GetPlanetPin(ctx context.Context, characterPlanetID, pinID int64) (*app.PlanetPin, error) {
	arg := queries.GetPlanetPinParams{
		CharacterPlanetID: characterPlanetID,
		PinID:             pinID,
	}
	r, err := st.q.GetPlanetPin(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("get PlanetPin for %+v: %w", arg, err)
	}
	return st.planetPinFromDBModel(ctx, r)
}

func (st *Storage) ListPlanetPins(ctx context.Context, characterPlanetID int64) ([]*app.PlanetPin, error) {
	rows, err := st.q.ListPlanetPins(ctx, characterPlanetID)
	if err != nil {
		return nil, fmt.Errorf("list planet pins for %d: %w", characterPlanetID, err)
	}
	oo := make([]*app.PlanetPin, len(rows))
	for i, r := range rows {
		o, err := st.planetPinFromDBModel(ctx, queries.GetPlanetPinRow(r))
		if err != nil {
			return nil, err
		}
		oo[i] = o
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
		o.Schematic = eveSchematicFromDBModel(queries.EveSchematic{
			ID:        r.PlanetPin.SchematicID.Int64,
			Name:      r.SchematicName.String,
			CycleTime: r.SchematicCycle.Int64,
		})
	}
	if r.FactorySchematicName.Valid {
		o.FactorySchematic = eveSchematicFromDBModel(queries.EveSchematic{
			ID:        r.PlanetPin.FactorySchemaID.Int64,
			Name:      r.FactorySchematicName.String,
			CycleTime: r.FactorySchematicCycle.Int64,
		})
	}
	if r.PlanetPin.ExtractorProductTypeID.Valid {
		et, err := st.GetEveType(ctx, int32(r.PlanetPin.ExtractorProductTypeID.Int64))
		if err != nil {
			return nil, err
		}
		o.ExtractorProductType = et
	}
	return o, nil
}
