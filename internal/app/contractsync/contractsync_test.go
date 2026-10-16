package contractsync_test

import (
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

func TestParseESI(t *testing.T) {
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
			got, _, _, err := contractsync.ParseESI(c)
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
			_, got, _, err := contractsync.ParseESI(c)
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
			_, _, got, err := contractsync.ParseESI(c)
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
			_, _, _, err := contractsync.ParseESI(c)
			assert.Error(t, err)
		}
	})
}

func TestParseStatus(t *testing.T) {
	got, err := contractsync.ParseStatus("in_progress")
	require.NoError(t, err)
	xassert.Equal(t, app.ContractStatusInProgress, got)
	_, err = contractsync.ParseStatus("invalid")
	assert.Error(t, err)
}

func TestFilterDeleted(t *testing.T) {
	got := contractsync.FilterDeleted([]contractsync.ESIContract{
		{ContractId: 1, Status: "outstanding"},
		{ContractId: 2, Status: "deleted"},
		{ContractId: 3, Status: "finished"},
	})
	xassert.Equal(t, []int64{1, 3}, contractIDs(got))
}

func TestRelatedIDs(t *testing.T) {
	entityIDs, locationIDs := contractsync.RelatedIDs(
		contractsync.ESIContract{
			AcceptorId:          1,
			AssigneeId:          2,
			IssuerId:            3,
			IssuerCorporationId: 4,
			StartLocationId:     new(int64(10)),
			EndLocationId:       new(int64(11)),
		},
		contractsync.ESIContract{
			IssuerId:            5,
			IssuerCorporationId: 4,
		},
	)
	xassert.Equal(t, set.Of[int64](0, 1, 2, 3, 4, 5), entityIDs)
	xassert.Equal(t, set.Of[int64](10, 11), locationIDs)
}

func TestSplit(t *testing.T) {
	contracts := []contractsync.ESIContract{
		{ContractId: 1},
		{ContractId: 2},
		{ContractId: 3},
	}
	current := []*app.Contract{{ContractID: 2}, {ContractID: 4}}
	newContracts, existing := contractsync.Split(contracts, current)
	xassert.Equal(t, []int64{1, 3}, contractIDs(newContracts))
	xassert.Equal(t, []int64{2}, contractIDs(existing))
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

func TestStaleIDs(t *testing.T) {
	cases := []struct {
		status    app.ContractStatus
		wantStale bool
	}{
		{app.ContractStatusCancelled, true},
		{app.ContractStatusDeleted, true},
		{app.ContractStatusFailed, true},
		{app.ContractStatusInProgress, true},
		{app.ContractStatusOutstanding, true},
		{app.ContractStatusRejected, true},
		{app.ContractStatusFinished, false},
		{app.ContractStatusFinishedContractor, false},
		{app.ContractStatusFinishedIssuer, false},
		{app.ContractStatusReversed, false},
	}
	for _, tc := range cases {
		t.Run(tc.status.String(), func(t *testing.T) {
			current := []*app.Contract{
				{ContractID: 1, Status: tc.status},
				{ContractID: 2, Status: tc.status}, // still in ESI
			}
			contracts := []contractsync.ESIContract{{ContractId: 2}}
			got := contractsync.StaleIDs(current, contracts)
			xassert.Equal(t, tc.wantStale, got.Contains(1))
			xassert.Equal(t, false, got.Contains(2))
		})
	}
}

func contractIDs(contracts []contractsync.ESIContract) []int64 {
	var ids []int64
	for _, c := range contracts {
		ids = append(ids, c.ContractId)
	}
	return ids
}
