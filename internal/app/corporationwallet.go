package app

import (
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

type CorporationWalletBalance struct {
	Balance       float64
	CorporationID int64
	DivisionID    int64
}

type CorporationWalletBalanceWithName struct {
	Balance       float64
	CorporationID int64
	DivisionID    int64
	Name          string
}

type CorporationWalletName struct {
	CorporationID int64
	DivisionID    int64
	Name          string
}

type CorporationWalletJournalEntry struct {
	Amount        optional.Optional[float64]
	Balance       optional.Optional[float64]
	CorporationID int64
	ContextID     optional.Optional[int64]
	ContextIDType optional.Optional[string]
	Date          time.Time
	Description   string
	DivisionID    int64
	FirstParty    optional.Optional[*EveEntity]
	ID            int64
	Reason        optional.Optional[string]
	RefID         int64
	RefType       string
	SecondParty   optional.Optional[*EveEntity]
	Tax           optional.Optional[float64]
	TaxReceiver   optional.Optional[*EveEntity]
}

func (we CorporationWalletJournalEntry) RefTypeDisplay() string {
	return xstrings.Title(strings.ReplaceAll(we.RefType, "_", " "))
}

type CorporationWalletTransaction struct {
	CorporationID int64
	Client        *EveEntity
	Date          time.Time
	DivisionID    int64 // TODO: Should that be a division type?
	ID            int64
	IsBuy         bool
	JournalRefID  int64
	Location      *EveLocationShort
	Region        *EntityShort
	Quantity      int64
	TransactionID int64
	Type          *EveType
	UnitPrice     float64
}

func (wt *CorporationWalletTransaction) Total() float64 {
	return wt.UnitPrice * float64(wt.Quantity)
}
