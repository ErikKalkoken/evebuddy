package corporationservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/contracttest"
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

func TestCountContractBids(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can count bids for a contract", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		o := factory.CreateCorporationContract()
		factory.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{ContractID: o.ID})
		factory.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{ContractID: o.ID})
		got, err := s.CountContractBids(ctx, o.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, got)
	})
	t.Run("returns zero when contract has no bids", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		o := factory.CreateCorporationContract()
		got, err := s.CountContractBids(ctx, o.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, got)
	})
}

func TestGetContractTopBid(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("returns the bid with the highest amount", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		o := factory.CreateCorporationContract()
		factory.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{ContractID: o.ID, Amount: 100})
		top := factory.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{ContractID: o.ID, Amount: 500})
		factory.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{ContractID: o.ID, Amount: 200})
		got, err := s.GetContractTopBid(ctx, o.ID)
		require.NoError(t, err)
		xassert.Equal(t, top.BidID, got.BidID)
	})
	t.Run("should return error when contract has no bids", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		o := factory.CreateCorporationContract()
		_, err := s.GetContractTopBid(ctx, o.ID)
		assert.ErrorIs(t, err, app.ErrNotFound)
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
			Status:        app.ContractStatusInProgress,
		})
		factory.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{
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
		i1 := factory.CreateCorporationContractItem(storage.CreateCorporationContractItemParams{ContractID: o.ID})
		i2 := factory.CreateCorporationContractItem(storage.CreateCorporationContractItemParams{ContractID: o.ID})
		got, err := s.ListContractItems(ctx, o.ID)
		require.NoError(t, err)
		ids := set.Collect(xiter.MapSlice(got, func(x *app.ContractItem) int64 { return x.RecordID }))
		xassert.Equal(t, set.Of(i1.RecordID, i2.RecordID), ids)
	})
}

func TestUpdateContractESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should create new courier contract from scratch", func(t *testing.T) {
		// given
		const (
			contractID = 42
			quantity   = 7
			recordID   = 123456
			volume     = 0.01
		)
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		startLocation := factory.CreateEveLocationStructure()
		endLocation := factory.CreateEveLocationStructure()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           contractID,
				"date_accepted":         "2017-06-06T13:12:32Z",
				"date_completed":        "2017-06-07T13:12:32Z",
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"days_to_complete":      0,
				"end_location_id":       endLocation.ID,
				"for_corporation":       true,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"price":                 1000000.01,
				"reward":                0.01,
				"start_location_id":     startLocation.ID,
				"status":                "finished",
				"type":                  "courier",
				"volume":                volume,
			}}),
		)
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/%d/items", c.ID, contractID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_included":  true,
				"is_singleton": false,
				"quantity":     quantity,
				"record_id":    recordID,
				"type_id":      et.ID,
			}}),
		)
		// when
		changed, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationContracts,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCorporationContract(ctx, c.ID, contractID)
		require.NoError(t, err)
		xassert.EqualOptional(t, time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC), o.DateAccepted)
		xassert.EqualOptional(t, time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC), o.DateCompleted)
		xassert.Equal(t, time.Date(2017, 6, 13, 13, 12, 32, 0, time.UTC), o.DateExpired)
		xassert.Equal(t, time.Date(2017, 6, 5, 13, 12, 32, 0, time.UTC), o.DateIssued)
		xassert.Equal(t, startLocation.ID, o.StartLocation.MustValue().ID)
		xassert.Equal(t, endLocation.ID, o.EndLocation.MustValue().ID)
		xassert.Equal(t, app.ContractStatusFinished, o.Status)
		xassert.Equal(t, app.ContractTypeCourier, o.Type)
		xassert.EqualOptional(t, volume, o.Volume)
		ii, err := st.ListCorporationContractItems(ctx, o.ID)

		require.NoError(t, err)
		require.Len(t, ii, 1)
		xassert.Equal(t, quantity, ii[0].Quantity)
		xassert.Equal(t, recordID, ii[0].RecordID)
		xassert.Equal(t, et, ii[0].Type)

	})
	t.Run("should create new auction contract from scratch", func(t *testing.T) {
		// given
		const (
			buyout     = 10000000000.01
			contractID = 42
			quantity   = 7
			recordID   = 123456
			volume     = 0.01
		)
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		startLocation := factory.CreateEveLocationStructure()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"buyout":                buyout,
				"contract_id":           contractID,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"days_to_complete":      0,
				"for_corporation":       false,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"start_location_id":     startLocation.ID,
				"status":                "outstanding",
				"type":                  "auction",
				"volume":                volume,
			}}),
		)
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/%d/items", c.ID, contractID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_included":  true,
				"is_singleton": false,
				"quantity":     quantity,
				"record_id":    recordID,
				"type_id":      et.ID,
			}}),
		)
		const (
			bidID  = 123456
			amount = 123.45
		)
		bidder := factory.CreateEveEntityCorporation()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts/%d/bids", c.ID, contractID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"amount":    amount,
				"bid_id":    bidID,
				"bidder_id": bidder.ID,
				"date_bid":  "2017-01-01T10:10:10Z",
			}}),
		)
		// when
		changed, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationContracts,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCorporationContract(ctx, c.ID, contractID)
		require.NoError(t, err)
		xassert.Equal(t, time.Date(2017, 6, 13, 13, 12, 32, 0, time.UTC), o.DateExpired)
		xassert.Equal(t, time.Date(2017, 6, 5, 13, 12, 32, 0, time.UTC), o.DateIssued)
		xassert.Equal(t, startLocation.ID, o.StartLocation.MustValue().ID)
		xassert.Equal(t, app.ContractStatusOutstanding, o.Status)
		xassert.Equal(t, app.ContractTypeAuction, o.Type)
		xassert.EqualOptional(t, buyout, o.Buyout)
		ii, err := st.ListCorporationContractItems(ctx, o.ID)
		require.NoError(t, err)
		assert.Len(t, ii, 1)
		xassert.Equal(t, quantity, ii[0].Quantity)
		xassert.Equal(t, recordID, ii[0].RecordID)
		xassert.Equal(t, et, ii[0].Type)
		bb, err := st.ListCorporationContractBids(ctx, o.ID)
		require.NoError(t, err)
		assert.Len(t, bb, 1)
		assert.InDelta(t, amount, bb[0].Amount, 0.1)
		xassert.Equal(t, bidder, bb[0].Bidder)
		xassert.Equal(t, time.Date(2017, 1, 1, 10, 10, 10, 0, time.UTC), bb[0].DateBid)
	})
	t.Run("should update existing contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		o1 := factory.CreateCorporationContractCourier(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			Availability:  app.ContractAvailabilityPublic,
			Status:        app.ContractStatusOutstanding,
		})
		acceptor := factory.CreateEveEntityCorporation()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           acceptor.ID,
				"assignee_id":           0,
				"availability":          "public",
				"buyout":                o1.Buyout,
				"contract_id":           o1.ContractID,
				"date_accepted":         "2017-06-06T13:12:32Z",
				"date_completed":        "2017-06-07T13:12:32Z",
				"date_expired":          o1.DateExpired.Format(time.RFC3339),
				"date_issued":           o1.DateIssued.Format(time.RFC3339),
				"days_to_complete":      o1.DaysToComplete,
				"for_corporation":       true,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"price":                 o1.Price,
				"reward":                o1.Reward,
				"status":                "finished",
				"type":                  "courier",
				"volume":                o1.Volume,
			}}),
		)
		// when
		changed, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationContracts,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o2, err := st.GetCorporationContract(ctx, c.ID, o1.ContractID)
		require.NoError(t, err)
		xassert.EqualOptional(t, acceptor, o2.Acceptor)
		xassert.Equal(t, app.ContractStatusFinished, o2.Status)
		xassert.EqualOptional(t, time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC), o2.DateAccepted)
		xassert.EqualOptional(t, time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC), o2.DateCompleted)
		xassert.Equal(t, o1.DateIssued, o2.DateIssued)
		xassert.Equal(t, o1.DateExpired, o2.DateExpired)
	})
	t.Run("should not update unchanged contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		o1 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			Availability:  app.ContractAvailabilityPublic,
			Status:        app.ContractStatusOutstanding,
			Type:          app.ContractTypeCourier,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"buyout":                o1.Buyout,
				"contract_id":           o1.ContractID,
				"date_expired":          o1.DateExpired.Format(time.RFC3339),
				"date_issued":           o1.DateIssued.Format(time.RFC3339),
				"days_to_complete":      o1.DaysToComplete,
				"for_corporation":       true,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"price":                 o1.Price,
				"reward":                o1.Reward,
				"status":                "outstanding",
				"type":                  "courier",
				"volume":                o1.Volume,
			}}),
		)
		// when
		changed, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationContracts,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o2, err := st.GetCorporationContract(ctx, c.ID, o1.ContractID)
		require.NoError(t, err)
		xassert.Equal(t, o1.UpdatedAt, o2.UpdatedAt)
	})
	t.Run("should not create deleted contracts", func(t *testing.T) {
		// given
		const (
			contractID = 42
			volume     = 0.01
		)
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		startLocation := factory.CreateEveLocationStructure()
		endLocation := factory.CreateEveLocationStructure()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           contractID,
				"date_accepted":         "2017-06-06T13:12:32Z",
				"date_completed":        "2017-06-07T13:12:32Z",
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"days_to_complete":      0,
				"end_location_id":       endLocation.ID,
				"for_corporation":       true,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"price":                 1000000.01,
				"reward":                0.01,
				"start_location_id":     startLocation.ID,
				"status":                "deleted",
				"type":                  "courier",
				"volume":                volume,
			}}),
		)
		// when
		_, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationContracts,
		})
		// then
		require.NoError(t, err)
		ids, err := st.ListCorporationContractIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0, ids.Size())
	})
	t.Run("should delete unfinished stale contracts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		o1 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			Availability:  app.ContractAvailabilityPublic,
			Status:        app.ContractStatusOutstanding,
			Type:          app.ContractTypeCourier,
		})
		factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			Availability:  app.ContractAvailabilityPublic,
			Status:        app.ContractStatusOutstanding,
			Type:          app.ContractTypeCourier,
		})
		o2 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			Availability:  app.ContractAvailabilityPublic,
			Status:        app.ContractStatusFinished,
			Type:          app.ContractTypeCourier,
		})
		o3 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			Availability:  app.ContractAvailabilityPublic,
			Status:        app.ContractStatusFinished,
			Type:          app.ContractTypeCourier,
		})
		o4 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			Availability:  app.ContractAvailabilityPublic,
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
				"buyout":                o1.Buyout,
				"contract_id":           o1.ContractID,
				"date_expired":          o1.DateExpired.Format(time.RFC3339),
				"date_issued":           o1.DateIssued.Format(time.RFC3339),
				"days_to_complete":      o1.DaysToComplete,
				"for_corporation":       true,
				"issuer_corporation_id": c.ID,
				"issuer_id":             c.ID,
				"price":                 o1.Price,
				"reward":                o1.Reward,
				"status":                "deleted",
				"type":                  "courier",
				"volume":                o1.Volume,
			}}),
		)
		// when
		changed, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationContracts,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCorporationContractIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(o2.ContractID, o3.ContractID, o4.ContractID), got)
	})
}

func TestUpdateContractsESIShared(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
		AccessToken: "accessToken",
	}}})
	contracttest.RunUpdateContractsESITests(t, db, factory, contracttest.Owner{
		ESIPath: "corporations",
		Setup: func(t *testing.T) contracttest.Fixture {
			c := factory.CreateCorporation()
			factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
			return contracttest.Fixture{
				OwnerID:             c.ID,
				IssuerID:            c.ID,
				IssuerCorporationID: c.ID,
			}
		},
		Sync: func(ctx context.Context, corporationID int64) error {
			_, err := s.updateContractsESI(ctx, corporationSectionUpdateParams{
				corporationID: corporationID,
				forceUpdate:   true,
				section:       app.SectionCorporationContracts,
			})
			return err
		},
		CreateContract: func(t *testing.T, corporationID int64, arg contracttest.CreateParams) *app.Contract {
			o := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
				CorporationID: corporationID,
				Status:        arg.Status,
				Type:          arg.Type,
			})
			return &o.Contract
		},
		CreateBid: func(t *testing.T, contractPK, bidID int64, amount float64) {
			factory.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{
				Amount:     amount,
				BidID:      bidID,
				ContractID: contractPK,
			})
		},
		GetContract: func(ctx context.Context, corporationID, contractID int64) (*app.Contract, error) {
			o, err := st.GetCorporationContract(ctx, corporationID, contractID)
			if err != nil {
				return nil, err
			}
			return &o.Contract, nil
		},
		ListContractIDs: st.ListCorporationContractIDs,
		ListItems:       st.ListCorporationContractItems,
		ListBids:        st.ListCorporationContractBids,
	})
}
