package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateCharacterAssetParams struct {
	CharacterID     int32
	EveTypeID       int32
	IsBlueprintCopy bool
	IsSingleton     bool
	ItemID          int64
	LocationFlag    string
	LocationID      int64
	LocationType    string
	Name            string
	Quantity        int32
}

func (st *Storage) CreateCharacterAsset(ctx context.Context, arg CreateCharacterAssetParams) error {
	if arg.CharacterID == 0 || arg.EveTypeID == 0 || arg.ItemID == 0 {
		return fmt.Errorf("IDs must not be zero %v", arg)
	}
	arg2 := queries.CreateCharacterAssetParams{
		CharacterID:     int64(arg.CharacterID),
		EveTypeID:       int64(arg.EveTypeID),
		IsBlueprintCopy: arg.IsBlueprintCopy,
		IsSingleton:     arg.IsSingleton,
		ItemID:          arg.ItemID,
		LocationFlag:    arg.LocationFlag,
		LocationID:      arg.LocationID,
		LocationType:    arg.LocationType,
		Name:            arg.Name,
		Quantity:        int64(arg.Quantity),
	}
	if err := st.q.CreateCharacterAsset(ctx, arg2); err != nil {
		return fmt.Errorf("failed to create character asset %v, %w", arg, err)
	}
	return nil
}

func (st *Storage) DeleteExcludedCharacterAssets(ctx context.Context, characterID int32, itemIDs []int64) error {
	arg := queries.DeleteExcludedCharacterAssetsParams{
		CharacterID: int64(characterID),
		ItemIds:     itemIDs,
	}
	return st.q.DeleteExcludedCharacterAssets(ctx, arg)
}

func (st *Storage) GetCharacterAsset(ctx context.Context, characterID int32, itemID int64) (*model.CharacterAsset, error) {
	arg := queries.GetCharacterAssetParams{
		CharacterID: int64(characterID),
		ItemID:      itemID,
	}
	row, err := st.q.GetCharacterAsset(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get character asset for character %d: %w", characterID, err)
	}
	o := characterAssetFromDBModel(row.CharacterAsset, row.EveType, row.EveGroup, row.EveCategory)
	return o, nil
}

func (st *Storage) GetCharacterAssetTotalValue(ctx context.Context, characterID int32) (float64, error) {
	v, err := st.q.GetCharacterAssetTotalValue(ctx, int64(characterID))
	if err != nil {
		return 0, err
	}
	return v.Float64, nil
}

func (st *Storage) ListCharacterAssetIDs(ctx context.Context, characterID int32) ([]int64, error) {
	return st.q.ListCharacterAssetIDs(ctx, int64(characterID))
}

func (st *Storage) ListCharacterAssetsInShipHangar(ctx context.Context, characterID int32, locationID int64) ([]*model.CharacterAsset, error) {
	arg := queries.ListCharacterAssetsInShipHangarParams{
		CharacterID:   int64(characterID),
		LocationID:    locationID,
		LocationFlag:  "Hangar",
		EveCategoryID: model.EveCategoryShip,
	}
	rows, err := st.q.ListCharacterAssetsInShipHangar(ctx, arg)
	if err != nil {
		return nil, err
	}
	ii2 := make([]*model.CharacterAsset, len(rows))
	for i, row := range rows {
		ii2[i] = characterAssetFromDBModel(row.CharacterAsset, row.EveType, row.EveGroup, row.EveCategory)
	}
	return ii2, nil
}

func (st *Storage) ListCharacterAssetsInItemHangar(ctx context.Context, characterID int32, locationID int64) ([]*model.CharacterAsset, error) {
	arg := queries.ListCharacterAssetsInItemHangarParams{
		CharacterID:   int64(characterID),
		LocationID:    locationID,
		LocationFlag:  "Hangar",
		EveCategoryID: model.EveCategoryShip,
	}
	rows, err := st.q.ListCharacterAssetsInItemHangar(ctx, arg)
	if err != nil {
		return nil, err
	}
	ii2 := make([]*model.CharacterAsset, len(rows))
	for i, row := range rows {
		ii2[i] = characterAssetFromDBModel(row.CharacterAsset, row.EveType, row.EveGroup, row.EveCategory)
	}
	return ii2, nil
}

func (st *Storage) ListCharacterAssetsInLocation(ctx context.Context, characterID int32, locationID int64) ([]*model.CharacterAsset, error) {
	arg := queries.ListCharacterAssetsInLocationParams{
		CharacterID: int64(characterID),
		LocationID:  locationID,
	}
	rows, err := st.q.ListCharacterAssetsInLocation(ctx, arg)
	if err != nil {
		return nil, err
	}
	ii2 := make([]*model.CharacterAsset, len(rows))
	for i, row := range rows {
		ii2[i] = characterAssetFromDBModel(row.CharacterAsset, row.EveType, row.EveGroup, row.EveCategory)
	}
	return ii2, nil
}

type UpdateCharacterAssetParams struct {
	CharacterID  int32
	ItemID       int64
	LocationFlag string
	LocationID   int64
	LocationType string
	Name         string
	Quantity     int32
}

func (st *Storage) UpdateCharacterAsset(ctx context.Context, arg UpdateCharacterAssetParams) error {
	if arg.CharacterID == 0 || arg.ItemID == 0 {
		return fmt.Errorf("IDs must not be zero %v", arg)
	}
	arg2 := queries.UpdateCharacterAssetParams{
		CharacterID:  int64(arg.CharacterID),
		ItemID:       arg.ItemID,
		LocationFlag: arg.LocationFlag,
		LocationID:   arg.LocationID,
		LocationType: arg.LocationType,
		Name:         arg.Name,
		Quantity:     int64(arg.Quantity),
	}
	if err := st.q.UpdateCharacterAsset(ctx, arg2); err != nil {
		return fmt.Errorf("failed to update character asset %v, %w", arg, err)
	}
	return nil
}

func (st *Storage) ListCharacterAssetLocations(ctx context.Context, characterID int32) ([]*model.CharacterAssetLocation, error) {
	arg := queries.ListCharacterAssetLocationsParams{
		CharacterID:  int64(characterID),
		LocationFlag: "Hangar",
	}
	rows, err := st.q.ListCharacterAssetLocations(ctx, arg)
	if err != nil {
		return nil, err
	}
	oo := make([]*model.CharacterAssetLocation, len(rows))
	for i, r := range rows {
		oo[i] = characterAssetLocationFromDBModel(r)
	}
	return oo, nil
}

func characterAssetLocationFromDBModel(x queries.ListCharacterAssetLocationsRow) *model.CharacterAssetLocation {
	if x.LocationID == 0 || x.CharacterID == 0 {
		panic(fmt.Sprintf("invalid IDs: %v", x))
	}
	o := &model.CharacterAssetLocation{
		CharacterID:    int32(x.CharacterID),
		ID:             x.LocationID,
		Location:       &model.EntityShort[int64]{ID: x.LocationID, Name: x.LocationName},
		SecurityStatus: x.SecurityStatus,
	}
	if x.SystemID.Valid && x.SystemName.Valid {
		o.SolarSystem = &model.EntityShort[int32]{ID: int32(x.SystemID.Int64), Name: x.SystemName.String}
	}
	return o
}

func (st *Storage) ListAllCharacterAssets(ctx context.Context) ([]*model.CharacterAsset, error) {
	rows, err := st.q.ListAllCharacterAssets(ctx)
	if err != nil {
		return nil, err
	}
	oo := make([]*model.CharacterAsset, len(rows))
	for i, r := range rows {
		oo[i] = characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory)
	}
	return oo, nil
}

func (st *Storage) ListCharacterAssets(ctx context.Context, characterID int32) ([]*model.CharacterAsset, error) {
	rows, err := st.q.ListCharacterAssets(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	oo := make([]*model.CharacterAsset, len(rows))
	for i, r := range rows {
		oo[i] = characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory)
	}
	return oo, nil
}

func characterAssetFromDBModel(ca queries.CharacterAsset, t queries.EveType, g queries.EveGroup, c queries.EveCategory) *model.CharacterAsset {
	if ca.CharacterID == 0 {
		panic("missing character ID")
	}
	o := &model.CharacterAsset{
		ID:              ca.ID,
		CharacterID:     int32(ca.CharacterID),
		EveType:         eveTypeFromDBModel(t, g, c),
		IsBlueprintCopy: ca.IsBlueprintCopy,
		IsSingleton:     ca.IsSingleton,
		ItemID:          ca.ItemID,
		LocationFlag:    ca.LocationFlag,
		LocationID:      ca.LocationID,
		LocationType:    ca.LocationType,
		Name:            ca.Name,
		Quantity:        int32(ca.Quantity),
	}
	return o
}
