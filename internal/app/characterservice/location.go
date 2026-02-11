package characterservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/fnt-eve/goesi-openapi/esi"

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
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdLocation")
			location, _, err := s.esiClient.LocationAPI.GetCharactersCharacterIdLocation(ctx, characterID).Execute()
			if err != nil {
				return false, err
			}
			return location, nil
		},
		func(ctx context.Context, characterID int64, data any) error {
			location := data.(*esi.CharactersCharacterIdLocationGet)
			var locationID int64
			if x := location.StructureId; x != nil {
				locationID = *x
			} else if y := location.StationId; y != nil {
				locationID = *y
			} else {
				locationID = location.SolarSystemId
			}
			el, err := s.eus.GetOrCreateLocationESI(ctx, locationID)
			if err != nil {
				return err
			}
			if el.Variant() == app.EveLocationStructure && el.SolarSystem.IsEmpty() {
				err := func() error {
					_, err := s.eus.GetOrCreateSolarSystemESI(ctx, location.SolarSystemId)
					if err != nil {
						return err
					}
					err = s.st.UpdateOrCreateEveLocation(ctx, storage.UpdateOrCreateLocationParams{
						ID:            locationID,
						Name:          el.Name,
						SolarSystemID: optional.New(location.SolarSystemId),
						UpdatedAt:     el.UpdatedAt,
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
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdOnline")
			online, _, err := s.esiClient.LocationAPI.GetCharactersCharacterIdOnline(ctx, characterID).Execute()
			if err != nil {
				return false, err
			}
			return online, nil
		},
		func(ctx context.Context, characterID int64, data any) error {
			online := data.(*esi.CharactersCharacterIdOnlineGet)
			err := s.st.UpdateCharacterLastLoginAt(ctx, characterID, optional.FromPtr(online.LastLogin))
			if err != nil {
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
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdShip")
			ship, _, err := s.esiClient.LocationAPI.GetCharactersCharacterIdShip(ctx, characterID).Execute()
			if err != nil {
				return false, err
			}
			return ship, nil
		},
		func(ctx context.Context, characterID int64, data any) error {
			ship := data.(*esi.CharactersCharacterIdShipGet)
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
