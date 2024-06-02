package eveuniverse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/model"

	"github.com/antihax/goesi/esi"
)

var ErrEveEntityNameNoMatch = errors.New("no matching EveEntity name")
var ErrEveEntityNameMultipleMatches = errors.New("multiple matching EveEntity names")

func eveEntityCategoryFromESICategory(c string) model.EveEntityCategory {
	categoryMap := map[string]model.EveEntityCategory{
		"alliance":       model.EveEntityAlliance,
		"character":      model.EveEntityCharacter,
		"corporation":    model.EveEntityCorporation,
		"constellation":  model.EveEntityConstellation,
		"faction":        model.EveEntityFaction,
		"inventory_type": model.EveEntityInventoryType,
		"mailing_list":   model.EveEntityMailList,
		"region":         model.EveEntityRegion,
		"solar_system":   model.EveEntitySolarSystem,
		"station":        model.EveEntityStation,
	}
	c2, ok := categoryMap[c]
	if !ok {
		panic(fmt.Sprintf("Can not map invalid category: %v", c))
	}
	return c2
}

// AddMissingEveEntities adds EveEntities from ESI for IDs missing in the database.
func (eu *EveUniverse) AddMissingEveEntities(ctx context.Context, ids []int32) ([]int32, error) {
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
		slog.Info("Trying to resolve EveEntity IDs from ESI", "ids", missingIDs)
	}
	var ee []esi.PostUniverseNames200Ok
	var badIDs []int32
	for _, chunk := range chunkBy(missingIDs, 1000) { // PostUniverseNames max is 1000 IDs
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
			eu.st.GetOrCreateEveEntity(ctx, id, "?", model.EveEntityUnknown)
		}
		slog.Warn("Marking unresolvable EveEntity IDs as unknown", "ids", badIDs)
	}
	return missingIDs, nil
}

func (eu *EveUniverse) resolveIDs(ctx context.Context, ids []int32) ([]esi.PostUniverseNames200Ok, []int32, error) {
	slog.Info("Trying to resolve IDs", "count", len(ids))
	ee, resp, err := eu.esiClient.ESI.UniverseApi.PostUniverseNames(ctx, ids, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			if len(ids) == 1 {
				slog.Warn("found unresolvable ID", "id", ids)
				return []esi.PostUniverseNames200Ok{}, ids, nil
			} else {
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
		}
		return nil, nil, err
	}
	return ee, []int32{}, nil
}

func (eu *EveUniverse) ListEveEntitiesByPartialName(ctx context.Context, partial string) ([]*model.EveEntity, error) {
	return eu.st.ListEveEntitiesByPartialName(ctx, partial)
}

// Resolve slice of unclean EveEntity objects and return as new slice with resolved objects.
// Will return an error if some entities can not be resolved.
func (eu *EveUniverse) ResolveUncleanEveEntities(ctx context.Context, ee []*model.EveEntity) ([]*model.EveEntity, error) {
	ee1, names, err := eu.resolveEveEntityLocally(ctx, ee)
	if err != nil {
		return nil, err
	}
	if err := eu.resolveEveEntityNamesRemotely(ctx, names); err != nil {
		return nil, err
	}
	ee2, err := eu.findEveEntitiesByName(ctx, names)
	if err != nil {
		return nil, err
	}
	r := slices.Concat(ee1, ee2)
	return r, nil
}

// resolveEveEntityLocally tries to resolve EveEntities locally.
// It returns resolved recipients and a list of remaining unresolved names (if any)
func (eu *EveUniverse) resolveEveEntityLocally(ctx context.Context, ee []*model.EveEntity) ([]*model.EveEntity, []string, error) {
	ee2 := make([]*model.EveEntity, 0, len(ee))
	names := make([]string, 0, len(ee))
	for _, r := range ee {
		if r.Category == model.EveEntityUndefined {
			names = append(names, r.Name)
			continue
		}
		ee3, err := eu.st.ListEveEntityByNameAndCategory(ctx, r.Name, r.Category)
		if err != nil {
			return nil, nil, err
		}
		if len(ee3) == 0 {
			names = append(names, r.Name)
			continue
		}
		if len(ee3) > 1 {
			return nil, nil, fmt.Errorf("entity %v: %w", r, ErrEveEntityNameMultipleMatches)
		}
		ee2 = append(ee2, ee3[0])
	}
	return ee2, names, nil
}

// resolveEveEntityNamesRemotely resolves a list of names remotely and stores them as EveEntity objects.
func (eu *EveUniverse) resolveEveEntityNamesRemotely(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return nil
	}
	r, _, err := eu.esiClient.ESI.UniverseApi.PostUniverseIds(ctx, names, nil)
	if err != nil {
		return err
	}
	ee := make([]model.EveEntity, 0, len(names))
	for _, o := range r.Alliances {
		e := model.EveEntity{ID: o.Id, Name: o.Name, Category: model.EveEntityAlliance}
		ee = append(ee, e)
	}
	for _, o := range r.Characters {
		e := model.EveEntity{ID: o.Id, Name: o.Name, Category: model.EveEntityCharacter}
		ee = append(ee, e)
	}
	for _, o := range r.Corporations {
		e := model.EveEntity{ID: o.Id, Name: o.Name, Category: model.EveEntityCorporation}
		ee = append(ee, e)
	}
	ids := make([]int32, len(ee))
	for i, e := range ee {
		ids[i] = e.ID
	}
	missing, err := eu.st.MissingEveEntityIDs(ctx, ids)
	if err != nil {
		return err
	}
	if missing.Size() == 0 {
		return nil
	}
	for _, e := range ee {
		if missing.Has(int32(e.ID)) {
			_, err := eu.st.GetOrCreateEveEntity(ctx, e.ID, e.Name, e.Category)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// findEveEntitiesByName tries to build EveEntity objects from given names
// by checking against EveEntity objects in the database.
// Will abort with errors if no match is found or if multiple matches are found for a name.
func (eu *EveUniverse) findEveEntitiesByName(ctx context.Context, names []string) ([]*model.EveEntity, error) {
	ee2 := make([]*model.EveEntity, 0, len(names))
	for _, n := range names {
		ee, err := eu.st.ListEveEntitiesByName(ctx, n)
		if err != nil {
			return nil, err
		}
		if len(ee) == 0 {
			return nil, fmt.Errorf("%s: %w", n, ErrEveEntityNameNoMatch)
		}
		if len(ee) > 1 {
			return nil, fmt.Errorf("%s: %w", n, ErrEveEntityNameMultipleMatches)
		}
		e := ee[0]
		ee2 = append(ee2, e)
	}
	return ee2, nil
}
