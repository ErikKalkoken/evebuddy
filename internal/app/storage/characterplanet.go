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

type CreateCharacterPlanetParams struct {
	CharacterID  int64
	EvePlanetID  int64
	LastNotified time.Time
	LastUpdate   time.Time
	UpgradeLevel int64
}

func (st *Storage) CreateCharacterPlanet(ctx context.Context, arg CreateCharacterPlanetParams) (int64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharacterPlanet: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.EvePlanetID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	id, err := st.qRW.CreateCharacterPlanet(ctx, queries.CreateCharacterPlanetParams{
		CharacterID:  arg.CharacterID,
		EvePlanetID:  arg.EvePlanetID,
		LastNotified: NewNullTimeFromTime(arg.LastNotified),
		LastUpdate:   arg.LastUpdate,
		UpgradeLevel: arg.UpgradeLevel,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	return id, nil
}

func (st *Storage) DeleteCharacterPlanet(ctx context.Context, characterID int64, planetIDs set.Set[int64]) error {
	arg := queries.DeleteCharacterPlanetsParams{
		CharacterID:  characterID,
		EvePlanetIds: slices.Collect(planetIDs.All()),
	}
	if err := st.qRW.DeleteCharacterPlanets(ctx, arg); err != nil {
		return fmt.Errorf("delete character planets: %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetCharacterPlanet(ctx context.Context, characterID int64, planetID int64) (*app.CharacterPlanet, error) {
	arg := queries.GetCharacterPlanetParams{
		CharacterID: characterID,
		EvePlanetID: planetID,
	}
	oo, err := st.readCharacterPlanets(ctx, func(q *queries.Queries) ([]queries.GetCharacterPlanetRow, error) {
		r, err := q.GetCharacterPlanet(ctx, arg)
		if err != nil {
			return nil, convertGetError(err)
		}
		return []queries.GetCharacterPlanetRow{r}, nil
	})
	if err != nil {
		return nil, err
	}
	return oo[0], nil
}

func (st *Storage) ListAllCharacterPlanets(ctx context.Context) ([]*app.CharacterPlanet, error) {
	oo, err := st.readCharacterPlanets(ctx, func(q *queries.Queries) ([]queries.GetCharacterPlanetRow, error) {
		rows, err := q.ListAllCharacterPlanets(ctx)
		if err != nil {
			return nil, err
		}
		rows2 := make([]queries.GetCharacterPlanetRow, len(rows))
		for i, r := range rows {
			rows2[i] = queries.GetCharacterPlanetRow(r)
		}
		return rows2, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list all planets: %w", err)
	}
	return oo, nil
}

func (st *Storage) ListCharacterPlanets(ctx context.Context, id int64) ([]*app.CharacterPlanet, error) {
	oo, err := st.readCharacterPlanets(ctx, func(q *queries.Queries) ([]queries.GetCharacterPlanetRow, error) {
		rows, err := q.ListCharacterPlanets(ctx, id)
		if err != nil {
			return nil, err
		}
		rows2 := make([]queries.GetCharacterPlanetRow, len(rows))
		for i, r := range rows {
			rows2[i] = queries.GetCharacterPlanetRow(r)
		}
		return rows2, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list planets for character %d: %w", id, err)
	}
	return oo, nil
}

// readCharacterPlanets reads planets with their pins and routes in one transaction,
// so concurrent writes can not mix old and new data.
func (st *Storage) readCharacterPlanets(ctx context.Context, listRows func(q *queries.Queries) ([]queries.GetCharacterPlanetRow, error)) ([]*app.CharacterPlanet, error) {
	tx, err := st.dbRO.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	qtx := st.qRO.WithTx(tx)
	rows, err := listRows(qtx)
	if err != nil {
		return nil, err
	}
	var ids set.Set[int64]
	for _, r := range rows {
		ids.Add(r.CharacterPlanet.ID)
	}
	pins, err := st.listPlanetPinsByPlanet(ctx, qtx, ids)
	if err != nil {
		return nil, err
	}
	routes, err := st.listPlanetRoutesByPlanet(ctx, qtx, ids)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	oo := make([]*app.CharacterPlanet, len(rows))
	for i, r := range rows {
		o := characterPlanetFromDBModel(r, pins[r.CharacterPlanet.ID])
		o.Routes = routes[r.CharacterPlanet.ID]
		oo[i] = o
	}
	return oo, nil
}

func characterPlanetFromDBModel(r queries.GetCharacterPlanetRow, pp []*app.PlanetPin) *app.CharacterPlanet {
	et := eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory)
	ess := eveSolarSystemFromDBModel(r.EveSolarSystem, r.EveConstellation, r.EveRegion)
	ep := evePlanetFromDBModel(r.EvePlanet, ess, et)
	o := &app.CharacterPlanet{
		ID:           r.CharacterPlanet.ID,
		CharacterID:  r.CharacterPlanet.CharacterID,
		EvePlanet:    ep,
		LastNotified: optional.FromNullTime(r.CharacterPlanet.LastNotified),
		LastUpdate:   r.CharacterPlanet.LastUpdate,
		UpgradeLevel: r.CharacterPlanet.UpgradeLevel,
	}
	o.Pins = pp
	return o
}

type UpdateCharacterPlanetLastNotifiedParams struct {
	CharacterID  int64
	EvePlanetID  int64
	LastNotified time.Time
}

func (st *Storage) UpdateCharacterPlanetLastNotified(ctx context.Context, arg UpdateCharacterPlanetLastNotifiedParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterPlanetLastNotified: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.EvePlanetID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	arg2 := queries.UpdateCharacterPlanetLastNotifiedParams{
		CharacterID:  arg.CharacterID,
		EvePlanetID:  arg.EvePlanetID,
		LastNotified: NewNullTimeFromTime(arg.LastNotified),
	}
	if err := st.qRW.UpdateCharacterPlanetLastNotified(ctx, arg2); err != nil {
		return wrapErr(err)
	}
	return nil
}

type ReplaceCharacterPlanetParams struct {
	CharacterID  int64
	EvePlanetID  int64
	LastUpdate   time.Time
	Pins         []CreatePlanetPinParams   // CharacterPlanetID is ignored
	Routes       []CreatePlanetRouteParams // CharacterPlanetID is ignored
	UpgradeLevel int64
}

// ReplaceCharacterPlanet creates or updates a colony and replaces its pins and routes in one transaction,
// so readers never see a partially updated colony. It returns the ID of the colony.
func (st *Storage) ReplaceCharacterPlanet(ctx context.Context, arg ReplaceCharacterPlanetParams) (int64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceCharacterPlanet: %d %d: %w", arg.CharacterID, arg.EvePlanetID, err)
	}
	if arg.CharacterID == 0 || arg.EvePlanetID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return 0, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	id, err := qtx.UpdateOrCreateCharacterPlanet(ctx, queries.UpdateOrCreateCharacterPlanetParams{
		CharacterID:  arg.CharacterID,
		EvePlanetID:  arg.EvePlanetID,
		LastUpdate:   arg.LastUpdate,
		UpgradeLevel: arg.UpgradeLevel,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	if err := qtx.DeletePlanetPins(ctx, id); err != nil {
		return 0, wrapErr(err)
	}
	if err := qtx.DeletePlanetRoutes(ctx, id); err != nil {
		return 0, wrapErr(err)
	}
	for _, p := range arg.Pins {
		p.CharacterPlanetID = id
		if err := createPlanetPin(ctx, qtx, p); err != nil {
			return 0, wrapErr(fmt.Errorf("pin %+v: %w", p, err))
		}
	}
	for _, r := range arg.Routes {
		r.CharacterPlanetID = id
		if err := createPlanetRoute(ctx, qtx, r); err != nil {
			return 0, wrapErr(fmt.Errorf("route %+v: %w", r, err))
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, wrapErr(err)
	}
	return id, nil
}
