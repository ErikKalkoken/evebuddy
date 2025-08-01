package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
	"golang.org/x/sync/errgroup"
)

func (s *CharacterService) GetCharacterIndustryJob(ctx context.Context, characterID, jobID int32) (*app.CharacterIndustryJob, error) {
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

func (s *CharacterService) updateIndustryJobsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionCharacterIndustryJobs {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			jobs, _, err := s.esiClient.ESI.IndustryApi.GetCharactersCharacterIdIndustryJobs(ctx, characterID, &esi.GetCharactersCharacterIdIndustryJobsOpts{
				IncludeCompleted: esioptional.NewBool(true),
			})
			if err != nil {
				return false, err
			}
			slog.Debug("Received industry jobs from ESI", "characterID", characterID, "count", len(jobs))
			return jobs, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			jobs := data.([]esi.GetCharactersCharacterIdIndustryJobs200Ok)
			entityIDs := set.Of[int32]()
			typeIDs := set.Of[int32]()
			locationIDs := set.Of[int64]()
			for _, j := range jobs {
				entityIDs.Add(j.InstallerId, j.CompletedCharacterId)
				locationIDs.Add(j.BlueprintLocationId, j.OutputLocationId, j.StationId)
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
				arg := storage.UpdateOrCreateCharacterIndustryJobParams{
					ActivityID:           j.ActivityId,
					BlueprintID:          j.BlueprintId,
					BlueprintLocationID:  j.BlueprintLocationId,
					BlueprintTypeID:      j.BlueprintTypeId,
					CharacterID:          characterID,
					CompletedCharacterID: j.CompletedCharacterId,
					CompletedDate:        j.CompletedDate,
					Cost:                 j.Cost,
					Duration:             j.Duration,
					EndDate:              j.EndDate,
					FacilityID:           j.FacilityId,
					InstallerID:          j.InstallerId,
					LicensedRuns:         j.LicensedRuns,
					JobID:                j.JobId,
					OutputLocationID:     j.OutputLocationId,
					Runs:                 j.Runs,
					PauseDate:            j.PauseDate,
					Probability:          j.Probability,
					ProductTypeID:        j.ProductTypeId,
					StartDate:            j.StartDate,
					Status:               status,
					StationID:            j.StationId,
					SuccessfulRuns:       j.SuccessfulRuns,
				}
				if err := s.st.UpdateOrCreateCharacterIndustryJob(ctx, arg); err != nil {
					return err
				}
			}
			slog.Info("Updated industry jobs", "characterID", characterID, "count", len(jobs))

			incoming := set.Collect(xiter.MapSlice(jobs, func(x esi.GetCharactersCharacterIdIndustryJobs200Ok) int32 {
				return x.JobId
			}))
			current, err := s.st.ListCharacterIndustryJobs(ctx, characterID)
			if err != nil {
				return err
			}
			running := set.Collect(xiter.Map(xiter.FilterSlice(current, func(x *app.CharacterIndustryJob) bool {
				return !x.Status.IsHistory()
			}), func(x *app.CharacterIndustryJob) int32 {
				return x.JobID
			}))
			orphans := set.Difference(running, incoming)
			if orphans.Size() > 0 {
				// The ESI response only returns jobs from the last 90 days.
				// It can therefore happen that a long running job vanishes from the response,
				// without the app having received a final status (e.g. delivered or canceled).
				// Since the status of the job is undetermined we can only delete it.
				err := s.st.DeleteCharacterIndustryJobsByID(ctx, characterID, orphans)
				if err != nil {
					return err
				}
				slog.Info("Deleted orphaned industry jobs", "characterID", characterID, "count", orphans.Size())
			}
			return nil
		})
}
