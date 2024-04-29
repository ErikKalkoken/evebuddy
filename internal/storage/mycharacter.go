package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	islices "github.com/ErikKalkoken/evebuddy/internal/helper/slices"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

func (r *Storage) DeleteMyCharacter(ctx context.Context, characterID int32) error {
	err := r.q.DeleteMyCharacter(ctx, int64(characterID))
	if err != nil {
		return fmt.Errorf("failed to delete MyCharacter %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) GetMyCharacter(ctx context.Context, characterID int32) (model.MyCharacter, error) {
	var dummy model.MyCharacter
	row, err := r.q.GetMyCharacter(ctx, int64(characterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return dummy, fmt.Errorf("failed to get MyCharacter %d: %w", characterID, err)
	}
	c := myCharacterFromDBModel(
		row.MyCharacter,
		row.EveCategory,
		row.EveGroup,
		row.EveType,
		row.EveRegion,
		row.EveConstellation,
		row.EveSolarSystem,
		row.EveCharacter,
		row.EveEntity,
		row.EveRace,
		row.EveCharacterAlliance,
		row.EveCharacterFaction,
	)
	return c, nil
}

func (r *Storage) GetFirstMyCharacter(ctx context.Context) (model.MyCharacter, error) {
	ids, err := r.ListMyCharacterIDs(ctx)
	if err != nil {
		return model.MyCharacter{}, nil
	}
	if len(ids) == 0 {
		return model.MyCharacter{}, ErrNotFound
	}
	return r.GetMyCharacter(ctx, ids[0])

}

func (r *Storage) ListMyCharacters(ctx context.Context) ([]model.MyCharacterShort, error) {
	rows, err := r.q.ListMyCharacters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list MyCharacter objects: %w", err)

	}
	cc := make([]model.MyCharacterShort, len(rows))
	for i, row := range rows {
		cc[i] = model.MyCharacterShort{ID: int32(row.ID), Name: row.Name}
	}
	return cc, nil
}

func (r *Storage) ListMyCharacterIDs(ctx context.Context) ([]int32, error) {
	ids, err := r.q.ListMyCharacterIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list MyCharacter IDs: %w", err)
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}

func (r *Storage) UpdateOrCreateMyCharacter(ctx context.Context, c *model.MyCharacter) error {
	arg := queries.UpdateOrCreateMyCharacterParams{
		ID:            int64(c.ID),
		LastLoginAt:   c.LastLoginAt,
		ShipID:        int64(c.Ship.ID),
		SkillPoints:   int64(c.SkillPoints),
		LocationID:    int64(c.Location.ID),
		WalletBalance: c.WalletBalance,
	}
	_, err := r.q.UpdateOrCreateMyCharacter(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to update or create MyCharacter %d: %w", c.ID, err)
	}
	return nil
}

func myCharacterFromDBModel(
	myCharacter queries.MyCharacter,
	shipCategory queries.EveCategory,
	shipGroup queries.EveGroup,
	shipType queries.EveType,
	region queries.EveRegion,
	constellation queries.EveConstellation,
	solar_system queries.EveSolarSystem,
	eveCharacter queries.EveCharacter,
	corporation queries.EveEntity,
	race queries.EveRace,
	alliance queries.EveCharacterAlliance,
	faction queries.EveCharacterFaction,
) model.MyCharacter {
	x := model.MyCharacter{
		Character:     eveCharacterFromDBModel(eveCharacter, corporation, race, alliance, faction),
		ID:            int32(myCharacter.ID),
		LastLoginAt:   myCharacter.LastLoginAt,
		Location:      eveSolarSystemFromDBModel(solar_system, constellation, region),
		Ship:          eveTypeFromDBModel(shipType, shipGroup, shipCategory),
		SkillPoints:   int(myCharacter.SkillPoints),
		WalletBalance: myCharacter.WalletBalance,
	}
	return x
}
