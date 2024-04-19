package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"example/evebuddy/internal/model"
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
		panic(fmt.Sprintf("Can not map unknown category: %v", c))
	}
	return c2
}

// AddEveEntitiesFromESISearch runs a search on ESI and adds the results as new EveEntity objects to the database.
func (s *Service) AddEveEntitiesFromESISearch(characterID int32, search string) ([]int32, error) {
	ctx := context.Background()
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return nil, err
	}
	categories := []string{
		"corporation",
		"character",
		"alliance",
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	r, _, err := s.esiClient.ESI.SearchApi.GetCharactersCharacterIdSearch(ctx, categories, characterID, search, nil)
	if err != nil {
		return nil, err
	}
	ids := slices.Concat(r.Alliance, r.Character, r.Corporation)
	missingIDs, err := s.addMissingEveEntities(ctx, ids)
	if err != nil {
		slog.Error("Failed to fetch missing IDs", "error", err)
		return nil, err
	}
	return missingIDs, nil
}

// addMissingEveEntities adds EveEntities from ESI for IDs missing in the database.
func (s *Service) addMissingEveEntities(ctx context.Context, ids []int32) ([]int32, error) {
	missing, err := s.r.MissingEveEntityIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if missing.Size() == 0 {
		return nil, nil
	}
	entities, _, err := s.esiClient.ESI.UniverseApi.PostUniverseNames(ctx, missing.ToSlice(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve IDs on ESI %v: %w", ids, err)
	}
	for _, entity := range entities {
		_, err := s.r.GetOrCreateEveEntity(ctx, entity.Id, entity.Name, eveEntityCategoryFromESICategory(entity.Category))
		if err != nil {
			return nil, err
		}
	}
	return missing.ToSlice(), nil
}

func (s *Service) ListEveEntitiesByPartialName(partial string) ([]model.EveEntity, error) {
	return s.r.ListEveEntitiesByPartialName(context.Background(), partial)
}

// Resolve slice of unclean EveEntity objects and return as new slice with resolved objects.
// Will return an error if some entities can not be resolved.
func (s *Service) ResolveUncleanEveEntities(ee []model.EveEntity) ([]model.EveEntity, error) {
	ctx := context.Background()
	ee1, names, err := s.resolveEveEntityLocally(ctx, ee)
	if err != nil {
		return nil, err
	}
	if err := s.resolveEveEntityNamesRemotely(ctx, names); err != nil {
		return nil, err
	}
	ee2, err := s.findEveEntitiesByName(ctx, names)
	if err != nil {
		return nil, err
	}
	r := slices.Concat(ee1, ee2)
	return r, nil
}

// resolveEveEntityLocally tries to resolve EveEntities locally.
// It returns resolved recipients and a list of remaining unresolved names (if any)
func (s *Service) resolveEveEntityLocally(ctx context.Context, ee []model.EveEntity) ([]model.EveEntity, []string, error) {
	ee2 := make([]model.EveEntity, 0, len(ee))
	names := make([]string, 0, len(ee))
	for _, r := range ee {
		if r.Category == model.EveEntityUndefined {
			names = append(names, r.Name)
			continue
		}
		ee3, err := s.r.ListEveEntityByNameAndCategory(ctx, r.Name, r.Category)
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
func (s *Service) resolveEveEntityNamesRemotely(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return nil
	}
	r, _, err := s.esiClient.ESI.UniverseApi.PostUniverseIds(ctx, names, nil)
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
	missing, err := s.r.MissingEveEntityIDs(ctx, ids)
	if err != nil {
		return err
	}
	if missing.Size() == 0 {
		return nil
	}
	for _, e := range ee {
		if missing.Has(int32(e.ID)) {
			_, err := s.r.GetOrCreateEveEntity(ctx, e.ID, e.Name, e.Category)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// findEveEntitiesByName tries to build MailRecipient objects from given names
// by checking against EveEntity objects in the database.
// Will abort with errors if no match is found or if multiple matches are found for a name.
func (s *Service) findEveEntitiesByName(ctx context.Context, names []string) ([]model.EveEntity, error) {
	ee2 := make([]model.EveEntity, 0, len(names))
	for _, n := range names {
		ee, err := s.r.ListEveEntitiesByName(ctx, n)
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
