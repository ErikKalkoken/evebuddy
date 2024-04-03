package logic

import (
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/helper/set"
	"example/esiapp/internal/model"
	"fmt"
	"log/slog"
)

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
