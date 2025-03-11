package app

import "time"

type MembershipHistoryItem struct {
	EndDate      time.Time
	Days         int
	IsDeleted    bool
	IsOldest     bool
	Organization *EveEntity
	RecordID     int
	StartDate    time.Time
}

func (hi MembershipHistoryItem) OrganizationName() string {
	if hi.Organization != nil {
		return hi.Organization.Name
	}
	return "?"
}
