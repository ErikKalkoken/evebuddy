package characterservice

import (
	"context"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
)

// AddEveEntitiesFromSearchESI runs a search on ESI and adds the results as new EveEntity objects to the database.
// This method performs a character specific search and needs a token.
func (s *CharacterService) AddEveEntitiesFromSearchESI(ctx context.Context, characterID int32, search string) ([]int32, error) {
	token, err := s.GetValidCharacterToken(ctx, characterID)
	if err != nil {
		return nil, err
	}
	categories := []string{
		"corporation",
		"character",
		"alliance",
	}
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
	r, _, err := s.esiClient.ESI.SearchApi.GetCharactersCharacterIdSearch(ctx, categories, characterID, search, nil)
	if err != nil {
		return nil, err
	}
	ids := set.Union(set.Of(r.Alliance...), set.Of(r.Character...), set.Of(r.Corporation...))
	missingIDs, err := s.eus.AddMissingEntities(ctx, ids)
	if err != nil {
		slog.Error("Failed to fetch missing IDs", "error", err)
		return nil, err
	}
	return missingIDs.Slice(), nil
}

// SearchESI performs a name search for items on the ESI server
// and returns the results by EveEntity category and sorted by name.
// It also returns the total number of results.
// A total of 500 indicates that we exceeded the server limit.
func (s *CharacterService) SearchESI(ctx context.Context, search string, categories []app.SearchCategory, strict bool) (map[app.SearchCategory][]*app.EveEntity, int, error) {
	c, err := s.GetAnyCharacter(ctx)
	if err != nil {
		return nil, 0, err
	}
	token, err := s.GetValidCharacterToken(ctx, c.ID)
	if err != nil {
		return nil, 0, err
	}
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
	cc := xslices.Map(categories, func(a app.SearchCategory) string {
		return string(a)
	})
	x, _, err := s.esiClient.ESI.SearchApi.GetCharactersCharacterIdSearch(
		ctx,
		cc,
		c.ID,
		search,
		&esi.GetCharactersCharacterIdSearchOpts{Strict: esioptional.NewBool(strict)},
	)
	if err != nil {
		return nil, 0, err
	}
	ids := set.Of(slices.Concat(
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
	)...)
	eeMap, err := s.eus.ToEntities(ctx, ids)
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
	for c, ids2 := range categoryMap {
		for _, id := range ids2 {
			r[c] = append(r[c], eeMap[id])
		}
	}
	for _, s := range r {
		slices.SortFunc(s, func(a, b *app.EveEntity) int {
			return a.Compare(b)
		})
	}
	return r, ids.Size(), nil
}
