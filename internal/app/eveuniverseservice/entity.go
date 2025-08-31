package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/antihax/goesi/esi"
)

const (
	esiPostUniverseNamesMax = 1000
)

// known invalid IDs
var invalidEveEntityIDs = []int32{
	1, // ID is used for fields, which are technically mandatory, but have no value (e.g. creator for NPC corps)
}

func (s *EveUniverseService) GetEntity(ctx context.Context, id int32) (*app.EveEntity, error) {
	return s.st.GetEveEntity(ctx, id)
}

// getValidEntity returns an EveEntity from storage for valid IDs and nil for invalid IDs.
func (s *EveUniverseService) getValidEntity(ctx context.Context, id int32) (*app.EveEntity, error) {
	if id == 0 || id == 1 {
		return nil, nil
	}
	return s.GetEntity(ctx, id)
}

func (s *EveUniverseService) GetOrCreateEntityESI(ctx context.Context, id int32) (*app.EveEntity, error) {
	o, err := s.st.GetEveEntity(ctx, id)
	if err == nil {
		return o, nil
	}
	if !errors.Is(err, app.ErrNotFound) {
		return nil, err
	}
	_, err = s.AddMissingEntities(ctx, set.Of(id))
	if err != nil {
		return nil, err
	}
	return s.st.GetEveEntity(ctx, id)
}

// ToEntities returns the resolved EveEntities for a list of valid entity IDs.
// It guarantees a result for every ID and will map unknown IDs (including 0 & 1) to empty EveEntity objects.
func (s *EveUniverseService) ToEntities(ctx context.Context, ids set.Set[int32]) (map[int32]*app.EveEntity, error) {
	r := make(map[int32]*app.EveEntity)
	if ids.Size() == 0 {
		return r, nil
	}
	ids2 := ids.Clone()
	ids2.Delete(0)
	if _, err := s.AddMissingEntities(ctx, ids2); err != nil {
		return nil, err
	}
	oo, err := s.st.ListEveEntitiesForIDs(ctx, ids2.Slice())
	if err != nil {
		return nil, err
	}
	for _, o := range oo {
		r[o.ID] = o
	}
	for id := range ids.All() {
		_, ok := r[id]
		if !ok {
			r[id] = &app.EveEntity{}
		}
	}
	return r, nil
}

// AddMissingEntities adds EveEntities from ESI for IDs missing in the database
// and returns which IDs where indeed missing.
//
// ID 0 will be ignored
func (s *EveUniverseService) AddMissingEntities(ctx context.Context, ids set.Set[int32]) (set.Set[int32], error) {
	var bad set.Set[int32]
	ids2 := ids.Clone()
	ids2.Delete(0) // ignore zero ID
	if ids2.Size() == 0 {
		return set.Set[int32]{}, nil
	}
	// Filter out known invalid IDs before continuing
	for _, id := range invalidEveEntityIDs {
		if ids2.Contains(id) {
			bad.Add(id)
			ids2.Delete(id)
		}
	}
	wrapErr := func(err error) error {
		return fmt.Errorf("AddMissingEntities: %w", err)
	}
	var missing set.Set[int32]
	if ids2.Size() > 0 {
		// Identify missing IDs
		ids, err := s.st.MissingEveEntityIDs(ctx, ids2)
		if err != nil {
			return set.Set[int32]{}, wrapErr(err)
		}
		missing.AddSeq(ids.All())
	}
	if missing.Size() > 0 {
		slog.Debug("Trying to resolve missing EveEntity IDs from ESI", "ids", missing)
		var ee []esi.PostUniverseNames200Ok
		for chunk := range slices.Chunk(missing.Slice(), esiPostUniverseNamesMax) {
			eeChunk, badChunk, err := s.resolveIDsFromESI(ctx, chunk)
			if err != nil {
				return set.Set[int32]{}, wrapErr(err)
			}
			ee = append(ee, eeChunk...)
			bad.AddSeq(slices.Values(badChunk))
		}
		for _, entity := range ee {
			_, err := s.st.UpdateOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
				ID:       entity.Id,
				Name:     entity.Name,
				Category: eveEntityCategoryFromESICategory(entity.Category),
			})
			if err != nil {
				return set.Set[int32]{}, wrapErr(err)
			}
		}
		slog.Info("Stored newly resolved EveEntities", "count", len(ee))
	}
	if bad.Size() > 0 {
		// Mark unresolvable IDs
		var marked set.Set[int32]
		for id := range bad.All() {
			arg := storage.CreateEveEntityParams{
				ID:       id,
				Name:     "?",
				Category: app.EveEntityUnknown,
			}
			if _, err := s.st.GetOrCreateEveEntity(ctx, arg); err != nil {
				slog.Error("Failed to mark unresolvable EveEntity", "id", id, "error", err)
				continue
			}
			marked.Add(id)
		}
		slog.Warn("Marked unresolvable EveEntity IDs as unknown", "ids", marked)
	}
	return set.Union(missing, bad), nil
}

func (s *EveUniverseService) resolveIDsFromESI(ctx context.Context, ids []int32) ([]esi.PostUniverseNames200Ok, []int32, error) {
	slog.Debug("Trying to resolve IDs from ESI", "count", len(ids))
	ee, resp, err := s.esiClient.ESI.UniverseApi.PostUniverseNames(ctx, ids, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			if len(ids) == 1 {
				slog.Warn("found unresolvable ID", "id", ids)
				return []esi.PostUniverseNames200Ok{}, ids, nil
			}
			i := len(ids) / 2
			ee1, bad1, err := s.resolveIDsFromESI(ctx, ids[:i])
			if err != nil {
				return nil, nil, err
			}
			ee2, bad2, err := s.resolveIDsFromESI(ctx, ids[i:])
			if err != nil {
				return nil, nil, err
			}
			return slices.Concat(ee1, ee2), slices.Concat(bad1, bad2), nil
		}
		return nil, nil, err
	}
	return ee, []int32{}, nil
}

func (s *EveUniverseService) ListEntitiesByPartialName(ctx context.Context, partial string) ([]*app.EveEntity, error) {
	return s.st.ListEveEntitiesByPartialName(ctx, partial)
}

func (s *EveUniverseService) ListEntitiesForIDs(ctx context.Context, ids []int32) ([]*app.EveEntity, error) {
	return s.st.ListEveEntitiesForIDs(ctx, ids)
}

func (s *EveUniverseService) UpdateAllEntitiesESI(ctx context.Context) (set.Set[int32], error) {
	var changed set.Set[int32]
	ee, err := s.st.ListEveEntities(ctx)
	if err != nil {
		return changed, err
	}
	selectedCategories := set.Of(
		app.EveEntityAlliance,
		app.EveEntityCharacter,
		app.EveEntityCorporation,
	)
	ids := slices.Collect(xiter.Map(xiter.Filter(slices.Values(ee), func(x *app.EveEntity) bool {
		return x.IsValid() && selectedCategories.Contains(x.Category)
	}), func(x *app.EveEntity) int32 {
		return x.ID
	}))
	var objs []esi.PostUniverseNames200Ok
	for chunk := range slices.Chunk(ids, esiPostUniverseNamesMax) {
		objsChunk, _, err := s.resolveIDsFromESI(ctx, chunk)
		if err != nil {
			return changed, err
		}
		objs = append(objs, objsChunk...)
	}
	incoming := make(map[int32]esi.PostUniverseNames200Ok)
	for _, o := range objs {
		incoming[o.Id] = o
	}
	for _, e := range ee {
		o, ok := incoming[e.ID]
		if !ok || o.Name == e.Name {
			continue
		}
		if err := s.st.UpdateEveEntity(ctx, o.Id, o.Name); err != nil {
			return changed, err
		}
		changed.Add(o.Id)
	}
	slog.Info("Updated Eve Entities", "count", len(objs), "changed", changed)
	return changed, nil
}

func (s *EveUniverseService) updateEntityNameIfExists(ctx context.Context, id int32, name string) error {
	o, err := s.st.GetEveEntity(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.st.UpdateEveEntity(ctx, o.ID, name)
}
