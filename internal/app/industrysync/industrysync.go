// Package industrysync provides the rules for updating stored industry jobs from ESI,
// which are shared by characters and corporations.
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

// orphanIDs returns the IDs of stored jobs which are active, but missing in ESI.
func orphanIDs(current map[int64]app.IndustryJobStatus, jobs []ESIJob) set.Set[int64] {
	var running, incoming set.Set[int64]
	for id, status := range current {
		if status.IsActive() {
			running.Add(id)
		}
	}
	for _, j := range jobs {
		incoming.Add(j.JobId)
	}
	return set.Difference(running, incoming)
}

// Sync updates stored jobs from ESI and reports whether anything has changed.
// All jobs are stored when any job is new or has a changed status.
// Active jobs missing in ESI are marked as unknown.
//
// current maps job IDs to their stored status.
// statuses maps the owner's ESI status values to their status.
// The callbacks do the owner specific work.
func Sync(
	ctx context.Context,
	jobs []ESIJob,
	current map[int64]app.IndustryJobStatus,
	statuses map[string]app.IndustryJobStatus,
	addRelated func(ctx context.Context) error,
	store func(ctx context.Context, job ESIJob, status app.IndustryJobStatus) error,
	markUnknown func(ctx context.Context, ids set.Set[int64]) error,
	logAttr slog.Attr,
) (bool, error) {
	now := time.Now()
	parsed := make([]app.IndustryJobStatus, len(jobs))
	var hasChanges bool
	for i, j := range jobs {
		parsed[i] = parseStatus(j, statuses, now, logAttr)
		if stored, found := current[j.JobId]; !found || stored != parsed[i] {
			hasChanges = true
		}
	}
	if hasChanges {
		if err := addRelated(ctx); err != nil {
			return false, err
		}
		for i, j := range jobs {
			if err := store(ctx, j, parsed[i]); err != nil {
				return false, err
			}
		}
		slog.Info("Updated industry jobs", logAttr, "count", len(jobs))
	}

	// The ESI response only returns jobs from the last 90 days, so a long
	// running job can vanish from it even when nothing else changed this cycle.
	orphans := orphanIDs(current, jobs)
	if orphans.Size() == 0 {
		return hasChanges, nil
	}
	if err := markUnknown(ctx, orphans); err != nil {
		return false, err
	}
	slog.Info("Marked orphaned industry jobs as unknown", logAttr, "count", orphans.Size())
	return true, nil
}
