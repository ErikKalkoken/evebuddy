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
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/contracttest"
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
		xassert.EqualOptional(t, volume, o.Volume)
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
		xassert.EqualOptional(t, time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC), o.DateAccepted)
		xassert.EqualOptional(t, time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC), o.DateCompleted)
		xassert.Equal(t, time.Date(2017, 6, 13, 13, 12, 32, 0, time.UTC), o.DateExpired)
		xassert.Equal(t, time.Date(2017, 6, 5, 13, 12, 32, 0, time.UTC), o.DateIssued)
		xassert.Equal(t, startLocation.ID, o.StartLocation.MustValue().ID)
		xassert.Equal(t, endLocation.ID, o.EndLocation.MustValue().ID)
		xassert.Equal(t, app.ContractStatusFinished, o.Status)
		xassert.Equal(t, app.ContractTypeCourier, o.Type)
		xassert.EqualOptional(t, volume, o.Volume)
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
		xassert.Equal(t, startLocation.ID, o.StartLocation.MustValue().ID)
		xassert.Equal(t, app.ContractStatusOutstanding, o.Status)
		xassert.Equal(t, app.ContractTypeAuction, o.Type)
		xassert.EqualOptional(t, buyout, o.Buyout)
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
		Status:      app.ContractStatusInProgress,
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

func TestUpdateContractsESIShared(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	contracttest.RunUpdateContractsESITests(t, db, factory, contracttest.Owner{
		ESIPath: "characters",
		Setup: func(t *testing.T) contracttest.Fixture {
			c := factory.CreateCharacter()
			factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
			factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
			return contracttest.Fixture{
				OwnerID:             c.ID,
				IssuerID:            c.ID,
				IssuerCorporationID: c.EveCharacter.Corporation.ID,
			}
		},
		Sync: func(ctx context.Context, characterID int64) error {
			_, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
				characterID: characterID,
				forceUpdate: true,
				section:     app.SectionCharacterContracts,
			})
			return err
		},
		CreateContract: func(t *testing.T, characterID int64, arg contracttest.CreateParams) *app.Contract {
			o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
				CharacterID: characterID,
				Status:      arg.Status,
				Type:        arg.Type,
			})
			return &o.Contract
		},
		CreateBid: func(t *testing.T, contractPK, bidID int64, amount float64) {
			factory.CreateCharacterContractBid(storage.CreateContractBidParams{
				Amount:     amount,
				BidID:      bidID,
				ContractID: contractPK,
			})
		},
		GetContract: func(ctx context.Context, characterID, contractID int64) (*app.Contract, error) {
			o, err := st.GetCharacterContract(ctx, characterID, contractID)
			if err != nil {
				return nil, err
			}
			return &o.Contract, nil
		},
		ListContractIDs: st.ListCharacterContractIDs,
		ListItems:       st.ListCharacterContractItems,
		ListBids:        st.ListCharacterContractBids,
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
