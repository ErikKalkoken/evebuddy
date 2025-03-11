package app

import "time"

type MembershipHistoryItem struct {
	EndDate      time.Time
	Days         int
	IsDeleted    bool
	Organization *EveEntity
	RecordID     int
	StartDate    time.Time
}
