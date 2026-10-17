package contractsync_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/contractsync"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestParseESIContract(t *testing.T) {
	valid := contractsync.ESIContract{
		Availability: "public",
		Status:       "outstanding",
		Type:         "courier",
	}
	t.Run("should map all availability values", func(t *testing.T) {
		cases := map[string]app.ContractAvailability{
			"alliance":    app.ContractAvailabilityAlliance,
			"corporation": app.ContractAvailabilityCorporation,
			"personal":    app.ContractAvailabilityPrivate,
			"public":      app.ContractAvailabilityPublic,
		}
		for v, want := range cases {
			c := valid
			c.Availability = v
			got, _, _, err := contractsync.ParseESIContract(c)
			require.NoError(t, err, v)
			xassert.Equal(t, want, got)
		}
	})
	t.Run("should map all status values", func(t *testing.T) {
		cases := map[string]app.ContractStatus{
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
		for v, want := range cases {
			c := valid
			c.Status = v
			_, got, _, err := contractsync.ParseESIContract(c)
			require.NoError(t, err, v)
			xassert.Equal(t, want, got)
		}
	})
	t.Run("should map all type values", func(t *testing.T) {
		cases := map[string]app.ContractType{
			"auction":       app.ContractTypeAuction,
			"courier":       app.ContractTypeCourier,
			"item_exchange": app.ContractTypeItemExchange,
			"loan":          app.ContractTypeLoan,
			"unknown":       app.ContractTypeUnknown,
		}
		for v, want := range cases {
			c := valid
			c.Type = v
			_, _, got, err := contractsync.ParseESIContract(c)
			require.NoError(t, err, v)
			xassert.Equal(t, want, got)
		}
	})
	t.Run("should return error for unknown values", func(t *testing.T) {
		for _, f := range []func(c *contractsync.ESIContract){
			func(c *contractsync.ESIContract) { c.Availability = "invalid" },
			func(c *contractsync.ESIContract) { c.Status = "invalid" },
			func(c *contractsync.ESIContract) { c.Type = "invalid" },
		} {
			c := valid
			f(&c)
			_, _, _, err := contractsync.ParseESIContract(c)
			assert.Error(t, err)
		}
	})
}

func TestParseESIStatus(t *testing.T) {
	got, err := contractsync.ParseESIStatus("in_progress")
	require.NoError(t, err)
	xassert.Equal(t, app.ContractStatusInProgress, got)
	_, err = contractsync.ParseESIStatus("invalid")
	assert.Error(t, err)
}

func TestRemoveDeleted(t *testing.T) {
	got := contractsync.RemoveDeleted([]contractsync.ESIContract{
		{ContractId: 1, Status: "outstanding"},
		{ContractId: 2, Status: "deleted"},
		{ContractId: 3, Status: "finished"},
	})
	xassert.Equal(t, []int64{1, 3}, contractIDs(got))
}

func TestExtractEntityAndLocationIDs(t *testing.T) {
	entityIDs, locationIDs := contractsync.ExtractEntityAndLocationIDs([]contractsync.ESIContract{
		{
			AcceptorId:          1,
			AssigneeId:          2,
			IssuerId:            3,
			IssuerCorporationId: 4,
			StartLocationId:     new(int64(10)),
			EndLocationId:       new(int64(11)),
		},
		{
			IssuerId:            5,
			IssuerCorporationId: 4,
		},
	})
	xassert.Equal(t, set.Of[int64](0, 1, 2, 3, 4, 5), entityIDs)
	xassert.Equal(t, set.Of[int64](10, 11), locationIDs)
}

func TestHasChanged(t *testing.T) {
	accepted := time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC)
	completed := time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC)
	stored := &app.Contract{
		Acceptor:      optional.New(&app.EveEntity{ID: 42}),
		DateAccepted:  optional.New(accepted),
		DateCompleted: optional.New(completed),
		Status:        app.ContractStatusInProgress,
	}
	unchanged := contractsync.ESIContract{
		AcceptorId:    42,
		DateAccepted:  &accepted,
		DateCompleted: &completed,
	}
	other := accepted.Add(time.Hour)
	cases := []struct {
		name   string
		change func(c *contractsync.ESIContract)
		status app.ContractStatus
		want   bool
	}{
		{"unchanged", func(c *contractsync.ESIContract) {}, app.ContractStatusInProgress, false},
		{"acceptor", func(c *contractsync.ESIContract) { c.AcceptorId = 7 }, app.ContractStatusInProgress, true},
		{"acceptor removed", func(c *contractsync.ESIContract) { c.AcceptorId = 0 }, app.ContractStatusInProgress, true},
		{"date accepted", func(c *contractsync.ESIContract) { c.DateAccepted = &other }, app.ContractStatusInProgress, true},
		{"date accepted removed", func(c *contractsync.ESIContract) { c.DateAccepted = nil }, app.ContractStatusInProgress, true},
		{"date completed", func(c *contractsync.ESIContract) { c.DateCompleted = &other }, app.ContractStatusInProgress, true},
		{"status", func(c *contractsync.ESIContract) {}, app.ContractStatusFinished, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := unchanged
			tc.change(&c)
			xassert.Equal(t, tc.want, contractsync.HasChanged(stored, c, tc.status))
		})
	}
	t.Run("stored contract without acceptor is unchanged when ESI has none", func(t *testing.T) {
		o := &app.Contract{Status: app.ContractStatusOutstanding}
		c := contractsync.ESIContract{}
		xassert.Equal(t, false, contractsync.HasChanged(o, c, app.ContractStatusOutstanding))
	})
}

func TestNewBids(t *testing.T) {
	bids := []contractsync.ESIContractBid{
		{BidId: 1, BidderId: 10},
		{BidId: 2, BidderId: 20},
		{BidId: 3, BidderId: 0},
		{BidId: 4, BidderId: 20},
	}
	got, bidderIDs := contractsync.NewBids(set.Of[int64](1), bids)
	var ids []int64
	for _, b := range got {
		ids = append(ids, b.BidId)
	}
	xassert.Equal(t, []int64{2, 3, 4}, ids)
	xassert.Equal(t, set.Of[int64](20), bidderIDs)
}

func contractIDs(contracts []contractsync.ESIContract) []int64 {
	var ids []int64
	for _, c := range contracts {
		ids = append(ids, c.ContractId)
	}
	return ids
}

func TestSync(t *testing.T) {
	logAttr := slog.Int64("characterID", 1)
	t.Run("should create new and update existing contracts", func(t *testing.T) {
		var created []int64
		updated := make(map[int64]int64) // contract ID -> stored ID
		current := []*app.Contract{
			{ID: 12, ContractID: 2, Status: app.ContractStatusOutstanding},
			{ID: 13, ContractID: 3, Status: app.ContractStatusOutstanding},
		}
		contracts := []contractsync.ESIContract{{ContractId: 1}, {ContractId: 2}, {ContractId: 3}}
		staleIDs, _, err := contractsync.Sync(t.Context(), contracts, current,
			func(ctx context.Context, c contractsync.ESIContract) (int64, error) {
				created = append(created, c.ContractId)
				return 100 + c.ContractId, nil
			},
			func(ctx context.Context, o *app.Contract, c contractsync.ESIContract) (bool, error) {
				updated[c.ContractId] = o.ID
				return true, nil
			},
			func(ctx context.Context, contractPK, contractID int64) (bool, error) {
				return false, nil
			},
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, []int64{1}, created)
		xassert.Equal(t, map[int64]int64{2: 12, 3: 13}, updated)
		xassert.Equal(t, 0, staleIDs.Size())
	})
	t.Run("should add bids for new and existing auctions with their stored ID", func(t *testing.T) {
		bids := make(map[int64]int64) // contract ID -> stored ID
		current := []*app.Contract{{ID: 12, ContractID: 2, Status: app.ContractStatusOutstanding}}
		contracts := []contractsync.ESIContract{
			{ContractId: 1, Type: "auction", Status: "outstanding"},
			{ContractId: 2, Type: "auction", Status: "outstanding"},
			{ContractId: 3, Type: "courier", Status: "outstanding"},
		}
		_, _, err := contractsync.Sync(t.Context(), contracts, current,
			func(ctx context.Context, c contractsync.ESIContract) (int64, error) {
				return 100 + c.ContractId, nil
			},
			func(ctx context.Context, o *app.Contract, c contractsync.ESIContract) (bool, error) {
				return false, nil
			},
			func(ctx context.Context, contractPK, contractID int64) (bool, error) {
				bids[contractID] = contractPK
				return false, nil
			},
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, map[int64]int64{1: 101, 2: 12}, bids)
	})
	t.Run("should fetch bids only for auctions which can have new bids", func(t *testing.T) {
		var bidContractIDs []int64
		current := []*app.Contract{
			{ID: 12, ContractID: 2, Status: app.ContractStatusFinished},
			{ID: 13, ContractID: 3, Status: app.ContractStatusOutstanding},
			{ID: 14, ContractID: 4, Status: app.ContractStatusOutstanding},
		}
		contracts := []contractsync.ESIContract{
			{ContractId: 1, Type: "auction", Status: "finished"},    // new
			{ContractId: 2, Type: "auction", Status: "finished"},    // unchanged
			{ContractId: 3, Type: "auction", Status: "finished"},    // has just finished
			{ContractId: 4, Type: "auction", Status: "outstanding"}, // unchanged
		}
		_, _, err := contractsync.Sync(t.Context(), contracts, current,
			func(ctx context.Context, c contractsync.ESIContract) (int64, error) {
				return 100 + c.ContractId, nil
			},
			func(ctx context.Context, o *app.Contract, c contractsync.ESIContract) (bool, error) {
				return c.ContractId == 3, nil
			},
			func(ctx context.Context, contractPK, contractID int64) (bool, error) {
				bidContractIDs = append(bidContractIDs, contractID)
				return false, nil
			},
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, []int64{1, 3, 4}, bidContractIDs)
	})
	t.Run("should report whether anything has changed", func(t *testing.T) {
		stored := &app.Contract{ID: 11, ContractID: 1, Status: app.ContractStatusOutstanding}
		auction := contractsync.ESIContract{ContractId: 1, Type: "auction", Status: "outstanding"}
		cases := []struct {
			name      string
			current   []*app.Contract
			contracts []contractsync.ESIContract
			updated   bool // update callback stores a change
			newBids   bool // bids callback stores new bids
			want      bool
		}{
			{"nothing", []*app.Contract{stored}, []contractsync.ESIContract{auction}, false, false, false},
			{"created", nil, []contractsync.ESIContract{auction}, false, false, true},
			{"updated", []*app.Contract{stored}, []contractsync.ESIContract{auction}, true, false, true},
			{"new bids", []*app.Contract{stored}, []contractsync.ESIContract{auction}, false, true, true},
			{"stale", []*app.Contract{stored}, nil, false, false, true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, changed, err := contractsync.Sync(t.Context(), tc.contracts, tc.current,
					func(ctx context.Context, c contractsync.ESIContract) (int64, error) {
						return 101, nil
					},
					func(ctx context.Context, o *app.Contract, c contractsync.ESIContract) (bool, error) {
						return tc.updated, nil
					},
					func(ctx context.Context, contractPK, contractID int64) (bool, error) {
						return tc.newBids, nil
					},
					logAttr,
				)
				require.NoError(t, err)
				xassert.Equal(t, tc.want, changed)
			})
		}
	})
	t.Run("should add bids for new auction when only storing related data failed", func(t *testing.T) {
		bids := make(map[int64]int64) // contract ID -> stored ID
		contracts := []contractsync.ESIContract{{ContractId: 1, Type: "auction", Status: "outstanding"}}
		_, _, err := contractsync.Sync(t.Context(), contracts, nil,
			func(ctx context.Context, c contractsync.ESIContract) (int64, error) {
				return 101, errors.New("items failed") // contract row is stored
			},
			func(ctx context.Context, o *app.Contract, c contractsync.ESIContract) (bool, error) {
				return false, nil
			},
			func(ctx context.Context, contractPK, contractID int64) (bool, error) {
				bids[contractID] = contractPK
				return false, nil
			},
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, map[int64]int64{1: 101}, bids)
	})
	t.Run("should skip bids for auction which could not be stored", func(t *testing.T) {
		var bidContractIDs []int64
		contracts := []contractsync.ESIContract{{ContractId: 1, Type: "auction", Status: "outstanding"}}
		_, _, err := contractsync.Sync(t.Context(), contracts, nil,
			func(ctx context.Context, c contractsync.ESIContract) (int64, error) {
				return 0, errors.New("create failed")
			},
			func(ctx context.Context, o *app.Contract, c contractsync.ESIContract) (bool, error) {
				return false, nil
			},
			func(ctx context.Context, contractPK, contractID int64) (bool, error) {
				bidContractIDs = append(bidContractIDs, contractID)
				return false, nil
			},
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, 0, len(bidContractIDs))
	})
	t.Run("should continue after failures of single contracts", func(t *testing.T) {
		var created []int64
		updated := make(map[int64]int64) // contract ID -> stored ID
		bids := make(map[int64]int64)    // contract ID -> stored ID
		current := []*app.Contract{
			{ID: 13, ContractID: 3, Status: app.ContractStatusOutstanding},
			{ID: 14, ContractID: 4, Status: app.ContractStatusOutstanding},
		}
		contracts := []contractsync.ESIContract{
			{ContractId: 1},
			{ContractId: 2},
			{ContractId: 3},
			{ContractId: 4},
			{ContractId: 5, Type: "auction", Status: "outstanding"},
			{ContractId: 6, Type: "auction", Status: "outstanding"},
		}
		_, _, err := contractsync.Sync(t.Context(), contracts, current,
			func(ctx context.Context, c contractsync.ESIContract) (int64, error) {
				if c.ContractId == 1 {
					return 0, errors.New("create failed")
				}
				created = append(created, c.ContractId)
				return 100 + c.ContractId, nil
			},
			func(ctx context.Context, o *app.Contract, c contractsync.ESIContract) (bool, error) {
				if c.ContractId == 3 {
					return false, errors.New("update failed")
				}
				updated[c.ContractId] = o.ID
				return true, nil
			},
			func(ctx context.Context, contractPK, contractID int64) (bool, error) {
				if contractID == 5 {
					return false, errors.New("bids failed")
				}
				bids[contractID] = contractPK
				return false, nil
			},
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, []int64{2, 5, 6}, created)
		xassert.Equal(t, map[int64]int64{4: 14}, updated)
		xassert.Equal(t, map[int64]int64{6: 106}, bids)
	})
	t.Run("should return error and stop when canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		var createCalls int
		contracts := []contractsync.ESIContract{{ContractId: 1}, {ContractId: 2}}
		_, _, err := contractsync.Sync(ctx, contracts, nil,
			func(ctx context.Context, c contractsync.ESIContract) (int64, error) {
				createCalls++
				cancel()
				return 0, ctx.Err()
			},
			func(ctx context.Context, o *app.Contract, c contractsync.ESIContract) (bool, error) {
				return false, nil
			},
			func(ctx context.Context, contractPK, contractID int64) (bool, error) {
				return false, nil
			},
			logAttr,
		)
		require.Error(t, err)
		xassert.Equal(t, 1, createCalls)
	})
	t.Run("should return stale IDs", func(t *testing.T) {
		current := []*app.Contract{
			{ID: 11, ContractID: 1, Status: app.ContractStatusOutstanding},
			{ID: 12, ContractID: 2, Status: app.ContractStatusFinished},
			{ID: 13, ContractID: 3, Status: app.ContractStatusOutstanding},
		}
		contracts := []contractsync.ESIContract{{ContractId: 3}}
		staleIDs, _, err := contractsync.Sync(t.Context(), contracts, current,
			func(ctx context.Context, c contractsync.ESIContract) (int64, error) {
				return 100 + c.ContractId, nil
			},
			func(ctx context.Context, o *app.Contract, c contractsync.ESIContract) (bool, error) {
				return false, nil
			},
			func(ctx context.Context, contractPK, contractID int64) (bool, error) {
				return false, nil
			},
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](1), staleIDs)
	})
}
