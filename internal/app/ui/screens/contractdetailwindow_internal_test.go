package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestFetchContractBids(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	cases := []struct {
		name          string
		isCorporation bool
		createAuction func() int64
		createBid     func(contractPK int64, amount float64)
	}{
		{
			"character",
			false,
			func() int64 {
				return factory.CreateCharacterContract(storage.CreateCharacterContractParams{Type: app.ContractTypeAuction}).ID
			},
			func(contractPK int64, amount float64) {
				factory.CreateCharacterContractBid(storage.CreateContractBidParams{ContractID: contractPK, Amount: amount})
			},
		},
		{
			"corporation",
			true,
			func() int64 {
				return factory.CreateCorporationContract(storage.CreateCorporationContractParams{Type: app.ContractTypeAuction}).ID
			},
			func(contractPK int64, amount float64) {
				factory.CreateCorporationContractBid(storage.CreateContractBidParams{ContractID: contractPK, Amount: amount})
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name+": auction without bids", func(t *testing.T) {
			testutil.MustTruncateTables(db)
			r := contractRow{contractType: app.ContractTypeAuction, isCorporation: tc.isCorporation, objectID: tc.createAuction()}
			count, top, err := fetchContractBids(t.Context(), u, r)
			require.NoError(t, err)
			assert.Equal(t, 0, count)
			assert.Equal(t, 0.0, top)
		})
		t.Run(tc.name+": auction with bids", func(t *testing.T) {
			testutil.MustTruncateTables(db)
			r := contractRow{contractType: app.ContractTypeAuction, isCorporation: tc.isCorporation, objectID: tc.createAuction()}
			tc.createBid(r.objectID, 100)
			tc.createBid(r.objectID, 300)
			count, top, err := fetchContractBids(t.Context(), u, r)
			require.NoError(t, err)
			assert.Equal(t, 2, count)
			assert.Equal(t, 300.0, top)
		})
	}
	t.Run("no bids for other contract types", func(t *testing.T) {
		count, top, err := fetchContractBids(t.Context(), u, contractRow{contractType: app.ContractTypeCourier})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.Equal(t, 0.0, top)
	})
}

func TestFetchContractItems(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	t.Run("character contract with items", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{Type: app.ContractTypeItemExchange})
		factory.CreateCharacterContractItem(storage.CreateContractItemParams{ContractID: o.ID})
		r := contractRow{contractType: app.ContractTypeItemExchange, objectID: o.ID}
		items, err := fetchContractItems(t.Context(), u, r)
		require.NoError(t, err)
		assert.Len(t, items, 1)
	})
	t.Run("corporation contract with items", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		o := factory.CreateCorporationContract(storage.CreateCorporationContractParams{Type: app.ContractTypeItemExchange})
		factory.CreateCorporationContractItem(storage.CreateContractItemParams{ContractID: o.ID})
		r := contractRow{contractType: app.ContractTypeItemExchange, isCorporation: true, objectID: o.ID}
		items, err := fetchContractItems(t.Context(), u, r)
		require.NoError(t, err)
		assert.Len(t, items, 1)
	})
	t.Run("courier contract has no items", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{Type: app.ContractTypeCourier})
		factory.CreateCharacterContractItem(storage.CreateContractItemParams{ContractID: o.ID})
		r := contractRow{contractType: app.ContractTypeCourier, objectID: o.ID}
		items, err := fetchContractItems(t.Context(), u, r)
		require.NoError(t, err)
		assert.Empty(t, items)
	})
}
