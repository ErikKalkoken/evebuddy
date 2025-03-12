package app

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
)

// An Eve Online corporation.
type EveCorporation struct {
	Alliance    *EveEntity
	Ceo         *EveEntity
	Creator     *EveEntity
	DateFounded time.Time
	Description string
	Faction     *EveEntity
	HomeStation *EveEntity
	ID          int32
	MemberCount int
	Name        string
	Shares      int
	TaxRate     float32
	Ticker      string
	URL         string
	WarEligible bool
	Timestamp   time.Time
}

func (ec EveCorporation) HasAlliance() bool {
	return ec.Alliance != nil
}

func (ec EveCorporation) HasFaction() bool {
	return ec.Faction != nil
}

func (ec EveCorporation) DescriptionPlain() string {
	return evehtml.ToPlain(ec.Description)
}

func (ec EveCorporation) ToEveEntity() *EveEntity {
	return &EveEntity{ID: ec.ID, Name: ec.Name, Category: EveEntityCorporation}
}
