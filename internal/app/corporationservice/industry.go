package corporationservice

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
)

// GetCorporationIndustryJob returns an industry job.
func (s *CorporationService) GetCorporationIndustryJob(ctx context.Context, corporationID, jobID int32) (*app.CorporationIndustryJob, error) {
	return s.st.GetCorporationIndustryJob(ctx, corporationID, jobID)
}

// ListAllCorporationIndustryJobs returns all industry jobs from corporations.
func (s *CorporationService) ListAllCorporationIndustryJobs(ctx context.Context) ([]*app.CorporationIndustryJob, error) {
	return s.st.ListAllCorporationIndustryJobs(ctx)
}

var jobStatusFromESIValue = map[string]app.IndustryJobStatus{
	"active":    app.JobActive,
	"cancelled": app.JobCancelled,
	"delivered": app.JobDelivered,
	"paused":    app.JobPaused,
	"ready":     app.JobReady,
	"reverted":  app.JobReverted,
}

func (s *CorporationService) updateIndustryJobsESI(ctx context.Context, arg app.CorporationUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionCorporationIndustryJobs {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, arg app.CorporationUpdateSectionParams) (any, error) {
			jobs, err := xesi.FetchWithPaging(
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
			slog.Debug("Received industry jobs from ESI", "corporationID", arg.CorporationID, "count", len(jobs))
			return jobs, nil
		},
		func(ctx context.Context, arg app.CorporationUpdateSectionParams, data any) error {
			jobs := data.([]esi.GetCorporationsCorporationIdIndustryJobs200Ok)
			entityIDs := set.Of[int32]()
			typeIDs := set.Of[int32]()
			locationIDs := set.Of[int64]()
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
				status, ok := jobStatusFromESIValue[j.Status]
				if !ok {
					status = app.JobUndefined
				}
				if status == app.JobActive && !j.EndDate.IsZero() && j.EndDate.Before(time.Now()) {
					// Workaround for known bug: https://github.com/esi/esi-issues/issues/752
					status = app.JobReady
				}
				arg := storage.UpdateOrCreateCorporationIndustryJobParams{
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
					Status:               status,
					SuccessfulRuns:       j.SuccessfulRuns,
				}
				if err := s.st.UpdateOrCreateCorporationIndustryJob(ctx, arg); err != nil {
					return nil
				}
			}
			slog.Info("Updated industry jobs", "corporationID", arg.CorporationID, "count", len(jobs))
			return nil
		})
}
