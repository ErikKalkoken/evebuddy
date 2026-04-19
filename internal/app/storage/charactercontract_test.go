package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCharacterContract(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		issuer := factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		issuerCorporation := c.EveCharacter.Corporation
		dateExpired := time.Now().Add(12 * time.Hour).UTC()
		dateIssued := time.Now().UTC()
		arg := storage.CreateCharacterContractParams{
			Availability:        app.ContractAvailabilityPrivate,
			CharacterID:         c.ID,
			ContractID:          42,
			DateExpired:         dateExpired,
			DateIssued:          dateIssued,
			IssuerCorporationID: issuerCorporation.ID,
			IssuerID:            issuer.ID,
			Status:              app.ContractStatusOutstanding,
			Type:                app.ContractTypeCourier,
		}
		// when
		id, err := st.CreateCharacterContract(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterContract(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, id, o.ID)
		xassert.Equal(t, issuer, o.Issuer)
		xassert.Equal(t, dateExpired, o.DateExpired)
		xassert.Equal(t, app.ContractAvailabilityPrivate, o.Availability)
		xassert.Equal(t, app.ContractStatusOutstanding, o.Status)
		xassert.Equal(t, app.ContractTypeCourier, o.Type)
		assert.WithinDuration(t, time.Now().UTC(), o.UpdatedAt, 5*time.Second)
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		issuer := factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		issuerCorporation := c.EveCharacter.Corporation
		dateExpired := time.Now().Add(12 * time.Hour).UTC()
		dateIssued := time.Now().UTC()
		startLocation := factory.CreateEveLocationStructure()
		endLocation := factory.CreateEveLocationStructure()
		arg := storage.CreateCharacterContractParams{
			Availability:        app.ContractAvailabilityPrivate,
			CharacterID:         c.ID,
			ContractID:          42,
			DateExpired:         dateExpired,
			DateIssued:          dateIssued,
			IssuerCorporationID: issuerCorporation.ID,
			IssuerID:            issuer.ID,
			Status:              app.ContractStatusOutstanding,
			Type:                app.ContractTypeCourier,
			EndLocationID:       optional.New(endLocation.ID),
			StartLocationID:     optional.New(startLocation.ID),
		}
		// when
		id, err := st.CreateCharacterContract(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterContract(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, id, o.ID)
		xassert.Equal(t, issuer, o.Issuer)
		xassert.Equal(t, dateExpired, o.DateExpired)
		xassert.Equal(t, app.ContractAvailabilityPrivate, o.Availability)
		xassert.Equal(t, app.ContractStatusOutstanding, o.Status)
		xassert.Equal(t, app.ContractTypeCourier, o.Type)
		xassert.Equal(t, endLocation.ToShort(), o.EndLocation.ValueOrZero())
		xassert.Equal(t, startLocation.ToShort(), o.StartLocation.ValueOrZero())
		xassert.Equal(t, endLocation.SolarSystem.ValueOrZero().ID, o.EndSolarSystem.ValueOrZero().ID)
		xassert.Equal(t, endLocation.SolarSystem.ValueOrZero().Name, o.EndSolarSystem.ValueOrZero().Name)
		xassert.Equal(t, startLocation.SolarSystem.ValueOrZero().ID, o.StartSolarSystem.ValueOrZero().ID)
		xassert.Equal(t, startLocation.SolarSystem.ValueOrZero().Name, o.StartSolarSystem.ValueOrZero().Name)
		assert.WithinDuration(t, time.Now().UTC(), o.UpdatedAt, 5*time.Second)
	})
	t.Run("can update contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			UpdatedAt: time.Now().UTC().Add(-5 * time.Second),
		})
		dateAccepted := time.Now().UTC()
		dateCompleted := time.Now().UTC()
		arg2 := storage.UpdateCharacterContractParams{
			CharacterID:   o1.CharacterID,
			ContractID:    o1.ContractID,
			DateAccepted:  optional.New(dateAccepted),
			DateCompleted: optional.New(dateCompleted),
			Status:        app.ContractStatusFinished,
		}
		// when
		err := st.UpdateCharacterContract(ctx, arg2)
		// then
		require.NoError(t, err)
		o2, err := st.GetCharacterContract(ctx, o1.CharacterID, o1.ContractID)
		require.NoError(t, err)
		xassert.Equal(t, app.ContractStatusFinished, o2.Status)
		xassert.Equal(t, optional.New(dateAccepted), o2.DateAccepted)
		xassert.Equal(t, optional.New(dateCompleted), o2.DateCompleted)
		assert.Less(t, o1.UpdatedAt, o2.UpdatedAt)
	})
	t.Run("can update notified", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			UpdatedAt: time.Now().UTC().Add(-5 * time.Second),
		})
		// when
		err := st.UpdateCharacterContractNotified(ctx, o1.ID, app.ContractStatusInProgress)
		// then
		require.NoError(t, err)
		o2, err := st.GetCharacterContract(ctx, o1.CharacterID, o1.ContractID)
		require.NoError(t, err)
		xassert.Equal(t, app.ContractStatusInProgress, o2.StatusNotified)
		assert.Less(t, o1.UpdatedAt, o2.UpdatedAt)
	})
	t.Run("can list IDs of existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		e1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		e2 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		e3 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		// when
		got, err := st.ListCharacterContractIDs(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(e1.ContractID, e2.ContractID, e3.ContractID)
		xassert.Equal(t, want, got)
	})
	t.Run("can list contracts for multiple characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character1 := factory.CreateCharacter()
		c1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: character1.ID})
		c2 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: character1.ID})
		character2 := factory.CreateCharacter()
		c3 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: character2.ID})
		// when
		oo, err := st.ListAllCharacterContracts(ctx)
		// then
		require.NoError(t, err)
		want := set.Of(c1.ID, c2.ID, c3.ID)
		got := set.Of(xslices.Map(oo, func(x *app.CharacterContract) int64 {
			return x.ID
		})...)
		xassert.Equal(t, want, got)
	})
	t.Run("can list existing contracts for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		// when
		oo, err := st.ListCharacterContracts(ctx, c.ID)
		// then
		require.NoError(t, err)
		assert.Len(t, oo, 1)
		xassert.Equal(t, o.ID, oo[0].ID)
	})
	t.Run("can delete contracts for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		e1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		e2 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		e3 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		// when
		err := st.DeleteCharacterContracts(ctx, c.ID, set.Of(e1.ContractID))
		// then
		require.NoError(t, err)
		got, err := st.ListCharacterContractIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(e2.ContractID, e3.ContractID)
		xassert.Equal(t, want, got)
	})
}

func TestCharacterContractBid(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterContract()
		bidder := factory.CreateEveEntityCharacter()
		const (
			amount = 123.45
			bidID  = 12345
		)
		dateBid := time.Now().UTC()
		arg := storage.CreateCharacterContractBidParams{
			ContractID: c.ID,
			Amount:     amount,
			BidID:      bidID,
			BidderID:   bidder.ID,
			DateBid:    dateBid,
		}
		// when
		err := st.CreateCharacterContractBid(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterContractBid(ctx, c.ID, bidID)
		require.NoError(t, err)
		assert.InDelta(t, amount, o.Amount, 0.1)
		xassert.Equal(t, bidder, o.Bidder)
		xassert.Equal(t, dateBid, o.DateBid)
	})
	t.Run("can list existing bids", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterContract()
		b1 := factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{ContractID: c.ID})
		b2 := factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{ContractID: c.ID})
		// when
		oo, err := st.ListCharacterContractBids(ctx, c.ID)
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.CharacterContractBid) int64 {
			return x.BidID
		}))
		want := set.Of(b1.BidID, b2.BidID)
		xassert.Equal(t, want, got)
	})
	t.Run("can list bid IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterContract()
		b1 := factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{ContractID: c.ID})
		b2 := factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{ContractID: c.ID})
		// when
		got, err := st.ListCharacterContractBidIDs(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(b1.BidID, b2.BidID)
		xassert.Equal(t, want, got)
	})
}

func TestCharacterContractItem(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterContract()
		et := factory.CreateEveType()
		arg := storage.CreateCharacterContractItemParams{
			ContractID:  c.ID,
			IsIncluded:  true,
			IsSingleton: true,
			Quantity:    7,
			RawQuantity: optional.New[int64](-5),
			RecordID:    42,
			TypeID:      et.ID,
		}
		// when
		err := st.CreateCharacterContractItem(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterContractItem(ctx, c.ID, 42)
		require.NoError(t, err)
		assert.True(t, o.IsIncluded)
		assert.True(t, o.IsSingleton)
		xassert.Equal(t, 7, o.Quantity)
		xassert.Equal(t, -5, o.RawQuantity.ValueOrZero())
		xassert.Equal(t, et, o.Type)
	})
	t.Run("can list existing items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterContract()
		i1 := factory.CreateCharacterContractItem(storage.CreateCharacterContractItemParams{ContractID: c.ID})
		i2 := factory.CreateCharacterContractItem(storage.CreateCharacterContractItemParams{ContractID: c.ID})
		// when
		oo, err := st.ListCharacterContractItems(ctx, c.ID)
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.CharacterContractItem) int64 {
			return x.RecordID
		}))
		want := set.Of(i1.RecordID, i2.RecordID)
		xassert.Equal(t, want, got)
	})
}

func TestCalculateCharacterContractItemsValue(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()

	const characterID = 42

	cases := []struct {
		name         string
		contractType app.ContractType
		status       app.ContractStatus
		price        float64
		want         float64
	}{
		{
			name:         "should include courier outstanding contract",
			contractType: app.ContractTypeCourier,
			status:       app.ContractStatusOutstanding,
			price:        10.0,
			want:         10.0,
		},
		{
			name:         "should include in progress courier contract",
			contractType: app.ContractTypeCourier,
			status:       app.ContractStatusInProgress,
			price:        10.0,
			want:         10.0,
		},
		{
			name:         "should include outstanding auction contract",
			contractType: app.ContractTypeAuction,
			status:       app.ContractStatusOutstanding,
			price:        10.0,
			want:         10.0,
		},
		{
			name:         "should exclude finished courier contract",
			contractType: app.ContractTypeCourier,
			status:       app.ContractStatusFinished,
			price:        10.0,
			want:         0,
		},
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
			o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
				CharacterID: c.ID,
				Status:      tc.status,
				Type:        tc.contractType,
			})
			u := factory.CreateCharacterContractItem(storage.CreateCharacterContractItemParams{
				ContractID: o.ID,
				IsIncluded: true,
				Quantity:   1,
			})
			factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
				TypeID:       u.Type.ID,
				AveragePrice: optional.New(tc.price),
			})

			// when
			got, err := st.CalculateCharacterContractItemsValue(t.Context(), c.ID)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("can calculate items value for multiple contracts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: c.ID,
			Status:      app.ContractStatusOutstanding,
			Type:        app.ContractTypeAuction,
		})
		u1 := factory.CreateCharacterContractItem(storage.CreateCharacterContractItemParams{
			ContractID: o1.ID,
			IsIncluded: true,
			Quantity:   1,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       u1.Type.ID,
			AveragePrice: optional.New(100.1),
		})
		o2 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: c.ID,
			Status:      app.ContractStatusOutstanding,
			Type:        app.ContractTypeAuction,
		})
		u2 := factory.CreateCharacterContractItem(storage.CreateCharacterContractItemParams{
			ContractID: o2.ID,
			IsIncluded: true,
			Quantity:   2,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       u2.Type.ID,
			AveragePrice: optional.New(200.2),
		})

		// when
		got, err := st.CalculateCharacterContractItemsValue(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		assert.Equal(t, 500.5, got)
	})

	t.Run("should ignore blueprints", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		blueprintCategory := factory.CreateEveCategory(storage.CreateEveCategoryParams{
			ID:   app.EveCategoryBlueprint,
			Name: "Blueprint",
		})
		blueprintGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{
			CategoryID: blueprintCategory.ID,
		})
		blueprintType := factory.CreateEveType(storage.CreateEveTypeParams{
			GroupID: blueprintGroup.ID,
		})
		o1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: c.ID,
			Status:      app.ContractStatusOutstanding,
			Type:        app.ContractTypeAuction,
		})
		factory.CreateCharacterContractItem(storage.CreateCharacterContractItemParams{
			ContractID: o1.ID,
			IsIncluded: true,
			Quantity:   1,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       blueprintType.ID,
			AveragePrice: optional.New(66.6),
		})

		// when
		got, err := st.CalculateCharacterOrderItemsValue(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0.0, got)
	})
}

func TestCalculateCharacterContractsCourierEscrow(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	characterID := int64(42)

	cases := []struct {
		name string

		acceptorID   int64
		contractType app.ContractType
		collateral   optional.Optional[float64]
		want         float64
	}{
		{"should include courier contract accepted by character", characterID, app.ContractTypeCourier, optional.New(123.0), 123.0},
		{"should not include courier contract accepted by other", 0, app.ContractTypeCourier, optional.New(123.0), 0.0},
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
			got, err := st.CalculateCharacterContractsCourierEscrow(t.Context(), c.ID)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("should sum over multiple courier contracts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
		factory.CreateCharacterContract() // should be ignored

		// when
		got, err := st.CalculateCharacterContractsCourierEscrow(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		assert.Equal(t, 300.0, got)
	})
}

func TestCalculateCharacterContractsAuctionEscrow(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("should include when own bid is highest", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: c.ID,
			Type:        app.ContractTypeAuction,
			Status:      app.ContractStatusInProgress,
		})
		factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{
			Amount:     12.3,
			BidderID:   c.ID,
			ContractID: o.ID,
		})
		factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{
			Amount:     3,
			ContractID: o.ID,
		})

		// when
		got, err := st.CalculateCharacterContractsAuctionEscrow(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		assert.Equal(t, 12.3, got)
	})

	t.Run("should not include when own bid is not highest", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: c.ID,
			Type:        app.ContractTypeAuction,
			Status:      app.ContractStatusInProgress,
		})
		factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{
			Amount:     12.3,
			BidderID:   c.ID,
			ContractID: o.ID,
		})
		factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{
			Amount:     15,
			ContractID: o.ID,
		})

		// when
		got, err := st.CalculateCharacterContractsAuctionEscrow(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0.0, got)
	})
}
