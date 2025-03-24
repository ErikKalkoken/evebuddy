package characterservice

import (
	"context"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
)

// SearchESI performs a name search for items on the ESI server
// and returns the results by EveEntity category and sorted by name.
// It also returns the total number of results.
// A total of 500 indicates that we exceeded the server limit.
func (s *CharacterService) SearchESI(
	ctx context.Context,
	characterID int32,
	search string,
	categories []app.SearchCategory, strict bool,
) (map[app.SearchCategory][]*app.EveEntity, int, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return nil, 0, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	cc := slices.Collect(xiter.MapSlice(categories, func(a app.SearchCategory) string {
		return string(a)
	}))
	x, _, err := s.esiClient.ESI.SearchApi.GetCharactersCharacterIdSearch(
		ctx,
		cc,
		characterID,
		search,
		&esi.GetCharactersCharacterIdSearchOpts{
			Strict: esioptional.NewBool(strict),
		})
	if err != nil {
		return nil, 0, err
	}
	ids := slices.Concat(
		x.Agent,
		x.Alliance,
		x.Character,
		x.Corporation,
		x.Constellation,
		x.Faction,
		x.InventoryType,
		x.SolarSystem,
		x.Station,
		x.Region,
	)
	eeMap, err := s.EveUniverseService.ToEveEntities(ctx, ids)
	if err != nil {
		slog.Error("SearchESI: resolve IDs to eve entities", "error", err)
		return nil, 0, err
	}
	categoryMap := map[app.SearchCategory][]int32{
		app.SearchAgent:         x.Agent,
		app.SearchAlliance:      x.Alliance,
		app.SearchCharacter:     x.Character,
		app.SearchConstellation: x.Constellation,
		app.SearchCorporation:   x.Corporation,
		app.SearchFaction:       x.Faction,
		app.SearchRegion:        x.Region,
		app.SearchSolarSystem:   x.SolarSystem,
		app.SearchStation:       x.Station,
		app.SearchType:          x.InventoryType,
	}
	r := make(map[app.SearchCategory][]*app.EveEntity)
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
