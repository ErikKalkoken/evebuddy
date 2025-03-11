package app

import "time"

type EveAlliance struct {
	CreatorCorporation  *EveEntity
	Creator             *EveEntity
	DateFounded         time.Time
	ExecutorCorporation *EveEntity
	Faction             *EveEntity
	ID                  int32
	Name                string
	Ticker              string
}
