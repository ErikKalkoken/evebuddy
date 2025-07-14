package app

import (
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Division represents a division in an EVE Online corporation.
type Division uint

const (
	Division1 Division = iota + 1
	Division2
	Division3
	Division4
	Division5
	Division6
	Division7
)

// TODO: Move ID resolution into storage layer

func (d Division) ID() int32 {
	return int32(d)
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
	ID             int32
	EveCorporation *EveCorporation
}

type CorporationHangarName struct {
	CorporationID int32
	DivisionID    int32
	Name          string
}

type CorporationWalletBalance struct {
	Balance       float64
	CorporationID int32
	DivisionID    int32
}

type CorporationWalletBalanceWithName struct {
	Balance       float64
	CorporationID int32
	DivisionID    int32
	Name          string
}

type CorporationWalletName struct {
	CorporationID int32
	DivisionID    int32
	Name          string
}

type CorporationWalletJournalEntry struct {
	Amount        float64
	Balance       float64
	CorporationID int32
	ContextID     int64
	ContextIDType string
	Date          time.Time
	Description   string
	DivisionID    int32
	FirstParty    *EveEntity
	ID            int64
	Reason        string
	RefID         int64
	RefType       string
	SecondParty   *EveEntity
	Tax           float64
	TaxReceiver   *EveEntity
}

func (we CorporationWalletJournalEntry) RefTypeDisplay() string {
	titler := cases.Title(language.English)
	return titler.String(strings.ReplaceAll(we.RefType, "_", " "))
}

type CorporationWalletTransaction struct {
	CorporationID int32
	Client        *EveEntity
	Date          time.Time
	DivisionID    int32
	ID            int64
	IsBuy         bool
	JournalRefID  int64
	Location      *EveLocationShort
	Region        *EntityShort[int32]
	Quantity      int32
	TransactionID int64
	Type          *EveType
	UnitPrice     float64
}

func (wt *CorporationWalletTransaction) Total() float64 {
	return wt.UnitPrice * float64(wt.Quantity)
}
