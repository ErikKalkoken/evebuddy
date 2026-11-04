// Package assetsync provides the rules for updating stored assets from ESI,
// which are shared by characters and corporations.
//
// Services convert ESI responses for characters and corporations
// to the ESI types of this package before use.
package assetsync

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

// namesMaxIDs is the maximum number of IDs ESI accepts in a request for asset names.
const namesMaxIDs = 999

var locationTypeFromESIValue = map[string]app.LocationType{
	"":             app.TypeUndefined,
	"station":      app.TypeStation,
	"solar_system": app.TypeSolarSystem,
	"item":         app.TypeItem,
	"other":        app.TypeOther,
}

// The ESI types below have the same fields as the ESI responses for characters and corporations,
// so services can convert them directly. Field names therefore follow ESI.

// ESIAsset is an asset from ESI.
type ESIAsset struct {
	IsBlueprintCopy *bool
	IsSingleton     bool
	ItemId          int64
	LocationFlag    string
	LocationId      int64
	LocationType    string
	Quantity        int64
	TypeId          int64
}

// ESIAssetName is the name of an asset from ESI.
type ESIAssetName struct {
	ItemId int64
	Name   string
}

// Item is an asset from ESI with parsed values, ready to be stored.
type Item struct {
	IsBlueprintCopy optional.Optional[bool]
	IsSingleton     bool
	ItemID          int64
	LocationFlag    app.LocationFlag
	LocationID      int64
	LocationType    app.LocationType
	Quantity        int64
	TypeID          int64
}

// ExtractIDs returns the IDs of the types and locations referenced by assets.
// Location IDs that reference other assets are not included.
func ExtractIDs(assets []ESIAsset) (typeIDs, locationIDs set.Set[int64]) {
	incomingIDs := itemIDs(assets)
	for _, a := range assets {
		typeIDs.Add(a.TypeId)
		if !incomingIDs.Contains(a.LocationId) {
			locationIDs.Add(a.LocationId)
		}
	}
	return typeIDs, locationIDs
}

func itemIDs(assets []ESIAsset) set.Set[int64] {
	return set.Collect(xiter.MapSlice(assets, func(x ESIAsset) int64 {
		return x.ItemId
	}))
}

// parse returns the item for an ESI asset.
// flags holds the location flags which ESI defines for the owner of the asset.
func parse(a ESIAsset, flags map[string]app.LocationFlag, logAttr slog.Attr) Item {
	locationFlag, found := flags[a.LocationFlag]
	if !found {
		locationFlag = app.FlagUnknown
		slog.Warn("Unknown location flag encountered", logAttr, "item", a)
	}
	locationType, found := locationTypeFromESIValue[a.LocationType]
	if !found {
		locationType = app.TypeUnknown
		slog.Warn("Unknown location type encountered", logAttr, "item", a)
	}
	quantity := a.Quantity
	if a.IsSingleton && quantity != 1 {
		slog.Warn("Singleton with invalid quantity encountered", logAttr, "item", a)
		quantity = 1
	}
	return Item{
		IsBlueprintCopy: optional.FromPtr(a.IsBlueprintCopy),
		IsSingleton:     a.IsSingleton,
		ItemID:          a.ItemId,
		LocationFlag:    locationFlag,
		LocationID:      a.LocationId,
		LocationType:    locationType,
		Quantity:        quantity,
		TypeID:          a.TypeId,
	}
}

// Sync creates and updates assets from ESI and deletes stored assets which are missing in ESI.
// currentIDs are the IDs of the stored assets.
// flags holds the location flags which ESI defines for the owner of the assets.
// The related types and locations of assets must already have been added.
//
// The callbacks do the owner specific work.
func Sync(
	ctx context.Context,
	assets []ESIAsset,
	currentIDs set.Set[int64],
	flags map[string]app.LocationFlag,
	create func(ctx context.Context, item Item) error,
	update func(ctx context.Context, item Item) error,
	deleteAssets func(ctx context.Context, ids set.Set[int64]) error,
	logAttr slog.Attr,
) error {
	var updated, created int
	for _, a := range assets {
		item := parse(a, flags, logAttr)
		if currentIDs.Contains(a.ItemId) {
			if err := update(ctx, item); err != nil {
				return err
			}
			updated++
		} else {
			if err := create(ctx, item); err != nil {
				return err
			}
			created++
		}
	}
	slog.Info("Stored assets", logAttr, "created", created, "updated", updated)

	if ids := set.Difference(currentIDs, itemIDs(assets)); ids.Size() > 0 {
		if err := deleteAssets(ctx, ids); err != nil {
			return err
		}
		slog.Info("Deleted obsolete assets", logAttr, "count", ids.Size())
	}
	return nil
}

// SyncNames updates the names of stored assets from ESI.
// Assets which can't have a name lose their stored name.
// When fetching names fails, the affected assets keep their stored names,
// and are fetched again the next time assets are processed.
// That can take a while, because assets are only processed
// when their ESI data has changed or an update is forced.
//
// The callbacks do the owner specific work.
// fetch returns the names for up to [namesMaxIDs] assets.
// adjustName is optional and returns the name to store for an asset.
func SyncNames(
	ctx context.Context,
	assets []*app.Asset,
	fetch func(ctx context.Context, ids []int64) ([]ESIAssetName, error),
	updateName func(ctx context.Context, itemID int64, name string) error,
	adjustName func(a *app.Asset, name string) string,
	logAttr slog.Attr,
) error {
	current := make(map[int64]string)
	for _, a := range assets {
		if a.CanHaveName() {
			current[a.ItemID] = a.Name
		} else if a.Name != "" {
			if err := updateName(ctx, a.ItemID, ""); err != nil {
				return err
			}
		}
	}
	names, err := fetchNames(ctx, slices.Collect(maps.Keys(current)), fetch)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// We can live without these names for now.
		slog.Warn("Failed to fetch asset names", logAttr, "err", err)
	}
	slog.Debug("Received asset names from ESI", logAttr, "count", len(names))

	if adjustName != nil {
		for _, a := range assets {
			if name, ok := names[a.ItemID]; ok {
				names[a.ItemID] = adjustName(a, name)
			}
		}
	}
	for id, name := range names {
		if current[id] == name {
			continue
		}
		if err := updateName(ctx, id, name); err != nil {
			return err
		}
	}
	return nil
}

// fetchNames returns the names of assets from ESI.
// The result contains all IDs that ESI has answered for, with an empty name for assets without a name.
// IDs from failed requests are left out and reported in the error.
func fetchNames(
	ctx context.Context,
	ids []int64,
	fetch func(ctx context.Context, ids []int64) ([]ESIAssetName, error),
) (map[int64]string, error) {
	m := make(map[int64]string)
	var errs []error
	for chunk := range slices.Chunk(ids, namesMaxIDs) {
		names, err := fetch(ctx, chunk)
		if err != nil {
			errs = append(errs, err)
			if ctx.Err() != nil {
				break
			}
			continue
		}
		for _, id := range chunk {
			m[id] = ""
		}
		for _, n := range names {
			if n.Name != "None" {
				m[n.ItemId] = n.Name
			}
		}
	}
	return m, errors.Join(errs...)
}
