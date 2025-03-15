package character

import (
	"context"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// SearchESI performs a name search for items on the ESI server
// and returns the results by EveEntity category and sorted by name.
// It also returns the total number of results.
// A total of 500 indicates that we exceeded the server limit.
func (s *CharacterService) SearchESI(ctx context.Context, characterID int32, search string) (map[app.EveEntityCategory][]*app.EveEntity, int, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return nil, 0, err
	}
	categories := []string{
		"alliance",
		"character",
		"corporation",
		"faction",
		"inventory_type",
		"solar_system",
		"station",
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	x, _, err := s.esiClient.ESI.SearchApi.GetCharactersCharacterIdSearch(ctx, categories, characterID, search, nil)
	if err != nil {
		return nil, 0, err
	}
	ids := slices.Concat(x.Alliance, x.Character, x.Corporation, x.Faction, x.InventoryType, x.SolarSystem, x.Station)
	oo, err := s.EveUniverseService.ToEveEntities(ctx, ids)
	if err != nil {
		slog.Error("SearchESI: resolve IDs to eve entities", "error", err)
		return nil, 0, err
	}
	r := make(map[app.EveEntityCategory][]*app.EveEntity)
	for _, o := range oo {
		r[o.Category] = append(r[o.Category], o)
	}
	for _, s := range r {
		slices.SortFunc(s, func(a, b *app.EveEntity) int {
			return a.Compare(b)
		})
	}
	return r, len(ids), nil
}
