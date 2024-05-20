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
		row.HomeID,
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
			row.HomeID,
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

func (r *Storage) UpdateMyCharacterHome(ctx context.Context, characterID int32, homeID sql.NullInt64) error {
	arg := queries.UpdateMyCharacterHomeIdParams{
		ID:     int64(characterID),
		HomeID: homeID,
	}
	if err := r.q.UpdateMyCharacterHomeId(ctx, arg); err != nil {
		return fmt.Errorf("failed to update home for character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) UpdateMyCharacterLastLoginAt(ctx context.Context, characterID int32, v sql.NullTime) error {
	arg := queries.UpdateMyCharacterLastLoginAtParams{
		ID:          int64(characterID),
		LastLoginAt: v,
	}
	if err := r.q.UpdateMyCharacterLastLoginAt(ctx, arg); err != nil {
		return fmt.Errorf("failed to update last login for character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) UpdateMyCharacterLocation(ctx context.Context, characterID int32, locationID sql.NullInt64) error {
	arg := queries.UpdateMyCharacterLocationIDParams{
		ID:         int64(characterID),
		LocationID: locationID,
	}
	if err := r.q.UpdateMyCharacterLocationID(ctx, arg); err != nil {
		return fmt.Errorf("failed to update last login for character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) UpdateMyCharacterShip(ctx context.Context, characterID int32, shipID sql.NullInt32) error {
	arg := queries.UpdateMyCharacterShipIDParams{
		ID:     int64(characterID),
		ShipID: sql.NullInt64{Int64: int64(shipID.Int32), Valid: shipID.Valid},
	}
	if err := r.q.UpdateMyCharacterShipID(ctx, arg); err != nil {
		return fmt.Errorf("failed to update ship for character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) UpdateMyCharacterSkillPoints(ctx context.Context, characterID int32, totalSP, unallocatedSP sql.NullInt64) error {
	arg := queries.UpdateMyCharacterSPParams{
		ID:            int64(characterID),
		TotalSp:       totalSP,
		UnallocatedSp: unallocatedSP,
	}
	if err := r.q.UpdateMyCharacterSP(ctx, arg); err != nil {
		return fmt.Errorf("failed to update sp for character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) UpdateMyCharacterWalletBalance(ctx context.Context, characterID int32, v sql.NullFloat64) error {
	arg := queries.UpdateMyCharacterWalletBalanceParams{
		ID:            int64(characterID),
		WalletBalance: v,
	}
	if err := r.q.UpdateMyCharacterWalletBalance(ctx, arg); err != nil {
		return fmt.Errorf("failed to update sp for character %d: %w", characterID, err)
	}
	return nil
}

type UpdateOrCreateMyCharacterParams struct {
	ID            int32
	HomeID        sql.NullInt64
	LastLoginAt   sql.NullTime
	LocationID    sql.NullInt64
	ShipID        sql.NullInt32
	TotalSP       sql.NullInt64
	UnallocatedSP sql.NullInt64
	WalletBalance sql.NullFloat64
}

func (r *Storage) UpdateOrCreateMyCharacter(ctx context.Context, arg UpdateOrCreateMyCharacterParams) error {
	arg2 := queries.UpdateOrCreateMyCharacterParams{
		ID:            int64(arg.ID),
		HomeID:        arg.HomeID,
		LastLoginAt:   arg.LastLoginAt,
		LocationID:    arg.LocationID,
		ShipID:        sql.NullInt64{Int64: int64(arg.ShipID.Int32), Valid: arg.ShipID.Valid},
		TotalSp:       arg.TotalSP,
		UnallocatedSp: arg.UnallocatedSP,
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
	homeID sql.NullInt64,
	locationID sql.NullInt64,
	shipID sql.NullInt64,
) (*model.MyCharacter, error) {
	c := model.MyCharacter{
		Character:     eveCharacterFromDBModel(eveCharacter, corporation, race, alliance, faction),
		ID:            int32(myCharacter.ID),
		LastLoginAt:   myCharacter.LastLoginAt,
		TotalSP:       myCharacter.TotalSp,
		UnallocatedSP: myCharacter.UnallocatedSp,
		WalletBalance: myCharacter.WalletBalance,
	}
	if homeID.Valid {
		x, err := r.GetLocation(ctx, homeID.Int64)
		if err != nil {
			return nil, err
		}
		c.Home = x
	}
	if locationID.Valid {
		x, err := r.GetLocation(ctx, locationID.Int64)
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
