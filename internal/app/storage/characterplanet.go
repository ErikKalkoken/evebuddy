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
	r, err := st.qRO.GetCharacterPlanet(ctx, arg)
	if err != nil {
		return nil, convertGetError(err)
	}
	oo, err := st.characterPlanetsFromDBModels(ctx, []queries.GetCharacterPlanetRow{r})
	if err != nil {
		return nil, err
	}
	return oo[0], nil
}

// HasCharacterPlanetsWithoutRoutes reports whether a character has colonies
// with configured pins, but without any routes.
func (st *Storage) HasCharacterPlanetsWithoutRoutes(ctx context.Context, characterID int64) (bool, error) {
	v, err := st.qRO.HasCharacterPlanetsWithoutRoutes(ctx, characterID)
	if err != nil {
		return false, fmt.Errorf("has character planets without routes for character %d: %w", characterID, err)
	}
	return v == 1, nil
}

func (st *Storage) ListAllCharacterPlanets(ctx context.Context) ([]*app.CharacterPlanet, error) {
	rows, err := st.qRO.ListAllCharacterPlanets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all planets: %w", err)
	}
	rows2 := make([]queries.GetCharacterPlanetRow, len(rows))
	for i, r := range rows {
		rows2[i] = queries.GetCharacterPlanetRow(r)
	}
	oo, err := st.characterPlanetsFromDBModels(ctx, rows2)
	if err != nil {
		return nil, fmt.Errorf("list all planets: %w", err)
	}
	return oo, nil
}

func (st *Storage) ListCharacterPlanets(ctx context.Context, id int64) ([]*app.CharacterPlanet, error) {
	rows, err := st.qRO.ListCharacterPlanets(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list planets for character %d: %w", id, err)
	}
	rows2 := make([]queries.GetCharacterPlanetRow, len(rows))
	for i, r := range rows {
		rows2[i] = queries.GetCharacterPlanetRow(r)
	}
	oo, err := st.characterPlanetsFromDBModels(ctx, rows2)
	if err != nil {
		return nil, fmt.Errorf("list planets for character %d: %w", id, err)
	}
	return oo, nil
}

// characterPlanetsFromDBModels converts rows to planets, batch loading their pins and routes.
func (st *Storage) characterPlanetsFromDBModels(ctx context.Context, rows []queries.GetCharacterPlanetRow) ([]*app.CharacterPlanet, error) {
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.CharacterPlanet.ID
	}
	pins, err := st.listPlanetPinsByPlanet(ctx, ids)
	if err != nil {
		return nil, err
	}
	routes, err := st.listPlanetRoutesByPlanet(ctx, ids)
	if err != nil {
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

type UpdateOrCreateCharacterPlanetParams struct {
	CharacterID  int64
	EvePlanetID  int64
	LastUpdate   time.Time
	UpgradeLevel int64
}

func (st *Storage) UpdateOrCreateCharacterPlanet(ctx context.Context, arg UpdateOrCreateCharacterPlanetParams) (int64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCharacterPlanet: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.EvePlanetID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	id, err := st.qRW.UpdateOrCreateCharacterPlanet(ctx, queries.UpdateOrCreateCharacterPlanetParams{
		CharacterID:  arg.CharacterID,
		EvePlanetID:  arg.EvePlanetID,
		LastUpdate:   arg.LastUpdate,
		UpgradeLevel: arg.UpgradeLevel,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	return id, nil
}
