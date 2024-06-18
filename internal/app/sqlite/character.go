// Package sqlite contains the logic for storing application data into a local SQLite database.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
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

func (st *Storage) UpdateCharacterHome(ctx context.Context, characterID int32, homeID optional.Optional[int64]) error {
	arg := queries.UpdateCharacterHomeIdParams{
		ID:     int64(characterID),
		HomeID: optional.ToNullInt64(homeID),
	}
	if err := st.q.UpdateCharacterHomeId(ctx, arg); err != nil {
		return fmt.Errorf("failed to update home for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterLastLoginAt(ctx context.Context, characterID int32, v optional.Optional[time.Time]) error {
	arg := queries.UpdateCharacterLastLoginAtParams{
		ID:          int64(characterID),
		LastLoginAt: optional.ToNullTime(v),
	}
	if err := st.q.UpdateCharacterLastLoginAt(ctx, arg); err != nil {
		return fmt.Errorf("failed to update last login for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterLocation(ctx context.Context, characterID int32, locationID optional.Optional[int64]) error {
	arg := queries.UpdateCharacterLocationIDParams{
		ID:         int64(characterID),
		LocationID: optional.ToNullInt64(locationID),
	}
	if err := st.q.UpdateCharacterLocationID(ctx, arg); err != nil {
		return fmt.Errorf("failed to update last login for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterShip(ctx context.Context, characterID int32, shipID optional.Optional[int32]) error {
	x := optional.ToNullInt32(shipID)
	arg := queries.UpdateCharacterShipIDParams{
		ID:     int64(characterID),
		ShipID: sql.NullInt64{Int64: int64(x.Int32), Valid: x.Valid},
	}
	if err := st.q.UpdateCharacterShipID(ctx, arg); err != nil {
		return fmt.Errorf("failed to update ship for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterSkillPoints(ctx context.Context, characterID int32, totalSP, unallocatedSP optional.Optional[int]) error {
	arg := queries.UpdateCharacterSPParams{
		ID:            int64(characterID),
		TotalSp:       optional.ToNullInt64(optional.ConvertNumeric[int, int64](totalSP)),
		UnallocatedSp: optional.ToNullInt64(optional.ConvertNumeric[int, int64](unallocatedSP)),
	}
	if err := st.q.UpdateCharacterSP(ctx, arg); err != nil {
		return fmt.Errorf("failed to update sp for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterWalletBalance(ctx context.Context, characterID int32, v optional.Optional[float64]) error {
	arg := queries.UpdateCharacterWalletBalanceParams{
		ID:            int64(characterID),
		WalletBalance: optional.ToNullFloat64(v),
	}
	if err := st.q.UpdateCharacterWalletBalance(ctx, arg); err != nil {
		return fmt.Errorf("failed to update sp for character %d: %w", characterID, err)
	}
	return nil
}

type UpdateOrCreateCharacterParams struct {
	ID            int32
	HomeID        optional.Optional[int64]
	LastLoginAt   optional.Optional[time.Time]
	LocationID    optional.Optional[int64]
	ShipID        optional.Optional[int32]
	TotalSP       optional.Optional[int]
	UnallocatedSP optional.Optional[int]
	WalletBalance optional.Optional[float64]
}

func (st *Storage) UpdateOrCreateCharacter(ctx context.Context, arg UpdateOrCreateCharacterParams) error {
	arg2 := queries.UpdateOrCreateCharacterParams{
		ID:            int64(arg.ID),
		HomeID:        optional.ToNullInt64(arg.HomeID),
		LastLoginAt:   optional.ToNullTime(arg.LastLoginAt),
		LocationID:    optional.ToNullInt64(arg.LocationID),
		ShipID:        optional.ToNullInt64(optional.ConvertNumeric[int32, int64](arg.ShipID)),
		TotalSp:       optional.ToNullInt64(optional.ConvertNumeric[int, int64](arg.TotalSP)),
		UnallocatedSp: optional.ToNullInt64(optional.ConvertNumeric[int, int64](arg.UnallocatedSP)),
		WalletBalance: optional.ToNullFloat64(arg.WalletBalance),
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
		LastLoginAt:   optional.FromNullTime(character.LastLoginAt),
		TotalSP:       optional.ConvertNumeric[int64, int](optional.FromNullInt64(character.TotalSp)),
		UnallocatedSP: optional.ConvertNumeric[int64, int](optional.FromNullInt64(character.UnallocatedSp)),
		WalletBalance: optional.FromNullFloat64(character.WalletBalance),
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
