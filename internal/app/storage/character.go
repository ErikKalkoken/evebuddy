// Package sqlite contains the logic for storing application data into a local SQLite database.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func (st *Storage) DeleteCharacter(ctx context.Context, characterID int32) error {
	err := st.q.DeleteCharacter(ctx, int64(characterID))
	if err != nil {
		return fmt.Errorf("delete Character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) GetCharacter(ctx context.Context, characterID int32) (*app.Character, error) {
	r, err := st.q.GetCharacter(ctx, int64(characterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("get Character %d: %w", characterID, err)
	}
	alliance := nullEveEntry{
		ID:       r.EveCharacter.AllianceID,
		Name:     r.AllianceName,
		Category: r.AllianceCategory,
	}
	faction := nullEveEntry{
		ID:       r.EveCharacter.FactionID,
		Name:     r.FactionName,
		Category: r.FactionCategory,
	}
	c, err := st.characterFromDBModel(
		ctx,
		r.Character,
		r.EveCharacter,
		r.EveEntity,
		r.EveRace,
		alliance,
		faction,
		r.HomeID,
		r.LocationID,
		r.ShipID,
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
		return nil, fmt.Errorf("list Characters: %w", err)
	}
	cc := make([]*app.Character, len(rows))
	for i, r := range rows {
		alliance := nullEveEntry{
			ID:       r.EveCharacter.AllianceID,
			Name:     r.AllianceName,
			Category: r.AllianceCategory,
		}
		faction := nullEveEntry{
			ID:       r.EveCharacter.FactionID,
			Name:     r.FactionName,
			Category: r.FactionCategory,
		}
		c, err := st.characterFromDBModel(
			ctx,
			r.Character,
			r.EveCharacter,
			r.EveEntity,
			r.EveRace,
			alliance,
			faction,
			r.HomeID,
			r.LocationID,
			r.ShipID,
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
		return nil, fmt.Errorf("list short characters: %w", err)

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
		return nil, fmt.Errorf("list character IDs: %w", err)
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
		return fmt.Errorf("update home for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterIsTrainingWatched(ctx context.Context, characterID int32, isWatched bool) error {
	arg := queries.UpdateCharacterIsTrainingWatchedParams{
		ID:                int64(characterID),
		IsTrainingWatched: isWatched,
	}
	if err := st.q.UpdateCharacterIsTrainingWatched(ctx, arg); err != nil {
		return fmt.Errorf("update is training watched for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterLastLoginAt(ctx context.Context, characterID int32, v optional.Optional[time.Time]) error {
	arg := queries.UpdateCharacterLastLoginAtParams{
		ID:          int64(characterID),
		LastLoginAt: optional.ToNullTime(v),
	}
	if err := st.q.UpdateCharacterLastLoginAt(ctx, arg); err != nil {
		return fmt.Errorf("update last login for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterLocation(ctx context.Context, characterID int32, locationID optional.Optional[int64]) error {
	arg := queries.UpdateCharacterLocationIDParams{
		ID:         int64(characterID),
		LocationID: optional.ToNullInt64(locationID),
	}
	if err := st.q.UpdateCharacterLocationID(ctx, arg); err != nil {
		return fmt.Errorf("update last login for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterShip(ctx context.Context, characterID int32, shipID optional.Optional[int32]) error {
	arg := queries.UpdateCharacterShipIDParams{
		ID:     int64(characterID),
		ShipID: optional.ToNullInt64(shipID),
	}
	if err := st.q.UpdateCharacterShipID(ctx, arg); err != nil {
		return fmt.Errorf("update ship for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterSkillPoints(ctx context.Context, characterID int32, totalSP, unallocatedSP optional.Optional[int]) error {
	arg := queries.UpdateCharacterSPParams{
		ID:            int64(characterID),
		TotalSp:       optional.ToNullInt64(totalSP),
		UnallocatedSp: optional.ToNullInt64(unallocatedSP),
	}
	if err := st.q.UpdateCharacterSP(ctx, arg); err != nil {
		return fmt.Errorf("update sp for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterWalletBalance(ctx context.Context, characterID int32, v optional.Optional[float64]) error {
	arg := queries.UpdateCharacterWalletBalanceParams{
		ID:            int64(characterID),
		WalletBalance: optional.ToNullFloat64(v),
	}
	if err := st.q.UpdateCharacterWalletBalance(ctx, arg); err != nil {
		return fmt.Errorf("update wallet balance for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterAssetValue(ctx context.Context, characterID int32, v optional.Optional[float64]) error {
	arg := queries.UpdateCharacterAssetValueParams{
		ID:         int64(characterID),
		AssetValue: optional.ToNullFloat64(v),
	}
	if err := st.q.UpdateCharacterAssetValue(ctx, arg); err != nil {
		return fmt.Errorf("update asset value for character %d: %w", characterID, err)
	}
	return nil
}

type UpdateOrCreateCharacterParams struct {
	AssetValue        optional.Optional[float64]
	ID                int32
	IsTrainingWatched bool
	HomeID            optional.Optional[int64]
	LastLoginAt       optional.Optional[time.Time]
	LocationID        optional.Optional[int64]
	ShipID            optional.Optional[int32]
	TotalSP           optional.Optional[int]
	UnallocatedSP     optional.Optional[int]
	WalletBalance     optional.Optional[float64]
}

func (st *Storage) UpdateOrCreateCharacter(ctx context.Context, arg UpdateOrCreateCharacterParams) error {
	arg2 := queries.UpdateOrCreateCharacterParams{
		ID:                int64(arg.ID),
		AssetValue:        optional.ToNullFloat64(arg.AssetValue),
		IsTrainingWatched: arg.IsTrainingWatched,
		HomeID:            optional.ToNullInt64(arg.HomeID),
		LastLoginAt:       optional.ToNullTime(arg.LastLoginAt),
		LocationID:        optional.ToNullInt64(arg.LocationID),
		ShipID:            optional.ToNullInt64(arg.ShipID),
		TotalSp:           optional.ToNullInt64(arg.TotalSP),
		UnallocatedSp:     optional.ToNullInt64(arg.UnallocatedSP),
		WalletBalance:     optional.ToNullFloat64(arg.WalletBalance),
	}
	if err := st.q.UpdateOrCreateCharacter(ctx, arg2); err != nil {
		return fmt.Errorf("update or create character %d: %w", arg.ID, err)
	}
	return nil
}

func (st *Storage) characterFromDBModel(
	ctx context.Context,
	character queries.Character,
	eveCharacter queries.EveCharacter,
	corporation queries.EveEntity,
	race queries.EveRace,
	alliance nullEveEntry,
	faction nullEveEntry,
	homeID sql.NullInt64,
	locationID sql.NullInt64,
	shipID sql.NullInt64,
) (*app.Character, error) {
	o := app.Character{
		AssetValue:        optional.FromNullFloat64(character.AssetValue),
		EveCharacter:      eveCharacterFromDBModel(eveCharacter, corporation, race, alliance, faction),
		ID:                int32(character.ID),
		IsTrainingWatched: character.IsTrainingWatched,
		LastLoginAt:       optional.FromNullTime(character.LastLoginAt),
		TotalSP:           optional.FromNullInt64ToInteger[int](character.TotalSp),
		UnallocatedSP:     optional.FromNullInt64ToInteger[int](character.UnallocatedSp),
		WalletBalance:     optional.FromNullFloat64(character.WalletBalance),
	}
	if homeID.Valid {
		x, err := st.GetEveLocation(ctx, homeID.Int64)
		if err != nil {
			return nil, err
		}
		o.Home = x
	}
	if locationID.Valid {
		x, err := st.GetEveLocation(ctx, locationID.Int64)
		if err != nil {
			return nil, err
		}
		o.Location = x
	}
	if shipID.Valid {
		x, err := st.GetEveType(ctx, int32(shipID.Int64))
		if err != nil {
			return nil, err
		}
		o.Ship = x
	}
	return &o, nil
}

func (st *Storage) GetCharacterAssetValue(ctx context.Context, characterID int32) (optional.Optional[float64], error) {
	v, err := st.q.GetCharacterAssetValue(ctx, int64(characterID))
	if err != nil {
		return optional.Optional[float64]{}, err
	}
	return optional.FromNullFloat64(v), nil
}
