package model

import (
	"time"
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

type WalletTransaction struct {
	Client        *EveEntity
	Date          time.Time
	EveType       *EntityShort[int32]
	IsBuy         bool
	IsPersonal    bool
	JournalRefID  int64
	Location      *EntityShort[int64]
	MyCharacterID int32
	Quantity      int32
	TransactionID int64
	UnitPrice     float64
}