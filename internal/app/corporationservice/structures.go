package corporationservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func (s *CorporationService) GetStructure(ctx context.Context, corporationID int64, structureID int64) (*app.CorporationStructure, error) {
	return s.st.GetCorporationStructure(ctx, corporationID, structureID)
}

func (s *CorporationService) ListStructures(ctx context.Context, corporationID int64) ([]*app.CorporationStructure, error) {
	return s.st.ListCorporationStructures(ctx, corporationID)
}

func (s *CorporationService) updateStructuresESI(ctx context.Context, arg corporationSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCorporationStructures {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, false,
		func(ctx context.Context, arg corporationSectionUpdateParams) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdStructures")
			structures, _, err := s.esiClient.CorporationAPI.GetCorporationsCorporationIdStructures(ctx, arg.corporationID).Execute()
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
				"armor_vulnerable":     app.StructureStateAnchorVulnerable,
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

			// Remove vanished structures
			incoming := set.Collect(xiter.MapSlice(structures, func(x esi.CorporationsCorporationIdStructuresGetInner) int64 {
				return x.StructureId
			}))
			current, err := s.st.ListCorporationStructureIDs(ctx, arg.corporationID)
			if err != nil {
				return false, err
			}
			removed := set.Difference(current, incoming)
			if removed.Size() > 0 {
				if err := s.st.DeleteCorporationStructures(ctx, arg.corporationID, removed); err != nil {
					return false, err
				}
				slog.Info("Removed vanished corporation structures", "corporationID", arg.corporationID, "count", removed.Size())
			}

			// Update structures
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
			for _, o := range structures {
				state, ok := structureStateFromESIValue[o.State]
				if !ok {
					state = app.StructureStateUnknown
				}
				services := xslices.Map(o.Services, func(x esi.CorporationsCorporationIdStructuresGetInnerServicesInner) storage.StructureServiceParams {
					return storage.StructureServiceParams{
						Name:  x.Name,
						State: structureServiceStateFromESIValue[x.State],
					}
				})
				err := s.st.UpdateOrCreateCorporationStructure(ctx, storage.UpdateOrCreateCorporationStructureParams{
					CorporationID:      arg.corporationID,
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
				})
				if err != nil {
					return false, err
				}
			}
			slog.Info("Updated corporation structures", "corporationID", arg.corporationID, "count", len(structures))
			return true, nil
		})
}
