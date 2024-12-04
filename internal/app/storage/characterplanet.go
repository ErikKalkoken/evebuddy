package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateCharacterPlanetParams struct {
	CharacterID  int32
	EvePlanetID  int32
	LastUpdate   time.Time
	UpgradeLevel int
}

func (st *Storage) CreateCharacterPlanet(ctx context.Context, arg CreateCharacterPlanetParams) (int64, error) {
	return createCharacterPlanet(ctx, st.q, arg)
}

func (st *Storage) GetCharacterPlanet(ctx context.Context, characterID int32, planetID int32) (*app.CharacterPlanet, error) {
	arg := queries.GetCharacterPlanetParams{
		CharacterID: int64(characterID),
		EvePlanetID: int64(planetID),
	}
	r, err := st.q.GetCharacterPlanet(ctx, arg)
	if err != nil {
		return nil, err
	}
	pp, err := st.ListPlanetPins(ctx, r.CharacterPlanet.ID)
	if err != nil {
		return nil, err
	}
	return characterPlanetFromDBModel(r, pp), err
}

func (st *Storage) ListCharacterPlanets(ctx context.Context, characterID int32) ([]*app.CharacterPlanet, error) {
	rows, err := st.q.ListCharacterPlanets(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	oo := make([]*app.CharacterPlanet, len(rows))
	for i, r := range rows {
		pp, err := st.ListPlanetPins(ctx, r.CharacterPlanet.ID)
		if err != nil {
			return nil, err
		}
		oo[i] = characterPlanetFromDBModel(queries.GetCharacterPlanetRow(r), pp)
	}
	return oo, nil
}

// ReplaceCharacterPlanets replaces all existing planets for a character with the new ones and returns their IDs.
func (st *Storage) ReplaceCharacterPlanets(ctx context.Context, characterID int32, args []CreateCharacterPlanetParams) ([]int64, error) {
	tx, err := st.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	qtx := st.q.WithTx(tx)
	if err := qtx.DeleteCharacterPlanets(ctx, int64(characterID)); err != nil {
		return nil, err
	}
	ids := make([]int64, 0)
	for _, arg := range args {
		id, err := createCharacterPlanet(ctx, qtx, arg)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ids, nil
}

func createCharacterPlanet(ctx context.Context, q *queries.Queries, arg CreateCharacterPlanetParams) (int64, error) {
	if arg.CharacterID == 0 || arg.EvePlanetID == 0 {
		return 0, fmt.Errorf("create planet: IDs can not be zero: %+v", arg)
	}
	arg2 := queries.CreateCharacterPlanetParams{
		CharacterID:  int64(arg.CharacterID),
		EvePlanetID:  int64(arg.EvePlanetID),
		LastUpdate:   arg.LastUpdate,
		UpgradeLevel: int64(arg.UpgradeLevel),
	}
	id, err := q.CreateCharacterPlanet(ctx, arg2)
	if err != nil {
		return 0, fmt.Errorf("create planet: %+v: %w", arg2, err)
	}
	return id, nil
}

func characterPlanetFromDBModel(r queries.GetCharacterPlanetRow, pp []*app.PlanetPin) *app.CharacterPlanet {
	et := eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory)
	ess := eveSolarSystemFromDBModel(r.EveSolarSystem, r.EveConstellation, r.EveRegion)
	ep := evePlanetFromDBModel(r.EvePlanet, ess, et)
	o := &app.CharacterPlanet{
		ID:           r.CharacterPlanet.ID,
		CharacterID:  int32(r.CharacterPlanet.CharacterID),
		EvePlanet:    ep,
		LastUpdate:   r.CharacterPlanet.LastUpdate,
		UpgradeLevel: int(r.CharacterPlanet.UpgradeLevel),
	}
	o.Pins = pp
	return o
}
