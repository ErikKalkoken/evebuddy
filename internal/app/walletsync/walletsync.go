// Package walletsync provides the wallet sync rules shared by characters and corporations.
package walletsync

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

const maxTransactionsPerPage = 2_500 // maximum objects returned per page

// ESIJournalEntry is a wallet journal entry from ESI.
// Its fields must match both ESI responses to allow direct conversion.
type ESIJournalEntry struct {
	Amount        *float64
	Balance       *float64
	ContextId     *int64
	ContextIdType *string
	Date          time.Time
	Description   string
	FirstPartyId  *int64
	Id            int64
	Reason        *string
	RefType       string
	SecondPartyId *int64
	Tax           *float64
	TaxReceiverId *int64
}

// ESITransaction is a wallet transaction from ESI.
// Its fields must match the character response to allow direct conversion.
// The corporation response has no IsPersonal and must be mapped.
type ESITransaction struct {
	ClientId      int64
	Date          time.Time
	IsBuy         bool
	IsPersonal    bool
	JournalRefId  int64
	LocationId    int64
	Quantity      int64
	TransactionId int64
	TypeId        int64
	UnitPrice     float64
}

// SyncJournal stores new entries and reports whether any were stored.
func SyncJournal(
	ctx context.Context,
	entries []ESIJournalEntry,
	existingIDs set.Set[int64],
	addEntities func(ctx context.Context, ids set.Set[int64]) error,
	store func(ctx context.Context, entries []ESIJournalEntry) error,
	logAttrs ...slog.Attr,
) (bool, error) {
	return syncNew(
		ctx,
		entries,
		func(e ESIJournalEntry) int64 { return e.Id },
		existingIDs,
		func(ctx context.Context, newEntries []ESIJournalEntry) error {
			var ids set.Set[int64]
			for _, e := range newEntries {
				for _, x := range []*int64{e.FirstPartyId, e.SecondPartyId, e.TaxReceiverId} {
					if x != nil {
						ids.Add(*x)
					}
				}
			}
			return addEntities(ctx, ids)
		},
		store,
		"wallet journal entries",
		logAttrs,
	)
}

// FetchTransactions fetches transactions page by page, starting with the newest (fromID 0).
// It stops at the last page or when checkLastID is set and a page contains lastID or an older ID.
func FetchTransactions(
	fetch func(fromID int64) ([]ESITransaction, error),
	lastID int64,
	checkLastID bool,
) ([]ESITransaction, error) {
	var transactions []ESITransaction
	var fromID int64
	for {
		page, err := fetch(fromID)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, page...)
		if len(page) < maxTransactionsPerPage {
			return transactions, nil
		}
		ids := xslices.Map(page, func(t ESITransaction) int64 {
			return t.TransactionId
		})
		if checkLastID && slices.ContainsFunc(ids, func(id int64) bool { return id <= lastID }) {
			return transactions, nil // stop reading once a known ID is found
		}
		fromID = slices.Min(ids)
	}
}

// SyncTransactions stores new transactions and reports whether any were stored.
func SyncTransactions(
	ctx context.Context,
	transactions []ESITransaction,
	existingIDs set.Set[int64],
	addRelated func(ctx context.Context, entityIDs, locationIDs, typeIDs set.Set[int64]) error,
	store func(ctx context.Context, transactions []ESITransaction) error,
	logAttrs ...slog.Attr,
) (bool, error) {
	return syncNew(
		ctx,
		transactions,
		func(t ESITransaction) int64 { return t.TransactionId },
		existingIDs,
		func(ctx context.Context, newTransactions []ESITransaction) error {
			var entityIDs, locationIDs, typeIDs set.Set[int64]
			for _, t := range newTransactions {
				entityIDs.Add(t.ClientId)
				locationIDs.Add(t.LocationId)
				typeIDs.Add(t.TypeId)
			}
			return addRelated(ctx, entityIDs, locationIDs, typeIDs)
		},
		store,
		"wallet transactions",
		logAttrs,
	)
}

// syncNew stores new items and reports whether any were stored.
func syncNew[T any](
	ctx context.Context,
	items []T,
	id func(T) int64,
	existingIDs set.Set[int64],
	addRelated func(ctx context.Context, newItems []T) error,
	store func(ctx context.Context, items []T) error,
	name string,
	logAttrs []slog.Attr,
) (bool, error) {
	// Items can repeat, because pages can shift while they are being fetched.
	var seen set.Set[int64]
	newItems := xslices.Filter(items, func(x T) bool {
		i := id(x)
		if existingIDs.Contains(i) || seen.Contains(i) {
			return false
		}
		seen.Add(i)
		return true
	})
	if len(newItems) == 0 {
		slog.LogAttrs(ctx, slog.LevelInfo, "No new "+name, logAttrs...)
		return false, nil
	}
	if err := addRelated(ctx, newItems); err != nil {
		return false, err
	}
	if err := store(ctx, newItems); err != nil {
		return false, err
	}
	attrs := slices.Concat(logAttrs, []slog.Attr{slog.Int("entries", len(newItems))})
	slog.LogAttrs(ctx, slog.LevelInfo, "Stored new "+name, attrs...)
	return true, nil
}
