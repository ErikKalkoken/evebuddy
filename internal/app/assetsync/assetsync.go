// Package assetsync provides the rules for updating stored assets from ESI,
// which are shared by characters and corporations.
//
// Services convert ESI responses for characters and corporations
// to the ESI types of this package before use.
package assetsync

import (
	"context"
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
	Name            optional.Optional[string] // empty keeps the stored name
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
// flags and types hold the location values which ESI defines for the owner of the asset.
// Unexpected values do not fail, because ESI can add new ones at any time.
func parse(a ESIAsset, flags map[string]app.LocationFlag, types map[string]app.LocationType, logAttr slog.Attr) Item {
	locationFlag, found := flags[a.LocationFlag]
	if !found {
		locationFlag = app.FlagUnknown
		slog.Warn("Unknown location flag encountered", logAttr, "item", a)
	}
	locationType, found := types[a.LocationType]
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

// Sync replaces the stored assets with assets from ESI.
// Assets missing in names keep their stored names.
// Of duplicate assets only the last one is kept.
// Related types and locations must exist.
func Sync(
	ctx context.Context,
	assets []ESIAsset,
	names map[int64]string,
	flags map[string]app.LocationFlag,
	types map[string]app.LocationType,
	replace func(ctx context.Context, items []Item) error,
	logAttr slog.Attr,
) error {
	items := make([]Item, 0, len(assets))
	index := make(map[int64]int, len(assets))
	var duplicates int
	for _, a := range assets {
		item := parse(a, flags, types, logAttr)
		if name, ok := names[a.ItemId]; ok {
			item.Name = optional.New(name)
		}
		if i, found := index[a.ItemId]; found {
			items[i] = item // pages can shift while being fetched
			duplicates++
			continue
		}
		index[a.ItemId] = len(items)
		items = append(items, item)
	}
	if duplicates > 0 {
		slog.Warn("Duplicate assets received from ESI", logAttr, "count", duplicates)
	}
	return replace(ctx, items)
}

// ResolveNames returns the names to store for assets by item ID.
// Assets which can't have a name get "". Assets not fetched are left out and keep their stored names.
// Fetching stops at the first failed request. adjustName is optional.
func ResolveNames(
	ctx context.Context,
	assets []ESIAsset,
	eveTypes []*app.EveType,
	fetch func(ctx context.Context, ids []int64) ([]ESIAssetName, error),
	adjustName func(typeID int64, name string) string,
	logAttr slog.Attr,
) (map[int64]string, error) {
	typeByID := make(map[int64]*app.EveType, len(eveTypes))
	for _, et := range eveTypes {
		typeByID[et.ID] = et
	}
	result := make(map[int64]string)
	candidates := make(map[int64]int64) // item ID -> type ID
	for _, x := range assets {
		et, found := typeByID[x.TypeId]
		if !found {
			continue
		}
		a := app.Asset{IsSingleton: x.IsSingleton, Type: et}
		if a.CanHaveName() {
			candidates[x.ItemId] = x.TypeId
		} else {
			result[x.ItemId] = ""
		}
	}
	for ids := range slices.Chunk(slices.Sorted(maps.Keys(candidates)), namesMaxIDs) {
		names, err := fetch(ctx, ids)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			// Further requests would likely fail too. We can live without these names for now.
			slog.Warn("Failed to fetch asset names", logAttr, "err", err)
			break
		}
		received := make(map[int64]string)
		for _, n := range names {
			if n.Name != "None" {
				received[n.ItemId] = n.Name
			}
		}
		for _, id := range ids {
			name := received[id] // assets without a name are missing or "None"
			if adjustName != nil {
				name = adjustName(candidates[id], name)
			}
			result[id] = name
		}
	}
	return result, nil
}
