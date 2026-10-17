package characterservice

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
)

func TestUpdateContractsEscrow(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	c := factory.CreateCharacter()
	factory.CreateCharacterContract(storage.CreateCharacterContractParams{
		CharacterID: c.ID,
		AcceptorID:  c.ID,
		Collateral:  optional.New(100.0),
		Type:        app.ContractTypeCourier,
		Status:      app.ContractStatusInProgress,
	})
	factory.CreateCharacterContract(storage.CreateCharacterContractParams{
		CharacterID: c.ID,
		AcceptorID:  c.ID,
		Collateral:  optional.New(200.0),
		Type:        app.ContractTypeCourier,
		Status:      app.ContractStatusInProgress,
	})
	o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
		CharacterID: c.ID,
		Type:        app.ContractTypeAuction,
		Status:      app.ContractStatusOutstanding,
	})
	factory.CreateCharacterContractBid(storage.CreateContractBidParams{
		Amount:     12.3,
		BidderID:   c.ID,
		ContractID: o.ID,
	})

	// when
	err := s.updateContractsEscrow(t.Context(), c.ID)

	// then
	require.NoError(t, err)
	c2, err := st.GetCharacter(t.Context(), c.ID)
	require.NoError(t, err)
	got := c2.ContractsEscrow
	assert.Equal(t, optional.New(312.3), got)
}

func TestUpdateContractsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	s := NewFake(Params{Storage: st})
	type fixture struct {
		ownerID, issuerID, issuerCorporationID int64
	}
	setup := func(t *testing.T) fixture {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		fx := fixture{ownerID: c.ID, issuerID: c.ID, issuerCorporationID: c.EveCharacter.Corporation.ID}
		empty := httpmock.NewJsonResponderOrPanic(200, []map[string]any{})
		base := fmt.Sprintf(`^https://esi\.evetech\.net/characters/%d/contracts/\d+/`, fx.ownerID)
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
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", fx.ownerID),
			httpmock.NewJsonResponderOrPanic(200, contracts),
		)
	}
	registerDetails := func(fx fixture, contractID int64, kind string, data []map[string]any) {
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/%d/%s", fx.ownerID, contractID, kind),
			httpmock.NewJsonResponderOrPanic(200, data),
		)
	}
	update := func(ownerID int64) error {
		_, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: ownerID,
			forceUpdate: true,
			section:     app.SectionCharacterContracts,
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
		c, err := st.GetCharacterContract(ctx, fx.ownerID, contractID)
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
		items, err := st.ListCharacterContractItems(ctx, c.ID)
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
		c, err := st.GetCharacterContract(ctx, fx.ownerID, contractID)
		require.NoError(t, err)
		bids, err := st.ListCharacterContractBids(ctx, c.ID)
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
		local := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: fx.ownerID,
			Type:        app.ContractTypeAuction,
		})
		factory.CreateCharacterContractBid(storage.CreateContractBidParams{ContractID: local.ID, BidID: 1, Amount: 100})
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
		ids, err := st.ListCharacterContractBidIDs(ctx, local.ID)
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
			changed, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
				characterID: fx.ownerID,
				section:     app.SectionCharacterContracts,
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
		c, err := st.GetCharacterContract(ctx, fx.ownerID, contractID)
		require.NoError(t, err)
		ids, err := st.ListCharacterContractBidIDs(ctx, c.ID)
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
		itemsURL := fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/%d/items", fx.ownerID, contractID)
		httpmock.RegisterResponder("GET", itemsURL, httpmock.NewErrorResponder(errors.New("items failed")))
		registerDetails(fx, contractID, "bids", []map[string]any{
			{"amount": 123.5, "bid_id": 7, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"},
		})
		// when
		err := update(fx.ownerID)
		// then
		require.NoError(t, err)
		_, err = st.GetCharacterContract(ctx, fx.ownerID, contractID)
		require.ErrorIs(t, err, app.ErrNotFound)
		// when ESI works again
		registerDetails(fx, contractID, "items", []map[string]any{
			{"is_included": true, "is_singleton": false, "quantity": 1, "record_id": 7, "type_id": et.ID},
		})
		err = update(fx.ownerID)
		// then
		require.NoError(t, err)
		c, err := st.GetCharacterContract(ctx, fx.ownerID, contractID)
		require.NoError(t, err)
		items, err := st.ListCharacterContractItems(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 1, len(items))
		ids, err := st.ListCharacterContractBidIDs(ctx, c.ID)
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
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/%d/items", fx.ownerID, contractID),
			func(*http.Request) (*http.Response, error) {
				itemRequests++
				return nil, errors.New("items failed")
			},
		)
		// when
		err := update(fx.ownerID)
		// then
		require.NoError(t, err)
		c, err := st.GetCharacterContract(ctx, fx.ownerID, contractID)
		require.NoError(t, err)
		xassert.Equal(t, app.ContractTypeCourier, c.Type)
		xassert.Equal(t, 0, itemRequests)
	})

	t.Run("should update existing contract", func(t *testing.T) {
		// given
		fx := setup(t)
		local := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: fx.ownerID,
			Status:      app.ContractStatusOutstanding,
			Type:        app.ContractTypeCourier,
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
		c, err := st.GetCharacterContract(ctx, fx.ownerID, local.ContractID)
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
			return factory.CreateCharacterContract(storage.CreateCharacterContractParams{
				CharacterID: fx.ownerID,
				Status:      status,
				Type:        app.ContractTypeCourier,
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
		ids, err := st.ListCharacterContractIDs(ctx, fx.ownerID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(finished), ids)
	})
}

func TestUpdateContractsESICalculatedValues(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	setup := func(t *testing.T) *app.Character {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           c.ID,
				"assignee_id":           0,
				"availability":          "public",
				"collateral":            100.0,
				"contract_id":           42,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"status":                "in_progress",
				"type":                  "courier",
			}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/42/items", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
		)
		return c
	}
	update := func(characterID int64) error {
		_, err := s.updateContractsESI(t.Context(), characterSectionUpdateParams{
			characterID: characterID,
			section:     app.SectionCharacterContracts,
		})
		return err
	}
	t.Run("should update escrow and items value when contracts changed", func(t *testing.T) {
		// given
		c := setup(t)
		// when
		err := update(c.ID)
		// then
		require.NoError(t, err)
		c2, err := st.GetCharacter(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.EqualOptional(t, 100.0, c2.ContractsEscrow)
		xassert.NotEmpty(t, c2.ContractItemsValue)
	})
	t.Run("should update missing values when contracts are unchanged", func(t *testing.T) {
		// given
		c := setup(t)
		require.NoError(t, update(c.ID))
		require.NoError(t, st.UpdateCharacterContractsEscrow(t.Context(), c.ID, optional.Optional[float64]{}))
		require.NoError(t, st.UpdateCharacterContractItemsValue(t.Context(), c.ID, optional.Optional[float64]{}))
		// when
		err := update(c.ID)
		// then
		require.NoError(t, err)
		c2, err := st.GetCharacter(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.EqualOptional(t, 100.0, c2.ContractsEscrow)
		xassert.NotEmpty(t, c2.ContractItemsValue)
	})
}
