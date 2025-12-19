package storage

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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

var structureServiceStateFromDBValue = map[string]app.StructureServiceState{
	"":        app.StructureServiceStateUndefined,
	"online":  app.StructureServiceStateOnline,
	"offline": app.StructureServiceStateOffline,
	"cleanup": app.StructureServiceStateCleanup,
}

var structureServiceStateToDBValue = map[app.StructureServiceState]string{}

func init() {
	for k, v := range structureStateFromDBValue {
		structureStateToDBValue[v] = k
	}
	for k, v := range structureServiceStateFromDBValue {
		structureServiceStateToDBValue[v] = k
	}
}

func (st *Storage) DeleteCorporationStructures(ctx context.Context, corporationID int32, structureIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCorporationStructuresByID for corporation %d and structures IDs: %v: %w", corporationID, structureIDs, err)
	}
	if corporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if structureIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.DeleteCorporationStructures(ctx, queries.DeleteCorporationStructuresParams{
		CorporationID: int64(corporationID),
		StructureIds:  slices.Collect(structureIDs.All()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Corporation structures deleted", "corporationID", corporationID, "jobIDs", structureIDs)
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
	services, err := st.ListStructureServices(ctx, r.CorporationStructure.ID)
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
		services:             services,
	}), nil
}

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
		services, err := st.ListStructureServices(ctx, r.CorporationStructure.ID)
		if err != nil {
			return nil, wrapErr(convertGetError(err))
		}
		oo[i] = corporationStructureFromDBModel(corporationStructureFromDBModelParams{
			corporationStructure: r.CorporationStructure,
			eveSolarSystem:       r.EveSolarSystem,
			eveConstellation:     r.EveConstellation,
			eveRegion:            r.EveRegion,
			eveType:              r.EveType,
			eveGroup:             r.EveGroup,
			eveCategory:          r.EveCategory,
			services:             services,
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
	services             []*app.StructureService
}

func corporationStructureFromDBModel(arg corporationStructureFromDBModelParams) *app.CorporationStructure {
	o2 := &app.CorporationStructure{
		CorporationID:      int32(arg.corporationStructure.CorporationID),
		FuelExpires:        optional.FromNullTime(arg.corporationStructure.FuelExpires),
		ID:                 arg.corporationStructure.ID,
		Name:               arg.corporationStructure.Name,
		NextReinforceApply: optional.FromNullTime(arg.corporationStructure.NextReinforceApply),
		NextReinforceHour:  optional.FromNullInt64(arg.corporationStructure.NextReinforceHour),
		ProfileID:          arg.corporationStructure.ProfileID,
		ReinforceHour:      optional.FromNullInt64(arg.corporationStructure.ReinforceHour),
		Services:           arg.services,
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

type StructureServiceParams struct {
	Name  string
	State app.StructureServiceState
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
	Services           []StructureServiceParams
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
	id, err := st.qRW.UpdateOrCreateCorporationStructure(ctx, queries.UpdateOrCreateCorporationStructureParams{
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
	if err := st.DeleteStructureServices(ctx, id); err != nil {
		return wrapErr(err)
	}
	for _, s := range arg.Services {
		err := st.CreateStructureService(ctx, CreateStructureServiceParams{
			CorporationStructureID: id,
			Name:                   s.Name,
			State:                  s.State,
		})
		if err != nil {
			return wrapErr(err)
		}
	}
	return nil
}

type CreateStructureServiceParams struct {
	CorporationStructureID int64
	Name                   string
	State                  app.StructureServiceState
}

func (st *Storage) CreateStructureService(ctx context.Context, arg CreateStructureServiceParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateStructureService: %+v: %w", arg, err)
	}
	if arg.CorporationStructureID == 0 || arg.Name == "" || arg.State == app.StructureServiceStateUndefined {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.CreateStructureService(ctx, queries.CreateStructureServiceParams{
		CorporationStructureID: arg.CorporationStructureID,
		Name:                   arg.Name,
		State:                  structureServiceStateToDBValue[arg.State],
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeleteStructureServices(ctx context.Context, corporationStructureID int64) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteStructureServices: %d: %w", corporationStructureID, err)
	}
	if corporationStructureID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if err := st.qRW.DeleteStructureServices(ctx, corporationStructureID); err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetStructureService(ctx context.Context, corporationStructureID int64, name string) (*app.StructureService, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetStructureService: %d %s: %w", corporationStructureID, name, err)
	}
	if corporationStructureID == 0 || name == "" {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetStructureService(ctx, queries.GetStructureServiceParams{
		CorporationStructureID: corporationStructureID,
		Name:                   name,
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	return structureServiceFromDBModel(r), nil
}

func (st *Storage) ListStructureServices(ctx context.Context, corporationStructureID int64) ([]*app.StructureService, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListStructureServices: %d: %w", corporationStructureID, err)
	}
	if corporationStructureID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListStructureServices(ctx, corporationStructureID)
	if err != nil {
		return nil, wrapErr(err)
	}
	return xslices.Map(rows, structureServiceFromDBModel), nil
}

func structureServiceFromDBModel(r queries.CorporationStructureService) *app.StructureService {
	o := &app.StructureService{
		CorporationStructureID: r.CorporationStructureID,
		Name:                   r.Name,
		State:                  structureServiceStateFromDBValue[r.State],
	}
	return o
}
