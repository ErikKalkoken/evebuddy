package characterservice

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
	"github.com/ErikKalkoken/evebuddy/internal/optional"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateContractESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("should create new item exchange contract from scratch", func(t *testing.T) {
		// given
		const (
			contractID = 42
			quantity   = 7
			recordID   = 123456
			volume     = 0.01
		)
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		acceptor := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           acceptor.ID,
				"assignee_id":           acceptor.ID,
				"availability":          "public",
				"contract_id":           contractID,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"days_to_complete":      3,
				"for_corporation":       false,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"price":                 1000000.01,
				"status":                "outstanding",
				"type":                  "item_exchange",
				"volume":                volume,
			}}),
		)
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/%d/items", c.ID, contractID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_included":  true,
				"is_singleton": false,
				"quantity":     quantity,
				"record_id":    recordID,
				"type_id":      et.ID,
			}}),
		)
		// when
		changed, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterContracts,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCharacterContract(ctx, c.ID, contractID)
		require.NoError(t, err)
		assert.True(t, o.DateAccepted.IsEmpty())
		assert.True(t, o.DateCompleted.IsEmpty())
		xassert.Equal(t, time.Date(2017, 6, 13, 13, 12, 32, 0, time.UTC), o.DateExpired)
		xassert.Equal(t, time.Date(2017, 6, 5, 13, 12, 32, 0, time.UTC), o.DateIssued)
		xassert.Equal(t, app.ContractStatusOutstanding, o.Status)
		xassert.Equal(t, app.ContractTypeItemExchange, o.Type)
		xassert.Equal(t, volume, o.Volume.ValueOrZero())
		ii, err := st.ListCharacterContractItems(ctx, o.ID)
		require.NoError(t, err)
		assert.Len(t, ii, 1)
		xassert.Equal(t, quantity, ii[0].Quantity)
		xassert.Equal(t, recordID, ii[0].RecordID)
		xassert.Equal(t, et, ii[0].Type)
	})
	t.Run("should create new courier contract from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		contractID := int64(42)
		startLocation := factory.CreateEveLocationStructure()
		endLocation := factory.CreateEveLocationStructure()
		volume := 0.01
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
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
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"price":                 1000000.01,
				"reward":                0.01,
				"start_location_id":     startLocation.ID,
				"status":                "finished",
				"type":                  "courier",
				"volume":                volume,
			}}),
		)
		const (
			recordID = 123456
			quantity = 7
		)
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/%d/items", c.ID, contractID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_included":  true,
				"is_singleton": false,
				"quantity":     quantity,
				"record_id":    recordID,
				"type_id":      et.ID,
			}}),
		)
		// when
		changed, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterContracts,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCharacterContract(ctx, c.ID, contractID)
		require.NoError(t, err)
		xassert.Equal(t, time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC), o.DateAccepted.MustValue())
		xassert.Equal(t, time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC), o.DateCompleted.MustValue())
		xassert.Equal(t, time.Date(2017, 6, 13, 13, 12, 32, 0, time.UTC), o.DateExpired)
		xassert.Equal(t, time.Date(2017, 6, 5, 13, 12, 32, 0, time.UTC), o.DateIssued)
		xassert.Equal(t, startLocation.ID, o.StartLocation.ValueOrZero().ID)
		xassert.Equal(t, endLocation.ID, o.EndLocation.ValueOrZero().ID)
		xassert.Equal(t, app.ContractStatusFinished, o.Status)
		xassert.Equal(t, app.ContractTypeCourier, o.Type)
		xassert.Equal(t, volume, o.Volume.ValueOrZero())
		ii, err := st.ListCharacterContractItems(ctx, o.ID)
		require.NoError(t, err)
		assert.Len(t, ii, 1)
		xassert.Equal(t, quantity, ii[0].Quantity)
		xassert.Equal(t, recordID, ii[0].RecordID)
		xassert.Equal(t, et, ii[0].Type)
	})
	t.Run("should create new auction contract from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		contractID := int64(42)
		startLocation := factory.CreateEveLocationStructure()
		buyout := 10000000000.01
		volume := 0.01
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
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
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"start_location_id":     startLocation.ID,
				"status":                "outstanding",
				"type":                  "auction",
				"volume":                volume,
			}}),
		)
		const (
			recordID = 123456
			quantity = 7
		)
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/%d/items", c.ID, contractID),
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
		bidder := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/%d/bids", c.ID, contractID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"amount":    amount,
				"bid_id":    bidID,
				"bidder_id": bidder.ID,
				"date_bid":  "2017-01-01T10:10:10Z",
			}}),
		)
		// when
		changed, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterContracts,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCharacterContract(ctx, c.ID, contractID)
		require.NoError(t, err)
		xassert.Equal(t, time.Date(2017, 6, 13, 13, 12, 32, 0, time.UTC), o.DateExpired)
		xassert.Equal(t, time.Date(2017, 6, 5, 13, 12, 32, 0, time.UTC), o.DateIssued)
		xassert.Equal(t, startLocation.ID, o.StartLocation.ValueOrZero().ID)
		xassert.Equal(t, app.ContractStatusOutstanding, o.Status)
		xassert.Equal(t, app.ContractTypeAuction, o.Type)
		xassert.Equal(t, buyout, o.Buyout.ValueOrZero())
		ii, err := st.ListCharacterContractItems(ctx, o.ID)
		require.NoError(t, err)
		assert.Len(t, ii, 1)
		xassert.Equal(t, quantity, ii[0].Quantity)
		xassert.Equal(t, recordID, ii[0].RecordID)
		xassert.Equal(t, et, ii[0].Type)
		bb, err := st.ListCharacterContractBids(ctx, o.ID)
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
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		o1 := factory.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
			CharacterID:  c.ID,
			Availability: app.ContractAvailabilityPublic,
			Status:       app.ContractStatusOutstanding,
		})
		acceptor := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           acceptor.ID,
				"assignee_id":           0,
				"availability":          "public",
				"buyout":                o1.Buyout.ValueOrZero(),
				"contract_id":           o1.ContractID,
				"date_accepted":         "2017-06-06T13:12:32Z",
				"date_completed":        "2017-06-07T13:12:32Z",
				"date_expired":          o1.DateExpired.Format(time.RFC3339),
				"date_issued":           o1.DateIssued.Format(time.RFC3339),
				"days_to_complete":      o1.DaysToComplete.ValueOrZero(),
				"for_corporation":       true,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"price":                 o1.Price.ValueOrZero(),
				"reward":                o1.Reward.ValueOrZero(),
				"status":                "finished",
				"type":                  "courier",
				"volume":                o1.Volume.ValueOrZero(),
			}}),
		)
		// when
		changed, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterContracts,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o2, err := st.GetCharacterContract(ctx, c.ID, o1.ContractID)
		require.NoError(t, err)
		xassert.Equal(t, acceptor, o2.Acceptor.ValueOrZero())
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
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		o1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID:  c.ID,
			Availability: app.ContractAvailabilityPublic,
			Status:       app.ContractStatusOutstanding,
			Type:         app.ContractTypeCourier,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"buyout":                o1.Buyout.ValueOrZero(),
				"contract_id":           o1.ContractID,
				"date_expired":          o1.DateExpired.Format(time.RFC3339),
				"date_issued":           o1.DateIssued.Format(time.RFC3339),
				"days_to_complete":      o1.DaysToComplete.ValueOrZero(),
				"for_corporation":       o1.ForCorporation,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"price":                 o1.Price.ValueOrZero(),
				"reward":                o1.Reward.ValueOrZero(),
				"status":                "outstanding",
				"type":                  "courier",
				"volume":                o1.Volume.ValueOrZero(),
			}}),
		)
		// when
		changed, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterContracts,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o2, err := st.GetCharacterContract(ctx, c.ID, o1.ContractID)
		require.NoError(t, err)
		xassert.Equal(t, o1.UpdatedAt, o2.UpdatedAt)
	})
	t.Run("should not create deleted contracts", func(t *testing.T) {
		// given
		const (
			contractID = 42
			volume     = 0.01
			recordID   = 123456
			quantity   = 7
		)
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		asignee := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           asignee.ID,
				"assignee_id":           asignee.ID,
				"availability":          "public",
				"contract_id":           contractID,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"days_to_complete":      3,
				"for_corporation":       false,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"price":                 1000000.01,
				"status":                "deleted",
				"type":                  "item_exchange",
				"volume":                volume,
			}}),
		)
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/%d/items", c.ID, contractID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_included":  true,
				"is_singleton": false,
				"quantity":     quantity,
				"record_id":    recordID,
				"type_id":      et.ID,
			}}),
		)
		// when
		_, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterContracts,
		})
		// then
		require.NoError(t, err)
		ids, err := st.ListCharacterContractIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0, ids.Size())
	})
	t.Run("should remove unfinished contracts missing in ESI response", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID:  c.ID,
			Availability: app.ContractAvailabilityPublic,
			Status:       app.ContractStatusOutstanding,
			Type:         app.ContractTypeCourier,
		})
		o1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID:  c.ID,
			Availability: app.ContractAvailabilityPublic,
			Status:       app.ContractStatusFinished,
			Type:         app.ContractTypeCourier,
		})
		o2 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID:  c.ID,
			Availability: app.ContractAvailabilityPublic,
			Status:       app.ContractStatusFinishedContractor,
			Type:         app.ContractTypeCourier,
		})
		o3 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID:  c.ID,
			Availability: app.ContractAvailabilityPublic,
			Status:       app.ContractStatusFinishedIssuer,
			Type:         app.ContractTypeCourier,
		})
		o4 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID:  c.ID,
			Availability: app.ContractAvailabilityPublic,
			Status:       app.ContractStatusOutstanding,
			Type:         app.ContractTypeCourier,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           o4.ContractID,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"days_to_complete":      3,
				"for_corporation":       false,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"price":                 1000000.01,
				"status":                "deleted",
				"type":                  "courier",
				"volume":                3,
			}}),
		)
		// when
		_, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterContracts,
		})
		// then
		require.NoError(t, err)
		ids, err := st.ListCharacterContractIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(o1.ContractID, o2.ContractID, o3.ContractID), ids)
	})
}

func TestUpdateContractsEscrow(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})

	characterID := int64(42)

	cases := []struct {
		name string

		acceptorID   int64
		contractType app.ContractType
		collateral   optional.Optional[float64]
		want         optional.Optional[float64]
	}{
		{"accepted courier contract", characterID, app.ContractTypeCourier, optional.New(123.0), optional.New(123.0)},
		{"courier contract accepted by other", 0, app.ContractTypeCourier, optional.New(123.0), optional.New(0.0)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
				ID: characterID,
			})
			c := factory.CreateCharacter(storage.CreateCharacterParams{
				ID: ec.ID,
			})
			factory.CreateCharacterContract(storage.CreateCharacterContractParams{
				CharacterID: c.ID,
				AcceptorID:  tc.acceptorID,
				Collateral:  tc.collateral,
				Type:        tc.contractType,
			})
			factory.CreateCharacterContract() // should be ignored

			// when
			err := s.updateContractsEscrow(t.Context(), c.ID)

			// then
			require.NoError(t, err)
			c2, err := st.GetCharacter(t.Context(), c.ID)
			require.NoError(t, err)
			got := c2.ContractsEscrow
			assert.Equal(t, tc.want, got)
		})
	}
}
