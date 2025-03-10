package app

import "time"

type CharacterCorporationHistoryItem struct {
	CharaterID  int32
	Corporation *EveEntity
	IsDeleted   bool
	RecordID    int
	StartDate   time.Time
	EndDate     time.Time
}

func (it CharacterCorporationHistoryItem) Days() int {
	var endDate time.Time
	if !it.EndDate.IsZero() {
		endDate = it.EndDate
	} else {
		endDate = time.Now().UTC()
	}
	return int(endDate.Sub(it.StartDate) / (time.Hour * 24))
}
