package character

import (
	"context"
	"log/slog"
	"slices"
	"strings"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type SearchCategory string

const (
	SearchAgent         SearchCategory = "agent"
	SearchAlliance      SearchCategory = "alliance"
	SearchCharacter     SearchCategory = "character"
	SearchCorporation   SearchCategory = "corporation"
	SearchFaction       SearchCategory = "faction"
	SearchInventoryType SearchCategory = "inventory_type"
	SearchSolarSystem   SearchCategory = "solar_system"
	SearchStation       SearchCategory = "station"
)

func (x SearchCategory) String() string {
	titler := cases.Title(language.English)
	return titler.String(strings.ReplaceAll(string(x), "_", " "))
}

// SearchCategories returns all available search categories
func SearchCategories() []SearchCategory {
	return []SearchCategory{
		SearchAgent,
		SearchAlliance,
		SearchCharacter,
		SearchCorporation,
		SearchFaction,
		SearchInventoryType,
		SearchSolarSystem,
		SearchStation,
	}
}

// SearchESI performs a name search for items on the ESI server
// and returns the results by EveEntity category and sorted by name.
// It also returns the total number of results.
// A total of 500 indicates that we exceeded the server limit.
func (s *CharacterService) SearchESI(ctx context.Context, characterID int32, search string, categories []SearchCategory, strict bool) (map[SearchCategory][]*app.EveEntity, int, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return nil, 0, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	cc := slices.Collect(xiter.MapSlice(categories, func(a SearchCategory) string {
		return string(a)
	}))
	x, _, err := s.esiClient.ESI.SearchApi.GetCharactersCharacterIdSearch(ctx, cc, characterID, search, &esi.GetCharactersCharacterIdSearchOpts{
		Strict: esioptional.NewBool(strict),
	})
	if err != nil {
		return nil, 0, err
	}
	ids := slices.Concat(x.Agent, x.Alliance, x.Character, x.Corporation, x.Faction, x.InventoryType, x.SolarSystem, x.Station)
	eeMap, err := s.EveUniverseService.ToEveEntities(ctx, ids)
	if err != nil {
		slog.Error("SearchESI: resolve IDs to eve entities", "error", err)
		return nil, 0, err
	}
	categoryMap := map[SearchCategory][]int32{
		SearchAgent:         x.Agent,
		SearchAlliance:      x.Alliance,
		SearchCharacter:     x.Character,
		SearchCorporation:   x.Corporation,
		SearchFaction:       x.Faction,
		SearchInventoryType: x.InventoryType,
		SearchSolarSystem:   x.SolarSystem,
		SearchStation:       x.Station,
	}
	r := make(map[SearchCategory][]*app.EveEntity)
	for c, ids := range categoryMap {
		for _, id := range ids {
			r[c] = append(r[c], eeMap[id])
		}
	}
	for _, s := range r {
		slices.SortFunc(s, func(a, b *app.EveEntity) int {
			return a.Compare(b)
		})
	}
	return r, len(ids), nil
}
