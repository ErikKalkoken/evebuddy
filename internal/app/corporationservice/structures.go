package corporationservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
	"golang.org/x/sync/errgroup"
)

func (s *CorporationService) GetStructure(ctx context.Context, corporationID int32, structureID int64) (*app.CorporationStructure, error) {
	return s.st.GetCorporationStructure(ctx, corporationID, structureID)
}

func (s *CorporationService) ListStructures(ctx context.Context, corporationID int32) ([]*app.CorporationStructure, error) {
	return s.st.ListCorporationStructures(ctx, corporationID)
}

var structureStateFromESIValue = map[string]app.StructureState{
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

func (s *CorporationService) updateStructuresESI(ctx context.Context, arg app.CorporationSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCorporationStructures {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, arg app.CorporationSectionUpdateParams) (any, error) {
			structures, _, err := s.esiClient.ESI.CorporationApi.GetCorporationsCorporationIdStructures(ctx, arg.CorporationID, nil)
			if err != nil {
				return false, err
			}
			return structures, nil
		},
		func(ctx context.Context, arg app.CorporationSectionUpdateParams, data any) error {
			structures := data.([]esi.GetCorporationsCorporationIdStructures200Ok)
			var typeIDs, systemIDs set.Set[int32]
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
				return err
			}
			for _, o := range structures {
				state, ok := structureStateFromESIValue[o.State]
				if !ok {
					state = app.StructureStateUndefined
				}
				err := s.st.UpdateOrCreateCorporationStructure(ctx, storage.UpdateOrCreateCorporationStructureParams{
					CorporationID:      arg.CorporationID,
					FuelExpires:        optional.FromTimeWithZero(o.FuelExpires),
					Name:               o.Name,
					NextReinforceApply: optional.FromTimeWithZero(o.NextReinforceApply),
					NextReinforceHour:  optional.FromIntegerWithZero(int64(o.NextReinforceHour)),
					ProfileID:          int64(o.ProfileId),
					ReinforceHour:      optional.FromIntegerWithZero(int64(o.ReinforceHour)),
					State:              state,
					StateTimerEnd:      optional.FromTimeWithZero(o.StateTimerEnd),
					StateTimerStart:    optional.FromTimeWithZero(o.StateTimerStart),
					StructureID:        o.StructureId,
					SystemID:           o.SystemId,
					TypeID:             o.TypeId,
					UnanchorsAt:        optional.FromTimeWithZero(o.UnanchorsAt),
				})
				if err != nil {
					return err
				}
			}
			// removed := set.Difference(current, incoming)
			// if err := s.st.DeleteCorporationStructures(ctx, arg.CorporationID, removed); err != nil {
			// 	return err
			// }
			slog.Info("Updated corporation structures", "corporationID", arg.CorporationID, "count", len(structures))
			return nil
		})
}
