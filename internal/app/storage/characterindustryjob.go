package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

var jobStatusFromDBValue = map[string]app.IndustryJobStatus{
	"":          app.JobUndefined,
	"active":    app.JobActive,
	"cancelled": app.JobCancelled,
	"delivered": app.JobDelivered,
	"paused":    app.JobPaused,
	"ready":     app.JobReady,
	"reverted":  app.JobReverted,
}

var jobStatusToDBValue = map[app.IndustryJobStatus]string{}

func init() {
	for k, v := range jobStatusFromDBValue {
		jobStatusToDBValue[v] = k
	}
}

func (st *Storage) DeleteCharacterIndustryJobsByID(ctx context.Context, characterID int32, jobIDs set.Set[int32]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCharacterIndustryJobs for character %d and job IDs: %v: %w", characterID, jobIDs, err)
	}
	if characterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if jobIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.DeleteCharacterIndustryJobs(ctx, queries.DeleteCharacterIndustryJobsParams{
		CharacterID: int64(characterID),
		JobIds:      convertNumericSlice[int64](jobIDs.Slice()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Industry jobs deleted for character", "characterID", characterID, "jobIDs", jobIDs)
	return nil
}

func (st *Storage) GetCharacterIndustryJob(ctx context.Context, characterID, jobID int32) (*app.CharacterIndustryJob, error) {
	arg := queries.GetCharacterIndustryJobParams{
		CharacterID: int64(characterID),
		JobID:       int64(jobID),
	}
	r, err := st.qRO.GetCharacterIndustryJob(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get industry job for character %d: %w", characterID, convertGetError(err))
	}
	o := characterIndustryJobFromDBModel(characterIndustryJobFromDBModelParams{
		blueprintLocationName:     r.BlueprintLocationName,
		blueprintLocationSecurity: r.BlueprintLocationSecurity,
		blueprintTypeName:         r.BlueprintTypeName,
		completedCharacterName:    r.CompletedCharacterName,
		facilityName:              r.FacilityName,
		facilitySecurity:          r.FacilitySecurity,
		installer:                 r.EveEntity,
		job:                       r.CharacterIndustryJob,
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
			completedCharacterName:    r.CompletedCharacterName,
			facilityName:              r.FacilityName,
			facilitySecurity:          r.FacilitySecurity,
			installer:                 r.EveEntity,
			job:                       r.CharacterIndustryJob,
			outputLocationName:        r.OutputLocationName,
			outputLocationSecurity:    r.OutputLocationSecurity,
			productTypeName:           r.ProductTypeName,
			stationName:               r.StationName,
			stationSecurity:           r.StationSecurity,
		})
	}
	return oo, nil
}

func (st *Storage) ListCharacterIndustryJobs(ctx context.Context, characterID int32) ([]*app.CharacterIndustryJob, error) {
	rows, err := st.qRO.ListCharacterIndustryJobs(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("ListCharacterIndustryJob for character %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterIndustryJob, len(rows))
	for i, r := range rows {
		oo[i] = characterIndustryJobFromDBModel(characterIndustryJobFromDBModelParams{
			blueprintLocationName:     r.BlueprintLocationName,
			blueprintLocationSecurity: r.BlueprintLocationSecurity,
			blueprintTypeName:         r.BlueprintTypeName,
			completedCharacterName:    r.CompletedCharacterName,
			facilityName:              r.FacilityName,
			facilitySecurity:          r.FacilitySecurity,
			installer:                 r.EveEntity,
			job:                       r.CharacterIndustryJob,
			outputLocationName:        r.OutputLocationName,
			outputLocationSecurity:    r.OutputLocationSecurity,
			productTypeName:           r.ProductTypeName,
			stationName:               r.StationName,
			stationSecurity:           r.StationSecurity,
		})
	}
	return oo, nil
}

type characterIndustryJobFromDBModelParams struct {
	blueprintLocationName     string
	blueprintLocationSecurity sql.NullFloat64
	blueprintTypeName         string
	completedCharacterName    sql.NullString
	facilityName              string
	facilitySecurity          sql.NullFloat64
	installer                 queries.EveEntity
	job                       queries.CharacterIndustryJob
	outputLocationName        string
	outputLocationSecurity    sql.NullFloat64
	productTypeName           sql.NullString
	stationName               string
	stationSecurity           sql.NullFloat64
}

func characterIndustryJobFromDBModel(arg characterIndustryJobFromDBModelParams) *app.CharacterIndustryJob {
	o2 := &app.CharacterIndustryJob{
		Activity:    app.IndustryActivity(arg.job.ActivityID),
		BlueprintID: arg.job.BlueprintID,
		BlueprintLocation: &app.EveLocationShort{
			ID:             arg.job.BlueprintLocationID,
			Name:           optional.New(arg.blueprintLocationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.blueprintLocationSecurity),
		},
		BlueprintType: &app.EntityShort[int32]{
			ID:   int32(arg.job.BlueprintTypeID),
			Name: arg.blueprintTypeName,
		},
		CharacterID:   int32(arg.job.CharacterID),
		CompletedDate: optional.FromNullTime(arg.job.CompletedDate),
		Cost:          optional.FromNullFloat64(arg.job.Cost),
		Duration:      int(arg.job.Duration),
		EndDate:       arg.job.EndDate,
		Facility: &app.EveLocationShort{
			ID:             arg.job.FacilityID,
			Name:           optional.New(arg.facilityName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.facilitySecurity),
		},
		ID:           arg.job.ID,
		Installer:    eveEntityFromDBModel(arg.installer),
		JobID:        int32(arg.job.JobID),
		LicensedRuns: optional.FromNullInt64ToInteger[int](arg.job.LicensedRuns),
		OutputLocation: &app.EveLocationShort{
			ID:             arg.job.OutputLocationID,
			Name:           optional.New(arg.outputLocationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.outputLocationSecurity),
		},
		PauseDate:   optional.FromNullTime(arg.job.PauseDate),
		Probability: optional.FromNullFloat64ToFloat32(arg.job.Probability),
		Runs:        int(arg.job.Runs),
		Station: &app.EveLocationShort{
			ID:             arg.job.StationID,
			Name:           optional.New(arg.stationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.stationSecurity),
		},
		StartDate:      arg.job.StartDate,
		Status:         jobStatusFromDBValue[arg.job.Status],
		SuccessfulRuns: optional.FromNullInt64ToInteger[int32](arg.job.SuccessfulRuns),
	}
	if arg.job.CompletedCharacterID.Valid && arg.completedCharacterName.Valid {
		o2.CompletedCharacter = optional.New(&app.EveEntity{
			ID:       int32(arg.job.CompletedCharacterID.Int64),
			Name:     arg.completedCharacterName.String,
			Category: app.EveEntityCharacter,
		})
	}
	if arg.job.ProductTypeID.Valid && arg.productTypeName.Valid {
		o2.ProductType = optional.New(&app.EntityShort[int32]{
			ID:   int32(arg.job.ProductTypeID.Int64),
			Name: arg.productTypeName.String,
		})
	}
	return o2
}

func (st *Storage) ListAllCharacterIndustryJobActiveCounts(ctx context.Context) ([]app.IndustryJobActivityCount, error) {
	rows, err := st.qRO.ListAllCharacterIndustryJobActiveCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListAllCharacterIndustryJobActiveCounts: %w", err)
	}
	result := make([]app.IndustryJobActivityCount, 0)
	for _, r := range rows {
		result = append(result, app.IndustryJobActivityCount{
			InstallerID: int32(r.InstallerID),
			Activity:    app.IndustryActivity(r.ActivityID),
			Status:      jobStatusFromDBValue[r.Status],
			Count:       int(r.Number),
		})
	}
	return result, nil
}

type UpdateOrCreateCharacterIndustryJobParams struct {
	ActivityID           int32
	BlueprintID          int64
	BlueprintLocationID  int64
	BlueprintTypeID      int32
	CharacterID          int32
	CompletedCharacterID int32     // optional
	CompletedDate        time.Time // optional
	Cost                 float64   // optional
	Duration             int32
	EndDate              time.Time
	FacilityID           int64
	InstallerID          int32
	JobID                int32
	LicensedRuns         int32 // optional
	OutputLocationID     int64
	PauseDate            time.Time // optional
	Probability          float32   // optional
	ProductTypeID        int32     // optional
	Runs                 int32
	StartDate            time.Time
	StationID            int64
	Status               app.IndustryJobStatus
	SuccessfulRuns       int32 // optional
}

func (st *Storage) UpdateOrCreateCharacterIndustryJob(ctx context.Context, arg UpdateOrCreateCharacterIndustryJobParams) error {
	if arg.CharacterID == 0 || arg.BlueprintTypeID == 0 || arg.BlueprintLocationID == 0 || arg.InstallerID == 0 || arg.OutputLocationID == 0 || arg.StationID == 0 {
		return fmt.Errorf("update or create character industry job: %+v: invalid parameters", arg)
	}
	arg2 := queries.UpdateOrCreateCharacterIndustryJobsParams{
		ActivityID:           int64(arg.ActivityID),
		BlueprintID:          arg.BlueprintID,
		BlueprintLocationID:  arg.BlueprintLocationID,
		BlueprintTypeID:      int64(arg.BlueprintTypeID),
		CharacterID:          int64(arg.CharacterID),
		CompletedCharacterID: NewNullInt64(int64(arg.CompletedCharacterID)),
		CompletedDate:        NewNullTimeFromTime(arg.CompletedDate),
		Cost:                 NewNullFloat64(arg.Cost),
		Duration:             int64(arg.Duration),
		EndDate:              arg.EndDate,
		FacilityID:           arg.FacilityID,
		InstallerID:          int64(arg.InstallerID),
		JobID:                int64(arg.JobID),
		LicensedRuns:         NewNullInt64(int64(arg.LicensedRuns)),
		OutputLocationID:     arg.OutputLocationID,
		PauseDate:            NewNullTimeFromTime(arg.PauseDate),
		Probability:          NewNullFloat64(float64(arg.Probability)),
		ProductTypeID:        NewNullInt64(int64(arg.ProductTypeID)),
		Runs:                 int64(arg.Runs),
		StartDate:            arg.StartDate,
		StationID:            arg.StationID,
		Status:               jobStatusToDBValue[arg.Status],
		SuccessfulRuns:       NewNullInt64(int64(arg.SuccessfulRuns)),
	}
	if err := st.qRW.UpdateOrCreateCharacterIndustryJobs(ctx, arg2); err != nil {
		return fmt.Errorf("update or create character industry job: %+v: %w", arg, err)
	}
	return nil
}
