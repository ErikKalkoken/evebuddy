package corporationservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"regexp"
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
	type fixture struct {
		ownerID, issuerID, issuerCorporationID int64
	}
	setup := func(t *testing.T) fixture {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		fx := fixture{ownerID: c.ID, issuerID: c.ID, issuerCorporationID: c.ID}
		empty := httpmock.NewJsonResponderOrPanic(200, []map[string]any{})
		base := fmt.Sprintf(`^https://esi\.evetech\.net/corporations/%d/contracts/\d+/`, fx.ownerID)
		httpmock.RegisterRegexpResponder("GET", regexp.MustCompile(base+"items"), empty)
		httpmock.RegisterRegexpResponder("GET", regexp.MustCompile(base+"bids"), empty)
		return fx
	}
	// esiContract returns an ESI contract with defaults for all required fields.
	esiContract := func(fx fixture, contractID int64, fields map[string]any) map[string]any {
		m := map[string]any{
			"acceptor_id":           0,
			"assignee_id":           0,
			"availability":          "public",
			"contract_id":           contractID,
			"date_expired":          "2017-06-13T13:12:32Z",
			"date_issued":           "2017-06-05T13:12:32Z",
			"for_corporation":       false,
			"issuer_corporation_id": fx.issuerCorporationID,
			"issuer_id":             fx.issuerID,
			"status":                "outstanding",
			"type":                  "item_exchange",
		}
		maps.Copy(m, fields)
		return m
	}
	registerContracts := func(fx fixture, contracts ...map[string]any) {
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", fx.ownerID),
			httpmock.NewJsonResponderOrPanic(200, contracts),
		)
	}
	registerDetails := func(fx fixture, contractID int64, kind string, data []map[string]any) {
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/%d/%s", fx.ownerID, contractID, kind),
			httpmock.NewJsonResponderOrPanic(200, data),
		)
	}
	update := func(ownerID int64) error {
		_, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
			corporationID: ownerID,
			forceUpdate:   true,
			section:       app.SectionCorporationContracts,
		})
		return err
	}

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
			{"is_included": false, "is_singleton": true, "quantity": 1, "raw_quantity": -1, "record_id": 7, "type_id": et.ID},
			{"is_included": true, "is_singleton": false, "quantity": 3, "record_id": 8, "type_id": et.ID},
		})
		// when
		err := update(fx.ownerID)
		// then
		require.NoError(t, err)
		c, err := st.GetCorporationContract(ctx, fx.ownerID, contractID)
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
		xassert.Equal(t, fx.issuerID, c.Issuer.ID)
		xassert.Equal(t, fx.issuerCorporationID, c.IssuerCorporation.ID)
		xassert.EqualOptional(t, 4.5, c.Price)
		xassert.EqualOptional(t, 5.5, c.Reward)
		xassert.Equal(t, startLocation.ID, c.StartLocation.MustValue().ID)
		xassert.Equal(t, app.ContractStatusInProgress, c.Status)
		xassert.EqualOptional(t, "My title", c.Title)
		xassert.Equal(t, app.ContractTypeItemExchange, c.Type)
		xassert.EqualOptional(t, 6.5, c.Volume)
		items, err := st.ListCorporationContractItems(ctx, c.ID)
		require.NoError(t, err)
		slices.SortFunc(items, func(a, b *app.ContractItem) int {
			return cmp.Compare(a.RecordID, b.RecordID)
		})
		want := []*app.ContractItem{
			{ContractID: c.ID, IsIncluded: false, IsSingleton: true, Quantity: 1, RawQuantity: optional.New[int64](-1), RecordID: 7, Type: et},
			{ContractID: c.ID, IsIncluded: true, IsSingleton: false, Quantity: 3, RecordID: 8, Type: et},
		}
		xassert.Equal(t, want, items)
	})

	t.Run("should store bids of a new auction", func(t *testing.T) {
		// given
		const contractID = 42
		fx := setup(t)
		bidder := factory.CreateEveEntityCharacter()
		registerContracts(fx, esiContract(fx, contractID, map[string]any{"type": "auction"}))
		registerDetails(fx, contractID, "bids", []map[string]any{
			{"amount": 123.5, "bid_id": 7, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"},
		})
		// when
		err := update(fx.ownerID)
		// then
		require.NoError(t, err)
		c, err := st.GetCorporationContract(ctx, fx.ownerID, contractID)
		require.NoError(t, err)
		bids, err := st.ListCorporationContractBids(ctx, c.ID)
		require.NoError(t, err)
		want := []*app.ContractBid{{
			ContractID: c.ID,
			Amount:     123.5,
			BidID:      7,
			Bidder:     bidder,
			DateBid:    time.Date(2017, 1, 1, 10, 10, 10, 0, time.UTC),
		}}
		xassert.Equal(t, want, bids)
	})

	t.Run("should add only new bids to an existing auction", func(t *testing.T) {
		// given
		fx := setup(t)
		local := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: fx.ownerID,
			Type:          app.ContractTypeAuction,
		})
		factory.CreateCorporationContractBid(storage.CreateContractBidParams{ContractID: local.ID, BidID: 1, Amount: 100})
		bidder := factory.CreateEveEntityCharacter()
		registerContracts(fx, esiContract(fx, local.ContractID, map[string]any{"type": "auction"}))
		registerDetails(fx, local.ContractID, "bids", []map[string]any{
			{"amount": 100, "bid_id": 1, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"},
			{"amount": 200, "bid_id": 2, "bidder_id": bidder.ID, "date_bid": "2017-01-02T10:10:10Z"},
		})
		// when
		err := update(fx.ownerID)
		// then
		require.NoError(t, err)
		ids, err := st.ListCorporationContractBidIDs(ctx, local.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](1, 2), ids)
	})

	t.Run("should store new bids when the contract list is unchanged", func(t *testing.T) {
		// given
		const contractID = 42
		fx := setup(t)
		bidder := factory.CreateEveEntityCharacter()
		registerContracts(fx, esiContract(fx, contractID, map[string]any{"type": "auction"}))
		bid1 := map[string]any{"amount": 100, "bid_id": 1, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"}
		bid2 := map[string]any{"amount": 200, "bid_id": 2, "bidder_id": bidder.ID, "date_bid": "2017-01-02T10:10:10Z"}
		registerDetails(fx, contractID, "bids", []map[string]any{bid1})
		updateUnforced := func() bool {
			changed, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
				corporationID: fx.ownerID,
				section:       app.SectionCorporationContracts,
			})
			require.NoError(t, err)
			return changed
		}
		xassert.Equal(t, true, updateUnforced())
		// when
		registerDetails(fx, contractID, "bids", []map[string]any{bid1, bid2})
		changed := updateUnforced()
		// then
		xassert.Equal(t, true, changed)
		c, err := st.GetCorporationContract(ctx, fx.ownerID, contractID)
		require.NoError(t, err)
		ids, err := st.ListCorporationContractBidIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](1, 2), ids)
		// when nothing is new
		changed = updateUnforced()
		// then
		xassert.Equal(t, false, changed)
	})

	t.Run("should store a new contract only after its items could be fetched", func(t *testing.T) {
		// given
		const contractID = 42
		fx := setup(t)
		et := factory.CreateEveType()
		bidder := factory.CreateEveEntityCharacter()
		registerContracts(fx, esiContract(fx, contractID, map[string]any{"type": "auction"}))
		itemsURL := fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/%d/items", fx.ownerID, contractID)
		httpmock.RegisterResponder("GET", itemsURL, httpmock.NewErrorResponder(errors.New("items failed")))
		registerDetails(fx, contractID, "bids", []map[string]any{
			{"amount": 123.5, "bid_id": 7, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"},
		})
		// when
		err := update(fx.ownerID)
		// then
		require.NoError(t, err)
		_, err = st.GetCorporationContract(ctx, fx.ownerID, contractID)
		require.ErrorIs(t, err, app.ErrNotFound)
		// when ESI works again
		registerDetails(fx, contractID, "items", []map[string]any{
			{"is_included": true, "is_singleton": false, "quantity": 1, "record_id": 7, "type_id": et.ID},
		})
		err = update(fx.ownerID)
		// then
		require.NoError(t, err)
		c, err := st.GetCorporationContract(ctx, fx.ownerID, contractID)
		require.NoError(t, err)
		items, err := st.ListCorporationContractItems(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 1, len(items))
		ids, err := st.ListCorporationContractBidIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](7), ids)
	})

	t.Run("should store a new courier without fetching items", func(t *testing.T) {
		// given
		const contractID = 42
		fx := setup(t)
		registerContracts(fx, esiContract(fx, contractID, map[string]any{"type": "courier"}))
		var itemRequests int
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/%d/items", fx.ownerID, contractID),
			func(*http.Request) (*http.Response, error) {
				itemRequests++
				return nil, errors.New("items failed")
			},
		)
		// when
		err := update(fx.ownerID)
		// then
		require.NoError(t, err)
		c, err := st.GetCorporationContract(ctx, fx.ownerID, contractID)
		require.NoError(t, err)
		xassert.Equal(t, app.ContractTypeCourier, c.Type)
		xassert.Equal(t, 0, itemRequests)
	})

	t.Run("should update existing contract", func(t *testing.T) {
		// given
		fx := setup(t)
		local := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: fx.ownerID,
			Status:        app.ContractStatusOutstanding,
			Type:          app.ContractTypeCourier,
		})
		acceptor := factory.CreateEveEntityCharacter()
		registerContracts(fx, esiContract(fx, local.ContractID, map[string]any{
			"acceptor_id":    acceptor.ID,
			"date_accepted":  "2017-06-06T13:12:32Z",
			"date_completed": "2017-06-07T13:12:32Z",
			"status":         "finished",
			"type":           "courier",
		}))
		// when
		err := update(fx.ownerID)
		// then
		require.NoError(t, err)
		c, err := st.GetCorporationContract(ctx, fx.ownerID, local.ContractID)
		require.NoError(t, err)
		xassert.EqualOptional(t, acceptor, c.Acceptor)
		xassert.EqualOptional(t, time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC), c.DateAccepted)
		xassert.EqualOptional(t, time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC), c.DateCompleted)
		xassert.Equal(t, app.ContractStatusFinished, c.Status)
	})

	t.Run("should delete stale and deleted contracts but keep finished ones", func(t *testing.T) {
		// given
		fx := setup(t)
		create := func(status app.ContractStatus) int64 {
			return factory.CreateCorporationContract(storage.CreateCorporationContractParams{
				CorporationID: fx.ownerID,
				Status:        status,
				Type:          app.ContractTypeCourier,
			}).ContractID
		}
		deleted := create(app.ContractStatusOutstanding)
		create(app.ContractStatusInProgress) // missing in ESI
		finished := create(app.ContractStatusFinished)
		registerContracts(fx, esiContract(fx, deleted, map[string]any{
			"status": "deleted",
			"type":   "courier",
		}))
		// when
		err := update(fx.ownerID)
		// then
		require.NoError(t, err)
		ids, err := st.ListCorporationContractIDs(ctx, fx.ownerID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(finished), ids)
	})
}
