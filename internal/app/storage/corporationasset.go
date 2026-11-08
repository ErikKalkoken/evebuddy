package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateCorporationAssetParams struct {
	CorporationID   int64
	EveTypeID       int64
	IsBlueprintCopy optional.Optional[bool]
	IsSingleton     bool
	ItemID          int64
	LocationFlag    app.LocationFlag
	LocationID      int64
	LocationType    app.LocationType
	Name            string
	Quantity        int64
}

func (st *Storage) CreateCorporationAsset(ctx context.Context, arg CreateCorporationAssetParams) error {
	return createCorporationAsset(ctx, st.qRW, arg)
}

func createCorporationAsset(ctx context.Context, q *queries.Queries, arg CreateCorporationAssetParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCorporationAsset: %+v, %w", arg, err)
	}
	if arg.CorporationID == 0 || arg.EveTypeID == 0 || arg.ItemID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if !isValidLocationFlag(app.CorporationLocationFlags, arg.LocationFlag) {
		return wrapErr(app.ErrInvalid)
	}
	if err := q.CreateCorporationAsset(ctx, queries.CreateCorporationAssetParams{
		CorporationID:   arg.CorporationID,
		EveTypeID:       arg.EveTypeID,
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

func (st *Storage) deleteCorporationAssets(ctx context.Context, q *queries.Queries, corporationID int64, itemIDs set.Set[int64]) error {
	// chunks keep queries below SQLite's limit for variables
	for idsChunk := range slices.Chunk(slices.Collect(itemIDs.All()), st.MaxIDsPerQuery) {
		err := q.DeleteCorporationAssets(ctx, queries.DeleteCorporationAssetsParams{
			CorporationID: corporationID,
			ItemIds:       idsChunk,
		})
		if err != nil {
			return fmt.Errorf("delete %d corporation assets: %w", len(idsChunk), err)
		}
	}
	return nil
}

type ReplaceCorporationAssetParams struct {
	IsBlueprintCopy optional.Optional[bool]
	IsSingleton     bool
	ItemID          int64
	LocationFlag    app.LocationFlag
	LocationID      int64
	LocationType    app.LocationType
	Name            optional.Optional[string] // empty keeps the stored name
	Quantity        int64
	TypeID          int64
}

// ReplaceCorporationAssets replaces all stored assets of a corporation with args.
// Existing assets keep their stored name when an arg has no name.
func (st *Storage) ReplaceCorporationAssets(ctx context.Context, corporationID int64, args []ReplaceCorporationAssetParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceCorporationAssets: %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	ids, err := qtx.ListCorporationAssetIDs(ctx, corporationID)
	if err != nil {
		return wrapErr(err)
	}
	stored := set.Collect(slices.Values(ids))
	obsolete := stored.Clone()
	var created, updated int
	for _, arg := range args {
		obsolete.Delete(arg.ItemID)
		if stored.Contains(arg.ItemID) {
			err := updateCorporationAsset(ctx, qtx, UpdateCorporationAssetParams{
				CorporationID: corporationID,
				IsSingleton:   arg.IsSingleton,
				ItemID:        arg.ItemID,
				LocationFlag:  arg.LocationFlag,
				LocationID:    arg.LocationID,
				LocationType:  arg.LocationType,
				Name:          arg.Name,
				Quantity:      arg.Quantity,
			})
			if err != nil {
				return wrapErr(err)
			}
			updated++
		} else {
			err := createCorporationAsset(ctx, qtx, CreateCorporationAssetParams{
				CorporationID:   corporationID,
				EveTypeID:       arg.TypeID,
				IsBlueprintCopy: arg.IsBlueprintCopy,
				IsSingleton:     arg.IsSingleton,
				ItemID:          arg.ItemID,
				LocationFlag:    arg.LocationFlag,
				LocationID:      arg.LocationID,
				LocationType:    arg.LocationType,
				Name:            arg.Name.ValueOrZero(),
				Quantity:        arg.Quantity,
			})
			if err != nil {
				return wrapErr(err)
			}
			stored.Add(arg.ItemID)
			created++
		}
	}
	if err := st.deleteCorporationAssets(ctx, qtx, corporationID, obsolete); err != nil {
		return wrapErr(err)
	}
	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	slog.Info("Stored corporation assets", "corporationID", corporationID, "created", created, "updated", updated, "deleted", obsolete.Size())
	return nil
}

func (st *Storage) GetCorporationAsset(ctx context.Context, corporationID int64, itemID int64) (*app.CorporationAsset, error) {
	r, err := st.qRO.GetCorporationAsset(ctx, queries.GetCorporationAssetParams{
		CorporationID: corporationID,
		ItemID:        itemID,
	})
	if err != nil {
		return nil, fmt.Errorf("get corporation asset for corporation %d: %w", corporationID, convertGetError(err))
	}
	o := corporationAssetFromDBModel(r.CorporationAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	return o, nil
}

func (st *Storage) CalculateCorporationAssetTotalValue(ctx context.Context, corporationID int64) (float64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CalculateCorporationAssetTotalValue: %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	v, err := st.qRO.CalculateCorporationAssetTotalValue(ctx, queries.CalculateCorporationAssetTotalValueParams{
		CorporationID: corporationID,
		EveCategoryID: app.EveCategoryBlueprint,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	return v.Float64, nil
}

func (st *Storage) ListCorporationAssetIDs(ctx context.Context, corporationID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCorporationAssetIDs(ctx, corporationID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list corporation asset IDs: %w", err)
	}
	return set.Collect(slices.Values(ids)), nil
}

type UpdateCorporationAssetParams struct {
	CorporationID int64
	IsSingleton   bool
	ItemID        int64
	LocationFlag  app.LocationFlag
	LocationID    int64
	LocationType  app.LocationType
	Name          optional.Optional[string] // empty keeps the stored name
	Quantity      int64
}

func updateCorporationAsset(ctx context.Context, q *queries.Queries, arg UpdateCorporationAssetParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("update corporation asset %+v, %w", arg, err)
	}
	if arg.CorporationID == 0 || arg.ItemID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if !isValidLocationFlag(app.CorporationLocationFlags, arg.LocationFlag) {
		return wrapErr(app.ErrInvalid)
	}
	if err := q.UpdateCorporationAsset(ctx, queries.UpdateCorporationAssetParams{
		CorporationID: arg.CorporationID,
		IsSingleton:   arg.IsSingleton,
		ItemID:        arg.ItemID,
		LocationFlag:  locationFlagToDBValue[arg.LocationFlag],
		LocationID:    arg.LocationID,
		LocationType:  locationTypeToDBValue[arg.LocationType],
		Name:          optional.ToNullString(arg.Name),
		Quantity:      arg.Quantity,
	}); err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) ListAllCorporationAssets(ctx context.Context) ([]*app.CorporationAsset, error) {
	rows, err := st.qRO.ListAllCorporationAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all corporation assets: %w", err)
	}
	oo := make([]*app.CorporationAsset, len(rows))
	for i, r := range rows {
		oo[i] = corporationAssetFromDBModel(r.CorporationAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return oo, nil
}

func (st *Storage) ListCorporationAssets(ctx context.Context, corporationID int64) ([]*app.CorporationAsset, error) {
	rows, err := st.qRO.ListCorporationAssets(ctx, corporationID)
	if err != nil {
		return nil, fmt.Errorf("list assets for corporation ID %d: %w", corporationID, err)
	}
	oo := make([]*app.CorporationAsset, len(rows))
	for i, r := range rows {
		oo[i] = corporationAssetFromDBModel(r.CorporationAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return oo, nil
}

func corporationAssetFromDBModel(ca queries.CorporationAsset, t queries.EveType, g queries.EveGroup, c queries.EveCategory, p sql.NullFloat64) *app.CorporationAsset {
	if ca.CorporationID == 0 {
		panic("missing corporation ID")
	}
	locationFlag, found := locationFlagFromDBValue[ca.LocationFlag]
	if !found {
		locationFlag = app.FlagUnknown
	}
	locationType, found := locationTypeFromDBValue[ca.LocationType]
	if !found {
		locationType = app.TypeUnknown
	}
	o := &app.CorporationAsset{
		CorporationID: ca.CorporationID,
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
