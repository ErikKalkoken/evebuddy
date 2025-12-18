package corporationservice

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

// GetCorporationIndustryJob returns an industry job.
func (s *CorporationService) GetCorporationIndustryJob(ctx context.Context, corporationID, jobID int32) (*app.CorporationIndustryJob, error) {
	return s.st.GetCorporationIndustryJob(ctx, corporationID, jobID)
}

// ListAllCorporationIndustryJobs returns all industry jobs from all corporations.
func (s *CorporationService) ListAllCorporationIndustryJobs(ctx context.Context) ([]*app.CorporationIndustryJob, error) {
	return s.st.ListAllCorporationIndustryJobs(ctx)
}

// ListCorporationIndustryJobs returns all industry jobs from a corporation.
func (s *CorporationService) ListCorporationIndustryJobs(ctx context.Context, corporationID int32) ([]*app.CorporationIndustryJob, error) {
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

func (s *CorporationService) updateIndustryJobsESI(ctx context.Context, arg app.CorporationSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCorporationIndustryJobs {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	var hasChanged bool
	_, err := s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, arg app.CorporationSectionUpdateParams) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdIndustryJobs")
			jobs, err := xgoesi.FetchPages(
				func(pageNum int) ([]esi.GetCorporationsCorporationIdIndustryJobs200Ok, *http.Response, error) {
					return s.esiClient.ESI.IndustryApi.GetCorporationsCorporationIdIndustryJobs(
						ctx, arg.CorporationID, &esi.GetCorporationsCorporationIdIndustryJobsOpts{
							IncludeCompleted: esioptional.NewBool(true),
							Page:             esioptional.NewInt32(int32(pageNum)),
						},
					)
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
			slog.Debug("Received industry jobs from ESI", "corporationID", arg.CorporationID, "count", len(jobs))
			return jobs, nil
		},
		func(ctx context.Context, arg app.CorporationSectionUpdateParams, data any) error {
			jobs := data.([]esi.GetCorporationsCorporationIdIndustryJobs200Ok)

			statusFromESIJob := func(j esi.GetCorporationsCorporationIdIndustryJobs200Ok) app.IndustryJobStatus {
				status, ok := jobStatusFromESIValue[j.Status]
				if !ok {
					return app.JobUndefined
				}
				return status
			}

			// Identify changed jobs
			jj, err := s.st.ListCorporationIndustryJobs(ctx, arg.CorporationID)
			if err != nil {
				return err
			}
			currentJobs := maps.Collect(xiter.MapSlice2(jj, func(j *app.CorporationIndustryJob) (int32, app.IndustryJobStatus) {
				return j.JobID, j.Status
			}))
			changedJobs := make([]esi.GetCorporationsCorporationIdIndustryJobs200Ok, 0)
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
			hasChanged = len(changedJobs) > 0
			if hasChanged {
				var entityIDs set.Set[int32]
				var typeIDs set.Set[int32]
				var locationIDs set.Set[int64]
				for _, j := range jobs {
					entityIDs.Add(j.InstallerId, j.CompletedCharacterId)
					locationIDs.Add(j.LocationId)
					typeIDs.Add(j.BlueprintTypeId, j.ProductTypeId)
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
					return err
				}
				for _, j := range jobs {
					if err := s.st.UpdateOrCreateCorporationIndustryJob(ctx, storage.UpdateOrCreateCorporationIndustryJobParams{
						ActivityID:           j.ActivityId,
						BlueprintID:          j.BlueprintId,
						BlueprintLocationID:  j.BlueprintLocationId,
						BlueprintTypeID:      j.BlueprintTypeId,
						CompletedCharacterID: j.CompletedCharacterId,
						CompletedDate:        j.CompletedDate,
						CorporationID:        arg.CorporationID,
						Cost:                 j.Cost,
						Duration:             j.Duration,
						EndDate:              j.EndDate,
						FacilityID:           j.FacilityId,
						InstallerID:          j.InstallerId,
						JobID:                j.JobId,
						LicensedRuns:         j.LicensedRuns,
						LocationID:           j.LocationId,
						OutputLocationID:     j.OutputLocationId,
						PauseDate:            j.PauseDate,
						Probability:          j.Probability,
						ProductTypeID:        j.ProductTypeId,
						Runs:                 j.Runs,
						StartDate:            j.StartDate,
						Status:               statusFromESIJob(j),
						SuccessfulRuns:       j.SuccessfulRuns,
					}); err != nil {
						return err
					}
				}
				slog.Info("Updated industry jobs", "corporationID", arg.CorporationID, "count", len(jobs))

				// Mark orphans
				incoming := set.Collect(xiter.MapSlice(jobs, func(x esi.GetCorporationsCorporationIdIndustryJobs200Ok) int32 {
					return x.JobId
				}))
				current, err := s.st.ListCorporationIndustryJobs(ctx, arg.CorporationID)
				if err != nil {
					return err
				}
				running := set.Collect(xiter.Map(xiter.FilterSlice(current, func(x *app.CorporationIndustryJob) bool {
					return x.Status.IsActive()
				}), func(x *app.CorporationIndustryJob) int32 {
					return x.JobID
				}))
				orphans := set.Difference(running, incoming)
				if orphans.Size() > 0 {
					// The ESI response only returns jobs from the last 90 days.
					// It can therefore happen that a long running job vanishes from the response,
					// without the app having received a final status (e.g. delivered or canceled).
					// The status of these orphaned job is therefore marked as undefined.
					err := s.st.UpdateCorporationIndustryJobStatus(ctx, storage.UpdateCorporationIndustryJobStatusParams{
						CorporationID: arg.CorporationID,
						JobIDs:        orphans,
						Status:        app.JobUnknown,
					})
					if err != nil {
						return err
					}
					slog.Info(
						"Marked orphaned industry jobs as unknown",
						"corporationID", arg.CorporationID,
						"count", orphans.Size(),
					)
				}
			}
			return nil
		},
	)
	if err != nil {
		return false, err
	}
	return hasChanged, nil
}
