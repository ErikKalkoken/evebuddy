// Package contractsync provides the rules for updating stored contracts from ESI,
// which are shared by characters and corporations.
//
// Services convert ESI responses for characters and corporations
// to the ESI types of this package before use.
package contractsync

import (
	"fmt"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

var availabilityFromESIValue = map[string]app.ContractAvailability{
	"alliance":    app.ContractAvailabilityAlliance,
	"corporation": app.ContractAvailabilityCorporation,
	"personal":    app.ContractAvailabilityPrivate,
	"public":      app.ContractAvailabilityPublic,
}

var statusFromESIValue = map[string]app.ContractStatus{
	"cancelled":           app.ContractStatusCancelled,
	"deleted":             app.ContractStatusDeleted,
	"failed":              app.ContractStatusFailed,
	"finished_contractor": app.ContractStatusFinishedContractor,
	"finished_issuer":     app.ContractStatusFinishedIssuer,
	"finished":            app.ContractStatusFinished,
	"in_progress":         app.ContractStatusInProgress,
	"outstanding":         app.ContractStatusOutstanding,
	"rejected":            app.ContractStatusRejected,
	"reversed":            app.ContractStatusReversed,
}

var typeFromESIValue = map[string]app.ContractType{
	"auction":       app.ContractTypeAuction,
	"courier":       app.ContractTypeCourier,
	"item_exchange": app.ContractTypeItemExchange,
	"loan":          app.ContractTypeLoan,
	"unknown":       app.ContractTypeUnknown,
}

// The ESI types below have the same fields as the ESI responses for characters and corporations,
// so services can convert them directly. Field names therefore follow ESI.

// ESIContract is a contract from ESI.
type ESIContract struct {
	AcceptorId          int64
	AssigneeId          int64
	Availability        string
	Buyout              *float64
	Collateral          *float64
	ContractId          int64
	DateAccepted        *time.Time
	DateCompleted       *time.Time
	DateExpired         time.Time
	DateIssued          time.Time
	DaysToComplete      *int64
	EndLocationId       *int64
	ForCorporation      bool
	IssuerCorporationId int64
	IssuerId            int64
	Price               *float64
	Reward              *float64
	StartLocationId     *int64
	Status              string
	Title               *string
	Type                string
	Volume              *float64
}

// ESIContractItem is an item of a contract from ESI.
type ESIContractItem struct {
	IsIncluded  bool
	IsSingleton bool
	Quantity    int64
	RawQuantity *int64
	RecordId    int64
	TypeId      int64
}

// ESIContractBid is a bid on a contract from ESI.
type ESIContractBid struct {
	Amount   float64
	BidId    int64
	BidderId int64
	DateBid  time.Time
}

// ParseESI returns the availability, status and type of an ESI contract.
func ParseESI(c ESIContract) (app.ContractAvailability, app.ContractStatus, app.ContractType, error) {
	availability, ok := availabilityFromESIValue[c.Availability]
	if !ok {
		return 0, 0, 0, fmt.Errorf("unknown availability: %s", c.Availability)
	}
	status, err := ParseStatus(c.Status)
	if err != nil {
		return 0, 0, 0, err
	}
	typ, ok := typeFromESIValue[c.Type]
	if !ok {
		return 0, 0, 0, fmt.Errorf("unknown type: %s", c.Type)
	}
	return availability, status, typ, nil
}

// ParseStatus returns the status for an ESI status value.
func ParseStatus(s string) (app.ContractStatus, error) {
	status, ok := statusFromESIValue[s]
	if !ok {
		return 0, fmt.Errorf("unknown status: %s", s)
	}
	return status, nil
}

// FilterDeleted removes contracts which ESI reports as deleted.
func FilterDeleted(contracts []ESIContract) []ESIContract {
	return slices.DeleteFunc(contracts, func(x ESIContract) bool {
		return x.Status == "deleted"
	})
}

// RelatedIDs returns the IDs of the entities and locations referenced by contracts.
func RelatedIDs(contracts ...ESIContract) (entityIDs, locationIDs set.Set[int64]) {
	for _, c := range contracts {
		entityIDs.Add(c.AcceptorId, c.AssigneeId, c.IssuerId, c.IssuerCorporationId)
		if c.StartLocationId != nil {
			locationIDs.Add(*c.StartLocationId)
		}
		if c.EndLocationId != nil {
			locationIDs.Add(*c.EndLocationId)
		}
	}
	return entityIDs, locationIDs
}

// Split separates contracts into those which are not yet stored and those which are.
func Split(contracts []ESIContract, current []*app.Contract) (newContracts, existing []ESIContract) {
	currentIDs := set.Collect(xiter.MapSlice(current, func(x *app.Contract) int64 {
		return x.ContractID
	}))
	for _, c := range contracts {
		if currentIDs.Contains(c.ContractId) {
			existing = append(existing, c)
		} else {
			newContracts = append(newContracts, c)
		}
	}
	return newContracts, existing
}

// HasChanged reports whether a stored contract differs from ESI in any field which can change.
// status is the parsed status of c.
func HasChanged(o *app.Contract, c ESIContract, status app.ContractStatus) bool {
	acceptorID := optional.Map(o.Acceptor, 0, func(x *app.EveEntity) int64 {
		return x.ID
	})
	return c.AcceptorId != acceptorID ||
		!optional.Equal2(optional.FromPtr(c.DateAccepted), o.DateAccepted) ||
		!optional.Equal2(optional.FromPtr(c.DateCompleted), o.DateCompleted) ||
		o.Status != status
}

// NewBids returns the bids which are not yet stored and the IDs of their bidders.
func NewBids(existing set.Set[int64], bids []ESIContractBid) ([]ESIContractBid, set.Set[int64]) {
	var newBids []ESIContractBid
	var bidderIDs set.Set[int64]
	for _, b := range bids {
		if existing.Contains(b.BidId) {
			continue
		}
		newBids = append(newBids, b)
		if b.BidderId != 0 {
			bidderIDs.Add(b.BidderId)
		}
	}
	return newBids, bidderIDs
}

// StaleIDs returns the IDs of stored contracts which are missing in ESI and should be deleted.
// Finished contracts are kept, since ESI drops them after a while.
func StaleIDs(current []*app.Contract, contracts []ESIContract) set.Set[int64] {
	var unfinishedIDs set.Set[int64]
	for _, o := range current {
		if !o.Status.IsFinished() {
			unfinishedIDs.Add(o.ContractID)
		}
	}
	incomingIDs := set.Collect(xiter.MapSlice(contracts, func(x ESIContract) int64 {
		return x.ContractId
	}))
	return set.Difference(unfinishedIDs, incomingIDs)
}
