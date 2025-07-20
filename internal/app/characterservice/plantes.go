package characterservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) GetPlanet(ctx context.Context, characterID, planetID int32) (*app.CharacterPlanet, error) {
	return s.st.GetCharacterPlanet(ctx, characterID, planetID)
}

func (s *CharacterService) ListAllPlanets(ctx context.Context) ([]*app.CharacterPlanet, error) {
	return s.st.ListAllCharacterPlanets(ctx)
}

func (s *CharacterService) ListPlanets(ctx context.Context, characterID int32) ([]*app.CharacterPlanet, error) {
	return s.st.ListCharacterPlanets(ctx, characterID)
}

// TODO: Improve update logic to only update changes

func (s *CharacterService) updatePlanetsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionCharacterPlanets {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			planets, _, err := s.esiClient.ESI.PlanetaryInteractionApi.GetCharactersCharacterIdPlanets(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			slog.Debug("Received planets from ESI", "characterID", characterID, "count", len(planets))
			return planets, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			// remove obsolete planets
			pp, err := s.st.ListCharacterPlanets(ctx, characterID)
			if err != nil {
				return err
			}
			existing := set.Of[int32]()
			for _, p := range pp {
				existing.Add(p.EvePlanet.ID)
			}
			planets := data.([]esi.GetCharactersCharacterIdPlanets200Ok)
			incoming := set.Of[int32]()
			for _, p := range planets {
				incoming.Add(p.PlanetId)
			}
			obsolete := set.Difference(existing, incoming)
			if err := s.st.DeleteCharacterPlanet(ctx, characterID, obsolete.Slice()); err != nil {
				return err
			}
			// update or create planet
			for _, o := range planets {
				_, err := s.eus.GetOrCreatePlanetESI(ctx, o.PlanetId)
				if err != nil {
					return err
				}
				arg := storage.UpdateOrCreateCharacterPlanetParams{
					CharacterID:  characterID,
					EvePlanetID:  o.PlanetId,
					LastUpdate:   o.LastUpdate,
					UpgradeLevel: int(o.UpgradeLevel),
				}
				characterPlanetID, err := s.st.UpdateOrCreateCharacterPlanet(ctx, arg)
				if err != nil {
					return err
				}
				planet, _, err := s.esiClient.ESI.PlanetaryInteractionApi.GetCharactersCharacterIdPlanetsPlanetId(ctx, characterID, o.PlanetId, nil)
				if err != nil {
					return err
				}
				// replace planet pins
				if err := s.st.DeletePlanetPins(ctx, characterPlanetID); err != nil {
					return err
				}
				for _, pin := range planet.Pins {
					et, err := s.eus.GetOrCreateTypeESI(ctx, pin.TypeId)
					if err != nil {
						return err
					}
					arg := storage.CreatePlanetPinParams{
						CharacterPlanetID: characterPlanetID,
						TypeID:            et.ID,
						PinID:             pin.PinId,
						ExpiryTime:        pin.ExpiryTime,
						InstallTime:       pin.InstallTime,
						LastCycleStart:    pin.LastCycleStart,
					}
					if pin.ExtractorDetails.ProductTypeId != 0 {
						et, err := s.eus.GetOrCreateTypeESI(ctx, pin.ExtractorDetails.ProductTypeId)
						if err != nil {
							return err
						}
						arg.ExtractorProductTypeID = optional.New(et.ID)
					}
					if pin.FactoryDetails.SchematicId != 0 {
						es, err := s.eus.GetOrCreateSchematicESI(ctx, pin.FactoryDetails.SchematicId)
						if err != nil {
							return err
						}
						arg.FactorySchemaID = optional.New(es.ID)
					}
					if pin.SchematicId != 0 {
						es, err := s.eus.GetOrCreateSchematicESI(ctx, pin.SchematicId)
						if err != nil {
							return err
						}
						arg.SchematicID = optional.New(es.ID)
					}
					if err := s.st.CreatePlanetPin(ctx, arg); err != nil {
						return err
					}
				}
			}
			slog.Info("Stored updated planets", "characterID", characterID, "count", len(planets))
			return nil
		})
}
