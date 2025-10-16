package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCorporationContract(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		issuer := factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		issuerCorporation := c.EveCorporation
		dateExpired := time.Now().Add(12 * time.Hour).UTC()
		dateIssued := time.Now().UTC()
		arg := storage.CreateCorporationContractParams{
			Availability:        app.ContractAvailabilityPrivate,
			CorporationID:       c.ID,
			ContractID:          42,
			DateExpired:         dateExpired,
			DateIssued:          dateIssued,
			IssuerCorporationID: issuerCorporation.ID,
			IssuerID:            issuer.ID,
			Status:              app.ContractStatusOutstanding,
			Type:                app.ContractTypeCourier,
		}
		// when
		id, err := st.CreateCorporationContract(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCorporationContract(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, id, o.ID)
				assert.Equal(t, issuer, o.Issuer)
				assert.Equal(t, dateExpired, o.DateExpired)
				assert.Equal(t, app.ContractAvailabilityPrivate, o.Availability)
				assert.Equal(t, app.ContractStatusOutstanding, o.Status)
				assert.Equal(t, app.ContractTypeCourier, o.Type)
				assert.WithinDuration(t, time.Now().UTC(), o.UpdatedAt, 5*time.Second)
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		issuer := factory.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		issuerCorporation := c.EveCorporation
		dateExpired := time.Now().Add(12 * time.Hour).UTC()
		dateIssued := time.Now().UTC()
		startLocation := factory.CreateEveLocationStructure()
		endLocation := factory.CreateEveLocationStructure()
		arg := storage.CreateCorporationContractParams{
			Availability:        app.ContractAvailabilityPrivate,
			CorporationID:       c.ID,
			ContractID:          42,
			DateExpired:         dateExpired,
			DateIssued:          dateIssued,
			IssuerCorporationID: issuerCorporation.ID,
			IssuerID:            issuer.ID,
			Status:              app.ContractStatusOutstanding,
			Type:                app.ContractTypeCourier,
			EndLocationID:       endLocation.ID,
			StartLocationID:     startLocation.ID,
		}
		// when
		id, err := st.CreateCorporationContract(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCorporationContract(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, id, o.ID)
				assert.Equal(t, issuer, o.Issuer)
				assert.Equal(t, dateExpired, o.DateExpired)
				assert.Equal(t, app.ContractAvailabilityPrivate, o.Availability)
				assert.Equal(t, app.ContractStatusOutstanding, o.Status)
				assert.Equal(t, app.ContractTypeCourier, o.Type)
				assert.Equal(t, endLocation.ToShort(), o.EndLocation)
				assert.Equal(t, startLocation.ToShort(), o.StartLocation)
				assert.Equal(t, endLocation.SolarSystem.ID, o.EndSolarSystem.ID)
				assert.Equal(t, endLocation.SolarSystem.Name, o.EndSolarSystem.Name)
				assert.Equal(t, startLocation.SolarSystem.ID, o.StartSolarSystem.ID)
				assert.Equal(t, startLocation.SolarSystem.Name, o.StartSolarSystem.Name)
				assert.WithinDuration(t, time.Now().UTC(), o.UpdatedAt, 5*time.Second)
			}
		}
	})
	t.Run("can update contract", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		o1 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			UpdatedAt: time.Now().UTC().Add(-5 * time.Second),
		})
		dateAccepted := time.Now().UTC()
		dateCompleted := time.Now().UTC()
		arg2 := storage.UpdateCorporationContractParams{
			CorporationID: o1.CorporationID,
			ContractID:    o1.ContractID,
			DateAccepted:  dateAccepted,
			DateCompleted: dateCompleted,
			Status:        app.ContractStatusFinished,
		}
		// when
		err := st.UpdateCorporationContract(ctx, arg2)
		// then
		if assert.NoError(t, err) {
			o2, err := st.GetCorporationContract(ctx, o1.CorporationID, o1.ContractID)
			if assert.NoError(t, err) {
				assert.Equal(t, app.ContractStatusFinished, o2.Status)
				assert.Equal(t, optional.New(dateAccepted), o2.DateAccepted)
				assert.Equal(t, optional.New(dateCompleted), o2.DateCompleted)
				assert.Less(t, o1.UpdatedAt, o2.UpdatedAt)
			}
		}
	})
	t.Run("can update notified", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		o1 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			UpdatedAt: time.Now().UTC().Add(-5 * time.Second),
		})
		// when
		err := st.UpdateCorporationContractNotified(ctx, o1.ID, app.ContractStatusInProgress)
		// then
		if assert.NoError(t, err) {
			o2, err := st.GetCorporationContract(ctx, o1.CorporationID, o1.ContractID)
			if assert.NoError(t, err) {
				assert.Equal(t, app.ContractStatusInProgress, o2.StatusNotified)
				assert.Less(t, o1.UpdatedAt, o2.UpdatedAt)
			}
		}
	})
	t.Run("can list IDs of existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		e1 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		e2 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		e3 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		// when
		ids, err := st.ListCorporationContractIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.Of(ids...)
			want := set.Of([]int32{e1.ContractID, e2.ContractID, e3.ContractID}...)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can list contracts for a corporation", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		o1 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		o2 := factory.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		factory.CreateCorporationContract()
		// when
		oo, err := st.ListCorporationContracts(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := set.Of(o1.ID, o2.ID)
			got := set.Of(xslices.Map(oo, func(x *app.CorporationContract) int64 {
				return x.ID
			})...)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}

func TestCorporationContractBid(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporationContract()
		bidder := factory.CreateEveEntityCorporation()
		const (
			amount = 123.45
			bidID  = 12345
		)
		dateBid := time.Now().UTC()
		arg := storage.CreateCorporationContractBidParams{
			ContractID: c.ID,
			Amount:     amount,
			BidID:      bidID,
			BidderID:   bidder.ID,
			DateBid:    dateBid,
		}
		// when
		err := st.CreateCorporationContractBid(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCorporationContractBid(ctx, c.ID, bidID)
			if assert.NoError(t, err) {
				assert.InDelta(t, amount, o.Amount, 0.1)
				assert.Equal(t, bidder, o.Bidder)
				assert.Equal(t, dateBid, o.DateBid)
			}
		}
	})
	t.Run("can list existing bids", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporationContract()
		b1 := factory.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{ContractID: c.ID})
		b2 := factory.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{ContractID: c.ID})
		// when
		oo, err := st.ListCorporationContractBids(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.Collect(xiter.MapSlice(oo, func(x *app.CorporationContractBid) int32 {
				return x.BidID
			}))
			want := set.Of(b1.BidID, b2.BidID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can list bid IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporationContract()
		b1 := factory.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{ContractID: c.ID})
		b2 := factory.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{ContractID: c.ID})
		// when
		got, err := st.ListCorporationContractBidIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := set.Of(b1.BidID, b2.BidID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}

func TestCorporationContractItem(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporationContract()
		et := factory.CreateEveType()
		arg := storage.CreateCorporationContractItemParams{
			ContractID:  c.ID,
			IsIncluded:  true,
			IsSingleton: true,
			Quantity:    7,
			RawQuantity: -5,
			RecordID:    42,
			TypeID:      et.ID,
		}
		// when
		err := st.CreateCorporationContractItem(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCorporationContractItem(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.True(t, o.IsIncluded)
				assert.True(t, o.IsSingleton)
				assert.Equal(t, 7, o.Quantity)
				assert.Equal(t, -5, o.RawQuantity)
				assert.Equal(t, et, o.Type)
			}
		}
	})
	t.Run("can list existing items", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporationContract()
		i1 := factory.CreateCorporationContractItem(storage.CreateCorporationContractItemParams{ContractID: c.ID})
		i2 := factory.CreateCorporationContractItem(storage.CreateCorporationContractItemParams{ContractID: c.ID})
		// when
		oo, err := st.ListCorporationContractItems(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.Collect(xiter.MapSlice(oo, func(x *app.CorporationContractItem) int64 {
				return x.RecordID
			}))
			want := set.Of(i1.RecordID, i2.RecordID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}
