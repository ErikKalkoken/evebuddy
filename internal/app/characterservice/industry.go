package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/industrysync"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func (s *CharacterService) GetCharacterIndustryJob(ctx context.Context, characterID, jobID int64) (*app.CharacterIndustryJob, error) {
	return s.st.GetCharacterIndustryJob(ctx, characterID, jobID)
}

// ListAllCharacterIndustryJob returns all industry jobs from characters.
func (s *CharacterService) ListAllCharacterIndustryJob(ctx context.Context) ([]*app.CharacterIndustryJob, error) {
	return s.st.ListAllCharacterIndustryJob(ctx)
}

func (s *CharacterService) ListAllCharactersIndustrySlots(ctx context.Context, typ app.IndustryJobType) ([]app.CharacterIndustrySlots, error) {
	total := make(map[int64]int)
	switch typ {
	case app.ManufacturingJob:
		industry1, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeIndustry)
		if err != nil {
			return nil, err
		}
		for _, r := range industry1 {
			if r.Level > 0 {
				total[r.CharacterID]++
			}
		}
		industry2, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeMassProduction)
		if err != nil {
			return nil, err
		}
		for _, r := range industry2 {
			total[r.CharacterID] += r.Level
		}
		industry3, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeAdvancedMassProduction)
		if err != nil {
			return nil, err
		}
		for _, r := range industry3 {
			total[r.CharacterID] += r.Level
		}
	case app.ScienceJob:
		research1, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeLaboratoryOperation)
		if err != nil {
			return nil, err
		}
		for _, r := range research1 {
			total[r.CharacterID] += r.Level + 1 // also adds base slot
		}
		research2, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeAdvancedLaboratoryOperation)
		if err != nil {
			return nil, err
		}
		for _, r := range research2 {
			total[r.CharacterID] += r.Level
		}
	case app.ReactionJob:
		reactions1, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeMassReactions)
		if err != nil {
			return nil, err
		}
		for _, r := range reactions1 {
			total[r.CharacterID] += r.Level + 1 // also adds base slot
		}
		reactions2, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeAdvancedMassReactions)
		if err != nil {
			return nil, err
		}
		for _, r := range reactions2 {
			total[r.CharacterID] += r.Level
		}
	}
	characters, err := s.st.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	results := make(map[int64]app.CharacterIndustrySlots)
	for _, c := range characters {
		results[c.ID] = app.CharacterIndustrySlots{
			CharacterID:   c.ID,
			CharacterName: c.Name,
			Type:          typ,
			Total:         total[c.ID],
		}
	}
	counts, err := s.st.ListAllCharacterIndustryJobActiveCounts(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range counts {
		if !typ.Activities().Contains(r.Activity) {
			continue
		}
		x := results[r.InstallerID]
		switch r.Status {
		case app.JobActive:
			x.Busy += r.Count
		case app.JobReady:
			x.Ready += r.Count
		}
		results[r.InstallerID] = x
	}
	for id, r := range results {
		r.Free = r.Total - r.Busy - r.Ready
		results[id] = r
	}
	rows := slices.SortedFunc(maps.Values(results), func(a, b app.CharacterIndustrySlots) int {
		return strings.Compare(a.CharacterName, b.CharacterName)
	})
	return rows, nil
}

var jobStatusFromESIValue = map[string]app.IndustryJobStatus{
	"active":    app.JobActive,
	"cancelled": app.JobCancelled,
	"delivered": app.JobDelivered,
	"paused":    app.JobPaused,
	"ready":     app.JobReady,
	"reverted":  app.JobReverted,
}

func (s *CharacterService) updateIndustryJobsESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterIndustryJobs {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdIndustryJobs")
			jobs, _, err := s.esiClient.IndustryAPI.GetCharactersCharacterIdIndustryJobs(ctx, characterID).IncludeCompleted(true).Execute()
			if err != nil {
				return false, err
			}
			slog.Debug("Received industry jobs from ESI", "characterID", characterID, "count", len(jobs))
			return jobs, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			jobs := xslices.Map(data.([]esi.CharactersCharacterIdIndustryJobsGetInner), industryJobFromESI)
			stored, err := s.st.ListCharacterIndustryJobs(ctx, characterID)
			if err != nil {
				return false, err
			}
			current := maps.Collect(xiter.MapSlice2(stored, func(j *app.CharacterIndustryJob) (int64, app.IndustryJobStatus) {
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
						locationIDs.Add(j.BlueprintLocationId, j.FacilityId, j.OutputLocationId, j.LocationId)
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
					return g.Wait()
				},
				func(ctx context.Context, j industrysync.ESIJob, status app.IndustryJobStatus) error {
					return s.st.UpdateOrCreateCharacterIndustryJob(ctx, storage.UpdateOrCreateCharacterIndustryJobParams{
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
						Status:               status,
						StationID:            j.LocationId,
						SuccessfulRuns:       optional.FromPtr(j.SuccessfulRuns),
					})
				},
				func(ctx context.Context, ids set.Set[int64]) error {
					return s.st.UpdateCharacterIndustryJobStatus(ctx, storage.UpdateCharacterIndustryJobStatusParams{
						CharacterID: characterID,
						JobIDs:      ids,
						Status:      app.JobUnknown,
					})
				},
				slog.Int64("characterID", characterID),
			)
		},
	)
}

// industryJobFromESI returns the job for an ESI response.
func industryJobFromESI(j esi.CharactersCharacterIdIndustryJobsGetInner) industrysync.ESIJob {
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
		LocationId:           j.StationId,
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
