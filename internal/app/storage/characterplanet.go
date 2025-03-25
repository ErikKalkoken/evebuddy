package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateCharacterPlanetParams struct {
	CharacterID  int32
	EvePlanetID  int32
	LastNotified time.Time
	LastUpdate   time.Time
	UpgradeLevel int
}

func (st *Storage) CreateCharacterPlanet(ctx context.Context, arg CreateCharacterPlanetParams) (int64, error) {
	if arg.CharacterID == 0 || arg.EvePlanetID == 0 {
		return 0, fmt.Errorf("create planet: IDs can not be zero: %+v", arg)
	}
	arg2 := queries.CreateCharacterPlanetParams{
		CharacterID:  int64(arg.CharacterID),
		EvePlanetID:  int64(arg.EvePlanetID),
		LastNotified: NewNullTimeFromTime(arg.LastNotified),
		LastUpdate:   arg.LastUpdate,
		UpgradeLevel: int64(arg.UpgradeLevel),
	}
	id, err := st.qRW.CreateCharacterPlanet(ctx, arg2)
	if err != nil {
		return 0, fmt.Errorf("create create planet: %+v: %w", arg2, err)
	}
	return id, nil
}

func (st *Storage) DeleteCharacterPlanet(ctx context.Context, characterID int32, planetIDs []int32) error {
	arg := queries.DeleteCharacterPlanetsParams{
		CharacterID:  int64(characterID),
		EvePlanetIds: convertNumericSlice[int64](planetIDs),
	}
	if err := st.qRW.DeleteCharacterPlanets(ctx, arg); err != nil {
		return fmt.Errorf("delete character planets: %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetCharacterPlanet(ctx context.Context, characterID int32, planetID int32) (*app.CharacterPlanet, error) {
	arg := queries.GetCharacterPlanetParams{
		CharacterID: int64(characterID),
		EvePlanetID: int64(planetID),
	}
	r, err := st.qRO.GetCharacterPlanet(ctx, arg)
	if err != nil {
		return nil, err
	}
	pp, err := st.ListPlanetPins(ctx, r.CharacterPlanet.ID)
	if err != nil {
		return nil, err
	}
	return characterPlanetFromDBModel(r, pp), err
}

func (st *Storage) ListAllCharacterPlanets(ctx context.Context) ([]*app.CharacterPlanet, error) {
	rows, err := st.qRO.ListAllCharacterPlanets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all planets: %w", err)
	}
	oo := make([]*app.CharacterPlanet, len(rows))
	for i, r := range rows {
		pp, err := st.ListPlanetPins(ctx, r.CharacterPlanet.ID)
		if err != nil {
			return nil, fmt.Errorf("list all planet pins: %w", err)
		}
		oo[i] = characterPlanetFromDBModel(queries.GetCharacterPlanetRow(r), pp)
	}
	return oo, nil
}

func (st *Storage) ListCharacterPlanets(ctx context.Context, id int32) ([]*app.CharacterPlanet, error) {
	rows, err := st.qRO.ListCharacterPlanets(ctx, int64(id))
	if err != nil {
		return nil, fmt.Errorf("list planets for character %d: %w", id, err)
	}
	oo := make([]*app.CharacterPlanet, len(rows))
	for i, r := range rows {
		pp, err := st.ListPlanetPins(ctx, r.CharacterPlanet.ID)
		if err != nil {
			return nil, fmt.Errorf("list planet pins for character %d: %w", id, err)
		}
		oo[i] = characterPlanetFromDBModel(queries.GetCharacterPlanetRow(r), pp)
	}
	return oo, nil
}

func characterPlanetFromDBModel(r queries.GetCharacterPlanetRow, pp []*app.PlanetPin) *app.CharacterPlanet {
	et := eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory)
	ess := eveSolarSystemFromDBModel(r.EveSolarSystem, r.EveConstellation, r.EveRegion)
	ep := evePlanetFromDBModel(r.EvePlanet, ess, et)
	o := &app.CharacterPlanet{
		ID:           r.CharacterPlanet.ID,
		CharacterID:  int32(r.CharacterPlanet.CharacterID),
		EvePlanet:    ep,
		LastNotified: optional.FromNullTime(r.CharacterPlanet.LastNotified),
		LastUpdate:   r.CharacterPlanet.LastUpdate,
		UpgradeLevel: int(r.CharacterPlanet.UpgradeLevel),
	}
	o.Pins = pp
	return o
}

type UpdateCharacterPlanetLastNotifiedParams struct {
	CharacterID  int32
	EvePlanetID  int32
	LastNotified time.Time
}

func (st *Storage) UpdateCharacterPlanetLastNotified(ctx context.Context, arg UpdateCharacterPlanetLastNotifiedParams) error {
	if arg.CharacterID == 0 || arg.EvePlanetID == 0 {
		return fmt.Errorf("update character planet last notified : IDs can not be zero: %+v", arg)
	}
	arg2 := queries.UpdateCharacterPlanetLastNotifiedParams{
		CharacterID:  int64(arg.CharacterID),
		EvePlanetID:  int64(arg.EvePlanetID),
		LastNotified: NewNullTimeFromTime(arg.LastNotified),
	}
	if err := st.qRW.UpdateCharacterPlanetLastNotified(ctx, arg2); err != nil {
		return fmt.Errorf("update character planet last notified: %+v: %w", arg2, err)
	}
	return nil
}

type UpdateOrCreateCharacterPlanetParams struct {
	CharacterID  int32
	EvePlanetID  int32
	LastUpdate   time.Time
	UpgradeLevel int
}

func (st *Storage) UpdateOrCreateCharacterPlanet(ctx context.Context, arg UpdateOrCreateCharacterPlanetParams) (int64, error) {
	if arg.CharacterID == 0 || arg.EvePlanetID == 0 {
		return 0, fmt.Errorf("update or create planet: IDs can not be zero: %+v", arg)
	}
	arg2 := queries.UpdateOrCreateCharacterPlanetParams{
		CharacterID:  int64(arg.CharacterID),
		EvePlanetID:  int64(arg.EvePlanetID),
		LastUpdate:   arg.LastUpdate,
		UpgradeLevel: int64(arg.UpgradeLevel),
	}
	id, err := st.qRW.UpdateOrCreateCharacterPlanet(ctx, arg2)
	if err != nil {
		return 0, fmt.Errorf("update or create create planet: %+v: %w", arg2, err)
	}
	return id, nil
}
