package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func (s *CharacterService) GetPlanet(ctx context.Context, characterID, planetID int64) (*app.CharacterPlanet, error) {
	return s.st.GetCharacterPlanet(ctx, characterID, planetID)
}

func (s *CharacterService) ListAllPlanets(ctx context.Context) ([]*app.CharacterPlanet, error) {
	return s.st.ListAllCharacterPlanets(ctx)
}

func (s *CharacterService) ListPlanets(ctx context.Context, characterID int64) ([]*app.CharacterPlanet, error) {
	return s.st.ListCharacterPlanets(ctx, characterID)
}

// NotifyExpiredExtractions sends notifications for expired extractions of a character.
// Expired notifications are notified once only.
// It will sent one notification covering all currently expired extractions.
func (s *CharacterService) NotifyExpiredExtractions(ctx context.Context, characterID int64, earliest time.Time, notify func(title, content string)) error {
	_, err, _ := s.sfg.Do(fmt.Sprintf("NotifyExpiredExtractions-%d", characterID), func() (any, error) {
		planets, err := s.ListPlanets(ctx, characterID)
		if err != nil {
			return nil, err
		}
		characterName, err := s.getCharacterName(ctx, characterID)
		if err != nil {
			return nil, err
		}
		var expired []string
		for _, p := range planets {
			expiration, ok := p.ExtractionsExpiry().Value()
			if !ok || expiration.After(time.Now()) || expiration.Before(earliest) {
				continue
			}
			if p.LastNotified.ValueOrZero().Equal(expiration) {
				continue
			}
			expired = append(expired, p.EvePlanet.Name)
			err := s.st.UpdateCharacterPlanetLastNotified(ctx, storage.UpdateCharacterPlanetLastNotifiedParams{
				CharacterID:  characterID,
				EvePlanetID:  p.EvePlanet.ID,
				LastNotified: expiration,
			})
			if err != nil {
				return nil, err
			}
		}
		if len(expired) > 0 {
			slices.Sort(expired)
			title := fmt.Sprintf("%s: PI extraction expired at %d planet(s)", characterName, len(expired))
			content := fmt.Sprintf("Extraction expired at %s", strings.Join(expired, ", "))
			notify(title, content)
			slog.Info("Notified expired planets", "characterID", characterID, "planets", expired)
		}
		return nil, nil
	})
	return err
}

// TODO: Improve update logic to only update changes to pins

func (s *CharacterService) updatePlanetsESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterPlanets {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdPlanets")
			planets, _, err := s.esiClient.PlanetaryInteractionAPI.GetCharactersCharacterIdPlanets(ctx, characterID).Execute()
			if err != nil {
				return false, err
			}
			slog.Debug("Received planets from ESI", "characterID", characterID, "count", len(planets))
			return planets, nil
		},
		func(ctx context.Context, characterID int64, data any) error {
			// remove obsolete planets
			pp, err := s.st.ListCharacterPlanets(ctx, characterID)
			if err != nil {
				return err
			}
			existing := set.Of[int64]()
			for _, p := range pp {
				existing.Add(p.EvePlanet.ID)
			}
			planets := data.([]esi.CharactersCharacterIdPlanetsGetInner)
			incoming := set.Of[int64]()
			for _, p := range planets {
				incoming.Add(p.PlanetId)
			}
			obsolete := set.Difference(existing, incoming)
			if obsolete.Size() > 0 {
				if err := s.st.DeleteCharacterPlanet(ctx, characterID, obsolete); err != nil {
					return err
				}
				slog.Info("Removed obsolete planets", "characterID", characterID, "count", obsolete.Size())
			}
			// update or create planet
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdPlanetsPlanetId")
			g := new(testutil.ErrGroupDebug)
			for _, o := range planets {
				g.Go(func() error {
					_, err := s.eus.GetOrCreatePlanetESI(ctx, o.PlanetId)
					if err != nil {
						return err
					}
					characterPlanetID, err := s.st.UpdateOrCreateCharacterPlanet(ctx, storage.UpdateOrCreateCharacterPlanetParams{
						CharacterID:  characterID,
						EvePlanetID:  o.PlanetId,
						LastUpdate:   o.LastUpdate,
						UpgradeLevel: o.UpgradeLevel,
					})
					if err != nil {
						return err
					}
					planet, _, err := s.esiClient.PlanetaryInteractionAPI.GetCharactersCharacterIdPlanetsPlanetId(ctx, characterID, o.PlanetId).Execute()
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
							ExpiryTime:        optional.FromPtr(pin.ExpiryTime),
							InstallTime:       optional.FromPtr(pin.InstallTime),
							LastCycleStart:    optional.FromPtr(pin.LastCycleStart),
						}
						if pin.ExtractorDetails != nil && pin.ExtractorDetails.ProductTypeId != nil {
							et, err := s.eus.GetOrCreateTypeESI(ctx, *pin.ExtractorDetails.ProductTypeId)
							if err != nil {
								return err
							}
							arg.ExtractorProductTypeID = optional.New(et.ID)
						}
						if pin.FactoryDetails != nil && pin.FactoryDetails.SchematicId != 0 {
							es, err := s.eus.GetOrCreateSchematicESI(ctx, pin.FactoryDetails.SchematicId)
							if err != nil {
								return err
							}
							arg.FactorySchematicID = optional.New(es.ID)
						}
						if x := pin.SchematicId; x != nil {
							es, err := s.eus.GetOrCreateSchematicESI(ctx, *x)
							if err != nil {
								return err
							}
							arg.SchematicID = optional.New(es.ID)
						}
						if err := s.st.CreatePlanetPin(ctx, arg); err != nil {
							return err
						}
					}
					return nil
				})
			}
			if err := g.Wait(); err != nil {
				return err
			}
			slog.Info("Stored updated planets", "characterID", characterID, "count", len(planets))
			return nil
		})
}
