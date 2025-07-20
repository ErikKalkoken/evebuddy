package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
}

var jobStatusToDBValue = map[app.IndustryJobStatus]string{}

func init() {
	for k, v := range jobStatusFromDBValue {
		jobStatusToDBValue[v] = k
	}
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
	o := characterIndustryJobFromDBModel(
		r.CharacterIndustryJob,
		r.EveEntity,
		r.BlueprintLocationName,
		r.BlueprintLocationSecurity,
		r.BlueprintTypeName,
		r.CompletedCharacterName,
		r.FacilityName,
		r.FacilitySecurity,
		r.OutputLocationName,
		r.OutputLocationSecurity,
		r.ProductTypeName,
		r.StationName,
		r.StationSecurity,
	)
	return o, err
}

func (st *Storage) ListAllCharacterIndustryJob(ctx context.Context) ([]*app.CharacterIndustryJob, error) {
	rows, err := st.qRO.ListAllCharacterIndustryJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list industry jobs for all characters: %w", err)
	}
	oo := make([]*app.CharacterIndustryJob, len(rows))
	for i, r := range rows {
		oo[i] = characterIndustryJobFromDBModel(
			r.CharacterIndustryJob,
			r.EveEntity,
			r.BlueprintLocationName,
			r.BlueprintLocationSecurity,
			r.BlueprintTypeName,
			r.CompletedCharacterName,
			r.FacilityName,
			r.FacilitySecurity,
			r.OutputLocationName,
			r.OutputLocationSecurity,
			r.ProductTypeName,
			r.StationName,
			r.StationSecurity,
		)
	}
	return oo, nil
}

func characterIndustryJobFromDBModel(
	o queries.CharacterIndustryJob,
	installer queries.EveEntity,
	blueprintLocationName string,
	blueprintLocationSecurity sql.NullFloat64,
	blueprintTypeName string,
	completedCharacterName sql.NullString,
	facilityName string,
	facilitySecurity sql.NullFloat64,
	outputLocationName string,
	outputLocationSecurity sql.NullFloat64,
	productTypeName sql.NullString,
	stationName string,
	stationSecurity sql.NullFloat64,
) *app.CharacterIndustryJob {
	o2 := &app.CharacterIndustryJob{
		Activity:    app.IndustryActivity(o.ActivityID),
		BlueprintID: o.BlueprintID,
		BlueprintLocation: &app.EveLocationShort{
			ID:             o.BlueprintLocationID,
			Name:           optional.New(blueprintLocationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(blueprintLocationSecurity),
		},
		BlueprintType: &app.EntityShort[int32]{
			ID:   int32(o.BlueprintTypeID),
			Name: blueprintTypeName,
		},
		CharacterID:   int32(o.CharacterID),
		CompletedDate: optional.FromNullTime(o.CompletedDate),
		Cost:          optional.FromNullFloat64(o.Cost),
		Duration:      int(o.Duration),
		EndDate:       o.EndDate,
		Facility: &app.EveLocationShort{
			ID:             o.FacilityID,
			Name:           optional.New(facilityName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(facilitySecurity),
		},
		Installer:    eveEntityFromDBModel(installer),
		JobID:        int32(o.JobID),
		LicensedRuns: optional.FromNullInt64ToInteger[int](o.LicensedRuns),
		OutputLocation: &app.EveLocationShort{
			ID:             o.OutputLocationID,
			Name:           optional.New(outputLocationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(outputLocationSecurity),
		},
		PauseDate:   optional.FromNullTime(o.PauseDate),
		Probability: optional.FromNullFloat64ToFloat32(o.Probability),
		Runs:        int(o.Runs),
		Station: &app.EveLocationShort{
			ID:             o.StationID,
			Name:           optional.New(stationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(stationSecurity),
		},
		StartDate:      o.StartDate,
		Status:         jobStatusFromDBValue[o.Status],
		SuccessfulRuns: optional.FromNullInt64ToInteger[int32](o.SuccessfulRuns),
	}
	if o.CompletedCharacterID.Valid && completedCharacterName.Valid {
		o2.CompletedCharacter = optional.New(&app.EveEntity{
			ID:       int32(o.CompletedCharacterID.Int64),
			Name:     completedCharacterName.String,
			Category: app.EveEntityCharacter,
		})
	}
	if o.ProductTypeID.Valid && productTypeName.Valid {
		o2.ProductType = optional.New(&app.EntityShort[int32]{
			ID:   int32(o.ProductTypeID.Int64),
			Name: productTypeName.String,
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
