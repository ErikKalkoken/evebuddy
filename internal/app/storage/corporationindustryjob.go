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

func (st *Storage) DeleteCorporationIndustryJobs(ctx context.Context, corporationID int32) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCorporationIndustryJobs: %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.DeleteCorporationIndustryJobs(ctx, int64(corporationID))
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Industry jobs deleted for corporation", "corporationID", corporationID)
	return nil
}

func (st *Storage) DeleteCorporationIndustryJobsByID(ctx context.Context, corporationID int32, jobIDs set.Set[int32]) error {
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
		CorporationID: int64(corporationID),
		JobIds:        convertNumericSlice[int64](jobIDs.Slice()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Industry jobs deleted for corporation", "corporationID", corporationID, "jobIDs", jobIDs)
	return nil
}
func (st *Storage) GetCorporationIndustryJob(ctx context.Context, corporationID, jobID int32) (*app.CorporationIndustryJob, error) {
	arg := queries.GetCorporationIndustryJobParams{
		CorporationID: int64(corporationID),
		JobID:         int64(jobID),
	}
	r, err := st.qRO.GetCorporationIndustryJob(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get industry job for corporation %d: %w", corporationID, convertGetError(err))
	}
	o := corporationIndustryJobFromDBModel(corporationIndustryJobFromDBModelParams{
		blueprintTypeName:      r.BlueprintTypeName,
		cij:                    r.CorporationIndustryJob,
		completedCharacterName: r.CompletedCharacterName,
		installer:              r.EveEntity,
		locationName:           r.LocationName,
		locationSecurity:       r.StationSecurity,
		productTypeName:        r.ProductTypeName,
	})
	return o, err
}

func (st *Storage) ListCorporationIndustryJobs(ctx context.Context, corporationID int32) ([]*app.CorporationIndustryJob, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationIndustryJobs: corporationID: %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCorporationIndustryJobs(ctx, int64(corporationID))
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]*app.CorporationIndustryJob, len(rows))
	for i, r := range rows {
		oo[i] = corporationIndustryJobFromDBModel(corporationIndustryJobFromDBModelParams{
			blueprintTypeName:      r.BlueprintTypeName,
			cij:                    r.CorporationIndustryJob,
			completedCharacterName: r.CompletedCharacterName,
			installer:              r.EveEntity,
			locationName:           r.LocationName,
			locationSecurity:       r.StationSecurity,
			productTypeName:        r.ProductTypeName,
		})
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
		oo[i] = corporationIndustryJobFromDBModel(corporationIndustryJobFromDBModelParams{
			blueprintTypeName:      r.BlueprintTypeName,
			cij:                    r.CorporationIndustryJob,
			completedCharacterName: r.CompletedCharacterName,
			installer:              r.EveEntity,
			locationName:           r.LocationName,
			locationSecurity:       r.StationSecurity,
			productTypeName:        r.ProductTypeName,
		})
	}
	return oo, nil
}

type corporationIndustryJobFromDBModelParams struct {
	blueprintTypeName      string
	cij                    queries.CorporationIndustryJob
	completedCharacterName sql.NullString
	installer              queries.EveEntity
	locationName           string
	locationSecurity       sql.NullFloat64
	productTypeName        sql.NullString
}

func corporationIndustryJobFromDBModel(arg corporationIndustryJobFromDBModelParams) *app.CorporationIndustryJob {
	o2 := &app.CorporationIndustryJob{
		Activity:            app.IndustryActivity(arg.cij.ActivityID),
		BlueprintID:         arg.cij.BlueprintID,
		BlueprintLocationID: arg.cij.BlueprintLocationID,
		BlueprintType: &app.EntityShort[int32]{
			ID:   int32(arg.cij.BlueprintTypeID),
			Name: arg.blueprintTypeName,
		},
		CorporationID: int32(arg.cij.CorporationID),
		CompletedDate: optional.FromNullTime(arg.cij.CompletedDate),
		Cost:          optional.FromNullFloat64(arg.cij.Cost),
		Duration:      int(arg.cij.Duration),
		EndDate:       arg.cij.EndDate,
		FacilityID:    arg.cij.FacilityID,
		ID:            arg.cij.ID,
		Installer:     eveEntityFromDBModel(arg.installer),
		JobID:         int32(arg.cij.JobID),
		LicensedRuns:  optional.FromNullInt64ToInteger[int](arg.cij.LicensedRuns),
		PauseDate:     optional.FromNullTime(arg.cij.PauseDate),
		Probability:   optional.FromNullFloat64ToFloat32(arg.cij.Probability),
		Runs:          int(arg.cij.Runs),
		Location: &app.EveLocationShort{
			ID:             arg.cij.LocationID,
			Name:           optional.New(arg.locationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.locationSecurity),
		},
		OutputLocationID: arg.cij.OutputLocationID,
		StartDate:        arg.cij.StartDate,
		Status:           jobStatusFromDBValue[arg.cij.Status],
		SuccessfulRuns:   optional.FromNullInt64ToInteger[int32](arg.cij.SuccessfulRuns),
	}
	if arg.cij.CompletedCharacterID.Valid && arg.completedCharacterName.Valid {
		o2.CompletedCharacter = optional.New(&app.EveEntity{
			ID:       int32(arg.cij.CompletedCharacterID.Int64),
			Name:     arg.completedCharacterName.String,
			Category: app.EveEntityCharacter,
		})
	}
	if arg.cij.ProductTypeID.Valid && arg.productTypeName.Valid {
		o2.ProductType = optional.New(&app.EntityShort[int32]{
			ID:   int32(arg.cij.ProductTypeID.Int64),
			Name: arg.productTypeName.String,
		})
	}
	return o2
}

type UpdateOrCreateCorporationIndustryJobParams struct {
	ActivityID           int32
	BlueprintID          int64
	BlueprintLocationID  int64
	BlueprintTypeID      int32
	CompletedCharacterID int32     // optional
	CompletedDate        time.Time // optional
	CorporationID        int32
	Cost                 float64 // optional
	Duration             int32
	EndDate              time.Time
	FacilityID           int64
	InstallerID          int32
	JobID                int32
	LicensedRuns         int32 // optional
	LocationID           int64
	OutputLocationID     int64
	PauseDate            time.Time // optional
	Probability          float32   // optional
	ProductTypeID        int32     // optional
	Runs                 int32
	StartDate            time.Time
	Status               app.IndustryJobState
	SuccessfulRuns       int32 // optional
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
		ActivityID:           int64(arg.ActivityID),
		BlueprintID:          arg.BlueprintID,
		BlueprintLocationID:  arg.BlueprintLocationID,
		BlueprintTypeID:      int64(arg.BlueprintTypeID),
		CorporationID:        int64(arg.CorporationID),
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
		LocationID:           arg.LocationID,
		Status:               jobStatusToDBValue[arg.Status],
		SuccessfulRuns:       NewNullInt64(int64(arg.SuccessfulRuns)),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}
