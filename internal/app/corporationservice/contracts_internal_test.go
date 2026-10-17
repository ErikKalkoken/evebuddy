package corporationservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestGetContract(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can return existing contract", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		o := factory.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		got, err := s.GetContract(ctx, c.ID, o.ContractID)
		require.NoError(t, err)
		xassert.Equal(t, o.ID, got.ID)
	})
	t.Run("should return error when contract not found", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		_, err := s.GetContract(ctx, c.ID, 42)
		assert.Error(t, err)
	})
}

func TestGetContractBidSummary(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	t.Run("should return count and top bid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o := factory.CreateCorporationContract()
		factory.CreateCorporationContractBid(storage.CreateContractBidParams{ContractID: o.ID, Amount: 100})
		top := factory.CreateCorporationContractBid(storage.CreateContractBidParams{ContractID: o.ID, Amount: 500})
		factory.CreateCorporationContractBid(storage.CreateContractBidParams{ContractID: o.ID, Amount: 200})
		// when
		count, got, err := s.GetContractBidSummary(t.Context(), o.ID)
		// then
		require.NoError(t, err)
		assert.Equal(t, 3, count)
		xassert.Equal(t, top.BidID, got.MustValue().BidID)
	})
	t.Run("should return no top bid when contract has no bids", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o := factory.CreateCorporationContract()
		// when
		count, got, err := s.GetContractBidSummary(t.Context(), o.ID)
		// then
		require.NoError(t, err)
		assert.Equal(t, 0, count)
		xassert.Empty(t, got)
	})
}

func TestListCorporationContracts(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can list contracts for a corporation", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		o1 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		o2 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		factory.CreateCorporationContract()
		got, err := s.ListCorporationContracts(ctx, c.ID)
		require.NoError(t, err)
		ids := set.Collect(xiter.MapSlice(got, func(x *app.CorporationContract) int64 { return x.ID }))
		xassert.Equal(t, set.Of(o1.ID, o2.ID), ids)
	})
}

func TestListActiveCorporationContracts(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	t.Run("should count outstanding and in progress contracts of a corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		for _, status := range []app.ContractStatus{
			app.ContractStatusOutstanding,
			app.ContractStatusInProgress,
			app.ContractStatusFinished,
			app.ContractStatusFailed,
			app.ContractStatusCancelled,
		} {
			factory.CreateCorporationContract(storage.CreateCorporationContractParams{
				CorporationID: c.ID,
				Status:        status,
			})
		}
		factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			Status: app.ContractStatusOutstanding,
		})
		// when
		got, err := s.ListActiveCorporationContracts(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 2, got)
	})
}

func TestCalculateContractsEscrow(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	t.Run("should sum courier collateral and winning auction bids", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		factory.CreateCorporationContractCourier(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			AcceptorID:    c.ID,
			Collateral:    optional.New(100.0),
			Status:        app.ContractStatusInProgress,
		})
		o := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			Type:          app.ContractTypeAuction,
			Status:        app.ContractStatusOutstanding,
		})
		factory.CreateCorporationContractBid(storage.CreateContractBidParams{
			Amount:     12.5,
			BidderID:   c.ID,
			ContractID: o.ID,
		})
		// when
		got, err := s.CalculateContractsEscrow(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 112.5, got)
	})
	t.Run("should return zero when corporation has no contracts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		// when
		got, err := s.CalculateContractsEscrow(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0.0, got)
	})
}

func TestListContractItems(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can list items for a contract", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		o := factory.CreateCorporationContract()
		i1 := factory.CreateCorporationContractItem(storage.CreateContractItemParams{ContractID: o.ID})
		i2 := factory.CreateCorporationContractItem(storage.CreateContractItemParams{ContractID: o.ID})
		got, err := s.ListContractItems(ctx, o.ID)
		require.NoError(t, err)
		ids := set.Collect(xiter.MapSlice(got, func(x *app.ContractItem) int64 { return x.RecordID }))
		xassert.Equal(t, set.Of(i1.RecordID, i2.RecordID), ids)
	})
}

func TestUpdateContractsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
		AccessToken: "accessToken",
	}}})

	t.Run("should store all fields of a new contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		acceptor := factory.CreateEveEntityCharacter()
		assignee := factory.CreateEveEntityCorporation()
		startLocation := factory.CreateEveLocationStructure()
		endLocation := factory.CreateEveLocationStructure()
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           acceptor.ID,
				"assignee_id":           assignee.ID,
				"availability":          "personal",
				"buyout":                1.5,
				"collateral":            2.5,
				"contract_id":           42,
				"date_accepted":         "2017-06-06T13:12:32Z",
				"date_completed":        "2017-06-07T13:12:32Z",
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"days_to_complete":      3,
				"end_location_id":       endLocation.ID,
				"for_corporation":       true,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"price":                 4.5,
				"reward":                5.5,
				"start_location_id":     startLocation.ID,
				"status":                "in_progress",
				"title":                 "My title",
				"type":                  "item_exchange",
				"volume":                6.5,
			}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/42/items", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"is_included": false, "is_singleton": true, "quantity": 1, "raw_quantity": -1, "record_id": 7, "type_id": et.ID},
				{"is_included": true, "is_singleton": false, "quantity": 3, "record_id": 8, "type_id": et.ID},
			}),
		)
		// when
		_, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			forceUpdate:   true,
			section:       app.SectionCorporationContracts,
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCorporationContract(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.EqualOptional(t, acceptor, o.Acceptor)
		xassert.EqualOptional(t, assignee, o.Assignee)
		xassert.Equal(t, app.ContractAvailabilityPrivate, o.Availability)
		xassert.EqualOptional(t, 1.5, o.Buyout)
		xassert.EqualOptional(t, 2.5, o.Collateral)
		xassert.EqualOptional(t, time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC), o.DateAccepted)
		xassert.EqualOptional(t, time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC), o.DateCompleted)
		xassert.Equal(t, time.Date(2017, 6, 13, 13, 12, 32, 0, time.UTC), o.DateExpired)
		xassert.Equal(t, time.Date(2017, 6, 5, 13, 12, 32, 0, time.UTC), o.DateIssued)
		xassert.EqualOptional(t, 3, o.DaysToComplete)
		xassert.Equal(t, endLocation.ID, o.EndLocation.MustValue().ID)
		xassert.Equal(t, true, o.ForCorporation)
		xassert.Equal(t, c.ID, o.Issuer.ID)
		xassert.Equal(t, c.ID, o.IssuerCorporation.ID)
		xassert.EqualOptional(t, 4.5, o.Price)
		xassert.EqualOptional(t, 5.5, o.Reward)
		xassert.Equal(t, startLocation.ID, o.StartLocation.MustValue().ID)
		xassert.Equal(t, app.ContractStatusInProgress, o.Status)
		xassert.EqualOptional(t, "My title", o.Title)
		xassert.Equal(t, app.ContractTypeItemExchange, o.Type)
		xassert.EqualOptional(t, 6.5, o.Volume)
		items, err := st.ListCorporationContractItems(ctx, o.ID)
		require.NoError(t, err)
		slices.SortFunc(items, func(a, b *app.ContractItem) int {
			return cmp.Compare(a.RecordID, b.RecordID)
		})
		want := []*app.ContractItem{
			{ContractID: o.ID, IsIncluded: false, IsSingleton: true, Quantity: 1, RawQuantity: optional.New[int64](-1), RecordID: 7, Type: et},
			{ContractID: o.ID, IsIncluded: true, IsSingleton: false, Quantity: 3, RecordID: 8, Type: et},
		}
		xassert.Equal(t, want, items)
	})

	t.Run("should store bids of a new auction", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		bidder := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           42,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"status":                "outstanding",
				"type":                  "auction",
			}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/42/items", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/42/bids", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"amount": 123.5, "bid_id": 7, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"},
			}),
		)
		// when
		_, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			forceUpdate:   true,
			section:       app.SectionCorporationContracts,
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCorporationContract(ctx, c.ID, 42)
		require.NoError(t, err)
		bids, err := st.ListCorporationContractBids(ctx, o.ID)
		require.NoError(t, err)
		want := []*app.ContractBid{{
			ContractID: o.ID,
			Amount:     123.5,
			BidID:      7,
			Bidder:     bidder,
			DateBid:    time.Date(2017, 1, 1, 10, 10, 10, 0, time.UTC),
		}}
		xassert.Equal(t, want, bids)
	})

	t.Run("should add only new bids to an existing auction", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		local := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			Status:        app.ContractStatusOutstanding,
			Type:          app.ContractTypeAuction,
		})
		factory.CreateCorporationContractBid(storage.CreateContractBidParams{ContractID: local.ID, BidID: 1, Amount: 100})
		bidder := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           local.ContractID,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"status":                "outstanding",
				"type":                  "auction",
			}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/%d/bids", c.ID, local.ContractID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"amount": 100, "bid_id": 1, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"},
				{"amount": 200, "bid_id": 2, "bidder_id": bidder.ID, "date_bid": "2017-01-02T10:10:10Z"},
			}),
		)
		// when
		_, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			forceUpdate:   true,
			section:       app.SectionCorporationContracts,
		})
		// then
		require.NoError(t, err)
		ids, err := st.ListCorporationContractBidIDs(ctx, local.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](1, 2), ids)
	})

	t.Run("should store new bids when the contract list is unchanged", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		bidder := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           42,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"status":                "outstanding",
				"type":                  "auction",
			}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/42/items", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
		)
		bidsURL := fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/42/bids", c.ID)
		bid1 := map[string]any{"amount": 100, "bid_id": 1, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"}
		bid2 := map[string]any{"amount": 200, "bid_id": 2, "bidder_id": bidder.ID, "date_bid": "2017-01-02T10:10:10Z"}
		httpmock.RegisterResponder("GET", bidsURL, httpmock.NewJsonResponderOrPanic(200, []map[string]any{bid1}))
		arg := corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationContracts,
		}
		changed, err := s.updateContractsESI(ctx, arg)
		require.NoError(t, err)
		xassert.Equal(t, true, changed)
		// when
		httpmock.RegisterResponder("GET", bidsURL, httpmock.NewJsonResponderOrPanic(200, []map[string]any{bid1, bid2}))
		changed, err = s.updateContractsESI(ctx, arg)
		// then
		require.NoError(t, err)
		xassert.Equal(t, true, changed)
		o, err := st.GetCorporationContract(ctx, c.ID, 42)
		require.NoError(t, err)
		ids, err := st.ListCorporationContractBidIDs(ctx, o.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](1, 2), ids)
		// when nothing is new
		changed, err = s.updateContractsESI(ctx, arg)
		// then
		require.NoError(t, err)
		xassert.Equal(t, false, changed)
	})

	t.Run("should store a new contract only after its items could be fetched", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		et := factory.CreateEveType()
		bidder := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           42,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"status":                "outstanding",
				"type":                  "auction",
			}}),
		)
		itemsURL := fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/42/items", c.ID)
		httpmock.RegisterResponder("GET", itemsURL, httpmock.NewErrorResponder(errors.New("items failed")))
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/42/bids", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"amount": 123.5, "bid_id": 7, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"},
			}),
		)
		arg := corporationSectionUpdateParams{
			corporationID: c.ID,
			forceUpdate:   true,
			section:       app.SectionCorporationContracts,
		}
		// when
		_, err := s.updateContractsESI(ctx, arg)
		// then
		require.NoError(t, err)
		_, err = st.GetCorporationContract(ctx, c.ID, 42)
		require.ErrorIs(t, err, app.ErrNotFound)
		// when ESI works again
		httpmock.RegisterResponder("GET", itemsURL, httpmock.NewJsonResponderOrPanic(200, []map[string]any{
			{"is_included": true, "is_singleton": false, "quantity": 1, "record_id": 7, "type_id": et.ID},
		}))
		_, err = s.updateContractsESI(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCorporationContract(ctx, c.ID, 42)
		require.NoError(t, err)
		items, err := st.ListCorporationContractItems(ctx, o.ID)
		require.NoError(t, err)
		xassert.Equal(t, 1, len(items))
		ids, err := st.ListCorporationContractBidIDs(ctx, o.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](7), ids)
	})

	t.Run("should store a new courier without fetching items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           42,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"status":                "outstanding",
				"type":                  "courier",
			}}),
		)
		var itemRequests int
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/42/items", c.ID),
			func(*http.Request) (*http.Response, error) {
				itemRequests++
				return nil, errors.New("items failed")
			},
		)
		// when
		_, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			forceUpdate:   true,
			section:       app.SectionCorporationContracts,
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCorporationContract(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, app.ContractTypeCourier, o.Type)
		xassert.Equal(t, 0, itemRequests)
	})

	t.Run("should update existing contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		local := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			Status:        app.ContractStatusOutstanding,
			Type:          app.ContractTypeCourier,
		})
		acceptor := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           acceptor.ID,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           local.ContractID,
				"date_accepted":         "2017-06-06T13:12:32Z",
				"date_completed":        "2017-06-07T13:12:32Z",
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"status":                "finished",
				"type":                  "courier",
			}}),
		)
		// when
		_, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			forceUpdate:   true,
			section:       app.SectionCorporationContracts,
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCorporationContract(ctx, c.ID, local.ContractID)
		require.NoError(t, err)
		xassert.EqualOptional(t, acceptor, o.Acceptor)
		xassert.EqualOptional(t, time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC), o.DateAccepted)
		xassert.EqualOptional(t, time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC), o.DateCompleted)
		xassert.Equal(t, app.ContractStatusFinished, o.Status)
	})

	t.Run("should delete stale and deleted contracts but keep finished ones", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		deleted := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			Status:        app.ContractStatusOutstanding,
			Type:          app.ContractTypeCourier,
		})
		factory.CreateCorporationContract(storage.CreateCorporationContractParams{ // missing in ESI
			CorporationID: c.ID,
			Status:        app.ContractStatusInProgress,
			Type:          app.ContractTypeCourier,
		})
		finished := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			Status:        app.ContractStatusFinished,
			Type:          app.ContractTypeCourier,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           deleted.ContractID,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"status":                "deleted",
				"type":                  "courier",
			}}),
		)
		// when
		_, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			forceUpdate:   true,
			section:       app.SectionCorporationContracts,
		})
		// then
		require.NoError(t, err)
		ids, err := st.ListCorporationContractIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(finished.ContractID), ids)
	})
}
