// Package contractsync provides the rules for updating stored contracts from ESI,
// which are shared by characters and corporations.
//
// Services convert ESI responses for characters and corporations
// to the ESI types of this package before use.
package contractsync

import (
	"context"
	"fmt"
	"log/slog"
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

// ParseESIContract returns the availability, status and type of an ESI contract.
func ParseESIContract(c ESIContract) (app.ContractAvailability, app.ContractStatus, app.ContractType, error) {
	availability, ok := availabilityFromESIValue[c.Availability]
	if !ok {
		return 0, 0, 0, fmt.Errorf("unknown availability: %s", c.Availability)
	}
	status, err := ParseESIStatus(c.Status)
	if err != nil {
		return 0, 0, 0, err
	}
	typ, ok := typeFromESIValue[c.Type]
	if !ok {
		return 0, 0, 0, fmt.Errorf("unknown type: %s", c.Type)
	}
	return availability, status, typ, nil
}

// ParseESIStatus returns the status for an ESI status value.
func ParseESIStatus(s string) (app.ContractStatus, error) {
	status, ok := statusFromESIValue[s]
	if !ok {
		return 0, fmt.Errorf("unknown status: %s", s)
	}
	return status, nil
}

// RemoveDeleted removes contracts which ESI reports as deleted.
func RemoveDeleted(contracts []ESIContract) []ESIContract {
	return slices.DeleteFunc(contracts, func(x ESIContract) bool {
		return x.Status == "deleted"
	})
}

// ExtractEntityAndLocationIDs returns the IDs of the entities and locations referenced by contracts.
func ExtractEntityAndLocationIDs(contracts []ESIContract) (entityIDs, locationIDs set.Set[int64]) {
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

// existingContract is a contract from ESI and its stored counterpart.
type existingContract struct {
	ESI    ESIContract
	Stored *app.Contract
}

// split separates contracts into those which are not yet stored and those which are.
func split(contracts []ESIContract, current []*app.Contract) (newContracts []ESIContract, existing []existingContract) {
	stored := make(map[int64]*app.Contract, len(current))
	for _, o := range current {
		stored[o.ContractID] = o
	}
	for _, c := range contracts {
		if o, ok := stored[c.ContractId]; ok {
			existing = append(existing, existingContract{ESI: c, Stored: o})
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

// staleIDs returns the IDs of stored contracts which are missing in ESI and should be deleted.
// Finished contracts are kept, since ESI drops them after a while.
func staleIDs(current []*app.Contract, contracts []ESIContract) set.Set[int64] {
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

// Sync creates, updates and adds bids for contracts from ESI.
// It returns the IDs of stored contracts which have become stale
// and reports whether anything has changed.
// contracts must already be filtered and their related entities added.
// Failures for single contracts are logged and skipped, unless ctx is canceled.
//
// The callbacks do the owner specific work and report whether they stored anything.
// create must return the stored ID also when only storing related data failed,
// so bids can still be added.
// Bids are only fetched for auctions which can still have new bids.
func Sync(
	ctx context.Context,
	contracts []ESIContract,
	current []*app.Contract,
	create func(ctx context.Context, c ESIContract) (int64, error),
	update func(ctx context.Context, stored *app.Contract, c ESIContract) (bool, error),
	updateBids func(ctx context.Context, contractPK, contractID int64) (bool, error),
	logAttr slog.Attr,
) (set.Set[int64], bool, error) {
	newContracts, existing := split(contracts, current)
	slog.Debug("contracts", logAttr, "current", len(current), "entries", contracts)
	contractPKs := make(map[int64]int64) // contract ID -> stored ID
	for _, o := range current {
		contractPKs[o.ContractID] = o.ID
	}
	var changedIDs set.Set[int64] // contracts created or updated in this run
	// create new entries
	if len(newContracts) > 0 {
		var count int
		for _, c := range newContracts {
			id, err := create(ctx, c)
			if id != 0 {
				contractPKs[c.ContractId] = id
				changedIDs.Add(c.ContractId)
			}
			if err != nil {
				if ctx.Err() != nil {
					return set.Set[int64]{}, false, err
				}
				slog.Error("create contract", "contract", c, "error", err)
				continue
			}
			count++
		}
		slog.Info("Stored new contracts", logAttr, "count", count)
	}
	// update existing entries
	if len(existing) > 0 {
		var count int
		for _, x := range existing {
			changed, err := update(ctx, x.Stored, x.ESI)
			if err != nil {
				if ctx.Err() != nil {
					return set.Set[int64]{}, false, err
				}
				slog.Error("update contract", "contract", x.ESI, "error", err)
				continue
			}
			if changed {
				changedIDs.Add(x.ESI.ContractId)
				count++
			}
		}
		if count > 0 {
			slog.Info("Updated contracts", logAttr, "count", count)
		}
	}
	// add new bids for auctions
	var hasNewBids bool
	for _, c := range contracts {
		// changed auctions can have bids from before they were finished
		if c.Type != "auction" || (c.Status != "outstanding" && !changedIDs.Contains(c.ContractId)) {
			continue
		}
		pk, ok := contractPKs[c.ContractId]
		if !ok {
			slog.Debug("Skipping bids for contract which could not be stored", logAttr, "contractID", c.ContractId)
			continue
		}
		added, err := updateBids(ctx, pk, c.ContractId)
		if err != nil {
			if ctx.Err() != nil {
				return set.Set[int64]{}, false, err
			}
			slog.Error("update contract bids", "contract", c, "error", err)
			continue
		}
		if added {
			hasNewBids = true
		}
	}
	stale := staleIDs(current, contracts)
	return stale, changedIDs.Size() > 0 || hasNewBids || stale.Size() > 0, nil
}
