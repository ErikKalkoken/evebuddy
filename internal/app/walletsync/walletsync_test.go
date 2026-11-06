package walletsync_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/walletsync"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

const pageSize = 2_500

var logAttr = slog.Int64("ownerID", 42)

func TestSyncJournal(t *testing.T) {
	ctx := context.Background()
	noAddEntities := func(t *testing.T) func(context.Context, set.Set[int64]) error {
		return func(context.Context, set.Set[int64]) error {
			t.Fatal("addEntities should not be called")
			return nil
		}
	}
	noStore := func(t *testing.T) func(context.Context, walletsync.ESIJournalEntry) error {
		return func(context.Context, walletsync.ESIJournalEntry) error {
			t.Fatal("store should not be called")
			return nil
		}
	}
	t.Run("stores only new entries and reports a change", func(t *testing.T) {
		var stored []int64
		changed, err := walletsync.SyncJournal(
			ctx,
			[]walletsync.ESIJournalEntry{{Id: 1}, {Id: 2}, {Id: 3}},
			set.Of[int64](2),
			func(context.Context, set.Set[int64]) error { return nil },
			func(_ context.Context, e walletsync.ESIJournalEntry) error {
				stored = append(stored, e.Id)
				return nil
			},
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, true, changed)
		xassert.Equal(t, []int64{1, 3}, stored)
	})
	t.Run("reports no change when all entries exist", func(t *testing.T) {
		changed, err := walletsync.SyncJournal(
			ctx,
			[]walletsync.ESIJournalEntry{{Id: 1}},
			set.Of[int64](1),
			noAddEntities(t),
			noStore(t),
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, false, changed)
	})
	t.Run("reports no change when there are no entries", func(t *testing.T) {
		changed, err := walletsync.SyncJournal(ctx, nil, set.Of[int64](1), noAddEntities(t), noStore(t), logAttr)
		require.NoError(t, err)
		xassert.Equal(t, false, changed)
	})
	t.Run("adds the entities of new entries only and ignores missing ones", func(t *testing.T) {
		first, second, tax, existing := int64(10), int64(20), int64(30), int64(40)
		var got set.Set[int64]
		_, err := walletsync.SyncJournal(
			ctx,
			[]walletsync.ESIJournalEntry{
				{Id: 1, FirstPartyId: &first, SecondPartyId: &second},
				{Id: 2, TaxReceiverId: &tax},
				{Id: 3},
				{Id: 4, FirstPartyId: &existing},
			},
			set.Of[int64](4),
			func(_ context.Context, ids set.Set[int64]) error {
				got = ids
				return nil
			},
			func(context.Context, walletsync.ESIJournalEntry) error { return nil },
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(first, second, tax), got)
	})
	t.Run("stores nothing when adding entities fails", func(t *testing.T) {
		wantErr := errors.New("add failed")
		changed, err := walletsync.SyncJournal(
			ctx,
			[]walletsync.ESIJournalEntry{{Id: 1}},
			set.Of[int64](),
			func(context.Context, set.Set[int64]) error { return wantErr },
			noStore(t),
			logAttr,
		)
		assert.ErrorIs(t, err, wantErr)
		xassert.Equal(t, false, changed)
	})
	t.Run("returns error when storing fails", func(t *testing.T) {
		wantErr := errors.New("store failed")
		changed, err := walletsync.SyncJournal(
			ctx,
			[]walletsync.ESIJournalEntry{{Id: 1}},
			set.Of[int64](),
			func(context.Context, set.Set[int64]) error { return nil },
			func(context.Context, walletsync.ESIJournalEntry) error { return wantErr },
			logAttr,
		)
		assert.ErrorIs(t, err, wantErr)
		xassert.Equal(t, false, changed)
	})
}

func TestFetchTransactions(t *testing.T) {
	// page returns a full page of transactions with descending IDs starting at first.
	page := func(first int64) []walletsync.ESITransaction {
		oo := make([]walletsync.ESITransaction, pageSize)
		for i := range oo {
			oo[i].TransactionId = first - int64(i)
		}
		return oo
	}
	// fetcher returns the given pages one after the other and records the requested IDs.
	fetcher := func(pages ...[]walletsync.ESITransaction) (func(int64) ([]walletsync.ESITransaction, error), *[]int64) {
		var fromIDs []int64
		return func(fromID int64) ([]walletsync.ESITransaction, error) {
			fromIDs = append(fromIDs, fromID)
			return pages[len(fromIDs)-1], nil
		}, &fromIDs
	}
	short := []walletsync.ESITransaction{{TransactionId: 5}}
	t.Run("stops after a page which is not full", func(t *testing.T) {
		fetch, fromIDs := fetcher(short)
		got, err := walletsync.FetchTransactions(fetch, 0, 0, false)
		require.NoError(t, err)
		xassert.Equal(t, 1, len(got))
		xassert.Equal(t, []int64{0}, *fromIDs)
	})
	t.Run("continues from the lowest ID of the previous page", func(t *testing.T) {
		fetch, fromIDs := fetcher(page(10_000), short)
		got, err := walletsync.FetchTransactions(fetch, 0, 0, false)
		require.NoError(t, err)
		xassert.Equal(t, pageSize+1, len(got))
		xassert.Equal(t, []int64{0, 10_000 - pageSize + 1}, *fromIDs)
	})
	t.Run("stops when another page would exceed the limit", func(t *testing.T) {
		fetch, fromIDs := fetcher(page(10_000), page(5_000))
		got, err := walletsync.FetchTransactions(fetch, 2*pageSize, 0, false)
		require.NoError(t, err)
		xassert.Equal(t, 2*pageSize, len(got))
		xassert.Equal(t, 2, len(*fromIDs))
	})
	t.Run("stops at a page containing the last ID", func(t *testing.T) {
		fetch, fromIDs := fetcher(page(10_000))
		got, err := walletsync.FetchTransactions(fetch, 0, 9_000, true)
		require.NoError(t, err)
		xassert.Equal(t, pageSize, len(got))
		xassert.Equal(t, 1, len(*fromIDs))
	})
	t.Run("ignores the last ID when it should not be checked", func(t *testing.T) {
		fetch, fromIDs := fetcher(page(10_000), short)
		got, err := walletsync.FetchTransactions(fetch, 0, 9_000, false)
		require.NoError(t, err)
		xassert.Equal(t, pageSize+1, len(got))
		xassert.Equal(t, 2, len(*fromIDs))
	})
	t.Run("returns error when fetching fails", func(t *testing.T) {
		wantErr := errors.New("fetch failed")
		_, err := walletsync.FetchTransactions(func(int64) ([]walletsync.ESITransaction, error) {
			return nil, wantErr
		}, 0, 0, false)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestSyncTransactions(t *testing.T) {
	ctx := context.Background()
	noAddRelated := func(t *testing.T) func(context.Context, set.Set[int64], set.Set[int64], set.Set[int64]) error {
		return func(context.Context, set.Set[int64], set.Set[int64], set.Set[int64]) error {
			t.Fatal("addRelated should not be called")
			return nil
		}
	}
	noStore := func(t *testing.T) func(context.Context, walletsync.ESITransaction) error {
		return func(context.Context, walletsync.ESITransaction) error {
			t.Fatal("store should not be called")
			return nil
		}
	}
	t.Run("stores only new transactions and adds their related objects", func(t *testing.T) {
		var entityIDs, locationIDs, typeIDs set.Set[int64]
		var stored []int64
		changed, err := walletsync.SyncTransactions(
			ctx,
			[]walletsync.ESITransaction{
				{TransactionId: 1, ClientId: 10, LocationId: 20, TypeId: 30},
				{TransactionId: 2, ClientId: 11, LocationId: 21, TypeId: 31},
			},
			set.Of[int64](2),
			func(_ context.Context, entities, locations, types set.Set[int64]) error {
				entityIDs, locationIDs, typeIDs = entities, locations, types
				return nil
			},
			func(_ context.Context, x walletsync.ESITransaction) error {
				stored = append(stored, x.TransactionId)
				return nil
			},
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, true, changed)
		xassert.Equal(t, []int64{1}, stored)
		xassert.Equal(t, set.Of[int64](10), entityIDs)
		xassert.Equal(t, set.Of[int64](20), locationIDs)
		xassert.Equal(t, set.Of[int64](30), typeIDs)
	})
	t.Run("returns error when storing fails", func(t *testing.T) {
		wantErr := errors.New("store failed")
		_, err := walletsync.SyncTransactions(
			ctx,
			[]walletsync.ESITransaction{{TransactionId: 1}},
			set.Of[int64](),
			func(context.Context, set.Set[int64], set.Set[int64], set.Set[int64]) error { return nil },
			func(context.Context, walletsync.ESITransaction) error { return wantErr },
			logAttr,
		)
		assert.ErrorIs(t, err, wantErr)
	})
	t.Run("reports no change when all transactions exist", func(t *testing.T) {
		changed, err := walletsync.SyncTransactions(
			ctx,
			[]walletsync.ESITransaction{{TransactionId: 1}},
			set.Of[int64](1),
			noAddRelated(t),
			noStore(t),
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, false, changed)
	})
	t.Run("stores nothing when adding related objects fails", func(t *testing.T) {
		wantErr := errors.New("add failed")
		changed, err := walletsync.SyncTransactions(
			ctx,
			[]walletsync.ESITransaction{{TransactionId: 1}},
			set.Of[int64](),
			func(context.Context, set.Set[int64], set.Set[int64], set.Set[int64]) error { return wantErr },
			noStore(t),
			logAttr,
		)
		assert.ErrorIs(t, err, wantErr)
		xassert.Equal(t, false, changed)
	})
}
