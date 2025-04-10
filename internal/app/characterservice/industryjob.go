package characterservice

import (
	"context"
	"log/slog"

	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

var jobStatusFromESIValue = map[string]app.IndustryJobStatus{
	"active":    app.JobActive,
	"cancelled": app.JobCancelled,
	"delivered": app.JobDelivered,
	"paused":    app.JobPaused,
	"ready":     app.JobReady,
	"reverted":  app.JobReverted,
}

func (s *CharacterService) updateIndustryJobsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionIndustryJobs {
		panic("called with wrong section")
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

			entityIDs := set.New[int32]()
			typeIDs := set.New[int32]()
			locationIDs := set.New[int64]()
			for _, j := range jobs {
				entityIDs.Add(j.InstallerId)
				if j.CompletedCharacterId != 0 {
					entityIDs.Add(j.CompletedCharacterId)
				}
				locationIDs.Add(j.BlueprintLocationId)
				locationIDs.Add(j.OutputLocationId)
				locationIDs.Add(j.StationId)
				typeIDs.Add(j.BlueprintTypeId)
				if j.ProductTypeId != 0 {
					typeIDs.Add(j.ProductTypeId)
				}
			}
			if _, err := s.EveUniverseService.AddMissingEntities(ctx, entityIDs.ToSlice()); err != nil {
				return err
			}
			for id := range locationIDs.Values() {
				if _, err := s.EveUniverseService.GetOrCreateLocationESI(ctx, id); err != nil {
					return err
				}
			}
			for id := range typeIDs.Values() {
				if _, err := s.EveUniverseService.GetOrCreateTypeESI(ctx, id); err != nil {
					return err
				}
			}
			for _, j := range jobs {
				status, ok := jobStatusFromESIValue[j.Status]
				if !ok {
					status = app.JobUndefined
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
					return nil
				}
			}
			slog.Info("Updated industry jobs", "characterID", characterID, "count", len(jobs))
			return nil
		})
}
