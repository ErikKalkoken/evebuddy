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
)

var jobStatusFromDBValue = map[string]app.IndustryJobStatus{
	"":          app.JobUndefined,
	"active":    app.JobActive,
	"cancelled": app.JobCancelled,
	"delivered": app.JobDelivered,
	"paused":    app.JobPaused,
	"ready":     app.JobReady,
	"reverted":  app.JobReverted,
	"unknown":   app.JobUnknown,
}

var jobStatusToDBValue = map[app.IndustryJobStatus]string{}

func init() {
	for k, v := range jobStatusFromDBValue {
		jobStatusToDBValue[v] = k
	}
}

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
	o := characterIndustryJobFromDBModel(characterIndustryJobFromDBModelParams{
		blueprintLocationName:     r.BlueprintLocationName,
		blueprintLocationSecurity: r.BlueprintLocationSecurity,
		blueprintTypeName:         r.BlueprintTypeName,
		cij:                       r.CharacterIndustryJob,
		completedCharacterName:    r.CompletedCharacterName,
		facilityName:              r.FacilityName,
		facilitySecurity:          r.FacilitySecurity,
		installer:                 r.EveEntity,
		outputLocationName:        r.OutputLocationName,
		outputLocationSecurity:    r.OutputLocationSecurity,
		productTypeName:           r.ProductTypeName,
		stationName:               r.StationName,
		stationSecurity:           r.StationSecurity,
	})
	return o, err
}

func (st *Storage) ListAllCharacterIndustryJob(ctx context.Context) ([]*app.CharacterIndustryJob, error) {
	rows, err := st.qRO.ListAllCharacterIndustryJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListAllCharacterIndustryJob: %w", err)
	}
	oo := make([]*app.CharacterIndustryJob, len(rows))
	for i, r := range rows {
		oo[i] = characterIndustryJobFromDBModel(characterIndustryJobFromDBModelParams{
			blueprintLocationName:     r.BlueprintLocationName,
			blueprintLocationSecurity: r.BlueprintLocationSecurity,
			blueprintTypeName:         r.BlueprintTypeName,
			cij:                       r.CharacterIndustryJob,
			completedCharacterName:    r.CompletedCharacterName,
			facilityName:              r.FacilityName,
			facilitySecurity:          r.FacilitySecurity,
			installer:                 r.EveEntity,
			outputLocationName:        r.OutputLocationName,
			outputLocationSecurity:    r.OutputLocationSecurity,
			productTypeName:           r.ProductTypeName,
			stationName:               r.StationName,
			stationSecurity:           r.StationSecurity,
		})
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
		oo[i] = characterIndustryJobFromDBModel(characterIndustryJobFromDBModelParams{
			blueprintLocationName:     r.BlueprintLocationName,
			blueprintLocationSecurity: r.BlueprintLocationSecurity,
			blueprintTypeName:         r.BlueprintTypeName,
			cij:                       r.CharacterIndustryJob,
			completedCharacterName:    r.CompletedCharacterName,
			facilityName:              r.FacilityName,
			facilitySecurity:          r.FacilitySecurity,
			installer:                 r.EveEntity,
			outputLocationName:        r.OutputLocationName,
			outputLocationSecurity:    r.OutputLocationSecurity,
			productTypeName:           r.ProductTypeName,
			stationName:               r.StationName,
			stationSecurity:           r.StationSecurity,
		})
	}
	return oo, nil
}

func (st *Storage) ListAllCharacterIndustryJobActiveCounts(ctx context.Context) ([]app.IndustryJobActivityCount, error) {
	rows, err := st.qRO.ListAllCharacterIndustryJobActiveCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListAllCharacterIndustryJobActiveCounts: %w", err)
	}
	result := make([]app.IndustryJobActivityCount, 0)
	for _, r := range rows {
		result = append(result, app.IndustryJobActivityCount{
			Activity:    app.IndustryActivity(r.ActivityID),
			Count:       int(r.Number),
			InstallerID: r.InstallerID,
			Status:      jobStatusFromDBValue[r.Status],
		})
	}
	return result, nil
}

type characterIndustryJobFromDBModelParams struct {
	blueprintLocationName     string
	blueprintLocationSecurity sql.NullFloat64
	blueprintTypeName         string
	cij                       queries.CharacterIndustryJob
	completedCharacterName    sql.NullString
	facilityName              string
	facilitySecurity          sql.NullFloat64
	installer                 queries.EveEntity
	outputLocationName        string
	outputLocationSecurity    sql.NullFloat64
	productTypeName           sql.NullString
	stationName               string
	stationSecurity           sql.NullFloat64
}

func characterIndustryJobFromDBModel(arg characterIndustryJobFromDBModelParams) *app.CharacterIndustryJob {
	o2 := &app.CharacterIndustryJob{
		Activity:    app.IndustryActivity(arg.cij.ActivityID),
		BlueprintID: arg.cij.BlueprintID,
		BlueprintLocation: &app.EveLocationShort{
			ID:             arg.cij.BlueprintLocationID,
			Name:           optional.New(arg.blueprintLocationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.blueprintLocationSecurity),
		},
		BlueprintType: &app.EntityShort[int64]{
			ID:   arg.cij.BlueprintTypeID,
			Name: arg.blueprintTypeName,
		},
		CharacterID:   arg.cij.CharacterID,
		CompletedDate: optional.FromNullTime(arg.cij.CompletedDate),
		Cost:          optional.FromNullFloat64(arg.cij.Cost),
		Duration:      int(arg.cij.Duration),
		EndDate:       arg.cij.EndDate,
		Facility: &app.EveLocationShort{
			ID:             arg.cij.FacilityID,
			Name:           optional.New(arg.facilityName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.facilitySecurity),
		},
		ID:           arg.cij.ID,
		Installer:    eveEntityFromDBModel(arg.installer),
		JobID:        arg.cij.JobID,
		LicensedRuns: optional.FromNullInt64ToInteger[int](arg.cij.LicensedRuns),
		OutputLocation: &app.EveLocationShort{
			ID:             arg.cij.OutputLocationID,
			Name:           optional.New(arg.outputLocationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.outputLocationSecurity),
		},
		PauseDate:   optional.FromNullTime(arg.cij.PauseDate),
		Probability: optional.FromNullFloat64ToFloat32(arg.cij.Probability),
		Runs:        int(arg.cij.Runs),
		Station: &app.EveLocationShort{
			ID:             arg.cij.StationID,
			Name:           optional.New(arg.stationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.stationSecurity),
		},
		StartDate:      arg.cij.StartDate,
		Status:         jobStatusFromDBValue[arg.cij.Status],
		SuccessfulRuns: optional.FromNullInt64ToInteger[int64](arg.cij.SuccessfulRuns),
	}
	if arg.cij.CompletedCharacterID.Valid && arg.completedCharacterName.Valid {
		o2.CompletedCharacter = optional.New(&app.EveEntity{
			ID:       arg.cij.CompletedCharacterID.Int64,
			Name:     arg.completedCharacterName.String,
			Category: app.EveEntityCharacter,
		})
	}
	if arg.cij.ProductTypeID.Valid && arg.productTypeName.Valid {
		o2.ProductType = optional.New(&app.EntityShort[int64]{
			ID:   arg.cij.ProductTypeID.Int64,
			Name: arg.productTypeName.String,
		})
	}
	return o2
}

type UpdateCharacterIndustryJobStatusParams struct {
	CharacterID int64
	JobIDs      set.Set[int64]
	Status      app.IndustryJobStatus
}

func (st *Storage) UpdateCharacterIndustryJobStatus(ctx context.Context, arg UpdateCharacterIndustryJobStatusParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterIndustryJobStatus %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.JobIDs.Contains(0) {
		return wrapErr(app.ErrInvalid)
	}
	if arg.JobIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.UpdateCharacterIndustryJobStatus(ctx, queries.UpdateCharacterIndustryJobStatusParams{
		CharacterID: arg.CharacterID,
		JobIds:      slices.Collect(arg.JobIDs.All()),
		Status:      jobStatusToDBValue[arg.Status],
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
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
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCharacterIndustryJob: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.BlueprintTypeID == 0 || arg.BlueprintLocationID == 0 || arg.InstallerID == 0 || arg.OutputLocationID == 0 || arg.StationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateCharacterIndustryJobs(ctx, queries.UpdateOrCreateCharacterIndustryJobsParams{
		ActivityID:           arg.ActivityID,
		BlueprintID:          arg.BlueprintID,
		BlueprintLocationID:  arg.BlueprintLocationID,
		BlueprintTypeID:      arg.BlueprintTypeID,
		CharacterID:          arg.CharacterID,
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
