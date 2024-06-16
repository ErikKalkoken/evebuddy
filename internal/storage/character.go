// Package storage contains the logic for storing data into a local SQLite database.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

func (st *Storage) DeleteCharacter(ctx context.Context, characterID int32) error {
	err := st.q.DeleteCharacter(ctx, int64(characterID))
	if err != nil {
		return fmt.Errorf("failed to delete Character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) GetCharacter(ctx context.Context, characterID int32) (*app.Character, error) {
	row, err := st.q.GetCharacter(ctx, int64(characterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get Character %d: %w", characterID, err)
	}
	c, err := st.characterFromDBModel(
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

func (st *Storage) GetFirstCharacter(ctx context.Context) (*app.Character, error) {
	ids, err := st.ListCharacterIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, ErrNotFound
	}
	return st.GetCharacter(ctx, ids[0])

}

func (st *Storage) ListCharacters(ctx context.Context) ([]*app.Character, error) {
	rows, err := st.q.ListCharacters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list Characters: %w", err)
	}
	cc := make([]*app.Character, len(rows))
	for i, row := range rows {
		c, err := st.characterFromDBModel(
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

func (st *Storage) ListCharactersShort(ctx context.Context) ([]*app.CharacterShort, error) {
	rows, err := st.q.ListCharactersShort(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list short characters: %w", err)

	}
	cc := make([]*app.CharacterShort, len(rows))
	for i, row := range rows {
		cc[i] = &app.CharacterShort{ID: int32(row.ID), Name: row.Name}
	}
	return cc, nil
}

func (st *Storage) ListCharacterIDs(ctx context.Context) ([]int32, error) {
	ids, err := st.q.ListCharacterIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list character IDs: %w", err)
	}
	ids2 := convertNumericSlice[int64, int32](ids)
	return ids2, nil
}

func (st *Storage) UpdateCharacterHome(ctx context.Context, characterID int32, homeID sql.NullInt64) error {
	arg := queries.UpdateCharacterHomeIdParams{
		ID:     int64(characterID),
		HomeID: homeID,
	}
	if err := st.q.UpdateCharacterHomeId(ctx, arg); err != nil {
		return fmt.Errorf("failed to update home for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterLastLoginAt(ctx context.Context, characterID int32, v sql.NullTime) error {
	arg := queries.UpdateCharacterLastLoginAtParams{
		ID:          int64(characterID),
		LastLoginAt: v,
	}
	if err := st.q.UpdateCharacterLastLoginAt(ctx, arg); err != nil {
		return fmt.Errorf("failed to update last login for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterLocation(ctx context.Context, characterID int32, locationID sql.NullInt64) error {
	arg := queries.UpdateCharacterLocationIDParams{
		ID:         int64(characterID),
		LocationID: locationID,
	}
	if err := st.q.UpdateCharacterLocationID(ctx, arg); err != nil {
		return fmt.Errorf("failed to update last login for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterShip(ctx context.Context, characterID int32, shipID sql.NullInt32) error {
	arg := queries.UpdateCharacterShipIDParams{
		ID:     int64(characterID),
		ShipID: sql.NullInt64{Int64: int64(shipID.Int32), Valid: shipID.Valid},
	}
	if err := st.q.UpdateCharacterShipID(ctx, arg); err != nil {
		return fmt.Errorf("failed to update ship for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterSkillPoints(ctx context.Context, characterID int32, totalSP, unallocatedSP sql.NullInt64) error {
	arg := queries.UpdateCharacterSPParams{
		ID:            int64(characterID),
		TotalSp:       totalSP,
		UnallocatedSp: unallocatedSP,
	}
	if err := st.q.UpdateCharacterSP(ctx, arg); err != nil {
		return fmt.Errorf("failed to update sp for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterWalletBalance(ctx context.Context, characterID int32, v sql.NullFloat64) error {
	arg := queries.UpdateCharacterWalletBalanceParams{
		ID:            int64(characterID),
		WalletBalance: v,
	}
	if err := st.q.UpdateCharacterWalletBalance(ctx, arg); err != nil {
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

func (st *Storage) UpdateOrCreateCharacter(ctx context.Context, arg UpdateOrCreateCharacterParams) error {
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

	if err := st.q.UpdateOrCreateCharacter(ctx, arg2); err != nil {
		return fmt.Errorf("failed to update or create Character %d: %w", arg.ID, err)
	}
	return nil
}

func (st *Storage) characterFromDBModel(
	ctx context.Context,
	character queries.Character,
	eveCharacter queries.EveCharacter,
	corporation queries.EveEntity,
	race queries.EveRace,
	alliance queries.EveCharacterAlliance,
	faction queries.EveCharacterFaction,
	homeID sql.NullInt64,
	locationID sql.NullInt64,
	shipID sql.NullInt64,
) (*app.Character, error) {
	c := app.Character{
		EveCharacter:  eveCharacterFromDBModel(eveCharacter, corporation, race, alliance, faction),
		ID:            int32(character.ID),
		LastLoginAt:   character.LastLoginAt,
		TotalSP:       character.TotalSp,
		UnallocatedSP: character.UnallocatedSp,
		WalletBalance: character.WalletBalance,
	}
	if homeID.Valid {
		x, err := st.GetEveLocation(ctx, homeID.Int64)
		if err != nil {
			return nil, err
		}
		c.Home = x
	}
	if locationID.Valid {
		x, err := st.GetEveLocation(ctx, locationID.Int64)
		if err != nil {
			return nil, err
		}
		c.Location = x
	}
	if shipID.Valid {
		x, err := st.GetEveType(ctx, int32(shipID.Int64))
		if err != nil {
			return nil, err
		}
		c.Ship = x
	}
	return &c, nil
}
