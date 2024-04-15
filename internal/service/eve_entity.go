package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"example/evebuddy/internal/repository"
)

func eveEntityCategoryFromESICategory(c string) repository.EveEntityCategory {
	categoryMap := map[string]repository.EveEntityCategory{
		"alliance":     repository.EveEntityAlliance,
		"character":    repository.EveEntityCharacter,
		"corporation":  repository.EveEntityCorporation,
		"faction":      repository.EveEntityFaction,
		"mailing:list": repository.EveEntityMailList,
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
	token, err := s.GetValidToken(ctx, characterID)
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
	missingIDs, err := s.addMissingEveEntities(ids)
	if err != nil {
		slog.Error("Failed to fetch missing IDs", "error", err)
		return nil, err
	}
	return missingIDs, nil
}

// addMissingEveEntities adds EveEntities from ESI for IDs missing in the database.
func (s *Service) addMissingEveEntities(ids []int32) ([]int32, error) {
	ctx := context.Background()
	missing, err := s.r.MissingEveEntityIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if missing.Size() == 0 {
		return nil, nil
	}
	entities, _, err := s.esiClient.ESI.UniverseApi.PostUniverseNames(ctx, missing.ToSlice(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve IDs: %v %v", err, ids)
	}
	for _, entity := range entities {
		_, err := s.r.CreateEveEntity(ctx, entity.Id, entity.Name, eveEntityCategoryFromESICategory(entity.Category))
		if err != nil {
			return nil, err
		}
	}
	slog.Debug("Added missing eve entities", "count", len(entities))
	return missing.ToSlice(), nil
}

func (s *Service) SearchEveEntitiesByName(partial string) ([]repository.EveEntity, error) {
	return s.r.SearchEveEntitiesByName(context.Background(), partial)
}
