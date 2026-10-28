package core

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestNewDesktopUI(t *testing.T) {
	// On disk, because settings are persisted in a goroutine, which would wipe an in-memory DB.
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	t.Run("starts with all pages disabled", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		// when
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		// then
		assert.False(t, u.rail.ItemEnabled(u.homeRailItem))
		assert.False(t, u.rail.ItemEnabled(u.characterRailItem))
		assert.False(t, u.rail.ItemEnabled(u.corporationRailItem))
		assert.False(t, u.rail.ItemEnabled(u.searchRailItem))
		assert.True(t, u.homeNav.Disabled())
		assert.Nil(t, u.rail.Selected())
	})
	t.Run("shows tray icon without mail indicator", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		fa := NewFakeApp(t)
		// when
		MakeFakeDesktopUI(t, st, fa)
		// then
		xassert.Equal[fyne.Resource](t, icons.IconPng, fa.trayIcon)
	})
	t.Run("shows mail indicator in tray icon", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		fa := NewFakeApp(t)
		u := MakeFakeDesktopUI(t, st, fa)
		// when
		u.showMailIndicator()
		// then
		xassert.Equal[fyne.Resource](t, icons.IconmarkedPng, fa.trayIcon)
	})
	t.Run("hides mail indicator in tray icon", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		fa := NewFakeApp(t)
		u := MakeFakeDesktopUI(t, st, fa)
		u.showMailIndicator()
		// when
		u.hideMailIndicator()
		// then
		xassert.Equal[fyne.Resource](t, icons.IconPng, fa.trayIcon)
	})
	t.Run("adds tray menu when tray is enabled", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		fa := NewFakeApp(t)
		// when
		MakeFakeDesktopUI(t, st, fa)
		// then
		m := fa.trayMenu
		if assert.NotNil(t, m) {
			var labels []string
			for _, it := range m.Items {
				if it.IsSeparator {
					labels = append(labels, "-")
				} else {
					labels = append(labels, it.Label)
				}
			}
			xassert.Equal(t, []string{"EVE Buddy", "-", "Open EVE Buddy", "-", "Quit"}, labels)
			assert.True(t, m.Items[0].Disabled)
			assert.True(t, m.Items[4].IsQuit)
		}
	})
	t.Run("closing main window shuts down when tray is disabled", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		fa := NewFakeApp(t)
		params := MakeFakeUIParams(t, st, fa)
		params.Settings.SetSysTrayEnabled(false)
		u := MakeFakeDesktopUIWithParams(t, params)
		// when
		u.MainWindow().Close()
		// then
		assert.True(t, u.signals.IsShuttingDown())
		assert.Nil(t, fa.trayMenu)
	})
}

func TestDesktopUI_HandleCurrentCharacterExchanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	t.Run("enables pages when a character is set", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		c := factory.CreateCharacter()
		// when
		u.handleCurrentCharacterExchanged(t.Context(), c)
		// then
		assert.True(t, u.rail.ItemEnabled(u.homeRailItem))
		assert.True(t, u.rail.ItemEnabled(u.characterRailItem))
		assert.False(t, u.rail.ItemEnabled(u.corporationRailItem))
		assert.True(t, u.rail.ItemEnabled(u.searchRailItem))
		assert.False(t, u.homeNav.Disabled())
		assert.Same(t, u.homeRailItem, u.rail.Selected())
	})
	t.Run("disables pages and shows character sheet when the character is removed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		c := factory.CreateCharacter()
		u.handleCurrentCharacterExchanged(t.Context(), c)
		u.rail.EnableItem(u.corporationRailItem)
		u.characterNav.Select(u.characterWalletItem)
		// when
		u.handleCurrentCharacterExchanged(t.Context(), nil)
		// then
		assert.False(t, u.rail.ItemEnabled(u.homeRailItem))
		assert.False(t, u.rail.ItemEnabled(u.characterRailItem))
		assert.False(t, u.rail.ItemEnabled(u.corporationRailItem))
		assert.False(t, u.rail.ItemEnabled(u.searchRailItem))
		assert.True(t, u.homeNav.Disabled())
		assert.Nil(t, u.rail.Selected())
		assert.Same(t, u.characterSheetItem, u.characterNav.Selected())
	})
}

func TestDesktopUI_OnSetCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
	c := factory.CreateCharacter()
	// when
	u.onSetCharacter(c)
	// then
	xassert.Equal(t, c.EveCharacter.Name, u.characterSelector.label.Text)
}

func TestDesktopUI_OnSetCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
	c := factory.CreateCorporation()
	// when
	u.onSetCorporation(c)
	// then
	xassert.Equal(t, c.EveCorporation.Name, u.corporationSelector.label.Text)
}

func TestDesktopUI_OnShowCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
	u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
	// when
	u.onShowCharacter()
	// then
	assert.Same(t, u.characterRailItem, u.rail.Selected())
}

func TestDesktopUI_OnShowCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
	u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
	u.rail.EnableItem(u.corporationRailItem)
	// when
	u.onShowCorporation()
	// then
	assert.Same(t, u.corporationRailItem, u.rail.Selected())
}

func TestDesktopUI_UpdateCorporationRailItem(t *testing.T) {
	// On disk, because settings are persisted in a goroutine, which would wipe an in-memory DB.
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	t.Run("enables pages when there are corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		u.character.Store(factory.CreateCharacter())
		factory.CreateCorporation()
		// when
		u.updateCorporationRailItem(t.Context())
		// then
		assert.True(t, u.rail.ItemEnabled(u.homeRailItem))
		assert.True(t, u.rail.ItemEnabled(u.characterRailItem))
		assert.True(t, u.rail.ItemEnabled(u.corporationRailItem))
	})
	t.Run("keeps pages disabled when there is no current character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: c.ID})
		factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID}) // so the corporation is listed
		// when
		u.updateCorporationRailItem(t.Context())
		// then
		assert.False(t, u.rail.ItemEnabled(u.homeRailItem))
		assert.False(t, u.rail.ItemEnabled(u.characterRailItem))
		assert.False(t, u.rail.ItemEnabled(u.corporationRailItem))
	})
	t.Run("disables corporation page and shows corporation sheet when there are no corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
		u.rail.EnableItem(u.corporationRailItem)
		u.rail.Select(u.corporationRailItem)
		u.corporationNav.Select(u.corporationAssetsItem)
		// when
		u.updateCorporationRailItem(t.Context())
		// then
		assert.False(t, u.rail.ItemEnabled(u.corporationRailItem))
		assert.Same(t, u.homeRailItem, u.rail.Selected())
		assert.Same(t, u.corporationSheetItem, u.corporationNav.Selected())
	})
}

func TestDesktopUI_UpdatePermittedSections(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	cases := []struct {
		name     string
		sections []app.CorporationSection
		want     []string // labels of enabled items
	}{
		{"no sections", nil, nil},
		{"assets", []app.CorporationSection{app.SectionCorporationAssets}, []string{"assets"}},
		{"contracts", []app.CorporationSection{app.SectionCorporationContracts}, []string{"contracts"}},
		{"industry jobs", []app.CorporationSection{app.SectionCorporationIndustryJobs}, []string{"industry"}},
		{"structures", []app.CorporationSection{app.SectionCorporationStructures}, []string{"structures"}},
		{"wallet balances", []app.CorporationSection{app.SectionCorporationWalletBalances}, []string{"wallets"}},
		{
			"wealth sections",
			[]app.CorporationSection{
				app.SectionCorporationAssets,
				app.SectionCorporationContracts,
				app.SectionCorporationWalletBalances,
			},
			[]string{"assets", "contracts", "wallets", "wealth"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
			c := factory.CreateCorporation()
			for _, s := range tc.sections {
				factory.CreateCorporationTokenForSection(c.ID, s)
			}
			u.corporation.Store(c)
			// when
			u.updatePermittedSections(t.Context())
			// then
			items := map[string]*xwidget.NavDrawerItem{
				"assets":     u.corporationAssetsItem,
				"contracts":  u.corporationContractsItem,
				"industry":   u.corporationIndustryItem,
				"structures": u.corporationStructuresItem,
				"wallets":    u.corporationWalletsItem,
				"wealth":     u.corporationWealthItem,
			}
			var got []string
			for _, label := range []string{"assets", "contracts", "industry", "structures", "wallets", "wealth"} {
				if u.corporationNav.ItemEnabled(items[label]) {
					got = append(got, label)
				}
			}
			xassert.Equal(t, tc.want, got)
			assert.True(t, u.corporationNav.ItemEnabled(u.corporationSheetItem))
		})
	}
	t.Run("keeps the selected page when it is still permitted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationAssets)
		u.corporation.Store(c)
		u.corporationNav.Select(u.corporationAssetsItem)
		// when
		u.updatePermittedSections(t.Context())
		// then
		assert.Same(t, u.corporationAssetsItem, u.corporationNav.Selected())
	})
	t.Run("shows corporation sheet when the selected page is no longer permitted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		u.corporation.Store(c)
		u.corporationNav.Select(u.corporationAssetsItem)
		// when
		u.updatePermittedSections(t.Context())
		// then
		assert.Same(t, u.corporationSheetItem, u.corporationNav.Selected())
	})
}

func TestDesktopUI_CorporationTotalBadges(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	t.Run("shows wallets total when permitted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		u.corporation.Store(c)
		u.updatePermittedSections(t.Context())
		// when
		u.corporationWallets.OnBalanceUpdate(optional.New(1.5e9))
		// then
		xassert.Equal(t, "1.5 B", u.corporationWalletsItem.Badge())
	})
	t.Run("shows wallets total received before permission", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		u.corporation.Store(c)
		u.corporationWallets.OnBalanceUpdate(optional.New(1.5e9))
		// when
		u.updatePermittedSections(t.Context())
		// then
		xassert.Equal(t, "1.5 B", u.corporationWalletsItem.Badge())
	})
	t.Run("hides wallets total when not permitted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		u.corporation.Store(factory.CreateCorporation())
		u.updatePermittedSections(t.Context())
		// when
		u.corporationWallets.OnBalanceUpdate(optional.New(1.5e9))
		// then
		xassert.Equal(t, "", u.corporationWalletsItem.Badge())
	})
	t.Run("shows wealth total when permitted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationAssets)
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationContracts)
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		u.corporation.Store(c)
		u.updatePermittedSections(t.Context())
		// when
		u.corporationWealth.OnUpdate(optional.New(1.5e9))
		// then
		xassert.Equal(t, "1.5 B", u.corporationWealthItem.Badge())
	})
	t.Run("shows wealth total received before permission", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationAssets)
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationContracts)
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		u.corporation.Store(c)
		u.corporationWealth.OnUpdate(optional.New(1.5e9))
		// when
		u.updatePermittedSections(t.Context())
		// then
		xassert.Equal(t, "1.5 B", u.corporationWealthItem.Badge())
	})
	t.Run("hides wealth total when not permitted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		u.corporation.Store(c)
		u.updatePermittedSections(t.Context())
		// when
		u.corporationWealth.OnUpdate(optional.New(1.5e9))
		// then
		xassert.Equal(t, "", u.corporationWealthItem.Badge())
	})
}

func TestDesktopUI_Badges(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
	cases := []struct {
		name   string
		update func()
		item   *xwidget.NavDrawerItem
		want   string
	}{
		{"home communications", func() { u.unifiedCommunications.OnUpdate(optional.New(5)) }, u.homeCommunicationsItem, "5"},
		{"home communications at limit", func() { u.unifiedCommunications.OnUpdate(optional.New(1000)) }, u.homeCommunicationsItem, "999+"},
		{"home communications unknown", func() { u.unifiedCommunications.OnUpdate(optional.Optional[int]{}) }, u.homeCommunicationsItem, "?"},
		{"home communications none", func() { u.unifiedCommunications.OnUpdate(optional.New(0)) }, u.homeCommunicationsItem, ""},
		{"home mail", func() { u.unifiedMails.OnUpdate(optional.New(5), 0) }, u.homeMailItem, "5"},
		{"home mail at limit", func() { u.unifiedMails.OnUpdate(optional.New(1000), 0) }, u.homeMailItem, "999+"},
		{"home contracts", func() { u.contractList.OnUpdate(optional.New(1234)) }, u.homeContractsItem, "1,234"},
		{"home colonies", func() { u.colonies.OnUpdate(optional.New(1234)) }, u.homeColoniesItem, "1,234"},
		{"home industry", func() { u.industryJobs.OnUpdate(optional.New(1234)) }, u.homeIndustryItem, "1,234"},
		{"home skills", func() { u.training.OnUpdate(optional.New(1234)) }, u.homeSkillsItem, "1,234"},
		{"home structures", func() { u.unifiedStructures.OnUpdate(optional.New(1234)) }, u.homeStructuresItem, "1,234"},
		{"home wealth", func() { u.wealth.OnUpdate(optional.New(1.5e9)) }, u.homeWealthItem, "1.5 B"},
		{"home wealth unknown", func() { u.wealth.OnUpdate(optional.Optional[float64]{}) }, u.homeWealthItem, "?"},
		{"home wealth zero", func() { u.wealth.OnUpdate(optional.New(0.0)) }, u.homeWealthItem, ""},
		{"character communications", func() { u.characterCommunications.OnUpdate(optional.New(5)) }, u.characterCommunicationsItem, "5"},
		{"character communications at limit", func() { u.characterCommunications.OnUpdate(optional.New(1000)) }, u.characterCommunicationsItem, "999+"},
		{"character mail", func() { u.characterMails.OnUpdate(optional.New(5), 0) }, u.characterMailItem, "5"},
		{"character mail at limit", func() { u.characterMails.OnUpdate(optional.New(100), 0) }, u.characterMailItem, "99+"},
		{"character skills", func() { u.characterSkillQueue.OnUpdate("3 skills", "") }, u.characterSkillsItem, "3 skills"},
		{"character wallet", func() { u.characterWallet.OnBalanceUpdate(optional.New(1.5e9)) }, u.characterWalletItem, "1.5 B"},
		{"character wallet unknown", func() { u.characterWallet.OnBalanceUpdate(optional.Optional[float64]{}) }, u.characterWalletItem, "?"},
		{"corporation contracts", func() { u.corporationContracts.OnUpdate(optional.New(1234)) }, u.corporationContractsItem, "1,234"},
		{"corporation industry", func() { u.corporationIndyJobs.OnUpdate(optional.New(1234)) }, u.corporationIndustryItem, "1,234"},
		{"corporation structures", func() { u.corporationStructures.OnUpdate(optional.New(1234)) }, u.corporationStructuresItem, "1,234"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			tc.update()
			// then
			xassert.Equal(t, tc.want, tc.item.Badge())
		})
	}
}

func TestDesktopUI_Shortcuts(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	t.Run("shows current character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		iv := &testdouble.InfoViewerFake{}
		u.iw = iv
		c := factory.CreateCharacter()
		u.character.Store(c)
		sc, ok := xdesktop.Shortcut("currentCharacter", u.MainWindow())
		require.True(t, ok)
		// when
		sc.Handler(sc.Shortcut)
		// then
		if assert.Len(t, iv.Shown, 1) {
			xassert.Equal(t, c.ID, iv.Shown[0].ID)
		}
	})
	t.Run("shows current location", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		iv := &testdouble.InfoViewerFake{}
		u.iw = iv
		c := factory.CreateCharacterFull()
		u.character.Store(c)
		sc, ok := xdesktop.Shortcut("currentLocation", u.MainWindow())
		require.True(t, ok)
		// when
		sc.Handler(sc.Shortcut)
		// then
		xassert.Equal(t, []int64{c.LocationID.ValueOrZero()}, iv.ShownLocations)
	})
	t.Run("shows current ship", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		iv := &testdouble.InfoViewerFake{}
		u.iw = iv
		c := factory.CreateCharacterFull()
		u.character.Store(c)
		sc, ok := xdesktop.Shortcut("currentShip", u.MainWindow())
		require.True(t, ok)
		// when
		sc.Handler(sc.Shortcut)
		// then
		xassert.Equal(t, []int64{c.ShipTypeID.ValueOrZero()}, iv.ShownTypes)
	})
	t.Run("shows nothing when there is no character", func(t *testing.T) {
		for _, name := range []string{"currentCharacter", "currentLocation", "currentShip"} {
			t.Run(name, func(t *testing.T) {
				testutil.MustTruncateTables(db)
				u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
				iv := &testdouble.InfoViewerFake{}
				u.iw = iv
				sc, ok := xdesktop.Shortcut(name, u.MainWindow())
				require.True(t, ok)
				// when
				sc.Handler(sc.Shortcut)
				// then
				assert.Empty(t, iv.Shown)
				assert.Empty(t, iv.ShownLocations)
				assert.Empty(t, iv.ShownTypes)
			})
		}
	})
	t.Run("shows no location when it is unknown", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		iv := &testdouble.InfoViewerFake{}
		u.iw = iv
		u.character.Store(factory.CreateCharacter())
		sc, ok := xdesktop.Shortcut("currentLocation", u.MainWindow())
		require.True(t, ok)
		// when
		sc.Handler(sc.Shortcut)
		// then
		assert.Empty(t, iv.ShownLocations)
	})
	t.Run("shows no ship when it is unknown", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		iv := &testdouble.InfoViewerFake{}
		u.iw = iv
		u.character.Store(factory.CreateCharacter())
		sc, ok := xdesktop.Shortcut("currentShip", u.MainWindow())
		require.True(t, ok)
		// when
		sc.Handler(sc.Shortcut)
		// then
		assert.Empty(t, iv.ShownTypes)
	})
	t.Run("quits app", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		sc, ok := xdesktop.Shortcut("quit", u.MainWindow())
		require.True(t, ok)
		// when
		sc.Handler(sc.Shortcut)
		// then
		assert.True(t, u.signals.IsShuttingDown())
	})
}

func TestDesktopUI_ShowSearchWindow(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	t.Run("creates search window", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		// when
		u.showSearchWindow()
		// then
		w, ok := u.windows["new-eden-search"]
		require.True(t, ok)
		assert.Contains(t, w.Title(), "Search New Eden")
		assert.Same(t, u.gameSearch, w.Content())
	})
	t.Run("reuses open search window", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		u.showSearchWindow()
		w := u.windows["new-eden-search"]
		// when
		u.showSearchWindow()
		// then
		assert.Same(t, w, u.windows["new-eden-search"])
	})
	t.Run("creates new search window after the previous one was closed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		u.showSearchWindow()
		w := u.windows["new-eden-search"]
		w.Close()
		// when
		u.showSearchWindow()
		// then
		assert.NotSame(t, w, u.windows["new-eden-search"])
	})
}

func TestDesktopUI_ShowUserDataDialog(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	fa := NewFakeApp(t)
	params := MakeFakeUIParams(t, st, fa)
	params.DataPaths = map[string]string{"log": "/alpha/../bravo/log"}
	u := MakeFakeDesktopUIWithParams(t, params)
	u.showUserDataDialog()
	var copyButton *widget.Button
	for _, o := range test.LaidOutObjects(u.MainWindow().Canvas().Overlays().Top()) {
		if b, ok := o.(*widget.Button); ok && b.Icon != nil && b.Icon.Name() == theme.ContentCopyIcon().Name() {
			copyButton = b
		}
	}
	require.NotNil(t, copyButton)
	// when
	test.Tap(copyButton)
	// then
	xassert.Equal(t, "/bravo/log", fa.Clipboard().Content())
}

func TestDesktopUI_Quit(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
	// when
	u.quit()
	// then
	assert.True(t, u.signals.IsShuttingDown())
}

func TestDesktopUI_WindowSize(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	t.Run("saves window size when app stops", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		u.MainWindow().Resize(fyne.NewSize(1000, 700))
		// when
		u.onAppStopped()
		// then
		xassert.Equal(t, fyne.NewSize(1000, 700), u.settings.WindowSize())
	})
	t.Run("restores window size when app starts", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeDesktopUI(t, st, NewFakeApp(t))
		u.settings.SetWindowSize(fyne.NewSize(1000, 700))
		// when
		u.onShowAndRun()
		// then
		xassert.Equal(t, fyne.NewSize(1000, 700), u.MainWindow().Canvas().Size())
	})
}
