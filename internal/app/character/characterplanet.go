package character

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) ListCharacterPlanets(ctx context.Context, characterID int32) ([]*app.CharacterPlanet, error) {
	return s.st.ListCharacterPlanets(ctx, characterID)
}

// TODO: Improve update logic to only update changes

func (s *CharacterService) updateCharacterPlanetsESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionPlanets {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			planets, _, err := s.esiClient.ESI.PlanetaryInteractionApi.GetCharactersCharacterIdPlanets(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return planets, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			planets := data.([]esi.GetCharactersCharacterIdPlanets200Ok)
			args := make([]storage.CreateCharacterPlanetParams, len(planets))
			for i, o := range planets {
				_, err := s.EveUniverseService.GetOrCreateEvePlanetESI(ctx, o.PlanetId)
				if err != nil {
					return err
				}
				args[i] = storage.CreateCharacterPlanetParams{
					CharacterID:  characterID,
					EvePlanetID:  o.PlanetId,
					LastUpdate:   o.LastUpdate,
					UpgradeLevel: int(o.UpgradeLevel),
				}
			}
			_, err := s.st.ReplaceCharacterPlanets(ctx, characterID, args)
			if err != nil {
				return err
			}
			planets2, err := s.st.ListCharacterPlanets(ctx, characterID)
			if err != nil {
				return err
			}
			for _, p := range planets2 {
				planet, _, err := s.esiClient.ESI.PlanetaryInteractionApi.GetCharactersCharacterIdPlanetsPlanetId(ctx, characterID, p.EvePlanet.ID, nil)
				if err != nil {
					return err
				}
				for _, pin := range planet.Pins {
					// create pins
					et, err := s.EveUniverseService.GetOrCreateEveTypeESI(ctx, pin.TypeId)
					if err != nil {
						return err
					}
					arg := storage.CreatePlanetPinParams{
						CharacterPlanetID: p.ID,
						TypeID:            et.ID,
						PinID:             pin.PinId,
						ExpiryTime:        pin.ExpiryTime,
						InstallTime:       pin.InstallTime,
						LastCycleStart:    pin.LastCycleStart,
					}
					if pin.ExtractorDetails.ProductTypeId != 0 {
						et, err := s.EveUniverseService.GetOrCreateEveTypeESI(ctx, pin.ExtractorDetails.ProductTypeId)
						if err != nil {
							return err
						}
						arg.ExtractorProductTypeID = optional.New(et.ID)
					}
					if pin.FactoryDetails.SchematicId != 0 {
						es, err := s.EveUniverseService.GetOrCreateEveSchematicESI(ctx, pin.FactoryDetails.SchematicId)
						if err != nil {
							return err
						}
						arg.FactorySchemaID = optional.New(es.ID)
					}
					if pin.SchematicId != 0 {
						es, err := s.EveUniverseService.GetOrCreateEveSchematicESI(ctx, pin.SchematicId)
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
			return nil
		})
}
