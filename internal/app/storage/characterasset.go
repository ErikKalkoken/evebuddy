package storage

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func (st *Storage) CalculateCharacterAssetTotalValue(ctx context.Context, characterID int64) (float64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CalculateCharacterAssetTotalValue: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	v, err := st.qRO.CalculateCharacterAssetTotalValue(ctx, queries.CalculateCharacterAssetTotalValueParams{
		EveCategoryID: app.EveCategoryBlueprint,
		CharacterID:   characterID,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	return v.Float64, nil
}

type CreateCharacterAssetParams struct {
	CharacterID     int64
	IsBlueprintCopy optional.Optional[bool]
	IsSingleton     bool
	ItemID          int64
	LocationFlag    app.LocationFlag
	LocationID      int64
	LocationType    app.LocationType
	Name            string
	Quantity        int64
	TypeID          int64
}

func (st *Storage) CreateCharacterAsset(ctx context.Context, arg CreateCharacterAssetParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharacterAsset: %+v: %w", arg, err)

	}
	if arg.CharacterID == 0 || arg.TypeID == 0 || arg.ItemID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if !isValidLocationFlag(app.CharacterLocationFlags, arg.LocationFlag) {
		return wrapErr(app.ErrInvalid)
	}
	if err := st.qRW.CreateCharacterAsset(ctx, queries.CreateCharacterAssetParams{
		CharacterID:     arg.CharacterID,
		EveTypeID:       arg.TypeID,
		IsBlueprintCopy: arg.IsBlueprintCopy.ValueOrZero(),
		IsSingleton:     arg.IsSingleton,
		ItemID:          arg.ItemID,
		LocationFlag:    locationFlagToDBValue[arg.LocationFlag],
		LocationID:      arg.LocationID,
		LocationType:    locationTypeToDBValue[arg.LocationType],
		Name:            arg.Name,
		Quantity:        arg.Quantity,
	}); err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeleteCharacterAssets(ctx context.Context, characterID int64, itemIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCharacterAssets: %d: %w", characterID, err)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	// chunks keep queries below SQLite's limit for variables
	for idsChunk := range slices.Chunk(slices.Collect(itemIDs.All()), st.MaxIDsPerQuery) {
		err := qtx.DeleteCharacterAssets(ctx, queries.DeleteCharacterAssetsParams{
			CharacterID: characterID,
			ItemIds:     idsChunk,
		})
		if err != nil {
			return wrapErr(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetCharacterAsset(ctx context.Context, characterID int64, itemID int64) (*app.CharacterAsset, error) {
	r, err := st.qRO.GetCharacterAsset(ctx, queries.GetCharacterAssetParams{
		CharacterID: characterID,
		ItemID:      itemID,
	})
	if err != nil {
		return nil, fmt.Errorf("get character asset for character %d: %w", characterID, convertGetError(err))
	}
	o := characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	return o, nil
}

func (st *Storage) ListCharacterAssetIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterAssetIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list character asset IDs: %w", err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) ListAllCharacterAssets(ctx context.Context) ([]*app.CharacterAsset, error) {
	rows, err := st.qRO.ListAllCharacterAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all character assets: %w", err)
	}
	oo := make([]*app.CharacterAsset, len(rows))
	for i, r := range rows {
		oo[i] = characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return oo, nil
}

func (st *Storage) ListCharacterAssets(ctx context.Context, characterID int64) ([]*app.CharacterAsset, error) {
	rows, err := st.qRO.ListCharacterAssets(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("list assets for character ID %d: %w", characterID, err)
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
	locationFlag, found := locationFlagFromDBValue[ca.LocationFlag]
	if !found {
		locationFlag = app.FlagUnknown
	}
	locationType, found := locationTypeFromDBValue[ca.LocationType]
	if !found {
		locationType = app.TypeUnknown
	}
	o := &app.CharacterAsset{
		CharacterID: ca.CharacterID,
		Asset: app.Asset{
			IsBlueprintCopy: optional.FromZeroValue(ca.IsBlueprintCopy),
			IsSingleton:     ca.IsSingleton,
			ItemID:          ca.ItemID,
			LocationFlag:    locationFlag,
			LocationID:      ca.LocationID,
			LocationType:    locationType,
			Name:            ca.Name,
			Price:           optional.FromNullFloat64(p),
			Quantity:        int(ca.Quantity),
			Type:            eveTypeFromDBModel(t, g, c),
		},
	}
	return o
}

type UpdateCharacterAssetParams struct {
	CharacterID  int64
	IsSingleton  bool
	ItemID       int64
	LocationFlag app.LocationFlag
	LocationID   int64
	LocationType app.LocationType
	Quantity     int64
}

func (st *Storage) UpdateCharacterAsset(ctx context.Context, arg UpdateCharacterAssetParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterAsset: %+v: %w", arg, err)

	}
	if arg.CharacterID == 0 || arg.ItemID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if !isValidLocationFlag(app.CharacterLocationFlags, arg.LocationFlag) {
		return wrapErr(app.ErrInvalid)
	}
	if err := st.qRW.UpdateCharacterAsset(ctx, queries.UpdateCharacterAssetParams{
		CharacterID:  arg.CharacterID,
		IsSingleton:  arg.IsSingleton,
		ItemID:       arg.ItemID,
		LocationFlag: locationFlagToDBValue[arg.LocationFlag],
		LocationID:   arg.LocationID,
		LocationType: locationTypeToDBValue[arg.LocationType],
		Quantity:     arg.Quantity,
	}); err != nil {
		return wrapErr(err)
	}
	return nil
}

type UpdateCharacterAssetNameParams struct {
	CharacterID int64
	ItemID      int64
	Name        string
}

func (st *Storage) UpdateCharacterAssetName(ctx context.Context, arg UpdateCharacterAssetNameParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterAssetName: %+v: %w", arg, err)

	}
	if arg.CharacterID == 0 || arg.ItemID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if err := st.qRW.UpdateCharacterAssetName(ctx, queries.UpdateCharacterAssetNameParams{
		CharacterID: arg.CharacterID,
		ItemID:      arg.ItemID,
		Name:        arg.Name,
	}); err != nil {
		return wrapErr(err)
	}
	return nil
}
