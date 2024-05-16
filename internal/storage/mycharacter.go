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

func (r *Storage) GetMyCharacter(ctx context.Context, characterID int32) (*model.MyCharacter, error) {
	row, err := r.q.GetMyCharacter(ctx, int64(characterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get MyCharacter %d: %w", characterID, err)
	}
	c, err := r.myCharacterFromDBModel(
		ctx,
		row.MyCharacter,
		row.EveCharacter,
		row.EveEntity,
		row.EveRace,
		row.EveCharacterAlliance,
		row.EveCharacterFaction,
		row.LocationID,
		row.ShipID,
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *Storage) GetFirstMyCharacter(ctx context.Context) (*model.MyCharacter, error) {
	ids, err := r.ListMyCharacterIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, ErrNotFound
	}
	return r.GetMyCharacter(ctx, ids[0])

}

func (r *Storage) ListMyCharacters(ctx context.Context) ([]*model.MyCharacter, error) {
	rows, err := r.q.ListMyCharacters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list MyCharacters: %w", err)
	}
	cc := make([]*model.MyCharacter, len(rows))
	for i, row := range rows {
		c, err := r.myCharacterFromDBModel(
			ctx,
			row.MyCharacter,
			row.EveCharacter,
			row.EveEntity,
			row.EveRace,
			row.EveCharacterAlliance,
			row.EveCharacterFaction,
			row.LocationID,
			row.ShipID,
		)
		if err != nil {
			return nil, err
		}
		cc[i] = c
	}
	return cc, nil
}

func (r *Storage) ListMyCharactersShort(ctx context.Context) ([]*model.MyCharacterShort, error) {
	rows, err := r.q.ListMyCharactersShort(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list MyCharacter objects: %w", err)

	}
	cc := make([]*model.MyCharacterShort, len(rows))
	for i, row := range rows {
		cc[i] = &model.MyCharacterShort{ID: int32(row.ID), Name: row.Name, CorporationName: row.Name_2}
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

type UpdateOrCreateMyCharacterParams struct {
	ID            int32
	LastLoginAt   sql.NullTime
	LocationID    sql.NullInt32
	ShipID        sql.NullInt32
	SkillPoints   sql.NullInt64
	WalletBalance sql.NullFloat64
}

func (r *Storage) UpdateOrCreateMyCharacter(ctx context.Context, arg UpdateOrCreateMyCharacterParams) error {
	arg2 := queries.UpdateOrCreateMyCharacterParams{
		ID:            int64(arg.ID),
		LastLoginAt:   arg.LastLoginAt,
		LocationID:    sql.NullInt64{Int64: int64(arg.LocationID.Int32), Valid: arg.LocationID.Valid},
		ShipID:        sql.NullInt64{Int64: int64(arg.ShipID.Int32), Valid: arg.ShipID.Valid},
		SkillPoints:   arg.SkillPoints,
		WalletBalance: arg.WalletBalance,
	}

	if err := r.q.UpdateOrCreateMyCharacter(ctx, arg2); err != nil {
		return fmt.Errorf("failed to update or create MyCharacter %d: %w", arg.ID, err)
	}
	return nil
}

func (r *Storage) myCharacterFromDBModel(
	ctx context.Context,
	myCharacter queries.MyCharacter,
	eveCharacter queries.EveCharacter,
	corporation queries.EveEntity,
	race queries.EveRace,
	alliance queries.EveCharacterAlliance,
	faction queries.EveCharacterFaction,
	locationID sql.NullInt64,
	shipID sql.NullInt64,
) (*model.MyCharacter, error) {
	c := model.MyCharacter{
		Character:     eveCharacterFromDBModel(eveCharacter, corporation, race, alliance, faction),
		ID:            int32(myCharacter.ID),
		LastLoginAt:   myCharacter.LastLoginAt,
		SkillPoints:   myCharacter.SkillPoints,
		WalletBalance: myCharacter.WalletBalance,
	}
	if locationID.Valid {
		x, err := r.GetEveSolarSystem(ctx, int32(locationID.Int64))
		if err != nil {
			return nil, err
		}
		c.Location = x
	}
	if shipID.Valid {
		x, err := r.GetEveType(ctx, int32(shipID.Int64))
		if err != nil {
			return nil, err
		}
		c.Ship = x
	}
	return &c, nil
}
