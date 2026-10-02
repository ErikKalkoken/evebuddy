package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/colonysim"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/evesde"
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

// ForecastPlanet returns the estimated state of a colony at now,
// simulated forward from its last ESI snapshot.
func (s *CharacterService) ForecastPlanet(cp *app.CharacterPlanet, now time.Time) *app.ColonyForecast {
	return colonysim.Forecast(cp, now)
}

// NotifyStoppedColonies sends notifications for colonies of a character which stopped working.
// A colony is notified once per snapshot and only when it was working at the snapshot.
// It will sent one notification covering all colonies which stopped working.
func (s *CharacterService) NotifyStoppedColonies(ctx context.Context, characterID int64, earliest time.Time, notify func(title, content string)) error {
	_, err, _ := s.sfg.Do(fmt.Sprintf("NotifyStoppedColonies-%d", characterID), func() (any, error) {
		planets, err := s.ListPlanets(ctx, characterID)
		if err != nil {
			return nil, err
		}
		characterName, err := s.getCharacterName(ctx, characterID)
		if err != nil {
			return nil, err
		}
		type stoppedPlanet struct {
			evePlanetID int64
			lastUpdate  time.Time
			text        string
		}
		now := time.Now()
		var stopped []stoppedPlanet
		for _, p := range planets {
			if p.LastNotified.ValueOrZero().Equal(p.LastUpdate) {
				continue
			}
			// work end is only set when the colony was working at the snapshot
			workEndsAt, ok := colonysim.Forecast(p, p.LastUpdate).WorkEndsAt.Value()
			if !ok || workEndsAt.After(now) || workEndsAt.Before(earliest) {
				continue
			}
			var reasons []string
			for _, x := range colonysim.Forecast(p, now).ProblemStatuses() {
				reasons = append(reasons, x.Display())
			}
			if len(reasons) == 0 {
				reasons = append(reasons, app.ColonyIdle.Display())
			}
			stopped = append(stopped, stoppedPlanet{
				evePlanetID: p.EvePlanet.ID,
				lastUpdate:  p.LastUpdate,
				text:        fmt.Sprintf("%s (%s)", p.EvePlanet.Name, strings.Join(reasons, ", ")),
			})
		}
		if len(stopped) > 0 {
			texts := make([]string, len(stopped))
			for i, p := range stopped {
				texts[i] = p.text
			}
			slices.Sort(texts)
			title := fmt.Sprintf("%s: PI colony stopped working at %d planet(s)", characterName, len(stopped))
			content := strings.Join(texts, ", ")
			notify(title, content)
			slog.Info("Notified stopped colonies", "characterID", characterID, "planets", texts)
			for _, p := range stopped {
				err := s.st.UpdateCharacterPlanetLastNotified(ctx, storage.UpdateCharacterPlanetLastNotifiedParams{
					CharacterID:  characterID,
					EvePlanetID:  p.evePlanetID,
					LastNotified: p.lastUpdate,
				})
				if err != nil {
					return nil, err
				}
			}
		}
		return nil, nil
	})
	return err
}

// TODO: Improve update logic to only update changes to pins

func (s *CharacterService) updatePlanetsESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterPlanets {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	// refetch colonies stored before routes were added, which the list of planets would not reveal
	hasOldColonies, err := s.st.HasCharacterPlanetsWithoutRoutes(ctx, arg.characterID)
	if err != nil {
		return false, err
	}
	return s.updateSectionIfChanged(
		ctx, arg, hasOldColonies,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdPlanets")
			planets, _, err := s.esiClient.PlanetaryInteractionAPI.GetCharactersCharacterIdPlanets(ctx, characterID).Execute()
			if err != nil {
				return false, err
			}
			slog.Debug("Received planets from ESI", "characterID", characterID, "count", len(planets))
			return planets, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			// remove obsolete planets
			pp, err := s.st.ListCharacterPlanets(ctx, characterID)
			if err != nil {
				return false, err
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
					return false, err
				}
				slog.Info("Removed obsolete planets", "characterID", characterID, "count", obsolete.Size())
			}
			// update or create planet
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdPlanetsPlanetId")
			g := new(errgroup.Group)
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
					// replace planet pins and routes
					if err := s.st.DeletePlanetPins(ctx, characterPlanetID); err != nil {
						return err
					}
					if err := s.st.DeletePlanetRoutes(ctx, characterPlanetID); err != nil {
						return err
					}
					var recipeTypeIDs set.Set[int64] // of all schematics, so their names and volumes are known
					addRecipeTypes := func(schematicID int64) {
						if x, ok := evesde.PlanetSchematicByID(schematicID); ok {
							recipeTypeIDs.Add(x.OutputTypeID)
							for _, it := range x.Inputs {
								recipeTypeIDs.Add(it.TypeID)
							}
						}
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
						if len(pin.Contents) > 0 {
							arg.Contents = make(map[int64]int64)
							for _, c := range pin.Contents {
								et, err := s.eus.GetOrCreateTypeESI(ctx, c.TypeId)
								if err != nil {
									return err
								}
								arg.Contents[et.ID] += c.Amount
							}
						}
						if d := pin.ExtractorDetails; d != nil {
							if d.ProductTypeId != nil {
								et, err := s.eus.GetOrCreateTypeESI(ctx, *d.ProductTypeId)
								if err != nil {
									return err
								}
								arg.ExtractorProductTypeID = optional.New(et.ID)
							}
							if d.CycleTime != nil {
								arg.ExtractorCycleTime = optional.New(time.Duration(*d.CycleTime) * time.Second)
							}
							arg.ExtractorHeadRadius = optional.FromPtr(d.HeadRadius)
							arg.ExtractorQtyPerCycle = optional.FromPtr(d.QtyPerCycle)
							arg.ExtractorNumHeads = optional.New(int64(len(d.Heads)))
						}
						if pin.FactoryDetails != nil && pin.FactoryDetails.SchematicId != 0 {
							es, err := s.eus.GetOrCreateSchematicESI(ctx, pin.FactoryDetails.SchematicId)
							if err != nil {
								return err
							}
							arg.FactorySchematicID = optional.New(es.ID)
							addRecipeTypes(es.ID)
						}
						if x := pin.SchematicId; x != nil {
							es, err := s.eus.GetOrCreateSchematicESI(ctx, *x)
							if err != nil {
								return err
							}
							arg.SchematicID = optional.New(es.ID)
							addRecipeTypes(es.ID)
						}
						if err := s.st.CreatePlanetPin(ctx, arg); err != nil {
							return err
						}
					}
					if err := s.eus.AddMissingTypes(ctx, recipeTypeIDs); err != nil {
						return err
					}
					for _, r := range planet.Routes {
						et, err := s.eus.GetOrCreateTypeESI(ctx, r.ContentTypeId)
						if err != nil {
							return err
						}
						err = s.st.CreatePlanetRoute(ctx, storage.CreatePlanetRouteParams{
							CharacterPlanetID: characterPlanetID,
							ContentTypeID:     et.ID,
							DestinationPinID:  r.DestinationPinId,
							Quantity:          int64(math.Round(r.Quantity)),
							RouteID:           r.RouteId,
							SourcePinID:       r.SourcePinId,
						})
						if err != nil {
							return err
						}
					}
					return nil
				})
			}
			if err := g.Wait(); err != nil {
				return false, err
			}
			slog.Info("Stored updated planets", "characterID", characterID, "count", len(planets))
			return true, nil
		})
}
