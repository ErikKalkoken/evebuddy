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

// known invalid IDs
var invalidEveEntityIDs = []int32{
	1, // ID is used for fields, which are technically mandatory, but have no value (e.g. creator for NPC corps)
}

func (s *EveUniverseService) GetEveEntity(ctx context.Context, id int32) (*app.EveEntity, error) {
	return s.st.GetEveEntity(ctx, id)
}

// getValidEveEntity returns an EveEntity from storage for valid IDs and nil for invalid IDs.
func (s *EveUniverseService) getValidEveEntity(ctx context.Context, id int32) (*app.EveEntity, error) {
	if id == 0 || id == 1 {
		return nil, nil
	}
	return s.GetEveEntity(ctx, id)
}

func (s *EveUniverseService) GetOrCreateEveEntityESI(ctx context.Context, id int32) (*app.EveEntity, error) {
	o, err := s.st.GetEveEntity(ctx, id)
	if err == nil {
		return o, nil
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	_, err = s.AddMissingEveEntities(ctx, []int32{id})
	if err != nil {
		return nil, err
	}
	return s.st.GetEveEntity(ctx, id)
}

// TODO: Reduce DB calls with AddMissingEveEntities

// ToEveEntities returns the resolved EveEntities for a list of valid entity IDs.
// ID 0 will be resolved to a zero value EveEntity object
// Will return an error if any ID can not be resolved.
func (s *EveUniverseService) ToEveEntities(ctx context.Context, ids []int32) (map[int32]*app.EveEntity, error) {
	r := make(map[int32]*app.EveEntity)
	ids2 := set.NewFromSlice(ids)
	if ids2.Contains(0) {
		r[0] = &app.EveEntity{}
		ids2.Remove(0)
	}
	if _, err := s.AddMissingEveEntities(ctx, ids2.ToSlice()); err != nil {
		return nil, err
	}
	for id := range ids2.Values() {
		x, err := s.GetEveEntity(ctx, id)
		if err != nil {
			return nil, err
		}
		r[id] = x
	}
	return r, nil
}

// AddMissingEveEntities adds EveEntities from ESI for IDs missing in the database
// and returns which IDs where indeed missing.
//
// Invalid IDs (e.g. 0, 1) will be ignored.
func (s *EveUniverseService) AddMissingEveEntities(ctx context.Context, ids []int32) ([]int32, error) {
	// Filter out known invalid IDs before continuing
	var badIDs, missingIDs []int32
	err := func() error {
		ids2 := set.NewFromSlice(ids)
		ids2.Remove(0) // do nothring with ID 0
		for _, id := range invalidEveEntityIDs {
			if ids2.Contains(id) {
				badIDs = append(badIDs, 1)
				ids2.Remove(1)
			}
		}
		if ids2.Size() == 0 {
			return nil
		}
		// Identify missing IDs
		missing, err := s.st.MissingEveEntityIDs(ctx, ids2.ToSlice())
		if err != nil {
			return err
		}
		if missing.Size() == 0 {
			return nil
		}
		// Call ESI to resolve missing IDs
		missingIDs = missing.ToSlice()
		slices.Sort(missingIDs)
		if len(missingIDs) > 0 {
			slog.Debug("Trying to resolve EveEntity IDs from ESI", "ids", missingIDs)
		}
		var ee []esi.PostUniverseNames200Ok
		for chunk := range slices.Chunk(missingIDs, 1000) { // PostUniverseNames max is 1000 IDs
			eeChunk, badChunk, err := s.resolveIDs(ctx, chunk)
			if err != nil {
				return err
			}
			ee = append(ee, eeChunk...)
			badIDs = append(badIDs, badChunk...)
		}
		for _, entity := range ee {
			_, err := s.st.GetOrCreateEveEntity(
				ctx,
				entity.Id,
				entity.Name,
				eveEntityCategoryFromESICategory(entity.Category),
			)
			if err != nil {
				return err
			}
		}
		return nil
	}()
	if err != nil {
		return nil, err
	}
	if len(badIDs) > 0 {
		for _, id := range badIDs {
			if _, err := s.st.GetOrCreateEveEntity(ctx, id, "?", app.EveEntityUnknown); err != nil {
				slog.Error("Failed to mark unresolvable EveEntity", "id", id, "error", err)
			}
		}
		slog.Warn("Marking unresolvable EveEntity IDs as unknown", "ids", badIDs)
	}
	return missingIDs, nil
}

func (s *EveUniverseService) resolveIDs(ctx context.Context, ids []int32) ([]esi.PostUniverseNames200Ok, []int32, error) {
	slog.Debug("Trying to resolve IDs", "count", len(ids))
	ee, resp, err := s.esiClient.ESI.UniverseApi.PostUniverseNames(ctx, ids, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			if len(ids) == 1 {
				slog.Warn("found unresolvable ID", "id", ids)
				return []esi.PostUniverseNames200Ok{}, ids, nil
			}
			i := len(ids) / 2
			ee1, bad1, err := s.resolveIDs(ctx, ids[:i])
			if err != nil {
				return nil, nil, err
			}
			ee2, bad2, err := s.resolveIDs(ctx, ids[i:])
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

func (s *EveUniverseService) ListEveEntitiesByPartialName(ctx context.Context, partial string) ([]*app.EveEntity, error) {
	return s.st.ListEveEntitiesByPartialName(ctx, partial)
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
