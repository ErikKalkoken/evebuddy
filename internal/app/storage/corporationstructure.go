package storage

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xmaps"
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
	"unknown": app.StructureServiceStateUnknown,
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

func (st *Storage) CountCorporationStructuresWithState(ctx context.Context, corporationID int64, states set.Set[app.StructureState]) (int64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CountCorporationStructuresWithState: %d %v: %w", corporationID, states, err)
	}
	if states.Size() == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	var states2 []string
	for s := range states.All() {
		states2 = append(states2, structureStateToDBValue[s])
	}
	v, err := st.qRO.CountCorporationStructuresWithState(ctx, queries.CountCorporationStructuresWithStateParams{
		CorporationID: corporationID,
		States:        states2,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	return v, nil
}

func (st *Storage) GetCorporationStructure(ctx context.Context, corporationID int64, structureID int64) (*app.CorporationStructure, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCorporationStructure %d %d: %w", corporationID, structureID, err)
	}
	if corporationID == 0 || structureID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	oo, err := st.readCorporationStructures(ctx, func(q *queries.Queries) ([]queries.GetCorporationStructureRow, error) {
		r, err := q.GetCorporationStructure(ctx, queries.GetCorporationStructureParams{
			CorporationID: corporationID,
			StructureID:   structureID,
		})
		if err != nil {
			return nil, convertGetError(err)
		}
		return []queries.GetCorporationStructureRow{r}, nil
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	return oo[0], nil
}

func (st *Storage) ListAllCorporationStructures(ctx context.Context) ([]*app.CorporationStructure, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListAllCorporationStructures: %w", err)
	}
	oo, err := st.readCorporationStructures(ctx, func(q *queries.Queries) ([]queries.GetCorporationStructureRow, error) {
		rows, err := q.ListAllCorporationStructures(ctx)
		if err != nil {
			return nil, err
		}
		rows2 := make([]queries.GetCorporationStructureRow, len(rows))
		for i, r := range rows {
			rows2[i] = queries.GetCorporationStructureRow(r)
		}
		return rows2, nil
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	return oo, nil
}

func (st *Storage) ListCorporationStructures(ctx context.Context, corporationID int64) ([]*app.CorporationStructure, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationStructures for id %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	oo, err := st.readCorporationStructures(ctx, func(q *queries.Queries) ([]queries.GetCorporationStructureRow, error) {
		rows, err := q.ListCorporationStructures(ctx, corporationID)
		if err != nil {
			return nil, err
		}
		rows2 := make([]queries.GetCorporationStructureRow, len(rows))
		for i, r := range rows {
			rows2[i] = queries.GetCorporationStructureRow(r)
		}
		return rows2, nil
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	return oo, nil
}

// readCorporationStructures reads structures with their services in one transaction,
// so concurrent writes can not mix old and new data.
func (st *Storage) readCorporationStructures(ctx context.Context, listRows func(q *queries.Queries) ([]queries.GetCorporationStructureRow, error)) ([]*app.CorporationStructure, error) {
	tx, err := st.dbRO.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	qtx := st.qRO.WithTx(tx)
	rows, err := listRows(qtx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.CorporationStructure.ID
	}
	services, err := st.listStructureServicesByStructure(ctx, qtx, ids)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	oo := make([]*app.CorporationStructure, len(rows))
	for i, r := range rows {
		ss := services[r.CorporationStructure.ID]
		if ss == nil {
			ss = []*app.StructureService{}
		}
		oo[i] = corporationStructureFromDBModel(
			r.CorporationStructure,
			r.EveSolarSystem,
			r.EveConstellation,
			r.EveRegion,
			r.EveType,
			r.EveGroup,
			r.EveCategory,
			ss,
		)
	}
	return oo, nil
}

func (st *Storage) ListCorporationStructureIDs(ctx context.Context, corporationID int64) (set.Set[int64], error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationStructureIDs for id %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return set.Set[int64]{}, wrapErr(app.ErrInvalid)
	}
	ids, err := st.qRO.ListCorporationStructureIDs(ctx, corporationID)
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func corporationStructureFromDBModel(
	corporationStructure queries.CorporationStructure,
	eveSolarSystem queries.EveSolarSystem,
	eveConstellation queries.EveConstellation,
	eveRegion queries.EveRegion,
	eveType queries.EveType,
	eveGroup queries.EveGroup,
	eveCategory queries.EveCategory,
	services []*app.StructureService,
) *app.CorporationStructure {
	o := &app.CorporationStructure{
		CorporationID:      corporationStructure.CorporationID,
		FuelExpires:        optional.FromNullTime(corporationStructure.FuelExpires),
		ID:                 corporationStructure.ID,
		Name:               optional.FromZeroValue(corporationStructure.Name),
		NextReinforceApply: optional.FromNullTime(corporationStructure.NextReinforceApply),
		NextReinforceHour:  optional.FromNullInt64(corporationStructure.NextReinforceHour),
		ProfileID:          corporationStructure.ProfileID,
		ReinforceHour:      optional.FromNullInt64(corporationStructure.ReinforceHour),
		Services:           services,
		State:              structureStateFromDBValue[corporationStructure.State],
		StateTimerEnd:      optional.FromNullTime(corporationStructure.StateTimerEnd),
		StateTimerStart:    optional.FromNullTime(corporationStructure.StateTimerStart),
		StructureID:        corporationStructure.StructureID,
		System:             eveSolarSystemFromDBModel(eveSolarSystem, eveConstellation, eveRegion),
		Type:               eveTypeFromDBModel(eveType, eveGroup, eveCategory),
		UnanchorsAt:        optional.FromNullTime(corporationStructure.UnanchorsAt),
	}
	return o
}

type StructureServiceParams struct {
	Name  string
	State app.StructureServiceState
}

func (x StructureServiceParams) isValid() bool {
	return x.Name != "" && x.State != app.StructureServiceStateUndefined
}

// CorporationStructureParams describes a structure of a corporation and its services.
type CorporationStructureParams struct {
	FuelExpires        optional.Optional[time.Time]
	Name               optional.Optional[string]
	NextReinforceApply optional.Optional[time.Time]
	NextReinforceHour  optional.Optional[int64]
	ProfileID          int64
	ReinforceHour      optional.Optional[int64]
	Services           []StructureServiceParams
	State              app.StructureState
	StateTimerEnd      optional.Optional[time.Time]
	StateTimerStart    optional.Optional[time.Time]
	StructureID        int64
	SystemID           int64
	TypeID             int64
	UnanchorsAt        optional.Optional[time.Time]
}

func (x CorporationStructureParams) isValid() bool {
	if x.StructureID == 0 || x.SystemID == 0 || x.TypeID == 0 || x.State == app.StructureStateUndefined {
		return false
	}
	for _, s := range x.Services {
		if !s.isValid() {
			return false
		}
	}
	return true
}

// corporationStructureState holds the columns of a structure which can change after it was created and its services.
type corporationStructureState struct {
	fuelExpires        optional.Optional[time.Time]
	name               string
	nextReinforceApply optional.Optional[time.Time]
	nextReinforceHour  optional.Optional[int64]
	profileID          int64
	reinforceHour      optional.Optional[int64]
	services           map[string]app.StructureServiceState
	state              app.StructureState
	stateTimerEnd      optional.Optional[time.Time]
	stateTimerStart    optional.Optional[time.Time]
	unanchorsAt        optional.Optional[time.Time]
}

// equal reports whether s and other are equal.
// Times are compared with Equal, because their location can differ after a round trip through the DB.
func (s corporationStructureState) equal(other corporationStructureState) bool {
	return optional.Equal2(s.fuelExpires, other.fuelExpires) &&
		s.name == other.name &&
		optional.Equal2(s.nextReinforceApply, other.nextReinforceApply) &&
		optional.Equal(s.nextReinforceHour, other.nextReinforceHour) &&
		s.profileID == other.profileID &&
		optional.Equal(s.reinforceHour, other.reinforceHour) &&
		maps.Equal(s.services, other.services) &&
		s.state == other.state &&
		optional.Equal2(s.stateTimerEnd, other.stateTimerEnd) &&
		optional.Equal2(s.stateTimerStart, other.stateTimerStart) &&
		optional.Equal2(s.unanchorsAt, other.unanchorsAt)
}

// ReplaceCorporationStructures replaces the structures of a corporation in one transaction.
// It deletes structures missing in args and writes only new or changed structures.
// It reports whether anything has changed.
// Structure IDs in args and service names of each structure must be unique.
func (st *Storage) ReplaceCorporationStructures(ctx context.Context, corporationID int64, args []CorporationStructureParams) (bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceCorporationStructures: corporation %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return false, wrapErr(app.ErrInvalid)
	}
	for _, arg := range args {
		if !arg.isValid() {
			return false, wrapErr(fmt.Errorf("structure %d: %w", arg.StructureID, app.ErrInvalid))
		}
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return false, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	rows, err := qtx.ListCorporationStructureStates(ctx, corporationID)
	if err != nil {
		return false, wrapErr(err)
	}
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	services, err := st.listStructureServicesByStructure(ctx, qtx, ids)
	if err != nil {
		return false, wrapErr(err)
	}
	current := make(map[int64]corporationStructureState, len(rows))
	for _, r := range rows {
		m := make(map[string]app.StructureServiceState)
		for _, s := range services[r.ID] {
			m[s.Name] = s.State
		}
		current[r.StructureID] = corporationStructureState{
			fuelExpires:        optional.FromNullTime(r.FuelExpires),
			name:               r.Name,
			nextReinforceApply: optional.FromNullTime(r.NextReinforceApply),
			nextReinforceHour:  optional.FromNullInt64(r.NextReinforceHour),
			profileID:          r.ProfileID,
			reinforceHour:      optional.FromNullInt64(r.ReinforceHour),
			services:           m,
			state:              structureStateFromDBValue[r.State],
			stateTimerEnd:      optional.FromNullTime(r.StateTimerEnd),
			stateTimerStart:    optional.FromNullTime(r.StateTimerStart),
			unanchorsAt:        optional.FromNullTime(r.UnanchorsAt),
		}
	}
	var changed []CorporationStructureParams
	for _, arg := range args {
		o, found := xmaps.Pop(current, arg.StructureID) // what remains in current is missing in args
		if found {
			m := make(map[string]app.StructureServiceState)
			for _, s := range arg.Services {
				m[s.Name] = s.State
			}
			if o.equal(corporationStructureState{
				fuelExpires:        arg.FuelExpires,
				name:               arg.Name.ValueOrZero(),
				nextReinforceApply: arg.NextReinforceApply,
				nextReinforceHour:  arg.NextReinforceHour,
				profileID:          arg.ProfileID,
				reinforceHour:      arg.ReinforceHour,
				services:           m,
				state:              arg.State,
				stateTimerEnd:      arg.StateTimerEnd,
				stateTimerStart:    arg.StateTimerStart,
				unanchorsAt:        arg.UnanchorsAt,
			}) {
				continue
			}
		}
		changed = append(changed, arg)
	}
	removed := slices.Collect(maps.Keys(current))
	if len(changed) == 0 && len(removed) == 0 {
		return false, nil
	}
	for ids := range slices.Chunk(removed, st.MaxIDsPerQuery) {
		err := qtx.DeleteCorporationStructures(ctx, queries.DeleteCorporationStructuresParams{
			CorporationID: corporationID,
			StructureIds:  ids,
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("delete %d structures: %w", len(ids), err))
		}
	}
	for _, arg := range changed {
		if err := updateOrCreateCorporationStructure(ctx, qtx, corporationID, arg); err != nil {
			return false, fmt.Errorf("ReplaceCorporationStructures: %w", err) // err already names the corporation
		}
	}
	if err := tx.Commit(); err != nil {
		return false, wrapErr(err)
	}
	slog.Info("Corporation structures replaced", "corporationID", corporationID, "changed", len(changed), "removed", len(removed))
	return true, nil
}

type UpdateOrCreateCorporationStructureParams struct {
	CorporationID      int64
	FuelExpires        optional.Optional[time.Time]
	Name               optional.Optional[string]
	NextReinforceApply optional.Optional[time.Time]
	NextReinforceHour  optional.Optional[int64]
	ProfileID          int64
	ReinforceHour      optional.Optional[int64]
	Services           []StructureServiceParams
	State              app.StructureState
	StateTimerEnd      optional.Optional[time.Time]
	StateTimerStart    optional.Optional[time.Time]
	StructureID        int64
	SystemID           int64
	TypeID             int64
	UnanchorsAt        optional.Optional[time.Time]
}

// UpdateOrCreateCorporationStructure creates or updates a structure and replaces its services in one transaction.
func (st *Storage) UpdateOrCreateCorporationStructure(ctx context.Context, arg UpdateOrCreateCorporationStructureParams) error {
	p := CorporationStructureParams{
		FuelExpires:        arg.FuelExpires,
		Name:               arg.Name,
		NextReinforceApply: arg.NextReinforceApply,
		NextReinforceHour:  arg.NextReinforceHour,
		ProfileID:          arg.ProfileID,
		ReinforceHour:      arg.ReinforceHour,
		Services:           arg.Services,
		State:              arg.State,
		StateTimerEnd:      arg.StateTimerEnd,
		StateTimerStart:    arg.StateTimerStart,
		StructureID:        arg.StructureID,
		SystemID:           arg.SystemID,
		TypeID:             arg.TypeID,
		UnanchorsAt:        arg.UnanchorsAt,
	}
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCorporationStructure: %+v: %w", struct {
			CorporationID int64
			StructureID   int64
		}{arg.CorporationID, arg.StructureID}, err)
	}
	if arg.CorporationID == 0 || !p.isValid() {
		return wrapErr(app.ErrInvalid)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	if err := updateOrCreateCorporationStructure(ctx, st.qRW.WithTx(tx), arg.CorporationID, p); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	return nil
}

// updateOrCreateCorporationStructure creates or updates a structure and replaces its services.
// It expects valid params and should run inside a transaction.
func updateOrCreateCorporationStructure(ctx context.Context, q *queries.Queries, corporationID int64, arg CorporationStructureParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCorporationStructure: %+v: %w", struct {
			CorporationID int64
			StructureID   int64
		}{corporationID, arg.StructureID}, err)
	}
	id, err := q.UpdateOrCreateCorporationStructure(ctx, queries.UpdateOrCreateCorporationStructureParams{
		CorporationID:      corporationID,
		FuelExpires:        optional.ToNullTime(arg.FuelExpires),
		Name:               arg.Name.ValueOrZero(),
		NextReinforceApply: optional.ToNullTime(arg.NextReinforceApply),
		NextReinforceHour:  optional.ToNullInt64(arg.NextReinforceHour),
		ProfileID:          arg.ProfileID,
		ReinforceHour:      optional.ToNullInt64(arg.ReinforceHour),
		State:              structureStateToDBValue[arg.State],
		StateTimerEnd:      optional.ToNullTime(arg.StateTimerEnd),
		StateTimerStart:    optional.ToNullTime(arg.StateTimerStart),
		StructureID:        arg.StructureID,
		SystemID:           arg.SystemID,
		TypeID:             arg.TypeID,
		UnanchorsAt:        optional.ToNullTime(arg.UnanchorsAt),
	})
	if err != nil {
		return wrapErr(err)
	}
	if err := q.DeleteStructureServices(ctx, id); err != nil {
		return wrapErr(err)
	}
	for _, s := range arg.Services {
		err := q.CreateStructureService(ctx, queries.CreateStructureServiceParams{
			CorporationStructureID: id,
			Name:                   s.Name,
			State:                  structureServiceStateToDBValue[s.State],
		})
		if err != nil {
			return wrapErr(fmt.Errorf("service %s: %w", s.Name, err))
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
		return nil, wrapErr(convertGetError(err))
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

// listStructureServicesByStructure returns the services for the given structures, keyed by corporation structure ID.
func (st *Storage) listStructureServicesByStructure(ctx context.Context, q *queries.Queries, corporationStructureIDs []int64) (map[int64][]*app.StructureService, error) {
	m := make(map[int64][]*app.StructureService)
	for idsChunk := range slices.Chunk(corporationStructureIDs, st.MaxIDsPerQuery) {
		rows, err := q.ListStructureServicesForStructureIDs(ctx, idsChunk)
		if err != nil {
			return nil, fmt.Errorf("list structure services for %d structures: %w", len(idsChunk), err)
		}
		for _, r := range rows {
			m[r.CorporationStructureID] = append(m[r.CorporationStructureID], structureServiceFromDBModel(r))
		}
	}
	return m, nil
}

func structureServiceFromDBModel(r queries.CorporationStructureService) *app.StructureService {
	o := &app.StructureService{
		CorporationStructureID: r.CorporationStructureID,
		Name:                   r.Name,
		State:                  structureServiceStateFromDBValue[r.State],
	}
	return o
}
