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

func (r *Storage) DeleteCharacter(ctx context.Context, characterID int32) error {
	err := r.q.DeleteCharacter(ctx, int64(characterID))
	if err != nil {
		return fmt.Errorf("failed to delete Character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) GetCharacter(ctx context.Context, characterID int32) (*model.Character, error) {
	row, err := r.q.GetCharacter(ctx, int64(characterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get Character %d: %w", characterID, err)
	}
	c, err := r.characterFromDBModel(
		ctx,
		row.Character,
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

func (r *Storage) GetFirstCharacter(ctx context.Context) (*model.Character, error) {
	ids, err := r.ListCharacterIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, ErrNotFound
	}
	return r.GetCharacter(ctx, ids[0])

}

func (r *Storage) ListCharacters(ctx context.Context) ([]*model.Character, error) {
	rows, err := r.q.ListCharacters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list Characters: %w", err)
	}
	cc := make([]*model.Character, len(rows))
	for i, row := range rows {
		c, err := r.characterFromDBModel(
			ctx,
			row.Character,
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

func (r *Storage) ListCharactersShort(ctx context.Context) ([]*model.CharacterShort, error) {
	rows, err := r.q.ListCharactersShort(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list Character objects: %w", err)

	}
	cc := make([]*model.CharacterShort, len(rows))
	for i, row := range rows {
		cc[i] = &model.CharacterShort{ID: int32(row.ID), Name: row.Name, CorporationName: row.Name_2}
	}
	return cc, nil
}

func (r *Storage) ListCharacterIDs(ctx context.Context) ([]int32, error) {
	ids, err := r.q.ListCharacterIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list Character IDs: %w", err)
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}

func (r *Storage) UpdateCharacterHome(ctx context.Context, characterID int32, homeID sql.NullInt64) error {
	arg := queries.UpdateCharacterHomeIdParams{
		ID:     int64(characterID),
		HomeID: homeID,
	}
	if err := r.q.UpdateCharacterHomeId(ctx, arg); err != nil {
		return fmt.Errorf("failed to update home for character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) UpdateCharacterLastLoginAt(ctx context.Context, characterID int32, v sql.NullTime) error {
	arg := queries.UpdateCharacterLastLoginAtParams{
		ID:          int64(characterID),
		LastLoginAt: v,
	}
	if err := r.q.UpdateCharacterLastLoginAt(ctx, arg); err != nil {
		return fmt.Errorf("failed to update last login for character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) UpdateCharacterLocation(ctx context.Context, characterID int32, locationID sql.NullInt64) error {
	arg := queries.UpdateCharacterLocationIDParams{
		ID:         int64(characterID),
		LocationID: locationID,
	}
	if err := r.q.UpdateCharacterLocationID(ctx, arg); err != nil {
		return fmt.Errorf("failed to update last login for character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) UpdateCharacterShip(ctx context.Context, characterID int32, shipID sql.NullInt32) error {
	arg := queries.UpdateCharacterShipIDParams{
		ID:     int64(characterID),
		ShipID: sql.NullInt64{Int64: int64(shipID.Int32), Valid: shipID.Valid},
	}
	if err := r.q.UpdateCharacterShipID(ctx, arg); err != nil {
		return fmt.Errorf("failed to update ship for character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) UpdateCharacterSkillPoints(ctx context.Context, characterID int32, totalSP, unallocatedSP sql.NullInt64) error {
	arg := queries.UpdateCharacterSPParams{
		ID:            int64(characterID),
		TotalSp:       totalSP,
		UnallocatedSp: unallocatedSP,
	}
	if err := r.q.UpdateCharacterSP(ctx, arg); err != nil {
		return fmt.Errorf("failed to update sp for character %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) UpdateCharacterWalletBalance(ctx context.Context, characterID int32, v sql.NullFloat64) error {
	arg := queries.UpdateCharacterWalletBalanceParams{
		ID:            int64(characterID),
		WalletBalance: v,
	}
	if err := r.q.UpdateCharacterWalletBalance(ctx, arg); err != nil {
		return fmt.Errorf("failed to update sp for character %d: %w", characterID, err)
	}
	return nil
}

type UpdateOrCreateCharacterParams struct {
	ID            int32
	HomeID        sql.NullInt64
	LastLoginAt   sql.NullTime
	LocationID    sql.NullInt64
	ShipID        sql.NullInt32
	TotalSP       sql.NullInt64
	UnallocatedSP sql.NullInt64
	WalletBalance sql.NullFloat64
}

func (r *Storage) UpdateOrCreateCharacter(ctx context.Context, arg UpdateOrCreateCharacterParams) error {
	arg2 := queries.UpdateOrCreateCharacterParams{
		ID:            int64(arg.ID),
		HomeID:        arg.HomeID,
		LastLoginAt:   arg.LastLoginAt,
		LocationID:    arg.LocationID,
		ShipID:        sql.NullInt64{Int64: int64(arg.ShipID.Int32), Valid: arg.ShipID.Valid},
		TotalSp:       arg.TotalSP,
		UnallocatedSp: arg.UnallocatedSP,
		WalletBalance: arg.WalletBalance,
	}

	if err := r.q.UpdateOrCreateCharacter(ctx, arg2); err != nil {
		return fmt.Errorf("failed to update or create Character %d: %w", arg.ID, err)
	}
	return nil
}

func (r *Storage) characterFromDBModel(
	ctx context.Context,
	myCharacter queries.Character,
	eveCharacter queries.EveCharacter,
	corporation queries.EveEntity,
	race queries.EveRace,
	alliance queries.EveCharacterAlliance,
	faction queries.EveCharacterFaction,
	homeID sql.NullInt64,
	locationID sql.NullInt64,
	shipID sql.NullInt64,
) (*model.Character, error) {
	c := model.Character{
		EveCharacter:  eveCharacterFromDBModel(eveCharacter, corporation, race, alliance, faction),
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
