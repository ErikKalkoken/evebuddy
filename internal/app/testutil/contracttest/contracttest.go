// Package contracttest provides contract tests shared by the character and corporation services.
package contracttest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

// Contract holds the contract fields shared by both owners.
type Contract struct {
	ID                int64
	Acceptor          optional.Optional[*app.EveEntity]
	Assignee          optional.Optional[*app.EveEntity]
	Availability      app.ContractAvailability
	Buyout            optional.Optional[float64]
	Collateral        optional.Optional[float64]
	ContractID        int64
	DateAccepted      optional.Optional[time.Time]
	DateCompleted     optional.Optional[time.Time]
	DateExpired       time.Time
	DateIssued        time.Time
	DaysToComplete    optional.Optional[int64]
	EndLocation       optional.Optional[*app.EveLocationShort]
	ForCorporation    bool
	Issuer            *app.EveEntity
	IssuerCorporation *app.EveEntity
	Price             optional.Optional[float64]
	Reward            optional.Optional[float64]
	StartLocation     optional.Optional[*app.EveLocationShort]
	Status            app.ContractStatus
	Title             optional.Optional[string]
	Type              app.ContractType
	UpdatedAt         time.Time
	Volume            optional.Optional[float64]
}

func FromCharacterContract(o *app.CharacterContract) Contract {
	return Contract{
		ID:                o.ID,
		Acceptor:          o.Acceptor,
		Assignee:          o.Assignee,
		Availability:      o.Availability,
		Buyout:            o.Buyout,
		Collateral:        o.Collateral,
		ContractID:        o.ContractID,
		DateAccepted:      o.DateAccepted,
		DateCompleted:     o.DateCompleted,
		DateExpired:       o.DateExpired,
		DateIssued:        o.DateIssued,
		DaysToComplete:    o.DaysToComplete,
		EndLocation:       o.EndLocation,
		ForCorporation:    o.ForCorporation,
		Issuer:            o.Issuer,
		IssuerCorporation: o.IssuerCorporation,
		Price:             o.Price,
		Reward:            o.Reward,
		StartLocation:     o.StartLocation,
		Status:            o.Status,
		Title:             o.Title,
		Type:              o.Type,
		UpdatedAt:         o.UpdatedAt,
		Volume:            o.Volume,
	}
}

func FromCorporationContract(o *app.CorporationContract) Contract {
	return Contract{
		ID:                o.ID,
		Acceptor:          o.Acceptor,
		Assignee:          o.Assignee,
		Availability:      o.Availability,
		Buyout:            o.Buyout,
		Collateral:        o.Collateral,
		ContractID:        o.ContractID,
		DateAccepted:      o.DateAccepted,
		DateCompleted:     o.DateCompleted,
		DateExpired:       o.DateExpired,
		DateIssued:        o.DateIssued,
		DaysToComplete:    o.DaysToComplete,
		EndLocation:       o.EndLocation,
		ForCorporation:    o.ForCorporation,
		Issuer:            o.Issuer,
		IssuerCorporation: o.IssuerCorporation,
		Price:             o.Price,
		Reward:            o.Reward,
		StartLocation:     o.StartLocation,
		Status:            o.Status,
		Title:             o.Title,
		Type:              o.Type,
		UpdatedAt:         o.UpdatedAt,
		Volume:            o.Volume,
	}
}

// Item has the same layout as the owner specific item types, so they can be converted directly.
type Item struct {
	ContractID  int64
	IsIncluded  bool
	IsSingleton bool
	Quantity    int64
	RawQuantity optional.Optional[int64]
	RecordID    int64
	Type        *app.EveType
}

// Bid has the same layout as the owner specific bid types, so they can be converted directly.
type Bid struct {
	ContractID int64
	Amount     float64
	BidID      int64
	Bidder     *app.EveEntity
	DateBid    time.Time
}

// Fixture describes a newly created owner.
type Fixture struct {
	OwnerID             int64
	IssuerID            int64 // issuer to use for contracts from ESI
	IssuerCorporationID int64 // issuer corporation to use for contracts from ESI
}

// CreateParams are the fields of a local contract which tests control.
type CreateParams struct {
	Status app.ContractStatus
	Type   app.ContractType
}

// Owner adapts a contract owner (character or corporation) to the shared tests.
type Owner struct {
	ESIPath         string // first segment of the owner's ESI routes, e.g. "characters"
	Setup           func(t *testing.T) Fixture
	Sync            func(ctx context.Context, ownerID int64) error // forced contracts update from ESI
	CreateContract  func(t *testing.T, ownerID int64, arg CreateParams) Contract
	CreateBid       func(t *testing.T, contractPK, bidID int64, amount float64)
	GetContract     func(ctx context.Context, ownerID, contractID int64) (Contract, error)
	ListContractIDs func(ctx context.Context, ownerID int64) (set.Set[int64], error)
	ListItems       func(ctx context.Context, contractPK int64) ([]Item, error)
	ListBids        func(ctx context.Context, contractPK int64) ([]Bid, error)
}

// RunUpdateContractsESITests runs the contract update tests against an owner.
// httpmock must be activated by the caller.
func RunUpdateContractsESITests(t *testing.T, db *sql.DB, factory testutil.Factory, o Owner) {
	setup := func(t *testing.T) Fixture {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		fx := o.Setup(t)
		// default responses for contract details, which tests can override
		empty := httpmock.NewJsonResponderOrPanic(200, []map[string]any{})
		base := fmt.Sprintf(`^https://esi\.evetech\.net/%s/%d/contracts/\d+/`, o.ESIPath, fx.OwnerID)
		httpmock.RegisterRegexpResponder("GET", regexp.MustCompile(base+"items"), empty)
		httpmock.RegisterRegexpResponder("GET", regexp.MustCompile(base+"bids"), empty)
		return fx
	}
	registerContracts := func(fx Fixture, contracts ...map[string]any) {
		if contracts == nil {
			contracts = []map[string]any{}
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/%s/%d/contracts", o.ESIPath, fx.OwnerID),
			httpmock.NewJsonResponderOrPanic(200, contracts),
		)
	}
	registerDetails := func(fx Fixture, contractID int64, kind string, data []map[string]any) {
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/%s/%d/contracts/%d/%s", o.ESIPath, fx.OwnerID, contractID, kind),
			httpmock.NewJsonResponderOrPanic(200, data),
		)
	}
	ctx := context.Background()

	t.Run("should store all fields of a new contract", func(t *testing.T) {
		// given
		const contractID = 42
		fx := setup(t)
		acceptor := factory.CreateEveEntityCharacter()
		assignee := factory.CreateEveEntityCorporation()
		startLocation := factory.CreateEveLocationStructure()
		endLocation := factory.CreateEveLocationStructure()
		et := factory.CreateEveType()
		registerContracts(fx, esiContract(fx, contractID, map[string]any{
			"acceptor_id":       acceptor.ID,
			"assignee_id":       assignee.ID,
			"availability":      "personal",
			"buyout":            1.5,
			"collateral":        2.5,
			"date_accepted":     "2017-06-06T13:12:32Z",
			"date_completed":    "2017-06-07T13:12:32Z",
			"days_to_complete":  3,
			"end_location_id":   endLocation.ID,
			"for_corporation":   true,
			"price":             4.5,
			"reward":            5.5,
			"start_location_id": startLocation.ID,
			"status":            "in_progress",
			"title":             "My title",
			"volume":            6.5,
		}))
		registerDetails(fx, contractID, "items", []map[string]any{
			{
				"is_included":  false,
				"is_singleton": true,
				"quantity":     1,
				"raw_quantity": -1,
				"record_id":    7,
				"type_id":      et.ID,
			},
			{
				"is_included":  true,
				"is_singleton": false,
				"quantity":     3,
				"record_id":    8,
				"type_id":      et.ID,
			},
		})
		// when
		err := o.Sync(ctx, fx.OwnerID)
		// then
		require.NoError(t, err)
		c, err := o.GetContract(ctx, fx.OwnerID, contractID)
		require.NoError(t, err)
		xassert.EqualOptional(t, acceptor, c.Acceptor)
		xassert.EqualOptional(t, assignee, c.Assignee)
		xassert.Equal(t, app.ContractAvailabilityPrivate, c.Availability)
		xassert.EqualOptional(t, 1.5, c.Buyout)
		xassert.EqualOptional(t, 2.5, c.Collateral)
		xassert.EqualOptional(t, time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC), c.DateAccepted)
		xassert.EqualOptional(t, time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC), c.DateCompleted)
		xassert.Equal(t, time.Date(2017, 6, 13, 13, 12, 32, 0, time.UTC), c.DateExpired)
		xassert.Equal(t, time.Date(2017, 6, 5, 13, 12, 32, 0, time.UTC), c.DateIssued)
		xassert.EqualOptional(t, 3, c.DaysToComplete)
		xassert.Equal(t, endLocation.ID, c.EndLocation.MustValue().ID)
		xassert.Equal(t, true, c.ForCorporation)
		xassert.Equal(t, fx.IssuerID, c.Issuer.ID)
		xassert.Equal(t, fx.IssuerCorporationID, c.IssuerCorporation.ID)
		xassert.EqualOptional(t, 4.5, c.Price)
		xassert.EqualOptional(t, 5.5, c.Reward)
		xassert.Equal(t, startLocation.ID, c.StartLocation.MustValue().ID)
		xassert.Equal(t, app.ContractStatusInProgress, c.Status)
		xassert.EqualOptional(t, "My title", c.Title)
		xassert.Equal(t, app.ContractTypeItemExchange, c.Type)
		xassert.EqualOptional(t, 6.5, c.Volume)
		items, err := o.ListItems(ctx, c.ID)
		require.NoError(t, err)
		slices.SortFunc(items, func(a, b Item) int {
			return int(a.RecordID - b.RecordID)
		})
		want := []Item{
			{
				ContractID:  c.ID,
				IsIncluded:  false,
				IsSingleton: true,
				Quantity:    1,
				RawQuantity: optional.New[int64](-1),
				RecordID:    7,
				Type:        et,
			},
			{
				ContractID:  c.ID,
				IsIncluded:  true,
				IsSingleton: false,
				Quantity:    3,
				RecordID:    8,
				Type:        et,
			},
		}
		xassert.Equal(t, want, items)
	})

	t.Run("should map all ESI values", func(t *testing.T) {
		// given
		fx := setup(t)
		type want struct {
			availability app.ContractAvailability
			status       app.ContractStatus
			typ          app.ContractType
		}
		cases := []struct {
			fields map[string]any
			want   want
		}{
			{map[string]any{"status": "cancelled"}, want{status: app.ContractStatusCancelled}},
			{map[string]any{"status": "failed"}, want{status: app.ContractStatusFailed}},
			{map[string]any{"status": "finished_contractor"}, want{status: app.ContractStatusFinishedContractor}},
			{map[string]any{"status": "finished_issuer"}, want{status: app.ContractStatusFinishedIssuer}},
			{map[string]any{"status": "finished"}, want{status: app.ContractStatusFinished}},
			{map[string]any{"status": "in_progress"}, want{status: app.ContractStatusInProgress}},
			{map[string]any{"status": "outstanding"}, want{status: app.ContractStatusOutstanding}},
			{map[string]any{"status": "rejected"}, want{status: app.ContractStatusRejected}},
			{map[string]any{"status": "reversed"}, want{status: app.ContractStatusReversed}},
			{map[string]any{"type": "auction"}, want{typ: app.ContractTypeAuction}},
			{map[string]any{"type": "courier"}, want{typ: app.ContractTypeCourier}},
			{map[string]any{"type": "item_exchange"}, want{typ: app.ContractTypeItemExchange}},
			{map[string]any{"type": "loan"}, want{typ: app.ContractTypeLoan}},
			{map[string]any{"type": "unknown"}, want{typ: app.ContractTypeUnknown}},
			{map[string]any{"availability": "alliance"}, want{availability: app.ContractAvailabilityAlliance}},
			{map[string]any{"availability": "corporation"}, want{availability: app.ContractAvailabilityCorporation}},
			{map[string]any{"availability": "personal"}, want{availability: app.ContractAvailabilityPrivate}},
			{map[string]any{"availability": "public"}, want{availability: app.ContractAvailabilityPublic}},
		}
		var contracts []map[string]any
		for i, tc := range cases {
			contracts = append(contracts, esiContract(fx, int64(i+1), tc.fields))
		}
		registerContracts(fx, contracts...)
		// when
		err := o.Sync(ctx, fx.OwnerID)
		// then
		require.NoError(t, err)
		for i, tc := range cases {
			t.Run(fmt.Sprint(tc.fields), func(t *testing.T) {
				c, err := o.GetContract(ctx, fx.OwnerID, int64(i+1))
				require.NoError(t, err)
				if tc.want.availability != app.ContractAvailabilityUndefined {
					xassert.Equal(t, tc.want.availability, c.Availability)
				}
				if tc.want.status != app.ContractStatusUndefined {
					xassert.Equal(t, tc.want.status, c.Status)
				}
				if tc.want.typ != app.ContractTypeUndefined {
					xassert.Equal(t, tc.want.typ, c.Type)
				}
			})
		}
	})

	t.Run("should update existing contract when one tracked field changed", func(t *testing.T) {
		cases := []struct {
			name   string
			fields func() map[string]any
			check  func(t *testing.T, c Contract)
		}{
			{
				"acceptor",
				func() map[string]any {
					return map[string]any{"acceptor_id": factory.CreateEveEntityCharacter().ID}
				},
				func(t *testing.T, c Contract) {
					xassert.NotEmpty(t, c.Acceptor)
				},
			},
			{
				"date accepted",
				func() map[string]any {
					return map[string]any{"date_accepted": "2017-06-06T13:12:32Z"}
				},
				func(t *testing.T, c Contract) {
					xassert.EqualOptional(t, time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC), c.DateAccepted)
				},
			},
			{
				"date completed",
				func() map[string]any {
					return map[string]any{"date_completed": "2017-06-07T13:12:32Z"}
				},
				func(t *testing.T, c Contract) {
					xassert.EqualOptional(t, time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC), c.DateCompleted)
				},
			},
			{
				"status",
				func() map[string]any {
					return map[string]any{"status": "in_progress"}
				},
				func(t *testing.T, c Contract) {
					xassert.Equal(t, app.ContractStatusInProgress, c.Status)
				},
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// given
				fx := setup(t)
				local := o.CreateContract(t, fx.OwnerID, CreateParams{
					Status: app.ContractStatusOutstanding,
					Type:   app.ContractTypeCourier,
				})
				fields := tc.fields()
				fields["type"] = "courier"
				registerContracts(fx, esiContract(fx, local.ContractID, fields))
				// when
				err := o.Sync(ctx, fx.OwnerID)
				// then
				require.NoError(t, err)
				c, err := o.GetContract(ctx, fx.OwnerID, local.ContractID)
				require.NoError(t, err)
				tc.check(t, c)
			})
		}
	})

	t.Run("should add only new bids to an existing auction", func(t *testing.T) {
		// given
		fx := setup(t)
		local := o.CreateContract(t, fx.OwnerID, CreateParams{
			Status: app.ContractStatusOutstanding,
			Type:   app.ContractTypeAuction,
		})
		o.CreateBid(t, local.ID, 1, 100)
		bidder := factory.CreateEveEntityCharacter()
		registerContracts(fx, esiContract(fx, local.ContractID, map[string]any{"type": "auction"}))
		registerDetails(fx, local.ContractID, "bids", []map[string]any{
			{"amount": 100, "bid_id": 1, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"},
			{"amount": 200, "bid_id": 2, "bidder_id": bidder.ID, "date_bid": "2017-01-02T10:10:10Z"},
		})
		// when
		err := o.Sync(ctx, fx.OwnerID)
		// then
		require.NoError(t, err)
		bids, err := o.ListBids(ctx, local.ID)
		require.NoError(t, err)
		got := make(map[int64]float64)
		for _, b := range bids {
			got[b.BidID] = b.Amount
		}
		xassert.Equal(t, map[int64]float64{1: 100, 2: 200}, got)
	})

	t.Run("should handle local contracts missing in ESI response", func(t *testing.T) {
		cases := []struct {
			status   app.ContractStatus
			wantKeep bool
		}{
			{app.ContractStatusCancelled, false},
			{app.ContractStatusFailed, false},
			{app.ContractStatusInProgress, false},
			{app.ContractStatusOutstanding, false},
			{app.ContractStatusRejected, false},
			{app.ContractStatusFinished, true},
			{app.ContractStatusFinishedContractor, true},
			{app.ContractStatusFinishedIssuer, true},
			{app.ContractStatusReversed, true},
		}
		for _, tc := range cases {
			t.Run(tc.status.String(), func(t *testing.T) {
				// given
				fx := setup(t)
				local := o.CreateContract(t, fx.OwnerID, CreateParams{
					Status: tc.status,
					Type:   app.ContractTypeCourier,
				})
				registerContracts(fx)
				// when
				err := o.Sync(ctx, fx.OwnerID)
				// then
				require.NoError(t, err)
				ids, err := o.ListContractIDs(ctx, fx.OwnerID)
				require.NoError(t, err)
				xassert.Equal(t, tc.wantKeep, ids.Contains(local.ContractID))
			})
		}
	})

	t.Run("should delete existing contract which ESI reports as deleted", func(t *testing.T) {
		// given
		fx := setup(t)
		local := o.CreateContract(t, fx.OwnerID, CreateParams{
			Status: app.ContractStatusOutstanding,
			Type:   app.ContractTypeCourier,
		})
		registerContracts(fx, esiContract(fx, local.ContractID, map[string]any{
			"status": "deleted",
			"type":   "courier",
		}))
		// when
		err := o.Sync(ctx, fx.OwnerID)
		// then
		require.NoError(t, err)
		ids, err := o.ListContractIDs(ctx, fx.OwnerID)
		require.NoError(t, err)
		xassert.Equal(t, 0, ids.Size())
	})

	t.Run("should skip new contracts with unknown values and store the others", func(t *testing.T) {
		// given
		fx := setup(t)
		registerContracts(fx,
			esiContract(fx, 1, map[string]any{"status": "invalid"}),
			esiContract(fx, 2, map[string]any{"type": "invalid"}),
			esiContract(fx, 3, map[string]any{"availability": "invalid"}),
			esiContract(fx, 4, nil),
		)
		// when
		err := o.Sync(ctx, fx.OwnerID)
		// then
		require.NoError(t, err)
		ids, err := o.ListContractIDs(ctx, fx.OwnerID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](4), ids)
	})

	t.Run("should keep existing contract when ESI reports an unknown status", func(t *testing.T) {
		// given
		fx := setup(t)
		local := o.CreateContract(t, fx.OwnerID, CreateParams{
			Status: app.ContractStatusOutstanding,
			Type:   app.ContractTypeCourier,
		})
		registerContracts(fx, esiContract(fx, local.ContractID, map[string]any{
			"status": "invalid",
			"type":   "courier",
		}))
		// when
		err := o.Sync(ctx, fx.OwnerID)
		// then
		require.NoError(t, err)
		c, err := o.GetContract(ctx, fx.OwnerID, local.ContractID)
		require.NoError(t, err)
		xassert.Equal(t, app.ContractStatusOutstanding, c.Status)
	})

	t.Run("should return error when canceled while creating contracts", func(t *testing.T) {
		// given
		fx := setup(t)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		registerContracts(fx, esiContract(fx, 1, nil), esiContract(fx, 2, nil))
		httpmock.RegisterRegexpResponder(
			"GET",
			regexp.MustCompile(fmt.Sprintf(`^https://esi\.evetech\.net/%s/%d/contracts/\d+/items`, o.ESIPath, fx.OwnerID)),
			func(*http.Request) (*http.Response, error) {
				cancel()
				return nil, errors.New("canceled")
			},
		)
		// when
		err := o.Sync(ctx, fx.OwnerID)
		// then
		require.Error(t, err)
	})

}

// esiContract returns an ESI contract response with defaults for all required fields.
func esiContract(fx Fixture, contractID int64, fields map[string]any) map[string]any {
	m := map[string]any{
		"acceptor_id":           0,
		"assignee_id":           0,
		"availability":          "public",
		"contract_id":           contractID,
		"date_expired":          "2017-06-13T13:12:32Z",
		"date_issued":           "2017-06-05T13:12:32Z",
		"for_corporation":       false,
		"issuer_corporation_id": fx.IssuerCorporationID,
		"issuer_id":             fx.IssuerID,
		"status":                "outstanding",
		"type":                  "item_exchange",
	}
	maps.Copy(m, fields)
	return m
}
