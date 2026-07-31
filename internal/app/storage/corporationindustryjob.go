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

func (st *Storage) DeleteCorporationIndustryJobs(ctx context.Context, corporationID int64) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCorporationIndustryJobs: %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.DeleteCorporationIndustryJobs(ctx, corporationID)
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Industry jobs deleted for corporation", "corporationID", corporationID)
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

type UpdateCorporationIndustryJobStatusParams struct {
	CorporationID int64
	JobIDs        set.Set[int64]
	Status        app.IndustryJobStatus
}

func (st *Storage) UpdateCorporationIndustryJobStatus(ctx context.Context, arg UpdateCorporationIndustryJobStatusParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCorporationIndustryJobStatus %+v: %w", arg, err)
	}
	if arg.CorporationID == 0 || arg.JobIDs.Contains(0) {
		return wrapErr(app.ErrInvalid)
	}
	if arg.JobIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.UpdateCorporationIndustryJobStatus(ctx, queries.UpdateCorporationIndustryJobStatusParams{
		CorporationID: arg.CorporationID,
		JobIds:        slices.Collect(arg.JobIDs.All()),
		Status:        jobStatusToDBValue[arg.Status],
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
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
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCorporationIndustryJob: %+v: %w", arg, err)
	}
	if arg.CorporationID == 0 ||
		arg.BlueprintTypeID == 0 ||
		arg.BlueprintLocationID == 0 ||
		arg.InstallerID == 0 ||
		arg.OutputLocationID == 0 ||
		arg.LocationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateCorporationIndustryJobs(ctx, queries.UpdateOrCreateCorporationIndustryJobsParams{
		ActivityID:           arg.ActivityID,
		BlueprintID:          arg.BlueprintID,
		BlueprintLocationID:  arg.BlueprintLocationID,
		BlueprintTypeID:      arg.BlueprintTypeID,
		CorporationID:        arg.CorporationID,
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
