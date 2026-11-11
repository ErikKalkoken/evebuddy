// Package industrysync provides the logic for updating stored industry jobs from ESI,
// which is shared by characters and corporations.
package industrysync

import (
	"context"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// ESIJob is an industry job from ESI. Services must map it field by field,
// because the ESI responses for characters and corporations differ in LocationId.
type ESIJob struct {
	ActivityId           int64
	BlueprintId          int64
	BlueprintLocationId  int64
	BlueprintTypeId      int64
	CompletedCharacterId *int64
	CompletedDate        *time.Time
	Cost                 *float64
	Duration             int64
	EndDate              time.Time
	FacilityId           int64
	InstallerId          int64
	JobId                int64
	LicensedRuns         *int64
	LocationId           int64 // station_id for characters
	OutputLocationId     int64
	PauseDate            *time.Time
	Probability          *float64
	ProductTypeId        *int64
	Runs                 int64
	StartDate            time.Time
	Status               string
	SuccessfulRuns       *int64
}

// ExtractIDs returns the IDs of the entities and types referenced by jobs.
func ExtractIDs(jobs []ESIJob) (entityIDs, typeIDs set.Set[int64]) {
	for _, j := range jobs {
		entityIDs.Add(j.InstallerId)
		if x := j.CompletedCharacterId; x != nil {
			entityIDs.Add(*x)
		}
		typeIDs.Add(j.BlueprintTypeId)
		if x := j.ProductTypeId; x != nil {
			typeIDs.Add(*x)
		}
	}
	return entityIDs, typeIDs
}

// parseStatus returns the status of an ESI job.
// Unknown values do not fail, because ESI can add new ones at any time.
func parseStatus(j ESIJob, statuses map[string]app.IndustryJobStatus, now time.Time, logAttr slog.Attr) app.IndustryJobStatus {
	status, found := statuses[j.Status]
	if !found {
		slog.Warn("Unknown industry job status encountered", logAttr, "jobID", j.JobId, "status", j.Status)
		return app.JobUndefined
	}
	// Fix incorrect status for known bug: https://github.com/esi/esi-issues/issues/752
	if status == app.JobActive && !j.EndDate.IsZero() && j.EndDate.Before(now) {
		return app.JobReady
	}
	return status
}

// Sync updates stored jobs from ESI and reports whether anything has changed.
// Of duplicate jobs only the last is stored.
//
// statuses maps the owner's ESI status values to their status.
// The callbacks do the owner specific work:
// addRelated adds missing related data and
// save stores jobs, where jobStatuses[i] is the status of jobs[i].
func Sync(
	ctx context.Context,
	jobs []ESIJob,
	statuses map[string]app.IndustryJobStatus,
	addRelated func(ctx context.Context) error,
	save func(ctx context.Context, jobs []ESIJob, jobStatuses []app.IndustryJobStatus) (bool, error),
	logAttr slog.Attr,
) (bool, error) {
	jobs = dedupeJobs(jobs, logAttr)
	now := time.Now()
	jobStatuses := make([]app.IndustryJobStatus, len(jobs))
	for i, j := range jobs {
		jobStatuses[i] = parseStatus(j, statuses, now, logAttr)
	}
	if err := addRelated(ctx); err != nil {
		return false, err
	}
	return save(ctx, jobs, jobStatuses)
}

// dedupeJobs returns jobs with only the last of each job ID.
// Duplicates can occur when ESI pages shift between fetches.
func dedupeJobs(jobs []ESIJob, logAttr slog.Attr) []ESIJob {
	last := make(map[int64]int, len(jobs))
	for i, j := range jobs {
		last[j.JobId] = i
	}
	if len(last) == len(jobs) {
		return jobs
	}
	slog.Warn("Duplicate industry jobs received from ESI", logAttr, "count", len(jobs)-len(last))
	unique := make([]ESIJob, 0, len(last))
	for i, j := range jobs {
		if last[j.JobId] == i {
			unique = append(unique, j)
		}
	}
	return unique
}
