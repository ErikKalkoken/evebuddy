package storage

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

var jobStatusFromDBValue = map[string]app.IndustryJobStatus{
	"":          app.JobUndefined,
	"active":    app.JobActive,
	"cancelled": app.JobCancelled,
	"delivered": app.JobDelivered,
	"paused":    app.JobPaused,
	"ready":     app.JobReady,
	"reverted":  app.JobReverted,
	"unknown":   app.JobUnknown,
}

var jobStatusToDBValue = map[app.IndustryJobStatus]string{}

func init() {
	for k, v := range jobStatusFromDBValue {
		jobStatusToDBValue[v] = k
	}
}

// industryJobState holds the columns of an industry job which can change after it was created.
type industryJobState struct {
	completedCharacterID optional.Optional[int64]
	completedDate        optional.Optional[time.Time]
	endDate              time.Time
	pauseDate            optional.Optional[time.Time]
	status               app.IndustryJobStatus
	successfulRuns       optional.Optional[int64]
}

// equal reports whether s and other are equal.
// Times are compared with Equal, because their location can differ after a round trip through the DB.
func (s industryJobState) equal(other industryJobState) bool {
	return optional.Equal(s.completedCharacterID, other.completedCharacterID) &&
		optional.Equal2(s.completedDate, other.completedDate) &&
		s.endDate.Equal(other.endDate) &&
		optional.Equal2(s.pauseDate, other.pauseDate) &&
		s.status == other.status &&
		optional.Equal(s.successfulRuns, other.successfulRuns)
}

// orphanedIndustryJobIDs returns the IDs of the active jobs in states.
// ESI only returns jobs from the last 90 days, so long running jobs can vanish from it.
func orphanedIndustryJobIDs(states map[int64]industryJobState) []int64 {
	var ids []int64
	for id, s := range states {
		if s.status.IsActive() {
			ids = append(ids, id)
		}
	}
	return ids
}
