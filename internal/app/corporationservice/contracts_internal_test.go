package corporationservice

import (
	"context"
	"testing"

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
		factory.CreateCorporationContractBid(storage.CreateContractBidParams{ContractID: o.ID})
		factory.CreateCorporationContractBid(storage.CreateContractBidParams{ContractID: o.ID})
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
		factory.CreateCorporationContractBid(storage.CreateContractBidParams{ContractID: o.ID, Amount: 100})
		top := factory.CreateCorporationContractBid(storage.CreateContractBidParams{ContractID: o.ID, Amount: 500})
		factory.CreateCorporationContractBid(storage.CreateContractBidParams{ContractID: o.ID, Amount: 200})
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
		factory.CreateCorporationContractBid(storage.CreateContractBidParams{
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
		i1 := factory.CreateCorporationContractItem(storage.CreateContractItemParams{ContractID: o.ID})
		i2 := factory.CreateCorporationContractItem(storage.CreateContractItemParams{ContractID: o.ID})
		got, err := s.ListContractItems(ctx, o.ID)
		require.NoError(t, err)
		ids := set.Collect(xiter.MapSlice(got, func(x *app.ContractItem) int64 { return x.RecordID }))
		xassert.Equal(t, set.Of(i1.RecordID, i2.RecordID), ids)
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
			factory.CreateCorporationContractBid(storage.CreateContractBidParams{
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
