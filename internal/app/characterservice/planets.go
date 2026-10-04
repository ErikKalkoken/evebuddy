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

// forecastMaxAge is the maximum age of a cached forecast.
// Covers the forecast horizon, which moves with time and is not part of ValidUntil.
const forecastMaxAge = time.Hour

type colonyKey struct {
	characterID int64
	planetID    int64
}

type forecastEntry struct {
	fingerprint uint64 // of the colony data, see [app.CharacterPlanet.Fingerprint]
	forecast    *app.ColonyForecast
	time        time.Time // of the forecast, wall clock only
}

// isValid reports whether the cached forecast can be used for colony data with fingerprint at now.
func (e forecastEntry) isValid(fingerprint uint64, now time.Time) bool {
	if e.fingerprint != fingerprint || now.Before(e.time) || now.Sub(e.time) >= forecastMaxAge {
		return false
	}
	v, ok := e.forecast.ValidUntil.Value()
	return !ok || now.Before(v)
}

// ForecastPlanet returns the estimated state of a colony at now,
// simulated forward from its last ESI snapshot.
// Forecasts are reused while the colony data is unchanged and the forecast stays the same.
// The returned forecast must not be modified.
func (s *CharacterService) ForecastPlanet(cp *app.CharacterPlanet, now time.Time) *app.ColonyForecast {
	key := colonyKey{characterID: cp.CharacterID, planetID: cp.EvePlanet.ID}
	fingerprint := cp.Fingerprint()
	e, ok := s.forecasts.Load(key)
	if !ok || !e.isValid(fingerprint, now) {
		// wall clock only: the monotonic clock stops while a device sleeps, which would extend the max age
		e = forecastEntry{fingerprint: fingerprint, forecast: colonysim.Forecast(cp, now), time: now.Round(0)}
		s.forecasts.Store(key, e)
	}
	f := *e.forecast
	f.Time = now
	return &f
}

// clearForecasts removes all cached forecasts of a character.
func (s *CharacterService) clearForecasts(characterID int64) {
	s.forecasts.Range(func(k colonyKey, _ forecastEntry) bool {
		if k.characterID == characterID {
			s.forecasts.Delete(k)
		}
		return true
	})
}

// NotifyStoppedColonies sends notifications for colonies of a character which stopped working.
// A colony is notified once per snapshot and only when it was working at the snapshot.
// It will send one notification covering all colonies which stopped working.
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
			f := s.ForecastPlanet(p, now)
			if f.Status.IsWorking() {
				continue
			}
			// work end is only set when the colony was working at the snapshot
			workEndsAt, ok := colonysim.WorkEndsAt(p, now).Value()
			if !ok || workEndsAt.Before(earliest) {
				continue
			}
			var reasons []string
			for _, pin := range p.Pins {
				if pf := f.Pins[pin.ID]; pf != nil && pf.Status.IsProblem() {
					reasons = append(reasons, p.PinName(pin)+": "+pf.Status.Display())
				}
			}
			slices.Sort(reasons)
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

// planetsDataVersion is part of the content hash. Bump it to refetch all colonies once,
// e.g. after storing more colony data.
const planetsDataVersion = 1

type planetsData struct {
	Planets []esi.CharactersCharacterIdPlanetsGetInner
	Version int
}

// TODO: Improve update logic to only update changes to pins

func (s *CharacterService) updatePlanetsESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterPlanets {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, false,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdPlanets")
			planets, _, err := s.esiClient.PlanetaryInteractionAPI.GetCharactersCharacterIdPlanets(ctx, characterID).Execute()
			if err != nil {
				return false, err
			}
			slog.Debug("Received planets from ESI", "characterID", characterID, "count", len(planets))
			return planetsData{Planets: planets, Version: planetsDataVersion}, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			// frees forecasts of replaced colonies
			defer s.clearForecasts(characterID)
			// remove obsolete planets
			pp, err := s.st.ListCharacterPlanets(ctx, characterID)
			if err != nil {
				return false, err
			}
			existing := set.Of[int64]()
			for _, p := range pp {
				existing.Add(p.EvePlanet.ID)
			}
			planets := data.(planetsData).Planets
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
					planet, _, err := s.esiClient.PlanetaryInteractionAPI.GetCharactersCharacterIdPlanetsPlanetId(ctx, characterID, o.PlanetId).Execute()
					if err != nil {
						return err
					}
					// fetch everything first, so the colony can be written in one transaction
					colony := storage.ReplaceCharacterPlanetParams{
						CharacterID:  characterID,
						EvePlanetID:  o.PlanetId,
						LastUpdate:   o.LastUpdate,
						UpgradeLevel: o.UpgradeLevel,
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
							TypeID:         et.ID,
							PinID:          pin.PinId,
							ExpiryTime:     optional.FromPtr(pin.ExpiryTime),
							InstallTime:    optional.FromPtr(pin.InstallTime),
							LastCycleStart: optional.FromPtr(pin.LastCycleStart),
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
						colony.Pins = append(colony.Pins, arg)
					}
					if err := s.eus.AddMissingTypes(ctx, recipeTypeIDs); err != nil {
						return err
					}
					for _, r := range planet.Routes {
						et, err := s.eus.GetOrCreateTypeESI(ctx, r.ContentTypeId)
						if err != nil {
							return err
						}
						colony.Routes = append(colony.Routes, storage.CreatePlanetRouteParams{
							ContentTypeID:    et.ID,
							DestinationPinID: r.DestinationPinId,
							Quantity:         int64(math.Round(r.Quantity)),
							RouteID:          r.RouteId,
							SourcePinID:      r.SourcePinId,
						})
					}
					_, err = s.st.ReplaceCharacterPlanet(ctx, colony)
					return err
				})
			}
			if err := g.Wait(); err != nil {
				return false, err
			}
			slog.Info("Stored updated planets", "characterID", characterID, "count", len(planets))
			return true, nil
		})
}
