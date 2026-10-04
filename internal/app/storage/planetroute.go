package storage

import (
	"context"
	"fmt"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreatePlanetRouteParams struct {
	CharacterPlanetID int64
	ContentTypeID     int64
	DestinationPinID  int64
	Quantity          int64
	RouteID           int64
	SourcePinID       int64
}

func (st *Storage) CreatePlanetRoute(ctx context.Context, arg CreatePlanetRouteParams) error {
	if err := createPlanetRoute(ctx, st.qRW, arg); err != nil {
		return fmt.Errorf("CreatePlanetRoute: %+v: %w", arg, err)
	}
	return nil
}

func createPlanetRoute(ctx context.Context, q *queries.Queries, arg CreatePlanetRouteParams) error {
	if arg.CharacterPlanetID == 0 || arg.RouteID == 0 || arg.ContentTypeID == 0 || arg.SourcePinID == 0 || arg.DestinationPinID == 0 {
		return app.ErrInvalid
	}
	return q.CreatePlanetRoute(ctx, queries.CreatePlanetRouteParams{
		CharacterPlanetID: arg.CharacterPlanetID,
		ContentTypeID:     arg.ContentTypeID,
		DestinationPinID:  arg.DestinationPinID,
		Quantity:          arg.Quantity,
		RouteID:           arg.RouteID,
		SourcePinID:       arg.SourcePinID,
	})
}

func (st *Storage) ListPlanetRoutes(ctx context.Context, characterPlanetID int64) ([]*app.PlanetRoute, error) {
	if characterPlanetID == 0 {
		return nil, fmt.Errorf("ListPlanetRoutes: %d: %w", characterPlanetID, app.ErrInvalid)
	}
	m, err := st.listPlanetRoutesByPlanet(ctx, st.qRO, set.Of(characterPlanetID))
	if err != nil {
		return nil, err
	}
	return m[characterPlanetID], nil
}

// listPlanetRoutesByPlanet returns the routes for the given character planets, keyed by character planet ID.
func (st *Storage) listPlanetRoutesByPlanet(ctx context.Context, q *queries.Queries, characterPlanetIDs set.Set[int64]) (map[int64][]*app.PlanetRoute, error) {
	m := make(map[int64][]*app.PlanetRoute)
	for idsChunk := range slices.Chunk(slices.Collect(characterPlanetIDs.All()), st.MaxIDsPerQuery) {
		rows, err := q.ListPlanetRoutesForCharacterPlanetIDs(ctx, idsChunk)
		if err != nil {
			return nil, fmt.Errorf("list planet routes for %d character planets: %w", len(idsChunk), err)
		}
		for _, r := range rows {
			id := r.PlanetRoute.CharacterPlanetID
			m[id] = append(m[id], &app.PlanetRoute{
				ContentType:      eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory),
				DestinationPinID: r.PlanetRoute.DestinationPinID,
				Quantity:         r.PlanetRoute.Quantity,
				RouteID:          r.PlanetRoute.RouteID,
				SourcePinID:      r.PlanetRoute.SourcePinID,
			})
		}
	}
	return m, nil
}
