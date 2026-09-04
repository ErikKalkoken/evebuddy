package storage_test

import (
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

func TestCorporationContract(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can create new minimal courier", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCorporation()
		issuer := f.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		issuerCorporation := c.EveCorporation
		dateExpired := time.Now().Add(12 * time.Hour).UTC()
		dateIssued := time.Now().UTC()

		// when
		id, err := st.CreateCorporationContract(t.Context(), storage.CreateCorporationContractParams{
			Availability:        app.ContractAvailabilityPrivate,
			CorporationID:       c.ID,
			ContractID:          42,
			DateExpired:         dateExpired,
			DateIssued:          dateIssued,
			IssuerCorporationID: issuerCorporation.ID,
			IssuerID:            issuer.ID,
			Status:              app.ContractStatusOutstanding,
			Type:                app.ContractTypeCourier,
		})

		// then
		require.NoError(t, err)
		o, err := st.GetCorporationContract(t.Context(), c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, id, o.ID)
		xassert.Equal(t, issuer, o.Issuer)
		xassert.Equal(t, dateExpired, o.DateExpired)
		xassert.Equal(t, app.ContractAvailabilityPrivate, o.Availability)
		xassert.Equal(t, app.ContractStatusOutstanding, o.Status)
		xassert.Equal(t, app.ContractTypeCourier, o.Type)
		assert.WithinDuration(t, time.Now().UTC(), o.UpdatedAt, 5*time.Second)
	})

	t.Run("can create new full courier", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCorporation()
		issuer := f.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		issuerCorporation := c.EveCorporation
		dateExpired := time.Now().Add(12 * time.Hour).UTC()
		dateIssued := time.Now().UTC()
		startLocation := f.CreateEveLocationStructure()
		endLocation := f.CreateEveLocationStructure()

		// when
		id, err := st.CreateCorporationContract(t.Context(), storage.CreateCorporationContractParams{
			Availability:        app.ContractAvailabilityPrivate,
			CorporationID:       c.ID,
			ContractID:          42,
			DateExpired:         dateExpired,
			DateIssued:          dateIssued,
			IssuerCorporationID: issuerCorporation.ID,
			IssuerID:            issuer.ID,
			Status:              app.ContractStatusOutstanding,
			Type:                app.ContractTypeCourier,
			EndLocationID:       optional.New(endLocation.ID),
			StartLocationID:     optional.New(startLocation.ID),
		})

		// then
		require.NoError(t, err)
		o, err := st.GetCorporationContract(t.Context(), c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, id, o.ID)
		xassert.Equal(t, issuer, o.Issuer)
		xassert.Equal(t, dateExpired, o.DateExpired)
		xassert.Equal(t, app.ContractAvailabilityPrivate, o.Availability)
		xassert.Equal(t, app.ContractStatusOutstanding, o.Status)
		xassert.Equal(t, app.ContractTypeCourier, o.Type)
		xassert.EqualOptional(t, endLocation.ToEveLocationShort(), o.EndLocation)
		xassert.EqualOptional(t, startLocation.ToEveLocationShort(), o.StartLocation)
		xassert.Equal(t, endLocation.SolarSystem.MustValue().ID, o.EndSolarSystem.MustValue().ID)
		xassert.Equal(t, endLocation.SolarSystem.MustValue().Name, o.EndSolarSystem.MustValue().Name)
		xassert.Equal(t, startLocation.SolarSystem.MustValue().ID, o.StartSolarSystem.MustValue().ID)
		xassert.Equal(t, startLocation.SolarSystem.MustValue().Name, o.StartSolarSystem.MustValue().Name)
		assert.WithinDuration(t, time.Now().UTC(), o.UpdatedAt, 5*time.Second)
	})

	t.Run("can create new minimal item exchange with item", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCorporation()
		issuer := f.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		issuerCorporation := c.EveCorporation
		dateExpired := time.Now().Add(12 * time.Hour).UTC()
		dateIssued := time.Now().UTC()
		item := f.CreateEveType()

		// when
		id, err := st.CreateCorporationContract(t.Context(), storage.CreateCorporationContractParams{
			Availability:        app.ContractAvailabilityPrivate,
			CorporationID:       c.ID,
			ContractID:          42,
			DateExpired:         dateExpired,
			DateIssued:          dateIssued,
			IssuerCorporationID: issuerCorporation.ID,
			IssuerID:            issuer.ID,
			Status:              app.ContractStatusOutstanding,
			Type:                app.ContractTypeItemExchange,
		})
		require.NoError(t, err)
		err2 := st.CreateCorporationContractItem(t.Context(), storage.CreateCorporationContractItemParams{
			ContractID: id,
			IsIncluded: true,
			Quantity:   1,
			RecordID:   42,
			TypeID:     item.ID,
		})

		// then
		require.NoError(t, err2)
		got, err := st.GetCorporationContract(t.Context(), c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, []string{item.Name + " x 1"}, got.Items)
	})

	t.Run("can create new minimal item exchange without items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCorporation()
		issuer := f.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		issuerCorporation := c.EveCorporation
		dateExpired := time.Now().Add(12 * time.Hour).UTC()
		dateIssued := time.Now().UTC()

		// when
		_, err := st.CreateCorporationContract(t.Context(), storage.CreateCorporationContractParams{
			Availability:        app.ContractAvailabilityPrivate,
			CorporationID:       c.ID,
			ContractID:          42,
			DateExpired:         dateExpired,
			DateIssued:          dateIssued,
			IssuerCorporationID: issuerCorporation.ID,
			IssuerID:            issuer.ID,
			Status:              app.ContractStatusOutstanding,
			Type:                app.ContractTypeItemExchange,
		})

		// then
		require.NoError(t, err)
		got, err := st.GetCorporationContract(t.Context(), c.ID, 42)
		require.NoError(t, err)
		assert.Len(t, got.Items, 0)
	})

	t.Run("can update contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := f.CreateCorporationContract(storage.CreateCorporationContractParams{
			UpdatedAt: time.Now().UTC().Add(-5 * time.Second),
		})
		dateAccepted := time.Now().UTC()
		dateCompleted := time.Now().UTC()
		arg2 := storage.UpdateCorporationContractParams{
			CorporationID: o1.CorporationID,
			ContractID:    o1.ContractID,
			DateAccepted:  optional.New(dateAccepted),
			DateCompleted: optional.New(dateCompleted),
			Status:        app.ContractStatusFinished,
		}

		// when
		err := st.UpdateCorporationContract(t.Context(), arg2)

		// then
		require.NoError(t, err)
		o2, err := st.GetCorporationContract(t.Context(), o1.CorporationID, o1.ContractID)
		require.NoError(t, err)
		xassert.Equal(t, app.ContractStatusFinished, o2.Status)
		xassert.Equal(t, optional.New(dateAccepted), o2.DateAccepted)
		xassert.Equal(t, optional.New(dateCompleted), o2.DateCompleted)
		assert.Less(t, o1.UpdatedAt, o2.UpdatedAt)
	})

	t.Run("can update notified", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := f.CreateCorporationContract(storage.CreateCorporationContractParams{
			UpdatedAt: time.Now().UTC().Add(-5 * time.Second),
		})

		// when
		err := st.UpdateCorporationContractNotified(t.Context(), o1.ID, app.ContractStatusInProgress)

		// then
		require.NoError(t, err)
		o2, err := st.GetCorporationContract(t.Context(), o1.CorporationID, o1.ContractID)
		require.NoError(t, err)
		xassert.Equal(t, app.ContractStatusInProgress, o2.StatusNotified)
		assert.Less(t, o1.UpdatedAt, o2.UpdatedAt)
	})

	t.Run("can list IDs of existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCorporation()
		e1 := f.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		e2 := f.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		e3 := f.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})

		// when
		got, err := st.ListCorporationContractIDs(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		want := set.Of(e1.ContractID, e2.ContractID, e3.ContractID)
		xassert.Equal(t, want, got)
	})

	t.Run("can list contracts for a corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCorporation()
		o1 := f.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		o2 := f.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		f.CreateCorporationContract()

		// when
		oo, err := st.ListCorporationContracts(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		want := set.Of(o1.ID, o2.ID)
		got := set.Of(xslices.Map(oo, func(x *app.CorporationContract) int64 {
			return x.ID
		})...)
		xassert.Equal(t, want, got)
	})

	t.Run("can delete contracts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCorporation()
		e1 := f.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		e2 := f.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})
		e3 := f.CreateCorporationContract(storage.CreateCorporationContractParams{CorporationID: c.ID})

		// when
		err := st.DeleteCorporationContracts(t.Context(), c.ID, set.Of(e1.ContractID))

		// then
		require.NoError(t, err)
		got, err := st.ListCorporationContractIDs(t.Context(), c.ID)
		require.NoError(t, err)
		want := set.Of(e2.ContractID, e3.ContractID)
		xassert.Equal(t, want, got)
	})
}

func TestCorporationContractBid(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCorporationContract()
		bidder := f.CreateEveEntityCorporation()
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
		err := st.CreateCorporationContractBid(t.Context(), arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCorporationContractBid(t.Context(), c.ID, bidID)
		require.NoError(t, err)
		assert.InDelta(t, amount, o.Amount, 0.1)
		xassert.Equal(t, bidder, o.Bidder)
		xassert.Equal(t, dateBid, o.DateBid)
	})
	t.Run("can list existing bids", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCorporationContract()
		b1 := f.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{ContractID: c.ID})
		b2 := f.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{ContractID: c.ID})
		// when
		oo, err := st.ListCorporationContractBids(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.CorporationContractBid) int64 {
			return x.BidID
		}))
		want := set.Of(b1.BidID, b2.BidID)
		xassert.Equal(t, want, got)
	})
	t.Run("can list bid IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCorporationContract()
		b1 := f.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{ContractID: c.ID})
		b2 := f.CreateCorporationContractBid(storage.CreateCorporationContractBidParams{ContractID: c.ID})
		// when
		got, err := st.ListCorporationContractBidIDs(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(b1.BidID, b2.BidID)
		xassert.Equal(t, want, got)
	})
}

func TestCorporationContractItem(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCorporationContract()
		et := f.CreateEveType()

		// when
		err := st.CreateCorporationContractItem(t.Context(), storage.CreateCorporationContractItemParams{
			ContractID:  c.ID,
			IsIncluded:  true,
			IsSingleton: true,
			Quantity:    7,
			RawQuantity: optional.New[int64](-5),
			RecordID:    42,
			TypeID:      et.ID,
		})

		// then
		require.NoError(t, err)
		o, err := st.GetCorporationContractItem(t.Context(), c.ID, 42)
		require.NoError(t, err)
		assert.True(t, o.IsIncluded)
		assert.True(t, o.IsSingleton)
		xassert.Equal(t, 7, o.Quantity)
		xassert.EqualOptional(t, -5, o.RawQuantity)
		xassert.Equal(t, et, o.Type)
	})

	t.Run("can list existing items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateCorporationContract()
		i1 := f.CreateCorporationContractItem(storage.CreateCorporationContractItemParams{ContractID: c.ID})
		i2 := f.CreateCorporationContractItem(storage.CreateCorporationContractItemParams{ContractID: c.ID})

		// when
		oo, err := st.ListCorporationContractItems(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.CorporationContractItem) int64 {
			return x.RecordID
		}))
		want := set.Of(i1.RecordID, i2.RecordID)
		xassert.Equal(t, want, got)
	})
}
