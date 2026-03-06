package corporationservice

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
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
			// Fix incorrect status for known bug: https://github.com/esi/esi-issues/issues/752
			for i, j := range jobs {
				if j.Status == "active" && !j.EndDate.IsZero() && j.EndDate.Before(time.Now()) {
					jobs[i].Status = "ready"
				}
			}
			slices.SortFunc(jobs, func(a, b esi.CorporationsCorporationIdIndustryJobsGetInner) int {
				return cmp.Compare(a.JobId, b.JobId)
			})
			slog.Debug("Received industry jobs from ESI", "corporationID", arg.corporationID, "count", len(jobs))
			return jobs, nil
		},
		func(ctx context.Context, arg corporationSectionUpdateParams, data any) (bool, error) {
			jobs := data.([]esi.CorporationsCorporationIdIndustryJobsGetInner)

			statusFromESIJob := func(j esi.CorporationsCorporationIdIndustryJobsGetInner) app.IndustryJobStatus {
				status, ok := jobStatusFromESIValue[j.Status]
				if !ok {
					return app.JobUndefined
				}
				return status
			}

			// Identify changed jobs
			jj, err := s.st.ListCorporationIndustryJobs(ctx, arg.corporationID)
			if err != nil {
				return false, err
			}
			currentJobs := maps.Collect(xiter.MapSlice2(jj, func(j *app.CorporationIndustryJob) (int64, app.IndustryJobStatus) {
				return j.JobID, j.Status
			}))
			var changedJobs []esi.CorporationsCorporationIdIndustryJobsGetInner
			for _, j := range jobs {
				status, found := currentJobs[j.JobId]
				if !found {
					changedJobs = append(changedJobs, j)
					continue
				}
				if statusFromESIJob(j) == status {
					continue
				}
				changedJobs = append(changedJobs, j)
			}

			// Process changed jobs
			if len(changedJobs) == 0 {
				return false, nil
			}

			var entityIDs set.Set[int64]
			var typeIDs set.Set[int64]
			var locationIDs set.Set[int64]
			for _, j := range jobs {
				entityIDs.Add(j.InstallerId)
				if x := j.CompletedCharacterId; x != nil {
					entityIDs.Add(*x)
				}
				locationIDs.Add(j.LocationId)
				typeIDs.Add(j.BlueprintTypeId)
				if x := j.ProductTypeId; x != nil {
					typeIDs.Add(*x)
				}
			}
			g := new(errgroup.Group)
			g.Go(func() error {
				_, err := s.eus.AddMissingEntities(ctx, entityIDs)
				return err
			})
			g.Go(func() error {
				return s.eus.AddMissingLocations(ctx, locationIDs)
			})
			g.Go(func() error {
				return s.eus.AddMissingTypes(ctx, typeIDs)
			})
			if err := g.Wait(); err != nil {
				return false, err
			}
			for _, j := range jobs {
				if err := s.st.UpdateOrCreateCorporationIndustryJob(ctx, storage.UpdateOrCreateCorporationIndustryJobParams{
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
					Status:               statusFromESIJob(j),
					SuccessfulRuns:       optional.FromPtr(j.SuccessfulRuns),
				}); err != nil {
					return false, err
				}
			}
			slog.Info("Updated industry jobs", "corporationID", arg.corporationID, "count", len(jobs))

			// Mark orphans
			incoming := set.Collect(xiter.MapSlice(jobs, func(x esi.CorporationsCorporationIdIndustryJobsGetInner) int64 {
				return x.JobId
			}))
			current, err := s.st.ListCorporationIndustryJobs(ctx, arg.corporationID)
			if err != nil {
				return false, err
			}
			running := set.Collect(xiter.Map(xiter.FilterSlice(current, func(x *app.CorporationIndustryJob) bool {
				return x.Status.IsActive()
			}), func(x *app.CorporationIndustryJob) int64 {
				return x.JobID
			}))
			orphans := set.Difference(running, incoming)
			if orphans.Size() == 0 {
				return true, nil
			}

			// The ESI response only returns jobs from the last 90 days.
			// It can therefore happen that a long running job vanishes from the response,
			// without the app having received a final status (e.g. delivered or canceled).
			// The status of these orphaned job is therefore marked as undefined.
			err = s.st.UpdateCorporationIndustryJobStatus(ctx, storage.UpdateCorporationIndustryJobStatusParams{
				CorporationID: arg.corporationID,
				JobIDs:        orphans,
				Status:        app.JobUnknown,
			})
			if err != nil {
				return false, err
			}
			slog.Info(
				"Marked orphaned industry jobs as unknown",
				"corporationID", arg.corporationID,
				"count", orphans.Size(),
			)

			return true, nil
		},
	)
}
