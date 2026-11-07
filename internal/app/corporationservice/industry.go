package corporationservice

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/industrysync"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// GetCorporationIndustryJob returns an industry job.
func (s *CorporationService) GetCorporationIndustryJob(ctx context.Context, corporationID, jobID int64) (*app.CorporationIndustryJob, error) {
	return s.st.GetCorporationIndustryJob(ctx, corporationID, jobID)
}

// ListAllCorporationIndustryJobs returns all industry jobs from all corporations.
func (s *CorporationService) ListAllCorporationIndustryJobs(ctx context.Context) ([]*app.CorporationIndustryJob, error) {
	return s.st.ListAllCorporationIndustryJobs(ctx)
}

// ListCorporationIndustryJobs returns all industry jobs from a corporation.
func (s *CorporationService) ListCorporationIndustryJobs(ctx context.Context, corporationID int64) ([]*app.CorporationIndustryJob, error) {
	return s.st.ListCorporationIndustryJobs(ctx, corporationID)
}

// ListActiveCorporationIndustryJobs returns the amount of active industry jobs in a corporation.
func (s *CorporationService) ListActiveCorporationIndustryJobs(ctx context.Context, corporationID int64) (int64, error) {
	return s.st.CountCorporationIndustryJobsWithStatus(ctx, corporationID, app.IndustryJobStatusActive)
}

var jobStatusFromESIValue = map[string]app.IndustryJobStatus{
	"active":    app.JobActive,
	"cancelled": app.JobCancelled,
	"delivered": app.JobDelivered,
	"paused":    app.JobPaused,
	"ready":     app.JobReady,
	"reverted":  app.JobReverted,
}

func (s *CorporationService) updateIndustryJobsESI(ctx context.Context, arg corporationSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCorporationIndustryJobs {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true,
		func(ctx context.Context, arg corporationSectionUpdateParams) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdIndustryJobs")
			jobs, err := xgoesi.FetchPages(
				func(page int32) ([]esi.CorporationsCorporationIdIndustryJobsGetInner, *http.Response, error) {
					return s.esiClient.IndustryAPI.GetCorporationsCorporationIdIndustryJobs(ctx, arg.corporationID).IncludeCompleted(true).Page(page).Execute()
				},
			)
			if err != nil {
				return false, err
			}
			slog.Debug("Received industry jobs from ESI", "corporationID", arg.corporationID, "count", len(jobs))
			return jobs, nil
		},
		func(ctx context.Context, arg corporationSectionUpdateParams, data any) (bool, error) {
			jobs := xslices.Map(data.([]esi.CorporationsCorporationIdIndustryJobsGetInner), industryJobFromESI)
			stored, err := s.st.ListCorporationIndustryJobs(ctx, arg.corporationID)
			if err != nil {
				return false, err
			}
			current := maps.Collect(xiter.MapSlice2(stored, func(j *app.CorporationIndustryJob) (int64, app.IndustryJobStatus) {
				return j.JobID, j.Status
			}))
			return industrysync.Sync(
				ctx,
				jobs,
				current,
				jobStatusFromESIValue,
				func(ctx context.Context) error {
					entityIDs, typeIDs := industrysync.ExtractIDs(jobs)
					var locationIDs set.Set[int64]
					for _, j := range jobs {
						locationIDs.Add(j.LocationId)
					}
					return s.eus.AddMissing(ctx, entityIDs, locationIDs, typeIDs)
				},
				func(ctx context.Context, j industrysync.ESIJob, status app.IndustryJobStatus) error {
					return s.st.UpdateOrCreateCorporationIndustryJob(ctx, storage.UpdateOrCreateCorporationIndustryJobParams{
						ActivityID:           j.ActivityId,
						BlueprintID:          j.BlueprintId,
						BlueprintLocationID:  j.BlueprintLocationId,
						BlueprintTypeID:      j.BlueprintTypeId,
						CompletedCharacterID: optional.FromPtr(j.CompletedCharacterId),
						CompletedDate:        optional.FromPtr(j.CompletedDate),
						CorporationID:        arg.corporationID,
						Cost:                 optional.FromPtr(j.Cost),
						Duration:             j.Duration,
						EndDate:              j.EndDate,
						FacilityID:           j.FacilityId,
						InstallerID:          j.InstallerId,
						JobID:                j.JobId,
						LicensedRuns:         optional.FromPtr(j.LicensedRuns),
						LocationID:           j.LocationId,
						OutputLocationID:     j.OutputLocationId,
						PauseDate:            optional.FromPtr(j.PauseDate),
						Probability:          optional.FromPtr(j.Probability),
						ProductTypeID:        optional.FromPtr(j.ProductTypeId),
						Runs:                 j.Runs,
						StartDate:            j.StartDate,
						Status:               status,
						SuccessfulRuns:       optional.FromPtr(j.SuccessfulRuns),
					})
				},
				func(ctx context.Context, ids set.Set[int64]) error {
					return s.st.UpdateCorporationIndustryJobStatus(ctx, storage.UpdateCorporationIndustryJobStatusParams{
						CorporationID: arg.corporationID,
						JobIDs:        ids,
						Status:        app.JobUnknown,
					})
				},
				slog.Int64("corporationID", arg.corporationID),
			)
		},
	)
}

// industryJobFromESI returns the job for an ESI response.
func industryJobFromESI(j esi.CorporationsCorporationIdIndustryJobsGetInner) industrysync.ESIJob {
	return industrysync.ESIJob{
		ActivityId:           j.ActivityId,
		BlueprintId:          j.BlueprintId,
		BlueprintLocationId:  j.BlueprintLocationId,
		BlueprintTypeId:      j.BlueprintTypeId,
		CompletedCharacterId: j.CompletedCharacterId,
		CompletedDate:        j.CompletedDate,
		Cost:                 j.Cost,
		Duration:             j.Duration,
		EndDate:              j.EndDate,
		FacilityId:           j.FacilityId,
		InstallerId:          j.InstallerId,
		JobId:                j.JobId,
		LicensedRuns:         j.LicensedRuns,
		LocationId:           j.LocationId,
		OutputLocationId:     j.OutputLocationId,
		PauseDate:            j.PauseDate,
		Probability:          j.Probability,
		ProductTypeId:        j.ProductTypeId,
		Runs:                 j.Runs,
		StartDate:            j.StartDate,
		Status:               j.Status,
		SuccessfulRuns:       j.SuccessfulRuns,
	}
}
