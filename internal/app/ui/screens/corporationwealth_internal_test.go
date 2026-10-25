package screens

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestCorporationWealth_ErrorDisplay(t *testing.T) {
	t.Run("shows error and clears charts when update fails", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.corporation.Store(factory.CreateCorporation())
		a.categorySplitTitleLabel.SetText("Total Net Worth By Category - Total: 1.0 B")
		a.walletsTitleLabel.SetText("Wallet Balance by Division")
		a.assetsTitleLabel.SetText("Asset Value by Division")
		if _, err := db.Exec("DROP TABLE corporation_section_status"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.statusLabel.Visible())
		assert.Contains(t, a.statusLabel.Text, "Failed to fetch data")
		assert.Empty(t, a.categorySplitTitleLabel.Text)
		assert.Empty(t, a.walletsTitleLabel.Text)
		assert.Empty(t, a.assetsTitleLabel.Text)
	})
	t.Run("shows error and clears charts when fetching values fails", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationContracts,
		})
		a.corporation.Store(c)
		a.categorySplitTitleLabel.SetText("Total Net Worth By Category - Total: 1.0 B")
		if _, err := db.Exec("DROP TABLE corporation_contracts"); err != nil { // let fetching the escrow fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.statusLabel.Visible())
		assert.Equal(t, widget.DangerImportance, a.statusLabel.Importance)
		assert.Contains(t, a.statusLabel.Text, "Failed to fetch data")
		assert.Empty(t, a.categorySplitTitleLabel.Text)
	})
	t.Run("shows error and clears charts when fetching hangar names fails", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletBalances,
		})
		a.corporation.Store(c)
		a.categorySplitTitleLabel.SetText("Total Net Worth By Category - Total: 1.0 B")
		if _, err := db.Exec("DROP TABLE corporation_hangar_names"); err != nil { // let fetching the hangar names fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.statusLabel.Visible())
		assert.Equal(t, widget.DangerImportance, a.statusLabel.Importance)
		assert.Contains(t, a.statusLabel.Text, "Failed to fetch data")
		assert.Empty(t, a.categorySplitTitleLabel.Text)
	})
	t.Run("shows error on desktop", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		w := test.NewWindow(a)
		defer w.Close()
		a.corporation.Store(factory.CreateCorporation())
		if _, err := db.Exec("DROP TABLE corporation_section_status"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.statusLabel.Visible())
		assert.Contains(t, a.statusLabel.Text, "Failed to fetch data")
		assert.True(t, slices.Contains(test.LaidOutObjects(a), fyne.CanvasObject(a.statusLabel)))
	})
	t.Run("shows error on mobile", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: true,
			Storage:  st,
		}))
		w := test.NewWindow(a)
		defer w.Close()
		a.corporation.Store(factory.CreateCorporation())
		if _, err := db.Exec("DROP TABLE corporation_section_status"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.statusLabel.Visible())
		assert.Contains(t, a.statusLabel.Text, "Failed to fetch data")
		assert.True(t, slices.Contains(test.LaidOutObjects(a), fyne.CanvasObject(a.statusLabel)))
	})
}

func TestCorporationWealth_OnUpdate(t *testing.T) {
	t.Run("reports total net worth", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var got optional.Optional[float64]
		a.OnUpdate = func(total optional.Optional[float64]) {
			got = total
		}
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletBalances,
		})
		factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: c.ID,
			DivisionID:    1,
			Balance:       1000,
		})
		factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: c.ID,
			DivisionID:    2,
			Balance:       234.5,
		})
		a.corporation.Store(c)

		a.update(t.Context())
		assert.Equal(t, optional.New(1234.5), got)
	})
	t.Run("includes contract items value", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var got optional.Optional[float64]
		a.OnUpdate = func(total optional.Optional[float64]) {
			got = total
		}
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationContracts,
		})
		o := factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID:  c.ID,
			ForCorporation: true,
			Status:         app.ContractStatusOutstanding,
			Type:           app.ContractTypeItemExchange,
		})
		u := factory.CreateCorporationContractItem(storage.CreateContractItemParams{
			ContractID: o.ID,
			IsIncluded: true,
			Quantity:   2,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       u.Type.ID,
			AveragePrice: optional.New(500.0),
		})
		a.corporation.Store(c)

		a.update(t.Context())
		assert.Equal(t, optional.New(1000.0), got)
	})
	t.Run("reports zero net worth", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var got optional.Optional[float64]
		a.OnUpdate = func(total optional.Optional[float64]) {
			got = total
		}
		c := factory.CreateCorporation()
		for _, s := range []app.CorporationSection{
			app.SectionCorporationAssets,
			app.SectionCorporationContracts,
			app.SectionCorporationWalletBalances,
		} {
			factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
				CorporationID: c.ID,
				Section:       s,
			})
		}
		a.corporation.Store(c)

		a.update(t.Context())
		assert.Equal(t, optional.New(0.0), got)
	})
	t.Run("reports empty without wealth data", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		got := optional.New(1.0)
		a.OnUpdate = func(total optional.Optional[float64]) {
			got = total
		}
		a.corporation.Store(factory.CreateCorporation())

		a.update(t.Context())
		assert.True(t, got.IsEmpty())
	})
	t.Run("reports empty without corporation", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		got := optional.New(1.0)
		a.OnUpdate = func(total optional.Optional[float64]) {
			got = total
		}

		a.update(t.Context())
		assert.True(t, got.IsEmpty())
	})
	t.Run("reports empty when update fails", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		got := optional.New(1.0)
		a.OnUpdate = func(total optional.Optional[float64]) {
			got = total
		}
		a.corporation.Store(factory.CreateCorporation())
		if _, err := db.Exec("DROP TABLE corporation_section_status"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, got.IsEmpty())
	})
}

func TestCorporationWealth_HelpButton(t *testing.T) {
	// given
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	}))
	w := test.NewWindow(a)
	defer w.Close()
	w.Resize(fyne.NewSize(800, 600))

	// when
	test.Tap(a.helpButton)

	// then
	assert.NotNil(t, w.Canvas().Overlays().Top())
}

func TestCorporationAssetRows(t *testing.T) {
	t.Run("adds bars for other assets and contract items", func(t *testing.T) {
		// given
		rows := []corporationDivisionValue{{division: app.Division1, name: "Hangar", assetValue: 1}}
		// when
		got := corporationAssetRows(rows, 3_000_000_000, 2_000_000_000)
		// then
		want := []corporationDivisionValue{
			{division: app.Division1, name: "Hangar", assetValue: 1},
			{name: corporationWealthOtherAssetsLabel, assetValue: 3},
			{name: corporationWealthContractItemsLabel, assetValue: 2},
		}
		assert.Equal(t, want, got)
		assert.Len(t, rows, 1)
	})
	t.Run("returns rows unchanged without other assets and contract items", func(t *testing.T) {
		// given
		rows := []corporationDivisionValue{{division: app.Division1, name: "Hangar", assetValue: 1}}
		// when
		got := corporationAssetRows(rows, 0, 0)
		// then
		assert.Equal(t, rows, got)
	})
	t.Run("ignores rounding differences for other assets", func(t *testing.T) {
		// given
		rows := []corporationDivisionValue{{division: app.Division1, name: "Hangar", assetValue: 1}}
		// when
		got := corporationAssetRows(rows, 0.0001, 0)
		// then
		assert.Equal(t, rows, got)
	})
	t.Run("ignores division sum above asset total", func(t *testing.T) {
		// given
		rows := []corporationDivisionValue{{division: app.Division1, name: "Hangar", assetValue: 1}}
		// when
		got := corporationAssetRows(rows, -5, 0)
		// then
		assert.Equal(t, rows, got)
	})
}

func TestCorporationWealth_SectionChanged(t *testing.T) {
	t.Run("updates when divisions change", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var got optional.Optional[float64]
		a.OnUpdate = func(total optional.Optional[float64]) {
			got = total
		}
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletBalances,
		})
		factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: c.ID,
			DivisionID:    1,
			Balance:       1000,
		})
		a.corporation.Store(c)
		// when
		a.u.Signals().CorporationSectionChanged.Emit(t.Context(), app.CorporationSectionUpdated{
			CorporationID: c.ID,
			Section:       app.SectionCorporationDivisions,
		})
		// then
		assert.Equal(t, optional.New(1000.0), got)
	})
	t.Run("updates when market prices change", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var got optional.Optional[float64]
		a.OnUpdate = func(total optional.Optional[float64]) {
			got = total
		}
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletBalances,
		})
		factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
			CorporationID: c.ID,
			DivisionID:    1,
			Balance:       1000,
		})
		a.corporation.Store(c)
		// when
		a.u.Signals().EveUniverseSectionChanged.Emit(t.Context(), app.EveUniverseSectionUpdated{
			Section: app.SectionEveMarketPrices,
		})
		// then
		assert.Equal(t, optional.New(1000.0), got)
	})
}

func TestCorporationHangarLabel(t *testing.T) {
	assert.Equal(t, "[3] Alpha", corporationHangarLabel(app.Division3, "Alpha"))
	assert.Equal(t, "[1] 1st Division", corporationHangarLabel(app.Division1, ""))
}
