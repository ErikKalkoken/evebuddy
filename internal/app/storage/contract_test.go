package storage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

// TestContractEnumsRoundTrip ensures every enum value is stored and read back unchanged for both owners.
func TestContractEnumsRoundTrip(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	type values struct {
		availability app.ContractAvailability
		status       app.ContractStatus
		typ          app.ContractType
	}
	var cases []values
	for _, v := range []app.ContractAvailability{
		app.ContractAvailabilityAlliance,
		app.ContractAvailabilityCorporation,
		app.ContractAvailabilityPrivate,
		app.ContractAvailabilityPublic,
	} {
		cases = append(cases, values{v, app.ContractStatusOutstanding, app.ContractTypeItemExchange})
	}
	for _, v := range []app.ContractStatus{
		app.ContractStatusCancelled,
		app.ContractStatusDeleted,
		app.ContractStatusFailed,
		app.ContractStatusFinished,
		app.ContractStatusFinishedContractor,
		app.ContractStatusFinishedIssuer,
		app.ContractStatusInProgress,
		app.ContractStatusOutstanding,
		app.ContractStatusRejected,
		app.ContractStatusReversed,
	} {
		cases = append(cases, values{app.ContractAvailabilityPublic, v, app.ContractTypeItemExchange})
	}
	for _, v := range []app.ContractType{
		app.ContractTypeAuction,
		app.ContractTypeCourier,
		app.ContractTypeItemExchange,
		app.ContractTypeLoan,
		app.ContractTypeUnknown,
	} {
		cases = append(cases, values{app.ContractAvailabilityPublic, app.ContractStatusOutstanding, v})
	}
	for _, tc := range cases {
		name := tc.availability.String() + "/" + tc.status.String() + "/" + tc.typ.String()
		t.Run("character "+name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			o := f.CreateCharacterContract(storage.CreateCharacterContractParams{
				Availability:   tc.availability,
				Status:         tc.status,
				StatusNotified: tc.status,
				Type:           tc.typ,
			})
			got, err := st.GetCharacterContract(t.Context(), o.CharacterID, o.ContractID)
			require.NoError(t, err)
			xassert.Equal(t, tc.availability, got.Availability)
			xassert.Equal(t, tc.status, got.Status)
			xassert.Equal(t, tc.status, got.StatusNotified)
			xassert.Equal(t, tc.typ, got.Type)
		})
		t.Run("corporation "+name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			o := f.CreateCorporationContract(storage.CreateCorporationContractParams{
				Availability:   tc.availability,
				Status:         tc.status,
				StatusNotified: tc.status,
				Type:           tc.typ,
			})
			got, err := st.GetCorporationContract(t.Context(), o.CorporationID, o.ContractID)
			require.NoError(t, err)
			xassert.Equal(t, tc.availability, got.Availability)
			xassert.Equal(t, tc.status, got.Status)
			xassert.Equal(t, tc.status, got.StatusNotified)
			xassert.Equal(t, tc.typ, got.Type)
		})
	}
}

func TestCalculateContractsEscrow(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	t.Run("should sum courier collateral and winning bids of a character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := f.CreateCharacterFull()
		f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
			CharacterID: c.ID,
			AcceptorID:  c.ID,
			Collateral:  optional.New(100.0),
			Status:      app.ContractStatusInProgress,
		})
		o := f.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: c.ID,
			Type:        app.ContractTypeAuction,
			Status:      app.ContractStatusOutstanding,
		})
		f.CreateCharacterContractBid(storage.CreateContractBidParams{
			Amount:     12.5,
			BidderID:   c.ID,
			ContractID: o.ID,
		})
		got, err := st.CalculateCharacterContractsEscrow(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 112.5, got)
	})
	t.Run("should return zero for character without contracts", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := f.CreateCharacterFull()
		got, err := st.CalculateCharacterContractsEscrow(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0.0, got)
	})
	t.Run("should sum courier collateral and winning bids of a corporation", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := f.CreateCorporation()
		f.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
		f.CreateCorporationContractCourier(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			AcceptorID:    c.ID,
			Collateral:    optional.New(100.0),
			Status:        app.ContractStatusInProgress,
		})
		o := f.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID: c.ID,
			Type:          app.ContractTypeAuction,
			Status:        app.ContractStatusOutstanding,
		})
		f.CreateCorporationContractBid(storage.CreateContractBidParams{
			Amount:     12.5,
			BidderID:   c.ID,
			ContractID: o.ID,
		})
		got, err := st.CalculateCorporationContractsEscrow(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 112.5, got)
	})
	t.Run("should return zero for corporation without contracts", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := f.CreateCorporation()
		got, err := st.CalculateCorporationContractsEscrow(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0.0, got)
	})
}

func TestContractLocations(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	check := func(t *testing.T, startLocation, endLocation *app.EveLocation, got app.Contract) {
		t.Helper()
		for _, x := range []struct {
			name        string
			want        *app.EveLocation
			location    optional.Optional[*app.EveLocationShort]
			solarSystem optional.Optional[*app.EntityShort]
		}{
			{"start", startLocation, got.StartLocation, got.StartSolarSystem},
			{"end", endLocation, got.EndLocation, got.EndSolarSystem},
		} {
			es := x.want.SolarSystem.MustValue()
			xassert.Equal(t, &app.EveLocationShort{
				ID:             x.want.ID,
				Name:           optional.New(x.want.Name),
				SecurityStatus: optional.New(es.SecurityStatus),
			}, x.location.MustValue())
			xassert.Equal(t, &app.EntityShort{ID: es.ID, Name: es.Name}, x.solarSystem.MustValue())
		}
	}
	t.Run("character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		startLocation := f.CreateEveLocationStructure()
		endLocation := f.CreateEveLocationStructure()
		o := f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
			StartLocationID: optional.New(startLocation.ID),
			EndLocationID:   optional.New(endLocation.ID),
		})
		got, err := st.GetCharacterContract(t.Context(), o.CharacterID, o.ContractID)
		require.NoError(t, err)
		check(t, startLocation, endLocation, got.Contract)
	})
	t.Run("corporation", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		startLocation := f.CreateEveLocationStructure()
		endLocation := f.CreateEveLocationStructure()
		o := f.CreateCorporationContractCourier(storage.CreateCorporationContractParams{
			StartLocationID: optional.New(startLocation.ID),
			EndLocationID:   optional.New(endLocation.ID),
		})
		got, err := st.GetCorporationContract(t.Context(), o.CorporationID, o.ContractID)
		require.NoError(t, err)
		check(t, startLocation, endLocation, got.Contract)
	})
	t.Run("without locations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		o := f.CreateCharacterContract(storage.CreateCharacterContractParams{Type: app.ContractTypeItemExchange})
		got, err := st.GetCharacterContract(t.Context(), o.CharacterID, o.ContractID)
		require.NoError(t, err)
		xassert.Empty(t, got.StartLocation)
		xassert.Empty(t, got.EndLocation)
		xassert.Empty(t, got.StartSolarSystem)
		xassert.Empty(t, got.EndSolarSystem)
	})
}

func TestCalculateContractsAuctionEscrowTiedBids(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	early := time.Now().UTC().Add(-2 * time.Hour)
	late := early.Add(time.Hour)
	cases := []struct {
		name      string
		ownerDate time.Time
		otherDate time.Time
		want      float64
	}{
		{"should count tied bid when owner bid first", early, late, 100},
		{"should not count tied bid when other bidder bid first", late, early, 0},
	}
	for _, tc := range cases {
		t.Run("character: "+tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			c := f.CreateCharacterFull()
			o := f.CreateCharacterContract(storage.CreateCharacterContractParams{
				CharacterID: c.ID,
				Type:        app.ContractTypeAuction,
				Status:      app.ContractStatusOutstanding,
			})
			f.CreateCharacterContractBid(storage.CreateContractBidParams{ContractID: o.ID, Amount: 100, BidderID: c.ID, DateBid: tc.ownerDate})
			f.CreateCharacterContractBid(storage.CreateContractBidParams{ContractID: o.ID, Amount: 100, DateBid: tc.otherDate})
			got, err := st.CalculateCharacterContractsAuctionEscrow(t.Context(), c.ID)
			require.NoError(t, err)
			xassert.Equal(t, tc.want, got)
		})
		t.Run("corporation: "+tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			c := f.CreateCorporation()
			f.CreateEveEntityCorporation(app.EveEntity{ID: c.ID})
			o := f.CreateCorporationContract(storage.CreateCorporationContractParams{
				CorporationID: c.ID,
				Type:          app.ContractTypeAuction,
				Status:        app.ContractStatusOutstanding,
			})
			f.CreateCorporationContractBid(storage.CreateContractBidParams{ContractID: o.ID, Amount: 100, BidderID: c.ID, DateBid: tc.ownerDate})
			f.CreateCorporationContractBid(storage.CreateContractBidParams{ContractID: o.ID, Amount: 100, DateBid: tc.otherDate})
			got, err := st.CalculateCorporationContractsAuctionEscrow(t.Context(), c.ID)
			require.NoError(t, err)
			xassert.Equal(t, tc.want, got)
		})
	}
}
