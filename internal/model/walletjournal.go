package model

import (
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type WalletJournalEntry struct {
	Amount        float64
	Balance       float64
	ContextID     int64
	ContextIDType string
	Date          time.Time
	Description   string
	FirstParty    *EveEntity
	ID            int64
	MyCharacterID int32
	Reason        string
	RefType       string
	SecondParty   *EveEntity
	Tax           float64
	TaxReceiver   *EveEntity
}

func (e *WalletJournalEntry) Type() string {
	s := strings.ReplaceAll(e.RefType, "_", " ")
	c := cases.Title(language.English)
	s = c.String(s)
	return s
}
