package eveuniverse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
)

func (eu *EveUniverseService) GetEveLocation(ctx context.Context, id int64) (*app.EveLocation, error) {
	o, err := eu.st.GetEveLocation(ctx, id)
	if errors.Is(err, sqlite.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return o, nil
}

func (eu *EveUniverseService) ListEveLocations(ctx context.Context) ([]*app.EveLocation, error) {
	return eu.st.ListEveLocation(ctx)
}

// GetOrCreateEveLocationESI return a structure when it already exists
// or else tries to fetch and create a new structure from ESI.
//
// Important: A token with the structure scope must be set in the context
func (eu *EveUniverseService) GetOrCreateEveLocationESI(ctx context.Context, id int64) (*app.EveLocation, error) {
	x, err := eu.st.GetEveLocation(ctx, id)
	if errors.Is(err, sqlite.ErrNotFound) {
		return eu.updateOrCreateEveLocationESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

// updateOrCreateEveLocationESI tries to fetch and create a new structure from ESI.
//
// Important: A token with the structure scope must be set in the context when trying to fetch a structure.
func (eu *EveUniverseService) updateOrCreateEveLocationESI(ctx context.Context, id int64) (*app.EveLocation, error) {
	key := fmt.Sprintf("updateOrCreateLocationESI-%d", id)
	y, err, _ := eu.sfg.Do(key, func() (any, error) {
		var arg sqlite.UpdateOrCreateLocationParams
		switch app.LocationVariantFromID(id) {
		case app.EveLocationUnknown:
			t, err := eu.GetOrCreateEveTypeESI(ctx, app.EveTypeSolarSystem)
			if err != nil {
				return nil, err
			}
			arg = sqlite.UpdateOrCreateLocationParams{
				ID:        id,
				EveTypeID: sql.NullInt32{Int32: t.ID, Valid: true},
			}
		case app.EveLocationAssetSafety:
			t, err := eu.GetOrCreateEveTypeESI(ctx, app.EveTypeAssetSafetyWrap)
			if err != nil {
				return nil, err
			}
			arg = sqlite.UpdateOrCreateLocationParams{
				ID:        id,
				EveTypeID: sql.NullInt32{Int32: t.ID, Valid: true},
			}
		case app.EveLocationSolarSystem:
			t, err := eu.GetOrCreateEveTypeESI(ctx, app.EveTypeSolarSystem)
			if err != nil {
				return nil, err
			}
			x, err := eu.GetOrCreateEveSolarSystemESI(ctx, int32(id))
			if err != nil {
				return nil, err
			}
			arg = sqlite.UpdateOrCreateLocationParams{
				ID:               id,
				EveTypeID:        sql.NullInt32{Int32: t.ID, Valid: true},
				EveSolarSystemID: sql.NullInt32{Int32: x.ID, Valid: true},
			}
		case app.EveLocationStation:
			station, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseStationsStationId(ctx, int32(id), nil)
			if err != nil {
				return nil, err
			}
			_, err = eu.GetOrCreateEveSolarSystemESI(ctx, station.SystemId)
			if err != nil {
				return nil, err
			}
			_, err = eu.GetOrCreateEveTypeESI(ctx, station.TypeId)
			if err != nil {
				return nil, err
			}
			arg.EveTypeID = sql.NullInt32{Int32: station.TypeId, Valid: true}
			arg = sqlite.UpdateOrCreateLocationParams{
				ID:               id,
				EveSolarSystemID: sql.NullInt32{Int32: station.SystemId, Valid: true},
				EveTypeID:        sql.NullInt32{Int32: station.TypeId, Valid: true},
				Name:             station.Name,
			}
			if station.Owner != 0 {
				_, err = eu.AddMissingEveEntities(ctx, []int32{station.Owner})
				if err != nil {
					return nil, err
				}
				arg.OwnerID = sql.NullInt32{Int32: station.Owner, Valid: true}
			}
		case app.EveLocationStructure:
			structure, r, err := eu.esiClient.ESI.UniverseApi.GetUniverseStructuresStructureId(ctx, id, nil)
			if err != nil {
				if r != nil && r.StatusCode == http.StatusForbidden {
					arg = sqlite.UpdateOrCreateLocationParams{ID: id}
					break
				}
				return nil, err
			}
			_, err = eu.GetOrCreateEveSolarSystemESI(ctx, structure.SolarSystemId)
			if err != nil {
				return nil, err
			}
			_, err = eu.AddMissingEveEntities(ctx, []int32{structure.OwnerId})
			if err != nil {
				return nil, err
			}
			arg = sqlite.UpdateOrCreateLocationParams{
				ID:               id,
				EveSolarSystemID: sql.NullInt32{Int32: structure.SolarSystemId, Valid: true},
				Name:             structure.Name,
				OwnerID:          sql.NullInt32{Int32: structure.OwnerId, Valid: true},
			}
			if structure.TypeId != 0 {
				myType, err := eu.GetOrCreateEveTypeESI(ctx, structure.TypeId)
				if err != nil {
					return nil, err
				}
				arg.EveTypeID = sql.NullInt32{Int32: myType.ID, Valid: true}
			}
		default:
			return nil, fmt.Errorf("can not update or create structure for invalid ID: %d", id)
		}
		arg.UpdatedAt = time.Now()
		if err := eu.st.UpdateOrCreateEveLocation(ctx, arg); err != nil {
			return nil, err
		}
		return eu.st.GetEveLocation(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveLocation), nil
}