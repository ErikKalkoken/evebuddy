package screens

import (
	"fmt"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
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

func TestContractHighestBidDisplay(t *testing.T) {
	t.Run("shows none without bids", func(t *testing.T) {
		// when
		got := contractHighestBidDisplay(0, 0)
		// then
		xassert.Equal(t, "(None)", got)
	})
	t.Run("shows amount with single bid", func(t *testing.T) {
		// when
		got := contractHighestBidDisplay(1, 500)
		// then
		xassert.Equal(t, "500.00 ISK (1 bid)", got)
	})
	t.Run("shows amount with number of bids", func(t *testing.T) {
		// when
		got := contractHighestBidDisplay(3, 500)
		// then
		xassert.Equal(t, "500.00 ISK (3 bids)", got)
	})
}

func TestMakeEveEntityActionLabel2(t *testing.T) {
	t.Run("shows dash when entity is missing", func(t *testing.T) {
		// when
		got := makeEveEntityActionLabel2(optional.Optional[*app.EveEntity]{}, func(*app.EveEntity) {})
		// then
		xassert.Equal(t, "-", got.(*widget.Label).Text)
	})
	t.Run("shows link to entity", func(t *testing.T) {
		// given
		var shown *app.EveEntity
		o := &app.EveEntity{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter}
		x := makeEveEntityActionLabel2(optional.New(o), func(o *app.EveEntity) {
			shown = o
		})
		// when
		test.Tap(x.(*widget.Hyperlink))
		// then
		xassert.Equal(t, "Bruce Wayne", x.(*widget.Hyperlink).Text)
		xassert.Equal(t, o, shown)
	})
}

func TestMakeLocationLabel2(t *testing.T) {
	t.Run("shows question mark when location is missing", func(t *testing.T) {
		// when
		got := makeLocationLabel2(optional.Optional[*app.EveLocationShort]{}, func(int64) {})
		// then
		xassert.Equal(t, "?", got.(*widget.Label).Text)
	})
	t.Run("shows link to location", func(t *testing.T) {
		// given
		var shown int64
		el := &app.EveLocationShort{ID: 42, Name: optional.New("Jita IV - Moon 4")}
		x := makeLocationLabel2(optional.New(el), func(id int64) {
			shown = id
		})
		// when
		test.Tap(x.(*widget.Hyperlink))
		// then
		xassert.Equal(t, "Jita IV - Moon 4", x.(*widget.Hyperlink).Text)
		xassert.Equal(t, 42, shown)
	})
}

func TestShowContract(t *testing.T) {
	t.Run("can show courier contract", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			Type:          app.ContractTypeCourier,
			DateAccepted:  optional.New(time.Now().Add(-time.Hour)),
			EndLocationID: optional.New(factory.CreateEveLocationStation().ID),
			Status:        app.ContractStatusInProgress,
		})
		r := newContractRow(&o.Contract, o.CharacterID, "Bruce", false, set.Of[int64](), set.Of[int64]())
		// when
		ShowContract(u, r)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("contract-%d-%d", o.CharacterID, o.ContractID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("can show item exchange contract with items", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			Type:   app.ContractTypeItemExchange,
			Price:  optional.New(1000.0),
			Reward: optional.New(500.0),
		})
		factory.CreateCharacterContractItem(storage.CreateContractItemParams{ContractID: o.ID, IsIncluded: true})
		factory.CreateCharacterContractItem(storage.CreateContractItemParams{ContractID: o.ID, IsIncluded: false})
		r := newContractRow(&o.Contract, o.CharacterID, "Bruce", false, set.Of[int64](), set.Of[int64]())
		// when
		ShowContract(u, r)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("contract-%d-%d", o.CharacterID, o.ContractID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("can show auction contract", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			Type:   app.ContractTypeAuction,
			Status: app.ContractStatusOutstanding,
		})
		factory.CreateCharacterContractBid(storage.CreateContractBidParams{ContractID: o.ID, Amount: 100})
		r := newContractRow(&o.Contract, o.CharacterID, "Bruce", false, set.Of[int64](), set.Of[int64]())
		// when
		ShowContract(u, r)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("contract-%d-%d", o.CharacterID, o.ContractID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("closes window when contract details can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{Type: app.ContractTypeItemExchange})
		r := newContractRow(&o.Contract, o.CharacterID, "Bruce", false, set.Of[int64](), set.Of[int64]())
		if _, err := db.Exec("DROP TABLE character_contract_items"); err != nil {
			t.Fatal(err)
		}
		// when
		ShowContract(u, r)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("contract-%d-%d", o.CharacterID, o.ContractID))
		assert.True(t, created)
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		r := contractRow{contractID: 42, ownerID: 7}
		w, _ := u.GetOrCreateWindow(fmt.Sprintf("contract-%d-%d", r.ownerID, r.contractID))
		// when
		ShowContract(u, r)
		// then
		assert.Nil(t, w.Content())
	})
}

func TestShowCharacterContractByID(t *testing.T) {
	t.Run("can show contract", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{Type: app.ContractTypeCourier})
		// when
		ShowCharacterContractByID(u, o.CharacterID, o.ContractID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("contract-%d-%d", o.CharacterID, o.ContractID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("shows no window when contract does not exist", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCharacterFull()
		// when
		ShowCharacterContractByID(u, c.ID, 42)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("contract-%d-%d", c.ID, 42))
		assert.True(t, created)
	})
}
