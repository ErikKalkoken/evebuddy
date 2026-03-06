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
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

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
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
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
		xassert.Equal(t, time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC), o.DateAccepted.MustValue())
		xassert.Equal(t, time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC), o.DateCompleted.MustValue())
		xassert.Equal(t, time.Date(2017, 6, 13, 13, 12, 32, 0, time.UTC), o.DateExpired)
		xassert.Equal(t, time.Date(2017, 6, 5, 13, 12, 32, 0, time.UTC), o.DateIssued)
		xassert.Equal(t, startLocation.ID, o.StartLocation.MustValue().ID)
		xassert.Equal(t, endLocation.ID, o.EndLocation.MustValue().ID)
		xassert.Equal(t, app.ContractStatusFinished, o.Status)
		xassert.Equal(t, app.ContractTypeCourier, o.Type)
		xassert.Equal(t, volume, o.Volume.ValueOrZero())
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
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
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
		xassert.Equal(t, buyout, o.Buyout.ValueOrZero())
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
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
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
		xassert.Equal(t, acceptor, o2.Acceptor.MustValue())
		xassert.Equal(t, app.ContractStatusFinished, o2.Status)
		xassert.Equal(t, time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC), o2.DateAccepted.MustValue())
		xassert.Equal(t, time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC), o2.DateCompleted.MustValue())
		xassert.Equal(t, o1.DateIssued, o2.DateIssued)
		xassert.Equal(t, o1.DateExpired, o2.DateExpired)
	})
	t.Run("should not update unchanged contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
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
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
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
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
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
