package storage

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreatePlanetPinParams struct {
	CharacterPlanetID      int64
	Contents               map[int64]int64 // amount by type ID
	ExpiryTime             optional.Optional[time.Time]
	ExtractorCycleTime     optional.Optional[time.Duration]
	ExtractorHeadRadius    optional.Optional[float64]
	ExtractorNumHeads      optional.Optional[int64]
	ExtractorProductTypeID optional.Optional[int64]
	ExtractorQtyPerCycle   optional.Optional[int64]
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
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	if err := createPlanetPin(ctx, st.qRW.WithTx(tx), arg); err != nil {
		return wrapErr(err)
	}
	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	return nil
}

// createPlanetPin creates a pin with its contents. Must be called within a transaction.
func createPlanetPin(ctx context.Context, q *queries.Queries, arg CreatePlanetPinParams) error {
	if arg.CharacterPlanetID == 0 || arg.PinID == 0 || arg.TypeID == 0 {
		return app.ErrInvalid
	}
	var cycleTime optional.Optional[int64]
	if v, ok := arg.ExtractorCycleTime.Value(); ok {
		cycleTime.Set(int64(v.Seconds()))
	}
	id, err := q.CreatePlanetPin(ctx, queries.CreatePlanetPinParams{
		CharacterPlanetID:      arg.CharacterPlanetID,
		ExpiryTime:             optional.ToNullTime(arg.ExpiryTime),
		ExtractorCycleTime:     optional.ToNullInt64(cycleTime),
		ExtractorHeadRadius:    optional.ToNullFloat64(arg.ExtractorHeadRadius),
		ExtractorNumHeads:      optional.ToNullInt64(arg.ExtractorNumHeads),
		ExtractorProductTypeID: optional.ToNullInt64(arg.ExtractorProductTypeID),
		ExtractorQtyPerCycle:   optional.ToNullInt64(arg.ExtractorQtyPerCycle),
		FactorySchemaID:        optional.ToNullInt64(arg.FactorySchematicID),
		InstallTime:            optional.ToNullTime(arg.InstallTime),
		LastCycleStart:         optional.ToNullTime(arg.LastCycleStart),
		PinID:                  arg.PinID,
		SchematicID:            optional.ToNullInt64(arg.SchematicID),
		TypeID:                 arg.TypeID,
	})
	if err != nil {
		return err
	}
	for typeID, amount := range arg.Contents {
		err := q.CreatePlanetPinContent(ctx, queries.CreatePlanetPinContentParams{
			PlanetPinID: id,
			TypeID:      typeID,
			Amount:      amount,
		})
		if err != nil {
			return err
		}
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
	oo, err := st.planetPinsFromDBModels(ctx, st.qRO, []queries.GetPlanetPinRow{r})
	if err != nil {
		return nil, wrapErr(err)
	}
	return oo[0], nil
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
	if len(rows) == 0 {
		return nil, nil
	}
	rows2 := make([]queries.GetPlanetPinRow, len(rows))
	for i, r := range rows {
		rows2[i] = queries.GetPlanetPinRow(r)
	}
	oo, err := st.planetPinsFromDBModels(ctx, st.qRO, rows2)
	if err != nil {
		return nil, wrapErr(err)
	}
	return oo, nil
}

// listPlanetPinsByPlanet returns the pins for the given character planets, keyed by character planet ID.
func (st *Storage) listPlanetPinsByPlanet(ctx context.Context, q *queries.Queries, characterPlanetIDs set.Set[int64]) (map[int64][]*app.PlanetPin, error) {
	var rows []queries.GetPlanetPinRow
	for idsChunk := range slices.Chunk(slices.Collect(characterPlanetIDs.All()), st.MaxIDsPerQuery) {
		r, err := q.ListPlanetPinsForCharacterPlanetIDs(ctx, idsChunk)
		if err != nil {
			return nil, fmt.Errorf("list planet pins for %d character planets: %w", len(idsChunk), err)
		}
		for _, x := range r {
			rows = append(rows, queries.GetPlanetPinRow(x))
		}
	}
	pins, err := st.planetPinsFromDBModels(ctx, q, rows)
	if err != nil {
		return nil, err
	}
	m := make(map[int64][]*app.PlanetPin)
	for i, r := range rows {
		id := r.PlanetPin.CharacterPlanetID
		m[id] = append(m[id], pins[i])
	}
	return m, nil
}

// planetPinsFromDBModels converts rows to pins, batch loading extractor product types and contents.
func (st *Storage) planetPinsFromDBModels(ctx context.Context, q *queries.Queries, rows []queries.GetPlanetPinRow) ([]*app.PlanetPin, error) {
	var typeIDs, pinIDs set.Set[int64]
	for _, r := range rows {
		if r.PlanetPin.ExtractorProductTypeID.Valid {
			typeIDs.Add(r.PlanetPin.ExtractorProductTypeID.Int64)
		}
		pinIDs.Add(r.PlanetPin.ID)
	}
	typeMap, err := st.listEveTypesForIDs(ctx, q, typeIDs)
	if err != nil {
		return nil, err
	}
	contents, err := st.listPlanetPinContentsByPin(ctx, q, pinIDs)
	if err != nil {
		return nil, err
	}
	oo := make([]*app.PlanetPin, len(rows))
	for i, r := range rows {
		o := planetPinFromDBModel(r, typeMap)
		o.Contents = contents[r.PlanetPin.ID]
		oo[i] = o
	}
	return oo, nil
}

// listPlanetPinContentsByPin returns the contents for the given planet pins, keyed by planet pin row ID.
func (st *Storage) listPlanetPinContentsByPin(ctx context.Context, q *queries.Queries, planetPinIDs set.Set[int64]) (map[int64][]*app.PlanetPinContent, error) {
	m := make(map[int64][]*app.PlanetPinContent)
	for idsChunk := range slices.Chunk(slices.Collect(planetPinIDs.All()), st.MaxIDsPerQuery) {
		rows, err := q.ListPlanetPinContentsForPlanetPinIDs(ctx, idsChunk)
		if err != nil {
			return nil, fmt.Errorf("list planet pin contents for %d pins: %w", len(idsChunk), err)
		}
		for _, r := range rows {
			m[r.PlanetPinID] = append(m[r.PlanetPinID], &app.PlanetPinContent{
				Amount: r.Amount,
				Type:   eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory),
			})
		}
	}
	return m, nil
}

func planetPinFromDBModel(r queries.GetPlanetPinRow, types map[int64]*app.EveType) *app.PlanetPin {
	o := &app.PlanetPin{
		ID:                   r.PlanetPin.PinID,
		ExpiryTime:           optional.FromNullTime(r.PlanetPin.ExpiryTime),
		ExtractorHeadRadius:  optional.FromNullFloat64(r.PlanetPin.ExtractorHeadRadius),
		ExtractorNumHeads:    optional.FromNullInt64(r.PlanetPin.ExtractorNumHeads),
		ExtractorQtyPerCycle: optional.FromNullInt64(r.PlanetPin.ExtractorQtyPerCycle),
		InstallTime:          optional.FromNullTime(r.PlanetPin.InstallTime),
		LastCycleStart:       optional.FromNullTime(r.PlanetPin.LastCycleStart),
		Type:                 eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory),
	}
	if r.PlanetPin.ExtractorCycleTime.Valid {
		o.ExtractorCycleTime.Set(time.Duration(r.PlanetPin.ExtractorCycleTime.Int64) * time.Second)
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
		o.ExtractorProductType.Set(types[r.PlanetPin.ExtractorProductTypeID.Int64])
	}
	return o
}
