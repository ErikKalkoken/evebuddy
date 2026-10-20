package characterservice

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

	// when
	err := s.updateContractsEscrow(t.Context(), c.ID)

	// then
	require.NoError(t, err)
	c2, err := st.GetCharacter(t.Context(), c.ID)
	require.NoError(t, err)
	got := c2.ContractsEscrow
	assert.Equal(t, optional.New(300.0), got)
}

func TestUpdateContractsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	s := NewFake(Params{Storage: st})

	t.Run("should store all fields of a new contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		acceptor := factory.CreateEveEntityCharacter()
		assignee := factory.CreateEveEntityCorporation()
		startLocation := factory.CreateEveLocationStructure()
		endLocation := factory.CreateEveLocationStructure()
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
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
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
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
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/42/items", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"is_included": false, "is_singleton": true, "quantity": 1, "raw_quantity": -1, "record_id": 7, "type_id": et.ID},
				{"is_included": true, "is_singleton": false, "quantity": 3, "record_id": 8, "type_id": et.ID},
			}),
		)
		// when
		_, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			forceUpdate: true,
			section:     app.SectionCharacterContracts,
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterContract(ctx, c.ID, 42)
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
		xassert.Equal(t, c.EveCharacter.Corporation.ID, o.IssuerCorporation.ID)
		xassert.EqualOptional(t, 4.5, o.Price)
		xassert.EqualOptional(t, 5.5, o.Reward)
		xassert.Equal(t, startLocation.ID, o.StartLocation.MustValue().ID)
		xassert.Equal(t, app.ContractStatusInProgress, o.Status)
		xassert.EqualOptional(t, "My title", o.Title)
		xassert.Equal(t, app.ContractTypeItemExchange, o.Type)
		xassert.EqualOptional(t, 6.5, o.Volume)
		items, err := st.ListCharacterContractItems(ctx, o.ID)
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
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		bidder := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           42,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"status":                "outstanding",
				"type":                  "auction",
			}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/42/items", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/42/bids", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"amount": 123.5, "bid_id": 7, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"},
			}),
		)
		// when
		_, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			forceUpdate: true,
			section:     app.SectionCharacterContracts,
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterContract(ctx, c.ID, 42)
		require.NoError(t, err)
		bids, err := st.ListCharacterContractBids(ctx, o.ID)
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
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		local := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: c.ID,
			Status:      app.ContractStatusOutstanding,
			Type:        app.ContractTypeAuction,
		})
		factory.CreateCharacterContractBid(storage.CreateContractBidParams{ContractID: local.ID, BidID: 1, Amount: 100})
		bidder := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           local.ContractID,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"status":                "outstanding",
				"type":                  "auction",
			}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/%d/bids", c.ID, local.ContractID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"amount": 100, "bid_id": 1, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"},
				{"amount": 200, "bid_id": 2, "bidder_id": bidder.ID, "date_bid": "2017-01-02T10:10:10Z"},
			}),
		)
		// when
		_, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			forceUpdate: true,
			section:     app.SectionCharacterContracts,
		})
		// then
		require.NoError(t, err)
		ids, err := st.ListCharacterContractBidIDs(ctx, local.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](1, 2), ids)
	})

	t.Run("should store new bids when the contract list is unchanged", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		bidder := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           42,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"status":                "outstanding",
				"type":                  "auction",
			}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/42/items", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
		)
		bidsURL := fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/42/bids", c.ID)
		bid1 := map[string]any{"amount": 100, "bid_id": 1, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"}
		bid2 := map[string]any{"amount": 200, "bid_id": 2, "bidder_id": bidder.ID, "date_bid": "2017-01-02T10:10:10Z"}
		httpmock.RegisterResponder("GET", bidsURL, httpmock.NewJsonResponderOrPanic(200, []map[string]any{bid1}))
		arg := characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterContracts,
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
		o, err := st.GetCharacterContract(ctx, c.ID, 42)
		require.NoError(t, err)
		ids, err := st.ListCharacterContractBidIDs(ctx, o.ID)
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
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		et := factory.CreateEveType()
		bidder := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           42,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"status":                "outstanding",
				"type":                  "auction",
			}}),
		)
		itemsURL := fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/42/items", c.ID)
		httpmock.RegisterResponder("GET", itemsURL, httpmock.NewErrorResponder(errors.New("items failed")))
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/42/bids", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{"amount": 123.5, "bid_id": 7, "bidder_id": bidder.ID, "date_bid": "2017-01-01T10:10:10Z"},
			}),
		)
		arg := characterSectionUpdateParams{
			characterID: c.ID,
			forceUpdate: true,
			section:     app.SectionCharacterContracts,
		}
		// when
		_, err := s.updateContractsESI(ctx, arg)
		// then
		require.NoError(t, err)
		_, err = st.GetCharacterContract(ctx, c.ID, 42)
		require.ErrorIs(t, err, app.ErrNotFound)
		// when ESI works again
		httpmock.RegisterResponder("GET", itemsURL, httpmock.NewJsonResponderOrPanic(200, []map[string]any{
			{"is_included": true, "is_singleton": false, "quantity": 1, "record_id": 7, "type_id": et.ID},
		}))
		_, err = s.updateContractsESI(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterContract(ctx, c.ID, 42)
		require.NoError(t, err)
		items, err := st.ListCharacterContractItems(ctx, o.ID)
		require.NoError(t, err)
		xassert.Equal(t, 1, len(items))
		ids, err := st.ListCharacterContractBidIDs(ctx, o.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](7), ids)
	})

	t.Run("should store a new courier without fetching items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           42,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"status":                "outstanding",
				"type":                  "courier",
			}}),
		)
		var itemRequests int
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/42/items", c.ID),
			func(*http.Request) (*http.Response, error) {
				itemRequests++
				return nil, errors.New("items failed")
			},
		)
		// when
		_, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			forceUpdate: true,
			section:     app.SectionCharacterContracts,
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterContract(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, app.ContractTypeCourier, o.Type)
		xassert.Equal(t, 0, itemRequests)
	})

	t.Run("should update existing contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		local := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: c.ID,
			Status:      app.ContractStatusOutstanding,
			Type:        app.ContractTypeCourier,
		})
		acceptor := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
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
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"status":                "finished",
				"type":                  "courier",
			}}),
		)
		// when
		_, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			forceUpdate: true,
			section:     app.SectionCharacterContracts,
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterContract(ctx, c.ID, local.ContractID)
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
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		deleted := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: c.ID,
			Status:      app.ContractStatusOutstanding,
			Type:        app.ContractTypeCourier,
		})
		factory.CreateCharacterContract(storage.CreateCharacterContractParams{ // missing in ESI
			CharacterID: c.ID,
			Status:      app.ContractStatusInProgress,
			Type:        app.ContractTypeCourier,
		})
		finished := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: c.ID,
			Status:      app.ContractStatusFinished,
			Type:        app.ContractTypeCourier,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           deleted.ContractID,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"status":                "deleted",
				"type":                  "courier",
			}}),
		)
		// when
		_, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			forceUpdate: true,
			section:     app.SectionCharacterContracts,
		})
		// then
		require.NoError(t, err)
		ids, err := st.ListCharacterContractIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(finished.ContractID), ids)
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
