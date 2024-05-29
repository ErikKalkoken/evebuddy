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

func (r *Storage) CreateCharacterAsset(ctx context.Context, arg CreateCharacterAssetParams) error {
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
	if err := r.q.CreateCharacterAsset(ctx, arg2); err != nil {
		return fmt.Errorf("failed to create character asset %v, %w", arg, err)
	}
	return nil
}

func (r *Storage) GetCharacterAsset(ctx context.Context, characterID int32, itemID int64) (*model.CharacterAsset, error) {
	arg := queries.GetCharacterAssetParams{
		CharacterID: int64(characterID),
		ItemID:      itemID,
	}
	row, err := r.q.GetCharacterAsset(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get character asset for character %d: %w", characterID, err)
	}
	o := characterAssetFromDBModel(row.CharacterAsset, row.EveTypeName)
	return o, nil
}

func (r *Storage) DeleteExcludedCharacterAssets(ctx context.Context, characterID int32, itemIDs []int64) error {
	arg := queries.DeleteExcludedCharacterAssetsParams{
		CharacterID: int64(characterID),
		ItemIds:     itemIDs,
	}
	return r.q.DeleteExcludedCharacterAssets(ctx, arg)
}

func (r *Storage) ListCharacterAssetIDs(ctx context.Context, characterID int32) ([]int64, error) {
	return r.q.ListCharacterAssetIDs(ctx, int64(characterID))
}
func (r *Storage) ListCharacterAssets(ctx context.Context, characterID int32) ([]*model.CharacterAsset, error) {
	rows, err := r.q.ListCharacterAssets(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	ii2 := make([]*model.CharacterAsset, len(rows))
	for i, row := range rows {
		ii2[i] = characterAssetFromDBModel(row.CharacterAsset, row.EveTypeName)
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

func (r *Storage) UpdateCharacterAsset(ctx context.Context, arg UpdateCharacterAssetParams) error {
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
	if err := r.q.UpdateCharacterAsset(ctx, arg2); err != nil {
		return fmt.Errorf("failed to update character asset %v, %w", arg, err)
	}
	return nil
}

func characterAssetFromDBModel(ca queries.CharacterAsset, eveTypeName string) *model.CharacterAsset {
	if ca.CharacterID == 0 {
		panic("missing character ID")
	}
	o := &model.CharacterAsset{
		ID:              ca.ID,
		CharacterID:     int32(ca.CharacterID),
		EveType:         &model.EntityShort[int32]{ID: int32(ca.EveTypeID), Name: eveTypeName},
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
