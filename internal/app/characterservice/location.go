package characterservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/antihax/goesi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func (s *CharacterService) updateLocationESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterLocation {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdLocation")
			location, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdLocation(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return location, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			location := data.(esi.GetCharactersCharacterIdLocationOk)
			var locationID int64
			switch {
			case location.StructureId != 0:
				locationID = location.StructureId
			case location.StationId != 0:
				locationID = int64(location.StationId)
			default:
				locationID = int64(location.SolarSystemId)
			}
			x, err := s.eus.GetOrCreateLocationESI(ctx, locationID)
			if err != nil {
				return err
			}
			if x.Variant() == app.EveLocationStructure && x.SolarSystem == nil {
				err := func() error {
					_, err := s.eus.GetOrCreateSolarSystemESI(ctx, location.SolarSystemId)
					if err != nil {
						return err
					}
					err = s.st.UpdateOrCreateEveLocation(ctx, storage.UpdateOrCreateLocationParams{
						ID:            locationID,
						Name:          x.Name,
						SolarSystemID: optional.New(location.SolarSystemId),
						UpdatedAt:     x.UpdatedAt,
					})
					return err
				}()
				if err != nil {
					slog.Error("Failed to update solar system for unknown character location", "characterID", characterID, "error", err)
				}
			}
			if err := s.st.UpdateCharacterLocation(ctx, characterID, optional.New(locationID)); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) updateOnlineESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterOnline {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdOnline")
			online, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdOnline(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return online, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			online := data.(esi.GetCharactersCharacterIdOnlineOk)
			if err := s.st.UpdateCharacterLastLoginAt(ctx, characterID, optional.New(online.LastLogin)); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) updateShipESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterShip {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdShip")
			ship, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdShip(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return ship, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			ship := data.(esi.GetCharactersCharacterIdShipOk)
			_, err := s.eus.GetOrCreateTypeESI(ctx, ship.ShipTypeId)
			if err != nil {
				return err
			}
			if err := s.st.UpdateCharacterShip(ctx, characterID, optional.New(ship.ShipTypeId)); err != nil {
				return err
			}
			return nil
		})
}
