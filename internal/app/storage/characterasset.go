package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
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

func (st *Storage) GetCharacterAsset(ctx context.Context, characterID int32, itemID int64) (*app.CharacterAsset, error) {
	arg := queries.GetCharacterAssetParams{
		CharacterID: int64(characterID),
		ItemID:      itemID,
	}
	r, err := st.q.GetCharacterAsset(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get character asset for character %d: %w", characterID, err)
	}
	o := characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	return o, nil
}

func (st *Storage) CalculateCharacterAssetTotalValue(ctx context.Context, characterID int32) (float64, error) {
	v, err := st.q.CalculateCharacterAssetTotalValue(ctx, int64(characterID))
	if err != nil {
		return 0, err
	}
	return v.Float64, nil
}

func (st *Storage) ListCharacterAssetIDs(ctx context.Context, characterID int32) ([]int64, error) {
	return st.q.ListCharacterAssetIDs(ctx, int64(characterID))
}

func (st *Storage) ListCharacterAssetsInShipHangar(ctx context.Context, characterID int32, locationID int64) ([]*app.CharacterAsset, error) {
	arg := queries.ListCharacterAssetsInShipHangarParams{
		CharacterID:   int64(characterID),
		LocationID:    locationID,
		LocationFlag:  "Hangar",
		EveCategoryID: app.EveCategoryShip,
	}
	rows, err := st.q.ListCharacterAssetsInShipHangar(ctx, arg)
	if err != nil {
		return nil, err
	}
	ii2 := make([]*app.CharacterAsset, len(rows))
	for i, r := range rows {
		ii2[i] = characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return ii2, nil
}

func (st *Storage) ListCharacterAssetsInItemHangar(ctx context.Context, characterID int32, locationID int64) ([]*app.CharacterAsset, error) {
	arg := queries.ListCharacterAssetsInItemHangarParams{
		CharacterID:   int64(characterID),
		LocationID:    locationID,
		LocationFlag:  "Hangar",
		EveCategoryID: app.EveCategoryShip,
	}
	rows, err := st.q.ListCharacterAssetsInItemHangar(ctx, arg)
	if err != nil {
		return nil, err
	}
	ii2 := make([]*app.CharacterAsset, len(rows))
	for i, r := range rows {
		ii2[i] = characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return ii2, nil
}

func (st *Storage) ListCharacterAssetsInLocation(ctx context.Context, characterID int32, locationID int64) ([]*app.CharacterAsset, error) {
	arg := queries.ListCharacterAssetsInLocationParams{
		CharacterID: int64(characterID),
		LocationID:  locationID,
	}
	rows, err := st.q.ListCharacterAssetsInLocation(ctx, arg)
	if err != nil {
		return nil, err
	}
	ii2 := make([]*app.CharacterAsset, len(rows))
	for i, r := range rows {
		ii2[i] = characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
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

func (st *Storage) ListAllCharacterAssets(ctx context.Context) ([]*app.CharacterAsset, error) {
	rows, err := st.q.ListAllCharacterAssets(ctx)
	if err != nil {
		return nil, err
	}
	oo := make([]*app.CharacterAsset, len(rows))
	for i, r := range rows {
		oo[i] = characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return oo, nil
}

func (st *Storage) ListCharacterAssets(ctx context.Context, characterID int32) ([]*app.CharacterAsset, error) {
	rows, err := st.q.ListCharacterAssets(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	oo := make([]*app.CharacterAsset, len(rows))
	for i, r := range rows {
		oo[i] = characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return oo, nil
}

func characterAssetFromDBModel(ca queries.CharacterAsset, t queries.EveType, g queries.EveGroup, c queries.EveCategory, p sql.NullFloat64) *app.CharacterAsset {
	if ca.CharacterID == 0 {
		panic("missing character ID")
	}
	o := &app.CharacterAsset{
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
		Price:           optional.FromNullFloat64(p),
	}
	return o
}
