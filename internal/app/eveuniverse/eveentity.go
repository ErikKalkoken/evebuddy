package eveuniverse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"

	"github.com/antihax/goesi/esi"
)

var ErrEveEntityNameNoMatch = errors.New("no match found with that name")
var ErrEveEntityNameMultipleMatches = errors.New("multiple matches with that name")

func (eu *EveUniverseService) GetEveEntity(ctx context.Context, id int32) (*app.EveEntity, error) {
	return eu.st.GetEveEntity(ctx, id)
}

func (eu *EveUniverseService) GetOrCreateEveEntityESI(ctx context.Context, id int32) (*app.EveEntity, error) {
	o, err := eu.st.GetEveEntity(ctx, id)
	if err == nil {
		return o, nil
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	_, err = eu.AddMissingEveEntities(ctx, []int32{id})
	if err != nil {
		return nil, err
	}
	return eu.st.GetEveEntity(ctx, id)
}

// TODO: Reduce DB calls with AddMissingEveEntities

// ToEveEntities returns the resolved EveEntities for a list of valid entity IDs.
// ID 0 will be resolved to a zero value EveEntity object
// Will return an error if any ID can not be resolved.
func (eu *EveUniverseService) ToEveEntities(ctx context.Context, ids []int32) (map[int32]*app.EveEntity, error) {
	r := make(map[int32]*app.EveEntity)
	ids2 := set.NewFromSlice(ids)
	if ids2.Contains(0) {
		r[0] = &app.EveEntity{}
		ids2.Remove(0)
	}
	if _, err := eu.AddMissingEveEntities(ctx, ids2.ToSlice()); err != nil {
		return nil, err
	}
	for id := range ids2.Values() {
		x, err := eu.GetOrCreateEveEntityESI(ctx, id)
		if err != nil {
			return nil, err
		}
		r[id] = x
	}
	return r, nil
}

// AddMissingEveEntities adds EveEntities from ESI for IDs missing in the database and returns which IDs where indeed missing.
func (eu *EveUniverseService) AddMissingEveEntities(ctx context.Context, ids []int32) ([]int32, error) {
	missing, err := eu.st.MissingEveEntityIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if missing.Size() == 0 {
		return nil, nil
	}
	missingIDs := missing.ToSlice()
	slices.Sort(missingIDs)
	if len(missingIDs) > 0 {
		slog.Debug("Trying to resolve EveEntity IDs from ESI", "ids", missingIDs)
	}
	var ee []esi.PostUniverseNames200Ok
	var badIDs []int32
	for chunk := range slices.Chunk(missingIDs, 1000) { // PostUniverseNames max is 1000 IDs
		eeChunk, badChunk, err := eu.resolveIDs(ctx, chunk)
		if err != nil {
			return nil, err
		}
		ee = append(ee, eeChunk...)
		badIDs = append(badIDs, badChunk...)
	}
	for _, entity := range ee {
		_, err := eu.st.GetOrCreateEveEntity(
			ctx,
			entity.Id,
			entity.Name,
			eveEntityCategoryFromESICategory(entity.Category),
		)
		if err != nil {
			return nil, err
		}
	}
	if len(badIDs) > 0 {
		for _, id := range badIDs {
			if _, err := eu.st.GetOrCreateEveEntity(ctx, id, "?", app.EveEntityUnknown); err != nil {
				slog.Error("Failed to mark unresolvable EveEntity", "id", id, "error", err)
			}
		}
		slog.Warn("Marking unresolvable EveEntity IDs as unknown", "ids", badIDs)
	}
	return missingIDs, nil
}

func (eu *EveUniverseService) resolveIDs(ctx context.Context, ids []int32) ([]esi.PostUniverseNames200Ok, []int32, error) {
	slog.Debug("Trying to resolve IDs", "count", len(ids))
	ee, resp, err := eu.esiClient.ESI.UniverseApi.PostUniverseNames(ctx, ids, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			if len(ids) == 1 {
				slog.Warn("found unresolvable ID", "id", ids)
				return []esi.PostUniverseNames200Ok{}, ids, nil
			}
			i := len(ids) / 2
			ee1, bad1, err := eu.resolveIDs(ctx, ids[:i])
			if err != nil {
				return nil, nil, err
			}
			ee2, bad2, err := eu.resolveIDs(ctx, ids[i:])
			if err != nil {
				return nil, nil, err
			}
			return slices.Concat(ee1, ee2), slices.Concat(bad1, bad2), nil
		}
		return nil, nil, err
	}
	slog.Info("Stored newly resolved EveEntities", "count", len(ee))
	return ee, []int32{}, nil
}

func (eu *EveUniverseService) ListEveEntitiesByPartialName(ctx context.Context, partial string) ([]*app.EveEntity, error) {
	return eu.st.ListEveEntitiesByPartialName(ctx, partial)
}

func eveEntityCategoryFromESICategory(c string) app.EveEntityCategory {
	categoryMap := map[string]app.EveEntityCategory{
		"alliance":       app.EveEntityAlliance,
		"character":      app.EveEntityCharacter,
		"corporation":    app.EveEntityCorporation,
		"constellation":  app.EveEntityConstellation,
		"faction":        app.EveEntityFaction,
		"inventory_type": app.EveEntityInventoryType,
		"mailing_list":   app.EveEntityMailList,
		"region":         app.EveEntityRegion,
		"solar_system":   app.EveEntitySolarSystem,
		"station":        app.EveEntityStation,
	}
	c2, ok := categoryMap[c]
	if !ok {
		panic(fmt.Sprintf("Can not map invalid category: %v", c))
	}
	return c2
}
