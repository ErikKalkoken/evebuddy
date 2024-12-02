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
	NumPins      int
	UpgradeLevel int
}

func (st *Storage) CreateCharacterPlanet(ctx context.Context, arg CreateCharacterPlanetParams) error {
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
	return characterPlanetFromDBModel(r), err
}

func (st *Storage) ListCharacterPlanets(ctx context.Context, characterID int32) ([]*app.CharacterPlanet, error) {
	rows, err := st.q.ListCharacterPlanets(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	oo := make([]*app.CharacterPlanet, len(rows))
	for i, r := range rows {
		oo[i] = characterPlanetFromDBModel(queries.GetCharacterPlanetRow(r))
	}
	return oo, nil
}

func (st *Storage) ReplaceCharacterPlanets(ctx context.Context, characterID int32, args []CreateCharacterPlanetParams) error {
	tx, err := st.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := st.q.WithTx(tx)
	if err := qtx.DeleteCharacterPlanets(ctx, int64(characterID)); err != nil {
		return err
	}
	for _, arg := range args {
		if err := createCharacterPlanet(ctx, qtx, arg); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func createCharacterPlanet(ctx context.Context, q *queries.Queries, arg CreateCharacterPlanetParams) error {
	if arg.CharacterID == 0 || arg.EvePlanetID == 0 {
		return fmt.Errorf("create planet: IDs can not be zero: %+v", arg)
	}
	arg2 := queries.CreateCharacterPlanetParams{
		CharacterID:  int64(arg.CharacterID),
		EvePlanetID:  int64(arg.EvePlanetID),
		LastUpdate:   arg.LastUpdate,
		NumPins:      int64(arg.NumPins),
		UpgradeLevel: int64(arg.UpgradeLevel),
	}
	if err := q.CreateCharacterPlanet(ctx, arg2); err != nil {
		return fmt.Errorf("create planet: %+v: %w", arg2, err)
	}
	return nil
}

func characterPlanetFromDBModel(r queries.GetCharacterPlanetRow) *app.CharacterPlanet {
	et := eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory)
	ess := eveSolarSystemFromDBModel(r.EveSolarSystem, r.EveConstellation, r.EveRegion)
	ep := evePlanetFromDBModel(r.EvePlanet, ess, et)
	o2 := &app.CharacterPlanet{
		CharacterID:  int32(r.CharacterPlanet.CharacterID),
		EvePlanet:    ep,
		LastUpdate:   r.CharacterPlanet.LastUpdate,
		NumPins:      int(r.CharacterPlanet.NumPins),
		UpgradeLevel: int(r.CharacterPlanet.UpgradeLevel),
	}
	return o2
}
