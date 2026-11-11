package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xmaps"
)

func (st *Storage) CountCorporationIndustryJobsWithStatus(ctx context.Context, corporationID int64, statuses set.Set[app.IndustryJobStatus]) (int64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CountCorporationIndustryJobsWithStatus: %d %v: %w", corporationID, statuses, err)
	}
	if statuses.Size() == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	var statuses2 []string
	for s := range statuses.All() {
		statuses2 = append(statuses2, jobStatusToDBValue[s])
	}
	v, err := st.qRO.CountCorporationIndustryJobsWithStatus(ctx, queries.CountCorporationIndustryJobsWithStatusParams{
		CorporationID: corporationID,
		Statuses:      statuses2,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	return v, nil
}

func (st *Storage) DeleteCorporationIndustryJobs(ctx context.Context, corporationID int64) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCorporationIndustryJobs: %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	n, err := st.qRW.DeleteCorporationIndustryJobs(ctx, corporationID)
	if err != nil {
		return wrapErr(err)
	}
	if n > 0 {
		slog.Info("Industry jobs deleted for corporation", "corporationID", corporationID)
	}
	return nil
}

func (st *Storage) DeleteCorporationIndustryJobsByID(ctx context.Context, corporationID int64, jobIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCorporationIndustryJobsByID: corporation %d jobIDs %v: %w", corporationID, jobIDs, err)
	}
	if corporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if jobIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.DeleteCorporationIndustryJobsByID(ctx, queries.DeleteCorporationIndustryJobsByIDParams{
		CorporationID: corporationID,
		JobIds:        slices.Collect(jobIDs.All()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Industry jobs deleted for corporation", "corporationID", corporationID, "jobIDs", jobIDs)
	return nil
}
func (st *Storage) GetCorporationIndustryJob(ctx context.Context, corporationID, jobID int64) (*app.CorporationIndustryJob, error) {
	arg := queries.GetCorporationIndustryJobParams{
		CorporationID: corporationID,
		JobID:         jobID,
	}
	r, err := st.qRO.GetCorporationIndustryJob(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get industry job for corporation %d: %w", corporationID, convertGetError(err))
	}
	o := corporationIndustryJobFromDBModel(
		blueprintTypeName(r.BlueprintTypeName),
		r.CorporationIndustryJob,
		completedCharacterName(r.CompletedCharacterName),
		r.EveEntity,
		locationName(r.LocationName),
		r.StationSecurity,
		productTypeName(r.ProductTypeName),
	)
	return o, err
}

func (st *Storage) ListCorporationIndustryJobs(ctx context.Context, corporationID int64) ([]*app.CorporationIndustryJob, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationIndustryJobs: corporationID: %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCorporationIndustryJobs(ctx, corporationID)
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]*app.CorporationIndustryJob, len(rows))
	for i, r := range rows {
		oo[i] = corporationIndustryJobFromDBModel(
			blueprintTypeName(r.BlueprintTypeName),
			r.CorporationIndustryJob,
			completedCharacterName(r.CompletedCharacterName),
			r.EveEntity,
			locationName(r.LocationName),
			r.StationSecurity,
			productTypeName(r.ProductTypeName),
		)
	}
	return oo, nil
}

func (st *Storage) ListAllCorporationIndustryJobs(ctx context.Context) ([]*app.CorporationIndustryJob, error) {
	rows, err := st.qRO.ListAllCorporationIndustryJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list industry jobs for all corporations: %w", err)
	}
	oo := make([]*app.CorporationIndustryJob, len(rows))
	for i, r := range rows {
		oo[i] = corporationIndustryJobFromDBModel(
			blueprintTypeName(r.BlueprintTypeName),
			r.CorporationIndustryJob,
			completedCharacterName(r.CompletedCharacterName),
			r.EveEntity,
			locationName(r.LocationName),
			r.StationSecurity,
			productTypeName(r.ProductTypeName),
		)
	}
	return oo, nil
}

func corporationIndustryJobFromDBModel(
	blueprintTypeName blueprintTypeName,
	cij queries.CorporationIndustryJob,
	completedCharacterName completedCharacterName,
	installer queries.EveEntity,
	locationName locationName,
	locationSecurity sql.NullFloat64,
	productTypeName productTypeName,
) *app.CorporationIndustryJob {
	o2 := &app.CorporationIndustryJob{
		Activity:            app.IndustryActivity(cij.ActivityID),
		BlueprintID:         cij.BlueprintID,
		BlueprintLocationID: cij.BlueprintLocationID,
		BlueprintType: &app.EntityShort{
			ID:   cij.BlueprintTypeID,
			Name: string(blueprintTypeName),
		},
		CorporationID: cij.CorporationID,
		CompletedDate: optional.FromNullTime(cij.CompletedDate),
		Cost:          optional.FromNullFloat64(cij.Cost),
		Duration:      int(cij.Duration),
		EndDate:       cij.EndDate,
		FacilityID:    cij.FacilityID,
		ID:            cij.ID,
		Installer:     eveEntityFromDBModel(installer),
		JobID:         cij.JobID,
		LicensedRuns:  optional.FromNullInt64ToInteger[int](cij.LicensedRuns),
		PauseDate:     optional.FromNullTime(cij.PauseDate),
		Probability:   optional.FromNullFloat64ToFloat32(cij.Probability),
		Runs:          int(cij.Runs),
		Location: &app.EveLocationShort{
			ID:             cij.LocationID,
			Name:           optional.New(string(locationName)),
			SecurityStatus: optional.FromNullFloat64ToFloat32(locationSecurity),
		},
		OutputLocationID: cij.OutputLocationID,
		StartDate:        cij.StartDate,
		Status:           jobStatusFromDBValue[cij.Status],
		SuccessfulRuns:   optional.FromNullInt64ToInteger[int64](cij.SuccessfulRuns),
	}
	if cij.CompletedCharacterID.Valid && completedCharacterName.Valid {
		o2.CompletedCharacter = optional.New(&app.EveEntity{
			ID:       cij.CompletedCharacterID.Int64,
			Name:     completedCharacterName.String,
			Category: app.EveEntityCharacter,
		})
	}
	if cij.ProductTypeID.Valid && productTypeName.Valid {
		o2.ProductType = optional.New(&app.EntityShort{
			ID:   cij.ProductTypeID.Int64,
			Name: productTypeName.String,
		})
	}
	return o2
}

// CorporationIndustryJobParams are the params for an industry job of a corporation.
type CorporationIndustryJobParams struct {
	ActivityID           int64
	BlueprintID          int64
	BlueprintLocationID  int64
	BlueprintTypeID      int64
	CompletedCharacterID optional.Optional[int64]
	CompletedDate        optional.Optional[time.Time]
	Cost                 optional.Optional[float64]
	Duration             int64
	EndDate              time.Time
	FacilityID           int64
	InstallerID          int64
	JobID                int64
	LicensedRuns         optional.Optional[int64]
	LocationID           int64
	OutputLocationID     int64
	PauseDate            optional.Optional[time.Time]
	Probability          optional.Optional[float64]
	ProductTypeID        optional.Optional[int64]
	Runs                 int64
	StartDate            time.Time
	Status               app.IndustryJobStatus
	SuccessfulRuns       optional.Optional[int64]
}

func (arg CorporationIndustryJobParams) isValid() bool {
	return arg.BlueprintLocationID != 0 &&
		arg.BlueprintTypeID != 0 &&
		arg.FacilityID != 0 &&
		arg.InstallerID != 0 &&
		arg.JobID != 0 &&
		arg.LocationID != 0 &&
		arg.OutputLocationID != 0
}

// ReplaceCorporationIndustryJobs stores the jobs of a corporation which are new or have changed
// and marks stored active jobs missing in args as unknown.
// It reports whether anything was written.
// Job IDs in args must be unique.
func (st *Storage) ReplaceCorporationIndustryJobs(ctx context.Context, corporationID int64, args []CorporationIndustryJobParams) (bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceCorporationIndustryJobs: corporation %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return false, wrapErr(app.ErrInvalid)
	}
	for _, arg := range args {
		if !arg.isValid() {
			return false, wrapErr(fmt.Errorf("job %d: %w", arg.JobID, app.ErrInvalid))
		}
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return false, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	rows, err := qtx.ListCorporationIndustryJobStates(ctx, corporationID)
	if err != nil {
		return false, wrapErr(err)
	}
	current := make(map[int64]industryJobState, len(rows))
	for _, r := range rows {
		current[r.JobID] = industryJobState{
			completedCharacterID: optional.FromNullInt64(r.CompletedCharacterID),
			completedDate:        optional.FromNullTime(r.CompletedDate),
			endDate:              r.EndDate,
			pauseDate:            optional.FromNullTime(r.PauseDate),
			status:               jobStatusFromDBValue[r.Status],
			successfulRuns:       optional.FromNullInt64(r.SuccessfulRuns),
		}
	}
	var changed []CorporationIndustryJobParams
	for _, arg := range args {
		o, found := xmaps.Pop(current, arg.JobID) // what remains in current is missing in args
		if found && o.equal(industryJobState{
			completedCharacterID: arg.CompletedCharacterID,
			completedDate:        arg.CompletedDate,
			endDate:              arg.EndDate,
			pauseDate:            arg.PauseDate,
			status:               arg.Status,
			successfulRuns:       arg.SuccessfulRuns,
		}) {
			continue
		}
		changed = append(changed, arg)
	}
	orphans := orphanedIndustryJobIDs(current)
	if len(changed) == 0 && len(orphans) == 0 {
		return false, nil
	}
	for ids := range slices.Chunk(orphans, st.MaxIDsPerQuery) {
		err := qtx.UpdateCorporationIndustryJobStatus(ctx, queries.UpdateCorporationIndustryJobStatusParams{
			CorporationID: corporationID,
			JobIds:        ids,
			Status:        jobStatusToDBValue[app.JobUnknown],
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("mark %d jobs as unknown: %w", len(ids), err))
		}
	}
	for _, arg := range changed {
		if err := updateOrCreateCorporationIndustryJob(ctx, qtx, corporationID, arg); err != nil {
			return false, fmt.Errorf("ReplaceCorporationIndustryJobs: %w", err) // err already names the corporation
		}
	}
	if err := tx.Commit(); err != nil {
		return false, wrapErr(err)
	}
	slog.Info("Industry jobs replaced", "corporationID", corporationID, "changed", len(changed), "orphaned", len(orphans))
	return true, nil
}

type UpdateOrCreateCorporationIndustryJobParams struct {
	ActivityID           int64
	BlueprintID          int64
	BlueprintLocationID  int64
	BlueprintTypeID      int64
	CompletedCharacterID optional.Optional[int64]
	CompletedDate        optional.Optional[time.Time]
	CorporationID        int64
	Cost                 optional.Optional[float64]
	Duration             int64
	EndDate              time.Time
	FacilityID           int64
	InstallerID          int64
	JobID                int64
	LicensedRuns         optional.Optional[int64]
	LocationID           int64
	OutputLocationID     int64
	PauseDate            optional.Optional[time.Time]
	Probability          optional.Optional[float64]
	ProductTypeID        optional.Optional[int64]
	Runs                 int64
	StartDate            time.Time
	Status               app.IndustryJobStatus
	SuccessfulRuns       optional.Optional[int64]
}

func (st *Storage) UpdateOrCreateCorporationIndustryJob(ctx context.Context, arg UpdateOrCreateCorporationIndustryJobParams) error {
	p := CorporationIndustryJobParams{
		ActivityID:           arg.ActivityID,
		BlueprintID:          arg.BlueprintID,
		BlueprintLocationID:  arg.BlueprintLocationID,
		BlueprintTypeID:      arg.BlueprintTypeID,
		CompletedCharacterID: arg.CompletedCharacterID,
		CompletedDate:        arg.CompletedDate,
		Cost:                 arg.Cost,
		Duration:             arg.Duration,
		EndDate:              arg.EndDate,
		FacilityID:           arg.FacilityID,
		InstallerID:          arg.InstallerID,
		JobID:                arg.JobID,
		LicensedRuns:         arg.LicensedRuns,
		LocationID:           arg.LocationID,
		OutputLocationID:     arg.OutputLocationID,
		PauseDate:            arg.PauseDate,
		Probability:          arg.Probability,
		ProductTypeID:        arg.ProductTypeID,
		Runs:                 arg.Runs,
		StartDate:            arg.StartDate,
		Status:               arg.Status,
		SuccessfulRuns:       arg.SuccessfulRuns,
	}
	if arg.CorporationID == 0 || !p.isValid() {
		return fmt.Errorf("UpdateOrCreateCorporationIndustryJob: %+v: %w", struct {
			CorporationID int64
			JobID         int64
		}{arg.CorporationID, arg.JobID}, app.ErrInvalid)
	}
	return updateOrCreateCorporationIndustryJob(ctx, st.qRW, arg.CorporationID, p)
}

// updateOrCreateCorporationIndustryJob expects valid params.
func updateOrCreateCorporationIndustryJob(ctx context.Context, q *queries.Queries, corporationID int64, arg CorporationIndustryJobParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCorporationIndustryJob: %+v: %w", struct {
			CorporationID int64
			JobID         int64
		}{corporationID, arg.JobID}, err)
	}
	err := q.UpdateOrCreateCorporationIndustryJobs(ctx, queries.UpdateOrCreateCorporationIndustryJobsParams{
		ActivityID:           arg.ActivityID,
		BlueprintID:          arg.BlueprintID,
		BlueprintLocationID:  arg.BlueprintLocationID,
		BlueprintTypeID:      arg.BlueprintTypeID,
		CorporationID:        corporationID,
		CompletedCharacterID: optional.ToNullInt64(arg.CompletedCharacterID),
		CompletedDate:        optional.ToNullTime(arg.CompletedDate),
		Cost:                 optional.ToNullFloat64(arg.Cost),
		Duration:             arg.Duration,
		EndDate:              arg.EndDate,
		FacilityID:           arg.FacilityID,
		InstallerID:          arg.InstallerID,
		JobID:                arg.JobID,
		LicensedRuns:         optional.ToNullInt64(arg.LicensedRuns),
		OutputLocationID:     arg.OutputLocationID,
		PauseDate:            optional.ToNullTime(arg.PauseDate),
		Probability:          optional.ToNullFloat64(arg.Probability),
		ProductTypeID:        optional.ToNullInt64(arg.ProductTypeID),
		Runs:                 arg.Runs,
		StartDate:            arg.StartDate,
		LocationID:           arg.LocationID,
		Status:               jobStatusToDBValue[arg.Status],
		SuccessfulRuns:       optional.ToNullInt64(arg.SuccessfulRuns),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}
