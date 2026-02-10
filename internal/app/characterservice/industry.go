package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func (s *CharacterService) GetCharacterIndustryJob(ctx context.Context, characterID, jobID int64) (*app.CharacterIndustryJob, error) {
	return s.st.GetCharacterIndustryJob(ctx, characterID, jobID)
}

// ListAllCharacterIndustryJob returns all industry jobs from characters.
func (s *CharacterService) ListAllCharacterIndustryJob(ctx context.Context) ([]*app.CharacterIndustryJob, error) {
	return s.st.ListAllCharacterIndustryJob(ctx)
}

var jobStatusFromESIValue = map[string]app.IndustryJobStatus{
	"active":    app.JobActive,
	"cancelled": app.JobCancelled,
	"delivered": app.JobDelivered,
	"paused":    app.JobPaused,
	"ready":     app.JobReady,
	"reverted":  app.JobReverted,
}

func (s *CharacterService) updateIndustryJobsESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterIndustryJobs {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	var hasChanged bool
	_, err := s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdIndustryJobs")
			jobs, _, err := s.esiClient.IndustryAPI.GetCharactersCharacterIdIndustryJobs(ctx, characterID).IncludeCompleted(true).Execute()
			if err != nil {
				return false, err
			}
			// Fix incorrect status for known bug: https://github.com/esi/esi-issues/issues/752
			for i, j := range jobs {
				if j.Status == "active" && !j.EndDate.IsZero() && j.EndDate.Before(time.Now()) {
					jobs[i].Status = "ready"
				}
			}
			slog.Debug("Received industry jobs from ESI", "characterID", characterID, "count", len(jobs))
			return jobs, nil
		},
		func(ctx context.Context, characterID int64, data any) error {
			jobs := data.([]esi.CharactersCharacterIdIndustryJobsGetInner)

			statusFromESIJob := func(j esi.CharactersCharacterIdIndustryJobsGetInner) app.IndustryJobStatus {
				status, ok := jobStatusFromESIValue[j.Status]
				if !ok {
					return app.JobUndefined
				}
				return status
			}

			// Identify changed jobs
			jj, err := s.st.ListCharacterIndustryJobs(ctx, characterID)
			if err != nil {
				return err
			}
			currentJobs := maps.Collect(xiter.MapSlice2(jj, func(j *app.CharacterIndustryJob) (int64, app.IndustryJobStatus) {
				return j.JobID, j.Status
			}))
			changedJobs := make([]esi.CharactersCharacterIdIndustryJobsGetInner, 0)
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
				var entityIDs set.Set[int64]
				var typeIDs set.Set[int64]
				var locationIDs set.Set[int64]
				for _, j := range jobs {
					entityIDs.Add(j.InstallerId)
					if x := j.CompletedCharacterId; x != nil {
						entityIDs.Add(*x)
					}
					locationIDs.Add(j.BlueprintLocationId, j.OutputLocationId, j.StationId)
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
					return err
				}
				for _, j := range jobs {
					err := s.st.UpdateOrCreateCharacterIndustryJob(ctx, storage.UpdateOrCreateCharacterIndustryJobParams{
						ActivityID:           j.ActivityId,
						BlueprintID:          j.BlueprintId,
						BlueprintLocationID:  j.BlueprintLocationId,
						BlueprintTypeID:      j.BlueprintTypeId,
						CharacterID:          characterID,
						CompletedCharacterID: optional.FromPtr(j.CompletedCharacterId),
						CompletedDate:        optional.FromPtr(j.CompletedDate),
						Cost:                 optional.FromPtr(j.Cost),
						Duration:             j.Duration,
						EndDate:              j.EndDate,
						FacilityID:           j.FacilityId,
						InstallerID:          j.InstallerId,
						LicensedRuns:         optional.FromPtr(j.LicensedRuns),
						JobID:                j.JobId,
						OutputLocationID:     j.OutputLocationId,
						Runs:                 j.Runs,
						PauseDate:            optional.FromPtr(j.PauseDate),
						Probability:          optional.FromPtr(j.Probability),
						ProductTypeID:        optional.FromPtr(j.ProductTypeId),
						StartDate:            j.StartDate,
						Status:               statusFromESIJob(j),
						StationID:            j.StationId,
						SuccessfulRuns:       optional.FromPtr(j.SuccessfulRuns),
					})
					if err != nil {
						return err
					}
				}
				slog.Info("Updated industry jobs", "characterID", characterID, "count", len(jobs))

				// Mark orphans
				incoming := set.Collect(xiter.MapSlice(jobs, func(x esi.CharactersCharacterIdIndustryJobsGetInner) int64 {
					return x.JobId
				}))
				jj2, err := s.st.ListCharacterIndustryJobs(ctx, characterID)
				if err != nil {
					return err
				}
				running := set.Collect(xiter.Map(xiter.FilterSlice(jj2, func(x *app.CharacterIndustryJob) bool {
					return x.Status.IsActive()
				}), func(x *app.CharacterIndustryJob) int64 {
					return x.JobID
				}))
				orphans := set.Difference(running, incoming)
				if orphans.Size() > 0 {
					// The ESI response only returns jobs from the last 90 days.
					// It can therefore happen that a long running job vanishes from the response,
					// without the app having received a final status (e.g. delivered or canceled).
					// The status of these orphaned job is therefore marked as undefined.
					err := s.st.UpdateCharacterIndustryJobStatus(ctx, storage.UpdateCharacterIndustryJobStatusParams{
						CharacterID: characterID,
						JobIDs:      orphans,
						Status:      app.JobUnknown,
					})
					if err != nil {
						return err
					}
					slog.Info(
						"Marked orphaned industry jobs as unknown",
						"characterID", characterID,
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
