package storage

import (
	"context"
	"fmt"
	"slices"

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
	wrapErr := func(err error) error {
		return fmt.Errorf("CreatePlanetRoute: %+v: %w", arg, err)
	}
	if arg.CharacterPlanetID == 0 || arg.RouteID == 0 || arg.ContentTypeID == 0 || arg.SourcePinID == 0 || arg.DestinationPinID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.CreatePlanetRoute(ctx, queries.CreatePlanetRouteParams{
		CharacterPlanetID: arg.CharacterPlanetID,
		ContentTypeID:     arg.ContentTypeID,
		DestinationPinID:  arg.DestinationPinID,
		Quantity:          arg.Quantity,
		RouteID:           arg.RouteID,
		SourcePinID:       arg.SourcePinID,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeletePlanetRoutes(ctx context.Context, characterPlanetID int64) error {
	if err := st.qRW.DeletePlanetRoutes(ctx, characterPlanetID); err != nil {
		return fmt.Errorf("delete planet routes for %d: %w", characterPlanetID, err)
	}
	return nil
}

func (st *Storage) ListPlanetRoutes(ctx context.Context, characterPlanetID int64) ([]*app.PlanetRoute, error) {
	if characterPlanetID == 0 {
		return nil, fmt.Errorf("ListPlanetRoutes: %d: %w", characterPlanetID, app.ErrInvalid)
	}
	m, err := st.listPlanetRoutesByPlanet(ctx, []int64{characterPlanetID})
	if err != nil {
		return nil, err
	}
	return m[characterPlanetID], nil
}

// listPlanetRoutesByPlanet returns the routes for the given character planets, keyed by character planet ID.
func (st *Storage) listPlanetRoutesByPlanet(ctx context.Context, characterPlanetIDs []int64) (map[int64][]*app.PlanetRoute, error) {
	m := make(map[int64][]*app.PlanetRoute)
	for idsChunk := range slices.Chunk(characterPlanetIDs, st.MaxIDsPerQuery) {
		rows, err := st.qRO.ListPlanetRoutesForCharacterPlanetIDs(ctx, idsChunk)
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
