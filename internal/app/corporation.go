package app

import (
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

// Division represents a division in an EVE Online corporation.
type Division uint

const (
	DivisionZero Division = iota
	Division1
	Division2
	Division3
	Division4
	Division5
	Division6
	Division7
)

// TODO: Move ID resolution into storage layer

func (d Division) ID() int64 {
	return int64(d)
}

func (d Division) DefaultWalletName() string {
	m := map[Division]string{
		Division1: "Master Wallet",
		Division2: "2nd Wallet",
		Division3: "3rd Wallet",
		Division4: "4th Wallet",
		Division5: "5th Wallet",
		Division6: "6th Wallet",
		Division7: "7th Wallet",
	}
	return m[d]
}

var Divisions = []Division{
	Division1,
	Division2,
	Division3,
	Division4,
	Division5,
	Division6,
	Division7,
}

type Corporation struct {
	ID             int64
	EveCorporation *EveCorporation
}

type CorporationHangarName struct {
	CorporationID int64
	DivisionID    int64
	Name          string
}

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
	DivisionID    int64
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

// CorporationMember represents a member in an EVE Online corporation.
type CorporationMember struct {
	CorporationID int64
	Character     *EveEntity
}
