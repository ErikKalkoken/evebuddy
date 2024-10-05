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

var ErrEveEntityNameNoMatch = errors.New("no matching EveEntity name")
var ErrEveEntityNameMultipleMatches = errors.New("multiple matching EveEntity names")

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
	for id := range ids2.All() {
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
			eu.st.GetOrCreateEveEntity(ctx, id, "?", app.EveEntityUnknown)
		}
		slog.Warn("Marking unresolvable EveEntity IDs as unknown", "ids", badIDs)
	}
	return missingIDs, nil
}

func (eu *EveUniverseService) resolveIDs(ctx context.Context, ids []int32) ([]esi.PostUniverseNames200Ok, []int32, error) {
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

func (eu *EveUniverseService) ListEveEntitiesByPartialName(ctx context.Context, partial string) ([]*app.EveEntity, error) {
	return eu.st.ListEveEntitiesByPartialName(ctx, partial)
}

// Resolve slice of unclean EveEntity objects and return as new slice with resolved objects.
// Will return an error if some entities can not be resolved.
func (eu *EveUniverseService) ResolveUncleanEveEntities(ctx context.Context, ee []*app.EveEntity) ([]*app.EveEntity, error) {
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
func (eu *EveUniverseService) resolveEveEntityLocally(ctx context.Context, ee []*app.EveEntity) ([]*app.EveEntity, []string, error) {
	ee2 := make([]*app.EveEntity, 0, len(ee))
	names := make([]string, 0, len(ee))
	for _, r := range ee {
		if r.Category == app.EveEntityUndefined {
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
func (eu *EveUniverseService) resolveEveEntityNamesRemotely(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return nil
	}
	r, _, err := eu.esiClient.ESI.UniverseApi.PostUniverseIds(ctx, names, nil)
	if err != nil {
		return err
	}
	ee := make([]app.EveEntity, 0, len(names))
	for _, o := range r.Alliances {
		e := app.EveEntity{ID: o.Id, Name: o.Name, Category: app.EveEntityAlliance}
		ee = append(ee, e)
	}
	for _, o := range r.Characters {
		e := app.EveEntity{ID: o.Id, Name: o.Name, Category: app.EveEntityCharacter}
		ee = append(ee, e)
	}
	for _, o := range r.Corporations {
		e := app.EveEntity{ID: o.Id, Name: o.Name, Category: app.EveEntityCorporation}
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
		if missing.Contains(int32(e.ID)) {
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
func (eu *EveUniverseService) findEveEntitiesByName(ctx context.Context, names []string) ([]*app.EveEntity, error) {
	ee2 := make([]*app.EveEntity, 0, len(names))
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
