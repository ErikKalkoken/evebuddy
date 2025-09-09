package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

var structureStateFromDBValue = map[string]app.StructureState{
	"":                     app.StructureStateUndefined,
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

var structureStateToDBValue = map[app.StructureState]string{}

func init() {
	for k, v := range structureStateFromDBValue {
		structureStateToDBValue[v] = k
	}
}

type UpdateOrCreateCorporationStructureParams struct {
	CorporationID      int32
	FuelExpires        optional.Optional[time.Time]
	Name               string
	NextReinforceApply optional.Optional[time.Time]
	NextReinforceHour  optional.Optional[int64]
	ProfileID          int64
	ReinforceHour      optional.Optional[int64]
	State              app.StructureState
	StateTimerEnd      optional.Optional[time.Time]
	StateTimerStart    optional.Optional[time.Time]
	StructureID        int64
	SystemID           int32
	TypeID             int32
	UnanchorsAt        optional.Optional[time.Time]
}

func (x UpdateOrCreateCorporationStructureParams) isValid() bool {
	return x.CorporationID != 0 &&
		x.StructureID != 0 &&
		x.SystemID != 0 &&
		x.TypeID != 0 &&
		x.State != app.StructureStateUndefined
}

func (st *Storage) UpdateOrCreateCorporationStructure(ctx context.Context, arg UpdateOrCreateCorporationStructureParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCorporationStructure %+v: %w", arg, err)
	}
	if !arg.isValid() {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateCorporationStructure(ctx, queries.UpdateOrCreateCorporationStructureParams{
		CorporationID:      int64(arg.CorporationID),
		FuelExpires:        optional.ToNullTime(arg.FuelExpires),
		Name:               arg.Name,
		NextReinforceApply: optional.ToNullTime(arg.NextReinforceApply),
		NextReinforceHour:  optional.ToNullInt64(arg.NextReinforceHour),
		ProfileID:          arg.ProfileID,
		ReinforceHour:      optional.ToNullInt64(arg.ReinforceHour),
		State:              structureStateToDBValue[arg.State],
		StateTimerEnd:      optional.ToNullTime(arg.StateTimerEnd),
		StateTimerStart:    optional.ToNullTime(arg.StateTimerStart),
		StructureID:        arg.StructureID,
		SystemID:           int64(arg.SystemID),
		TypeID:             int64(arg.TypeID),
		UnanchorsAt:        optional.ToNullTime(arg.UnanchorsAt),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetCorporationStructure(ctx context.Context, corporationID int32, structureID int64) (*app.CorporationStructure, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCorporationStructure %d %d: %w", corporationID, structureID, err)
	}
	if corporationID == 0 || structureID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCorporationStructure(ctx, queries.GetCorporationStructureParams{
		CorporationID: int64(corporationID),
		StructureID:   structureID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	return corporationStructureFromDBModel(corporationStructureFromDBModelParams{
		corporationStructure: r.CorporationStructure,
		eveSolarSystem:       r.EveSolarSystem,
		eveConstellation:     r.EveConstellation,
		eveRegion:            r.EveRegion,
		eveType:              r.EveType,
		eveGroup:             r.EveGroup,
		eveCategory:          r.EveCategory,
	}), nil
}

// func (st *Storage) DeleteCorporationStructures(ctx context.Context, corporationID int32, characterIDs set.Set[int32]) error {
// 	wrapErr := func(err error) error {
// 		return fmt.Errorf("DeleteCorporationStructures %d: %w", corporationID, err)
// 	}
// 	if corporationID == 0 {
// 		return wrapErr(app.ErrInvalid)
// 	}
// 	if characterIDs.Size() == 0 {
// 		return nil
// 	}
// 	err := st.qRW.DeleteCorporationStructures(ctx, queries.DeleteCorporationStructuresParams{
// 		CorporationID: int64(corporationID),
// 		CharacterIds:  convertNumericSlice[int64](characterIDs.Slice()),
// 	})
// 	if err != nil {
// 		return wrapErr(err)
// 	}
// 	return nil
// }

func (st *Storage) ListCorporationStructures(ctx context.Context, corporationID int32) ([]*app.CorporationStructure, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationStructures for id %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCorporationStructures(ctx, int64(corporationID))
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]*app.CorporationStructure, len(rows))
	for i, r := range rows {
		oo[i] = corporationStructureFromDBModel(corporationStructureFromDBModelParams{
			corporationStructure: r.CorporationStructure,
			eveSolarSystem:       r.EveSolarSystem,
			eveConstellation:     r.EveConstellation,
			eveRegion:            r.EveRegion,
			eveType:              r.EveType,
			eveGroup:             r.EveGroup,
			eveCategory:          r.EveCategory,
		})
	}
	return oo, nil
}

func (st *Storage) ListCorporationStructureIDs(ctx context.Context, corporationID int32) (set.Set[int64], error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationStructureIDs for id %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return set.Set[int64]{}, wrapErr(app.ErrInvalid)
	}
	ids, err := st.qRO.ListCorporationStructureIDs(ctx, int64(corporationID))
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	return set.Of(ids...), nil
}

type corporationStructureFromDBModelParams struct {
	corporationStructure queries.CorporationStructure
	eveSolarSystem       queries.EveSolarSystem
	eveConstellation     queries.EveConstellation
	eveRegion            queries.EveRegion
	eveType              queries.EveType
	eveGroup             queries.EveGroup
	eveCategory          queries.EveCategory
}

func corporationStructureFromDBModel(arg corporationStructureFromDBModelParams) *app.CorporationStructure {
	o2 := &app.CorporationStructure{
		CorporationID:      int32(arg.corporationStructure.CorporationID),
		FuelExpires:        optional.FromNullTime(arg.corporationStructure.FuelExpires),
		Name:               arg.corporationStructure.Name,
		NextReinforceApply: optional.FromNullTime(arg.corporationStructure.NextReinforceApply),
		NextReinforceHour:  optional.FromNullInt64(arg.corporationStructure.NextReinforceHour),
		ProfileID:          arg.corporationStructure.ProfileID,
		ReinforceHour:      optional.FromNullInt64(arg.corporationStructure.ReinforceHour),
		State:              structureStateFromDBValue[arg.corporationStructure.State],
		StateTimerEnd:      optional.FromNullTime(arg.corporationStructure.StateTimerEnd),
		StateTimerStart:    optional.FromNullTime(arg.corporationStructure.StateTimerStart),
		StructureID:        arg.corporationStructure.StructureID,
		System:             eveSolarSystemFromDBModel(arg.eveSolarSystem, arg.eveConstellation, arg.eveRegion),
		Type:               eveTypeFromDBModel(arg.eveType, arg.eveGroup, arg.eveCategory),
		UnanchorsAt:        optional.FromNullTime(arg.corporationStructure.UnanchorsAt),
	}
	return o2
}
