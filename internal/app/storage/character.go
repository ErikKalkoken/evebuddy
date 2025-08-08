// Package storage contains the logic for storing application data into a local SQLite database.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mattn/go-sqlite3"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type CreateCharacterParams struct {
	AssetValue        optional.Optional[float64]
	ID                int32
	IsTrainingWatched bool
	HomeID            optional.Optional[int64]
	LastCloneJumpAt   optional.Optional[time.Time]
	LastLoginAt       optional.Optional[time.Time]
	LocationID        optional.Optional[int64]
	ShipID            optional.Optional[int32]
	TotalSP           optional.Optional[int]
	UnallocatedSP     optional.Optional[int]
	WalletBalance     optional.Optional[float64]
}

func (st *Storage) CreateCharacter(ctx context.Context, arg CreateCharacterParams) error {
	arg2 := queries.CreateCharacterParams{
		ID:                int64(arg.ID),
		AssetValue:        optional.ToNullFloat64(arg.AssetValue),
		IsTrainingWatched: arg.IsTrainingWatched,
		HomeID:            optional.ToNullInt64(arg.HomeID),
		LastCloneJumpAt:   optional.ToNullTime(arg.LastCloneJumpAt),
		LastLoginAt:       optional.ToNullTime(arg.LastLoginAt),
		LocationID:        optional.ToNullInt64(arg.LocationID),
		ShipID:            optional.ToNullInt64(arg.ShipID),
		TotalSp:           optional.ToNullInt64(arg.TotalSP),
		UnallocatedSp:     optional.ToNullInt64(arg.UnallocatedSP),
		WalletBalance:     optional.ToNullFloat64(arg.WalletBalance),
	}
	if err := st.qRW.CreateCharacter(ctx, arg2); err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
				err = app.ErrAlreadyExists
			}
		}
		return fmt.Errorf("create character %d: %w", arg.ID, err)
	}
	return nil
}

func (st *Storage) DeleteCharacter(ctx context.Context, characterID int32) error {
	err := st.qRW.DeleteCharacter(ctx, int64(characterID))
	if err != nil {
		return fmt.Errorf("delete character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) DisableAllTrainingWatchers(ctx context.Context) error {
	if err := st.qRW.DisableAllTrainingWatchers(ctx); err != nil {
		return fmt.Errorf("disable all training watchers: %w", err)
	}
	return nil
}

func (st *Storage) GetCharacter(ctx context.Context, characterID int32) (*app.Character, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacter %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCharacter(ctx, int64(characterID))
	if err != nil {
		return nil, wrapErr(convertGetError(err))
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
		return nil, wrapErr(err)
	}
	return c, nil
}

func (st *Storage) GetAnyCharacter(ctx context.Context) (*app.Character, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetAnyCharacter: %w", err)
	}
	ids, err := st.ListCharacterIDs(ctx)
	if err != nil {
		return nil, wrapErr(err)
	}
	if ids.Size() == 0 {
		return nil, wrapErr(app.ErrNotFound)
	}
	var id int32
	for v := range ids.All() {
		id = v
		break
	}
	o, err := st.GetCharacter(ctx, id)
	if err != nil {
		return nil, wrapErr(err)
	}
	return o, nil
}

func (st *Storage) ListCharacters(ctx context.Context) ([]*app.Character, error) {
	rows, err := st.qRO.ListCharacters(ctx)
	if err != nil {
		return nil, fmt.Errorf("list characters: %w", err)
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
			return nil, fmt.Errorf("list characters: %w", err)
		}
		cc[i] = c
	}
	return cc, nil
}

func (st *Storage) ListCharactersShort(ctx context.Context) ([]*app.EntityShort[int32], error) {
	rows, err := st.qRO.ListCharactersShort(ctx)
	if err != nil {
		return nil, fmt.Errorf("list short characters: %w", err)

	}
	cc := make([]*app.EntityShort[int32], len(rows))
	for i, row := range rows {
		cc[i] = &app.EntityShort[int32]{ID: int32(row.ID), Name: row.Name}
	}
	return cc, nil
}

func (st *Storage) ListCharacterCorporationIDs(ctx context.Context) (set.Set[int32], error) {
	ids, err := st.qRO.ListCharacterCorporationIDs(ctx)
	if err != nil {
		return set.Set[int32]{}, fmt.Errorf("ListCharacterCorporationIDs: %w", err)
	}
	ids2 := set.Of(convertNumericSlice[int32](ids)...)
	return ids2, nil
}

func (st *Storage) ListCharacterIDs(ctx context.Context) (set.Set[int32], error) {
	ids, err := st.qRO.ListCharacterIDs(ctx)
	if err != nil {
		return set.Set[int32]{}, fmt.Errorf("list character IDs: %w", err)
	}
	ids2 := set.Of(convertNumericSlice[int32](ids)...)
	return ids2, nil
}

func (st *Storage) UpdateCharacterHome(ctx context.Context, characterID int32, homeID optional.Optional[int64]) error {
	arg := queries.UpdateCharacterHomeIdParams{
		ID:     int64(characterID),
		HomeID: optional.ToNullInt64(homeID),
	}
	if err := st.qRW.UpdateCharacterHomeId(ctx, arg); err != nil {
		return fmt.Errorf("update home for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterIsTrainingWatched(ctx context.Context, characterID int32, isWatched bool) error {
	arg := queries.UpdateCharacterIsTrainingWatchedParams{
		ID:                int64(characterID),
		IsTrainingWatched: isWatched,
	}
	if err := st.qRW.UpdateCharacterIsTrainingWatched(ctx, arg); err != nil {
		return fmt.Errorf("update is training watched for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterLastCloneJump(ctx context.Context, characterID int32, v optional.Optional[time.Time]) error {
	arg := queries.UpdateCharacterLastCloneJumpParams{
		ID:              int64(characterID),
		LastCloneJumpAt: optional.ToNullTime(v),
	}
	if err := st.qRW.UpdateCharacterLastCloneJump(ctx, arg); err != nil {
		return fmt.Errorf("update last clone jump for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterLastLoginAt(ctx context.Context, characterID int32, v optional.Optional[time.Time]) error {
	arg := queries.UpdateCharacterLastLoginAtParams{
		ID:          int64(characterID),
		LastLoginAt: optional.ToNullTime(v),
	}
	if err := st.qRW.UpdateCharacterLastLoginAt(ctx, arg); err != nil {
		return fmt.Errorf("update last login for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterLocation(ctx context.Context, characterID int32, locationID optional.Optional[int64]) error {
	arg := queries.UpdateCharacterLocationIDParams{
		ID:         int64(characterID),
		LocationID: optional.ToNullInt64(locationID),
	}
	if err := st.qRW.UpdateCharacterLocationID(ctx, arg); err != nil {
		return fmt.Errorf("update location for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterShip(ctx context.Context, characterID int32, shipID optional.Optional[int32]) error {
	arg := queries.UpdateCharacterShipIDParams{
		ID:     int64(characterID),
		ShipID: optional.ToNullInt64(shipID),
	}
	if err := st.qRW.UpdateCharacterShipID(ctx, arg); err != nil {
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
	if err := st.qRW.UpdateCharacterSP(ctx, arg); err != nil {
		return fmt.Errorf("update sp for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterWalletBalance(ctx context.Context, characterID int32, v optional.Optional[float64]) error {
	arg := queries.UpdateCharacterWalletBalanceParams{
		ID:            int64(characterID),
		WalletBalance: optional.ToNullFloat64(v),
	}
	if err := st.qRW.UpdateCharacterWalletBalance(ctx, arg); err != nil {
		return fmt.Errorf("update wallet balance for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterAssetValue(ctx context.Context, characterID int32, v optional.Optional[float64]) error {
	arg := queries.UpdateCharacterAssetValueParams{
		ID:         int64(characterID),
		AssetValue: optional.ToNullFloat64(v),
	}
	if err := st.qRW.UpdateCharacterAssetValue(ctx, arg); err != nil {
		return fmt.Errorf("update asset value for character %d: %w", characterID, err)
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
		LastCloneJumpAt:   optional.FromNullTime(character.LastCloneJumpAt),
		LastLoginAt:       optional.FromNullTime(character.LastLoginAt),
		TotalSP:           optional.FromNullInt64ToInteger[int](character.TotalSp),
		UnallocatedSP:     optional.FromNullInt64ToInteger[int](character.UnallocatedSp),
		WalletBalance:     optional.FromNullFloat64(character.WalletBalance),
	}
	if homeID.Valid {
		x, err := st.GetLocation(ctx, homeID.Int64)
		if err != nil {
			return nil, err
		}
		o.Home = x
	}
	if locationID.Valid {
		x, err := st.GetLocation(ctx, locationID.Int64)
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

func (st *Storage) GetCharacterAssetValue(ctx context.Context, id int32) (optional.Optional[float64], error) {
	v, err := st.qRO.GetCharacterAssetValue(ctx, int64(id))
	if err != nil {
		return optional.Optional[float64]{}, fmt.Errorf("get asset value for character %d: %w", id, convertGetError(err))
	}
	return optional.FromNullFloat64(v), nil
}
