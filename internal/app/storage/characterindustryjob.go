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

func (st *Storage) DeleteCharacterIndustryJobsByID(ctx context.Context, characterID int64, jobIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCharacterIndustryJobs for character %d and job IDs: %s: %w", characterID, jobIDs, err)
	}
	if characterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if jobIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.DeleteCharacterIndustryJobs(ctx, queries.DeleteCharacterIndustryJobsParams{
		CharacterID: characterID,
		JobIds:      slices.Collect(jobIDs.All()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Industry jobs deleted for character", "characterID", characterID, "jobIDs", jobIDs)
	return nil
}

func (st *Storage) GetCharacterIndustryJob(ctx context.Context, characterID, jobID int64) (*app.CharacterIndustryJob, error) {
	arg := queries.GetCharacterIndustryJobParams{
		CharacterID: characterID,
		JobID:       jobID,
	}
	r, err := st.qRO.GetCharacterIndustryJob(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get industry job for character %d: %w", characterID, convertGetError(err))
	}
	o := characterIndustryJobFromDBModel(
		blueprintLocationName(r.BlueprintLocationName),
		blueprintLocationSecurity(r.BlueprintLocationSecurity),
		blueprintTypeName(r.BlueprintTypeName),
		r.CharacterIndustryJob,
		completedCharacterName(r.CompletedCharacterName),
		facilityName(r.FacilityName),
		facilitySecurity(r.FacilitySecurity),
		r.EveEntity,
		outputLocationName(r.OutputLocationName),
		outputLocationSecurity(r.OutputLocationSecurity),
		productTypeName(r.ProductTypeName),
		stationName(r.StationName),
		stationSecurity(r.StationSecurity),
	)
	return o, err
}

func (st *Storage) ListAllCharacterIndustryJob(ctx context.Context) ([]*app.CharacterIndustryJob, error) {
	rows, err := st.qRO.ListAllCharacterIndustryJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListAllCharacterIndustryJob: %w", err)
	}
	oo := make([]*app.CharacterIndustryJob, len(rows))
	for i, r := range rows {
		oo[i] = characterIndustryJobFromDBModel(
			blueprintLocationName(r.BlueprintLocationName),
			blueprintLocationSecurity(r.BlueprintLocationSecurity),
			blueprintTypeName(r.BlueprintTypeName),
			r.CharacterIndustryJob,
			completedCharacterName(r.CompletedCharacterName),
			facilityName(r.FacilityName),
			facilitySecurity(r.FacilitySecurity),
			r.EveEntity,
			outputLocationName(r.OutputLocationName),
			outputLocationSecurity(r.OutputLocationSecurity),
			productTypeName(r.ProductTypeName),
			stationName(r.StationName),
			stationSecurity(r.StationSecurity),
		)
	}
	return oo, nil
}

func (st *Storage) ListCharacterIndustryJobs(ctx context.Context, characterID int64) ([]*app.CharacterIndustryJob, error) {
	rows, err := st.qRO.ListCharacterIndustryJobs(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("ListCharacterIndustryJob for character %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterIndustryJob, len(rows))
	for i, r := range rows {
		oo[i] = characterIndustryJobFromDBModel(
			blueprintLocationName(r.BlueprintLocationName),
			blueprintLocationSecurity(r.BlueprintLocationSecurity),
			blueprintTypeName(r.BlueprintTypeName),
			r.CharacterIndustryJob,
			completedCharacterName(r.CompletedCharacterName),
			facilityName(r.FacilityName),
			facilitySecurity(r.FacilitySecurity),
			r.EveEntity,
			outputLocationName(r.OutputLocationName),
			outputLocationSecurity(r.OutputLocationSecurity),
			productTypeName(r.ProductTypeName),
			stationName(r.StationName),
			stationSecurity(r.StationSecurity),
		)
	}
	return oo, nil
}

func (st *Storage) ListAllCharacterIndustryJobActiveCounts(ctx context.Context) ([]app.IndustryJobActivityCount, error) {
	rows, err := st.qRO.ListAllCharacterIndustryJobActiveCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListAllCharacterIndustryJobActiveCounts: %w", err)
	}
	result := make([]app.IndustryJobActivityCount, len(rows))
	for i, r := range rows {
		result[i] = app.IndustryJobActivityCount{
			Activity:    app.IndustryActivity(r.ActivityID),
			Count:       int(r.Number),
			InstallerID: r.InstallerID,
			Status:      jobStatusFromDBValue[r.Status],
		}
	}
	return result, nil
}

func characterIndustryJobFromDBModel(
	blueprintLocationName blueprintLocationName,
	blueprintLocationSecurity blueprintLocationSecurity,
	blueprintTypeName blueprintTypeName,
	cij queries.CharacterIndustryJob,
	completedCharacterName completedCharacterName,
	facilityName facilityName,
	facilitySecurity facilitySecurity,
	installer queries.EveEntity,
	outputLocationName outputLocationName,
	outputLocationSecurity outputLocationSecurity,
	productTypeName productTypeName,
	stationName stationName,
	stationSecurity stationSecurity,

) *app.CharacterIndustryJob {
	o2 := &app.CharacterIndustryJob{
		Activity:    app.IndustryActivity(cij.ActivityID),
		BlueprintID: cij.BlueprintID,
		BlueprintLocation: &app.EveLocationShort{
			ID:             cij.BlueprintLocationID,
			Name:           optional.New(string(blueprintLocationName)),
			SecurityStatus: optional.FromNullFloat64ToFloat32(sql.NullFloat64(blueprintLocationSecurity)),
		},
		BlueprintType: &app.EntityShort{
			ID:   cij.BlueprintTypeID,
			Name: string(blueprintTypeName),
		},
		CharacterID:   cij.CharacterID,
		CompletedDate: optional.FromNullTime(cij.CompletedDate),
		Cost:          optional.FromNullFloat64(cij.Cost),
		Duration:      int(cij.Duration),
		EndDate:       cij.EndDate,
		Facility: &app.EveLocationShort{
			ID:             cij.FacilityID,
			Name:           optional.New(string(facilityName)),
			SecurityStatus: optional.FromNullFloat64ToFloat32(sql.NullFloat64(facilitySecurity)),
		},
		ID:           cij.ID,
		Installer:    eveEntityFromDBModel(installer),
		JobID:        cij.JobID,
		LicensedRuns: optional.FromNullInt64ToInteger[int](cij.LicensedRuns),
		OutputLocation: &app.EveLocationShort{
			ID:             cij.OutputLocationID,
			Name:           optional.New(string(outputLocationName)),
			SecurityStatus: optional.FromNullFloat64ToFloat32(sql.NullFloat64(outputLocationSecurity)),
		},
		PauseDate:   optional.FromNullTime(cij.PauseDate),
		Probability: optional.FromNullFloat64ToFloat32(cij.Probability),
		Runs:        int(cij.Runs),
		Station: &app.EveLocationShort{
			ID:             cij.StationID,
			Name:           optional.New(string(stationName)),
			SecurityStatus: optional.FromNullFloat64ToFloat32(sql.NullFloat64(stationSecurity)),
		},
		StartDate:      cij.StartDate,
		Status:         jobStatusFromDBValue[cij.Status],
		SuccessfulRuns: optional.FromNullInt64ToInteger[int64](cij.SuccessfulRuns),
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

// CharacterIndustryJobParams are the params for an industry job of a character.
type CharacterIndustryJobParams struct {
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
	OutputLocationID     int64
	PauseDate            optional.Optional[time.Time]
	Probability          optional.Optional[float64]
	ProductTypeID        optional.Optional[int64]
	Runs                 int64
	StartDate            time.Time
	StationID            int64
	Status               app.IndustryJobStatus
	SuccessfulRuns       optional.Optional[int64]
}

func (arg CharacterIndustryJobParams) isValid() bool {
	return arg.BlueprintLocationID != 0 &&
		arg.BlueprintTypeID != 0 &&
		arg.FacilityID != 0 &&
		arg.InstallerID != 0 &&
		arg.JobID != 0 &&
		arg.OutputLocationID != 0 &&
		arg.StationID != 0
}

// ReplaceCharacterIndustryJobs stores the jobs of a character which are new or have changed
// and marks stored active jobs missing in args as unknown.
// It reports whether anything was written.
// Job IDs in args must be unique.
func (st *Storage) ReplaceCharacterIndustryJobs(ctx context.Context, characterID int64, args []CharacterIndustryJobParams) (bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceCharacterIndustryJobs: character %d: %w", characterID, err)
	}
	if characterID == 0 {
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
	rows, err := qtx.ListCharacterIndustryJobStates(ctx, characterID)
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
	var changed []CharacterIndustryJobParams
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
		err := qtx.UpdateCharacterIndustryJobStatus(ctx, queries.UpdateCharacterIndustryJobStatusParams{
			CharacterID: characterID,
			JobIds:      ids,
			Status:      jobStatusToDBValue[app.JobUnknown],
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("mark %d jobs as unknown: %w", len(ids), err))
		}
	}
	for _, arg := range changed {
		if err := updateOrCreateCharacterIndustryJob(ctx, qtx, characterID, arg); err != nil {
			return false, fmt.Errorf("ReplaceCharacterIndustryJobs: %w", err) // err already names the character
		}
	}
	if err := tx.Commit(); err != nil {
		return false, wrapErr(err)
	}
	slog.Info("Industry jobs replaced", "characterID", characterID, "changed", len(changed), "orphaned", len(orphans))
	return true, nil
}

type UpdateOrCreateCharacterIndustryJobParams struct {
	ActivityID           int64
	BlueprintID          int64
	BlueprintLocationID  int64
	BlueprintTypeID      int64
	CharacterID          int64
	CompletedCharacterID optional.Optional[int64]
	CompletedDate        optional.Optional[time.Time]
	Cost                 optional.Optional[float64]
	Duration             int64
	EndDate              time.Time
	FacilityID           int64
	InstallerID          int64
	JobID                int64
	LicensedRuns         optional.Optional[int64]
	OutputLocationID     int64
	PauseDate            optional.Optional[time.Time]
	Probability          optional.Optional[float64]
	ProductTypeID        optional.Optional[int64]
	Runs                 int64
	StartDate            time.Time
	StationID            int64
	Status               app.IndustryJobStatus
	SuccessfulRuns       optional.Optional[int64]
}

func (st *Storage) UpdateOrCreateCharacterIndustryJob(ctx context.Context, arg UpdateOrCreateCharacterIndustryJobParams) error {
	p := CharacterIndustryJobParams{
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
		OutputLocationID:     arg.OutputLocationID,
		PauseDate:            arg.PauseDate,
		Probability:          arg.Probability,
		ProductTypeID:        arg.ProductTypeID,
		Runs:                 arg.Runs,
		StartDate:            arg.StartDate,
		StationID:            arg.StationID,
		Status:               arg.Status,
		SuccessfulRuns:       arg.SuccessfulRuns,
	}
	if arg.CharacterID == 0 || !p.isValid() {
		return fmt.Errorf("UpdateOrCreateCharacterIndustryJob: %+v: %w", struct {
			CharacterID int64
			JobID       int64
		}{arg.CharacterID, arg.JobID}, app.ErrInvalid)
	}
	return updateOrCreateCharacterIndustryJob(ctx, st.qRW, arg.CharacterID, p)
}

// updateOrCreateCharacterIndustryJob expects valid params.
func updateOrCreateCharacterIndustryJob(ctx context.Context, q *queries.Queries, characterID int64, arg CharacterIndustryJobParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCharacterIndustryJob: %+v: %w", struct {
			CharacterID int64
			JobID       int64
		}{characterID, arg.JobID}, err)
	}
	err := q.UpdateOrCreateCharacterIndustryJobs(ctx, queries.UpdateOrCreateCharacterIndustryJobsParams{
		ActivityID:           arg.ActivityID,
		BlueprintID:          arg.BlueprintID,
		BlueprintLocationID:  arg.BlueprintLocationID,
		BlueprintTypeID:      arg.BlueprintTypeID,
		CharacterID:          characterID,
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
		StationID:            arg.StationID,
		Status:               jobStatusToDBValue[arg.Status],
		SuccessfulRuns:       optional.ToNullInt64(arg.SuccessfulRuns),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}
