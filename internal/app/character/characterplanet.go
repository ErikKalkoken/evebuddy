package character

import (
	"context"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) ListCharacterPlanets(ctx context.Context, characterID int32) ([]*app.CharacterPlanet, error) {
	return s.st.ListCharacterPlanets(ctx, characterID)
}

func (s *CharacterService) UpdateCharacterPlanetLastNotified(ctx context.Context, characterID, evePlanetID int32, t time.Time) error {
	arg := storage.UpdateCharacterPlanetLastNotifiedParams{
		CharacterID:  characterID,
		EvePlanetID:  evePlanetID,
		LastNotified: t,
	}
	return s.st.UpdateCharacterPlanetLastNotified(ctx, arg)
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
			// remove obsolete planets
			pp, err := s.st.ListCharacterPlanets(ctx, characterID)
			if err != nil {
				return err
			}
			existing := set.New[int32]()
			for _, p := range pp {
				existing.Add(p.EvePlanet.ID)
			}
			planets := data.([]esi.GetCharactersCharacterIdPlanets200Ok)
			incoming := set.New[int32]()
			for _, p := range planets {
				incoming.Add(p.PlanetId)
			}
			obsolete := existing.Difference(incoming)
			if err := s.st.DeleteCharacterPlanet(ctx, characterID, obsolete.ToSlice()); err != nil {
				return err
			}
			// update or create planet
			for _, o := range planets {
				_, err := s.EveUniverseService.GetOrCreateEvePlanetESI(ctx, o.PlanetId)
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
					et, err := s.EveUniverseService.GetOrCreateEveTypeESI(ctx, pin.TypeId)
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
