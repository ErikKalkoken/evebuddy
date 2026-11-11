package corporationservice

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func (s *CorporationService) GetStructure(ctx context.Context, corporationID int64, structureID int64) (*app.CorporationStructure, error) {
	return s.st.GetCorporationStructure(ctx, corporationID, structureID)
}

func (s *CorporationService) ListAllStructures(ctx context.Context) ([]*app.CorporationStructure, error) {
	return s.st.ListAllCorporationStructures(ctx)
}

func (s *CorporationService) ListStructures(ctx context.Context, corporationID int64) ([]*app.CorporationStructure, error) {
	return s.st.ListCorporationStructures(ctx, corporationID)
}

// ListReinforcedCorporationStructures returns the amount of reinforced structures in a corporation.
func (s *CorporationService) ListReinforcedCorporationStructures(ctx context.Context, corporationID int64) (int64, error) {
	return s.st.CountCorporationStructuresWithState(ctx, corporationID, app.StructureStateReinforced)
}

func (s *CorporationService) updateStructuresESI(ctx context.Context, arg corporationSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCorporationStructures {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, false,
		func(ctx context.Context, arg corporationSectionUpdateParams) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdStructures")
			structures, err := xgoesi.FetchPages(
				func(page int32) ([]esi.CorporationsCorporationIdStructuresGetInner, *http.Response, error) {
					return s.esiClient.CorporationAPI.GetCorporationsCorporationIdStructures(ctx, arg.corporationID).Page(page).Execute()
				})
			if err != nil {
				return false, err
			}
			return structures, nil
		},
		func(ctx context.Context, arg corporationSectionUpdateParams, data any) (bool, error) {
			structures := data.([]esi.CorporationsCorporationIdStructuresGetInner)

			structureStateFromESIValue := map[string]app.StructureState{
				"anchor_vulnerable":    app.StructureStateAnchorVulnerable,
				"anchoring":            app.StructureStateAnchoring,
				"armor_reinforce":      app.StructureStateArmorReinforce,
				"armor_vulnerable":     app.StructureStateArmorVulnerable,
				"deploy_vulnerable":    app.StructureStateDeployVulnerable,
				"fitting_invulnerable": app.StructureStateFittingInvulnerable,
				"hull_reinforce":       app.StructureStateHullReinforce,
				"hull_vulnerable":      app.StructureStateHullVulnerable,
				"online_deprecated":    app.StructureStateOnlineDeprecated,
				"onlining_vulnerable":  app.StructureStateOnliningVulnerable,
				"shield_vulnerable":    app.StructureStateShieldVulnerable,
				"unanchored":           app.StructureStateUnanchored,
				"unknown":              app.StructureStateUnknown,
			}

			structureServiceStateFromESIValue := map[string]app.StructureServiceState{
				"online":  app.StructureServiceStateOnline,
				"offline": app.StructureServiceStateOffline,
				"cleanup": app.StructureServiceStateCleanup,
			}

			// Add related objects first, so a failure here leaves stored structures untouched
			var typeIDs, systemIDs set.Set[int64]
			for _, o := range structures {
				typeIDs.Add(o.TypeId)
				systemIDs.Add(o.SystemId)
			}
			g := new(errgroup.Group)
			g.Go(func() error {
				return s.eus.AddMissingTypes(ctx, typeIDs)
			})
			g.Go(func() error {
				return s.eus.AddMissingSolarSystems(ctx, systemIDs)
			})
			if err := g.Wait(); err != nil {
				return false, err
			}

			structures = dedupeStructures(structures, arg.corporationID)
			args := make([]storage.CorporationStructureParams, len(structures))
			for i, o := range structures {
				state, ok := structureStateFromESIValue[o.State]
				if !ok {
					state = app.StructureStateUnknown
					slog.Warn(
						"Unknown structure state encountered",
						"corporationID", arg.corporationID,
						"structureID", o.StructureId,
						"state", o.State,
					)
				}
				services := xslices.Map(o.Services, func(x esi.CorporationsCorporationIdStructuresGetInnerServicesInner) storage.StructureServiceParams {
					serviceState, ok := structureServiceStateFromESIValue[x.State]
					if !ok {
						serviceState = app.StructureServiceStateUnknown
						slog.Warn(
							"Unknown structure service state encountered",
							"corporationID", arg.corporationID,
							"structureID", o.StructureId,
							"service", x.Name,
							"state", x.State,
						)
					}
					return storage.StructureServiceParams{
						Name:  x.Name,
						State: serviceState,
					}
				})
				services = dedupeStructureServices(services, arg.corporationID, o.StructureId)
				args[i] = storage.CorporationStructureParams{
					FuelExpires:        optional.FromPtr(o.FuelExpires),
					Name:               optional.FromPtr(o.Name),
					NextReinforceApply: optional.FromPtr(o.NextReinforceApply),
					NextReinforceHour:  optional.FromPtr(o.NextReinforceHour),
					ProfileID:          o.ProfileId,
					ReinforceHour:      optional.FromPtr(o.ReinforceHour),
					Services:           services,
					State:              state,
					StateTimerEnd:      optional.FromPtr(o.StateTimerEnd),
					StateTimerStart:    optional.FromPtr(o.StateTimerStart),
					StructureID:        o.StructureId,
					SystemID:           o.SystemId,
					TypeID:             o.TypeId,
					UnanchorsAt:        optional.FromPtr(o.UnanchorsAt),
				}
			}
			return s.st.ReplaceCorporationStructures(ctx, arg.corporationID, args)
		})
}

// dedupeStructures returns structures without duplicate IDs. The last structure wins.
func dedupeStructures(structures []esi.CorporationsCorporationIdStructuresGetInner, corporationID int64) []esi.CorporationsCorporationIdStructuresGetInner {
	last := make(map[int64]int, len(structures))
	for i, o := range structures {
		last[o.StructureId] = i
	}
	if len(last) == len(structures) {
		return structures
	}
	slog.Warn("Duplicate corporation structures received from ESI", "corporationID", corporationID, "count", len(structures)-len(last))
	unique := make([]esi.CorporationsCorporationIdStructuresGetInner, 0, len(last))
	for i, o := range structures {
		if last[o.StructureId] == i {
			unique = append(unique, o)
		}
	}
	return unique
}

// dedupeStructureServices returns services without duplicate names. The last service wins.
func dedupeStructureServices(services []storage.StructureServiceParams, corporationID, structureID int64) []storage.StructureServiceParams {
	last := make(map[string]int, len(services))
	for i, x := range services {
		last[x.Name] = i
	}
	if len(last) == len(services) {
		return services
	}
	slog.Warn("Duplicate structure services received from ESI", "corporationID", corporationID, "structureID", structureID, "count", len(services)-len(last))
	unique := make([]storage.StructureServiceParams, 0, len(last))
	for i, x := range services {
		if last[x.Name] == i {
			unique = append(unique, x)
		}
	}
	return unique
}
