package logic

import (
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/helper/set"
	"example/esiapp/internal/model"
	"fmt"
	"log/slog"
	"slices"
)

// AddEveEntitiesFromESISearch runs a search on ESI and adds the results as new EveEntity objects to the database.
func AddEveEntitiesFromESISearch(characterID int32, search string) ([]int32, error) {
	token, err := FetchValidToken(characterID)
	if err != nil {
		return nil, err
	}
	categories := []esi.SearchCategory{
		esi.SearchCategoryCorporation,
		esi.SearchCategoryCharacter,
		esi.SearchCategoryAlliance,
	}
	r, err := esi.Search(httpClient, characterID, search, categories, token.AccessToken)
	if err != nil {
		return nil, err
	}
	ids := slices.Concat(r.Alliance, r.Character, r.Corporation)
	missingIDs, err := AddMissingEveEntities(ids)
	if err != nil {
		slog.Error("Failed to fetch missing IDs", "error", err)
		return nil, err
	}
	return missingIDs, nil
}

// AddMissingEveEntities adds EveEntities from ESI for IDs missing in the database.
func AddMissingEveEntities(ids []int32) ([]int32, error) {
	c, err := model.FetchEveEntityIDs()
	if err != nil {
		return nil, err
	}
	current := set.NewFromSlice(c)
	incoming := set.NewFromSlice(ids)
	missing := incoming.Difference(current)

	if missing.Size() == 0 {
		return nil, nil
	}

	entities, err := esi.ResolveEntityIDs(httpClient, missing.ToSlice())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve IDs: %v %v", err, ids)
	}

	for _, entity := range entities {
		e := model.EveEntity{
			ID:       entity.ID,
			Category: model.EveEntityCategory(entity.Category),
			Name:     entity.Name,
		}
		err := e.Save()
		if err != nil {
			return nil, err
		}
	}
	slog.Debug("Added missing eve entities", "count", len(entities))
	return missing.ToSlice(), nil
}
