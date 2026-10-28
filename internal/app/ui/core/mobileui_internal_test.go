package core

import (
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestNewMobileUI(t *testing.T) {
	// On disk, because settings are persisted in a goroutine, which would wipe an in-memory DB.
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	t.Run("starts with only the manage destination enabled", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		// when
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		// then
		assert.False(t, u.navBar.Enabled(homeDestination))
		assert.False(t, u.navBar.Enabled(characterDestination))
		assert.False(t, u.navBar.Enabled(corporationDestination))
		assert.False(t, u.navBar.Enabled(searchDestination))
		assert.True(t, u.navBar.Enabled(moreDestination))
		id, ok := u.navBar.Selected()
		assert.True(t, ok)
		xassert.Equal(t, moreDestination, id)
		_, hasBadge := u.navBar.Badge(moreDestination)
		assert.False(t, hasBadge)
	})
	t.Run("resizes window when showing fake mobile", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		params := MakeFakeUIParams(t, st, NewFakeApp(t))
		params.IsMobile = true
		params.IsFakeMobile = true
		u := MakeFakeMobileUIWithParams(t, params)
		// when
		u.onShowAndRun()
		// then
		xassert.Equal(t, fyne.NewSize(340, 700), u.MainWindow().Canvas().Size())
		assert.True(t, u.MainWindow().FixedSize())
	})
}

func TestMobileUI_HandleCurrentCharacterExchanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	t.Run("enables destinations and shows home when a character is set", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		// when
		u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
		// then
		assert.True(t, u.navBar.Enabled(homeDestination))
		assert.True(t, u.navBar.Enabled(characterDestination))
		assert.False(t, u.navBar.Enabled(corporationDestination))
		assert.True(t, u.navBar.Enabled(searchDestination))
		id, _ := u.navBar.Selected()
		xassert.Equal(t, homeDestination, id)
	})
	t.Run("keeps selected destination when another character is set", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
		u.navBar.Select(characterDestination)
		// when
		u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
		// then
		id, _ := u.navBar.Selected()
		xassert.Equal(t, characterDestination, id)
	})
	t.Run("disables destinations and shows manage when the character is removed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
		u.navBar.Enable(corporationDestination)
		u.homeNav.Push(xwidget.NewAppBar("Dummy", widget.NewLabel("Dummy")))
		u.characterNav.Push(xwidget.NewAppBar("Dummy", widget.NewLabel("Dummy")))
		u.navBar.Select(characterDestination)
		// when
		u.handleCurrentCharacterExchanged(t.Context(), nil)
		// then
		assert.False(t, u.navBar.Enabled(homeDestination))
		assert.False(t, u.navBar.Enabled(characterDestination))
		assert.False(t, u.navBar.Enabled(corporationDestination))
		assert.False(t, u.navBar.Enabled(searchDestination))
		id, _ := u.navBar.Selected()
		xassert.Equal(t, moreDestination, id)
		assert.True(t, u.homeNav.IsRoot())
		assert.True(t, u.characterNav.IsRoot())
	})
}

func TestMobileUI_OnSetCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := MakeFakeMobileUI(t, st, NewFakeApp(t))
	c := factory.CreateCharacter()
	avatar := theme.AccountIcon()
	u.avatarCache.Store(c.ID, avatar) // avoids loading avatars in the background
	u.handleCurrentCharacterExchanged(t.Context(), c)
	u.characterNav.Push(xwidget.NewAppBar("Dummy", widget.NewLabel("Dummy")))
	// when
	u.onSetCharacter(c)
	// then
	xassert.Equal(t, c.EveCharacter.Name, u.characterPage.Title())
	assert.True(t, u.characterNav.IsRoot())
	xassert.Equal[fyne.Resource](t, avatar, u.characterSelector.Icon)
}

func TestMobileUI_OnSetCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := MakeFakeMobileUI(t, st, NewFakeApp(t))
	c := factory.CreateCorporation()
	avatar := theme.AccountIcon()
	u.avatarCache.Store(c.ID, avatar) // avoids loading avatars in the background
	u.corporationNav.Push(xwidget.NewAppBar("Dummy", widget.NewLabel("Dummy")))
	// when
	u.onSetCorporation(c)
	// then
	xassert.Equal(t, c.EveCorporation.Name, u.corporationPage.Title())
	assert.True(t, u.corporationNav.IsRoot())
	xassert.Equal[fyne.Resource](t, avatar, u.corporationSelector.Icon)
}

func TestMobileUI_OnShowCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := MakeFakeMobileUI(t, st, NewFakeApp(t))
	u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
	// when
	u.onShowCharacter()
	// then
	id, _ := u.navBar.Selected()
	xassert.Equal(t, characterDestination, id)
}

func TestMobileUI_OnShowCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := MakeFakeMobileUI(t, st, NewFakeApp(t))
	u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
	u.navBar.Enable(corporationDestination)
	// when
	u.onShowCorporation()
	// then
	id, _ := u.navBar.Selected()
	xassert.Equal(t, corporationDestination, id)
}

func TestMobileUI_UpdateCorporationDestination(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	t.Run("enables corporation destination when there are corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.character.Store(factory.CreateCharacter())
		factory.CreateCorporation()
		// when
		u.updateCorporationDestination(t.Context())
		// then
		assert.True(t, u.navBar.Enabled(corporationDestination))
	})
	t.Run("keeps corporation destination disabled when there is no current character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: c.ID})
		factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID}) // so the corporation is listed
		// when
		u.updateCorporationDestination(t.Context())
		// then
		assert.False(t, u.navBar.Enabled(corporationDestination))
	})
	t.Run("disables corporation destination and shows home when there are no corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
		u.navBar.Enable(corporationDestination)
		u.navBar.Select(corporationDestination)
		// when
		u.updateCorporationDestination(t.Context())
		// then
		assert.False(t, u.navBar.Enabled(corporationDestination))
		id, _ := u.navBar.Selected()
		xassert.Equal(t, homeDestination, id)
	})
	t.Run("keeps selected destination when there are no corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
		u.navBar.Select(characterDestination)
		// when
		u.updateCorporationDestination(t.Context())
		// then
		id, _ := u.navBar.Selected()
		xassert.Equal(t, characterDestination, id)
	})
}

func TestMobileUI_UpdatePermittedSections(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	cases := []struct {
		name     string
		sections []app.CorporationSection
		want     []string // labels of enabled items
	}{
		{"no sections", nil, nil},
		{"assets", []app.CorporationSection{app.SectionCorporationAssets}, []string{"assets", "asset search"}},
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
			[]string{"assets", "asset search", "contracts", "wallets", "wealth"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			u := MakeFakeMobileUI(t, st, NewFakeApp(t))
			c := factory.CreateCorporation()
			for _, s := range tc.sections {
				factory.CreateCorporationTokenForSection(c.ID, s)
			}
			u.corporation.Store(c)
			// when
			u.updatePermittedSections(t.Context())
			// then
			items := map[string]*xwidget.NavListItem{
				"assets":       u.corporationAssetBrowserItem,
				"asset search": u.corporationAssetSearchItem,
				"contracts":    u.corporationContractsItem,
				"industry":     u.corporationIndustryItem,
				"structures":   u.corporationStructuresItem,
				"wallets":      u.corporationWalletsItem,
				"wealth":       u.corporationWealthItem,
			}
			var got []string
			for _, label := range []string{"assets", "asset search", "contracts", "industry", "structures", "wallets", "wealth"} {
				if !items[label].Disabled() {
					got = append(got, label)
				}
			}
			xassert.Equal(t, tc.want, got)
			assert.False(t, u.corporationSheetItem.Disabled())
		})
	}
}

func TestMobileUI_CorporationTotals(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	t.Run("shows wallets total when permitted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		u.corporation.Store(c)
		u.updatePermittedSections(t.Context())
		// when
		u.corporationWallets.OnBalanceUpdate(optional.New(1.5e9))
		// then
		xassert.Equal(t, "1,500,000,000 ISK", u.corporationWalletsItem.Supporting)
	})
	t.Run("shows wallets total received before permission", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		u.corporation.Store(c)
		u.corporationWallets.OnBalanceUpdate(optional.New(1.5e9))
		// when
		u.updatePermittedSections(t.Context())
		// then
		xassert.Equal(t, "1,500,000,000 ISK", u.corporationWalletsItem.Supporting)
	})
	t.Run("hides wallets total when not permitted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.corporation.Store(factory.CreateCorporation())
		u.updatePermittedSections(t.Context())
		// when
		u.corporationWallets.OnBalanceUpdate(optional.New(1.5e9))
		// then
		xassert.Equal(t, "", u.corporationWalletsItem.Supporting)
	})
	t.Run("shows wealth total when permitted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationAssets)
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationContracts)
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		u.corporation.Store(c)
		u.updatePermittedSections(t.Context())
		// when
		u.corporationWealth.OnUpdate(optional.New(1.5e9))
		// then
		xassert.Equal(t, "1,500,000,000 ISK", u.corporationWealthItem.Supporting)
	})
	t.Run("shows wealth total received before permission", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationAssets)
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationContracts)
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		u.corporation.Store(c)
		u.corporationWealth.OnUpdate(optional.New(1.5e9))
		// when
		u.updatePermittedSections(t.Context())
		// then
		xassert.Equal(t, "1,500,000,000 ISK", u.corporationWealthItem.Supporting)
	})
	t.Run("hides wealth total when not permitted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletBalances)
		u.corporation.Store(c)
		u.updatePermittedSections(t.Context())
		// when
		u.corporationWealth.OnUpdate(optional.New(1.5e9))
		// then
		xassert.Equal(t, "", u.corporationWealthItem.Supporting)
	})
}

func TestMobileUI_SupportingTexts(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := MakeFakeMobileUI(t, st, NewFakeApp(t))
	cases := []struct {
		name   string
		update func()
		item   *xwidget.NavListItem
		want   string
	}{
		{"home characters", func() { u.characterOverview.OnUpdate(3) }, u.homeCharactersItem, "3 characters"},
		{"home corporations", func() { u.corporationOverview.OnUpdate(1) }, u.homeCorporationsItem, "1 corporation"},
		{"home colonies", func() { u.colonies.OnUpdate(optional.New(1234)) }, u.homeColoniesItem, "1,234 not working"},
		{"home communications", func() { u.unifiedCommunications.OnUpdate(optional.New(1234)) }, u.homeCommunicationsItem, "1,234 unread"},
		{"home communications none", func() { u.unifiedCommunications.OnUpdate(optional.New(0)) }, u.homeCommunicationsItem, ""},
		{"home contracts", func() { u.contractList.OnUpdate(optional.New(2)) }, u.homeContractsItem, "2 contracts outstanding"},
		{"home industry", func() { u.industryJobs.OnUpdate(optional.New(1)) }, u.homeIndustryItem, "1 job ready"},
		{"home mail", func() { u.unifiedMails.OnUpdate(optional.New(5), 0) }, u.homeMailItem, "5 unread"},
		{"home mail downloading", func() { u.unifiedMails.OnUpdate(optional.New(5), 10) }, u.homeMailItem, "5 unread • 90% downloaded"},
		{"home mail unknown", func() { u.unifiedMails.OnUpdate(optional.Optional[int]{}, 0) }, u.homeMailItem, "? unread"},
		{"home skills", func() { u.training.OnUpdate(optional.New(2)) }, u.homeSkillsItem, "2 expired"},
		{"home structures", func() { u.unifiedStructures.OnUpdate(optional.New(2)) }, u.homeStructuresItem, "2 structures reinforced"},
		{"home wealth", func() { u.wealth.OnUpdate(optional.New(1.5e9)) }, u.homeWealthItem, "1,500,000,000 ISK"},
		{"home wealth unknown", func() { u.wealth.OnUpdate(optional.Optional[float64]{}) }, u.homeWealthItem, "? ISK"},
		{"home wealth zero", func() { u.wealth.OnUpdate(optional.New(0.0)) }, u.homeWealthItem, ""},
		{"character communications", func() { u.characterCommunications.OnUpdate(optional.New(5)) }, u.characterCommunicationsItem, "5 unread"},
		{"character mail", func() { u.characterMails.OnUpdate(optional.New(5), 0) }, u.characterMailItem, "5 unread"},
		{"character mail downloading", func() { u.characterMails.OnUpdate(optional.New(0), 10) }, u.characterMailItem, "90% downloaded"},
		{"character skills", func() { u.characterSkillQueue.OnUpdate("3 skills", "Training") }, u.characterSkillsItem, "Training"},
		{"character wallet", func() { u.characterWallet.OnBalanceUpdate(optional.New(1.5e9)) }, u.characterWalletItem, "1,500,000,000 ISK"},
		{"character wallet unknown", func() { u.characterWallet.OnBalanceUpdate(optional.Optional[float64]{}) }, u.characterWalletItem, "? ISK"},
		{"corporation contracts", func() { u.corporationContracts.OnUpdate(optional.New(2)) }, u.corporationContractsItem, "2 contracts outstanding"},
		{"corporation industry", func() { u.corporationIndyJobs.OnUpdate(optional.New(2)) }, u.corporationIndustryItem, "2 jobs ready"},
		{"corporation structures", func() { u.corporationStructures.OnUpdate(optional.New(1)) }, u.corporationStructuresItem, "1 structure reinforced"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			tc.update()
			// then
			xassert.Equal(t, tc.want, tc.item.Supporting)
		})
	}
	t.Run("shows warning when colonies are not working", func(t *testing.T) {
		// when
		u.colonies.OnUpdate(optional.New(1))
		// then
		xassert.Equal(t, widget.WarningImportance, u.homeColoniesItem.SupportingImportance)
		assert.NotNil(t, u.homeColoniesItem.Trailing)
	})
	t.Run("shows no warning when all colonies are working", func(t *testing.T) {
		u.colonies.OnUpdate(optional.New(1))
		// when
		u.colonies.OnUpdate(optional.New(0))
		// then
		xassert.Equal(t, widget.MediumImportance, u.homeColoniesItem.SupportingImportance)
		assert.Nil(t, u.homeColoniesItem.Trailing)
	})
}

func TestMobileUI_RefreshMoreBadge(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	cases := []struct {
		name           string
		hasUpdateError bool
		hasScopeError  bool
		isOffline      bool
		hasUpdate      bool
		wantBadge      bool
		wantImportance widget.Importance
	}{
		{"nothing", false, false, false, false, false, 0},
		{"update error", true, false, false, false, true, widget.DangerImportance},
		{"scope error", false, true, false, false, true, widget.WarningImportance},
		{"offline", false, false, true, false, true, widget.WarningImportance},
		{"update available", false, false, false, true, true, widget.HighImportance},
		{"update error before others", true, true, true, true, true, widget.DangerImportance},
		{"scope error before update", false, true, false, true, true, widget.WarningImportance},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			u := MakeFakeMobileUI(t, st, NewFakeApp(t))
			u.hasUpdateError.Store(tc.hasUpdateError)
			u.hasScopeError.Store(tc.hasScopeError)
			u.isOffline.Store(tc.isOffline)
			u.hasUpdate.Store(tc.hasUpdate)
			// when
			u.refreshMoreBadge()
			// then
			importance, ok := u.navBar.Badge(moreDestination)
			xassert.Equal(t, tc.wantBadge, ok)
			xassert.Equal(t, tc.wantImportance, importance)
		})
	}
	t.Run("hides badge when nothing needs attention anymore", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.hasUpdate.Store(true)
		u.refreshMoreBadge()
		u.hasUpdate.Store(false)
		// when
		u.refreshMoreBadge()
		// then
		_, ok := u.navBar.Badge(moreDestination)
		assert.False(t, ok)
	})
}

func TestMobileUI_UpdateUpdateStatus(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	original := xgoesi.TimeNow
	t.Cleanup(func() {
		xgoesi.TimeNow = original
	})
	setNow := func(s string) {
		x, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		xgoesi.TimeNow = func() time.Time {
			return x
		}
	}
	t.Run("shows errors of the status cache", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      app.SectionCharacterSkills,
			ErrorMessage: "error",
		})
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		setNow("2026-09-24T12:00:00Z")
		// when
		u.updateUpdateStatus(t.Context())
		// then
		xassert.Equal(t, u.scs.Summary().Display(), u.moreUpdateStatusItem.Supporting)
		xassert.Equal(t, widget.DangerImportance, u.moreUpdateStatusItem.SupportingImportance)
		assert.NotNil(t, u.moreUpdateStatusItem.Trailing)
		xassert.Equal(t, "1 ERRORS", u.homeUpdateStatusItem.label.Text)
		xassert.Equal(t, widget.DangerImportance, u.homeUpdateStatusItem.label.Importance)
		importance, ok := u.navBar.Badge(moreDestination)
		assert.True(t, ok)
		xassert.Equal(t, widget.DangerImportance, importance)
		assert.False(t, u.IsOffline())
	})
	t.Run("shows no error icon when there are no errors", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		setNow("2026-09-24T12:00:00Z")
		// when
		u.updateUpdateStatus(t.Context())
		// then
		xassert.Equal(t, u.scs.Summary().Display(), u.moreUpdateStatusItem.Supporting)
		assert.Nil(t, u.moreUpdateStatusItem.Trailing)
		_, ok := u.navBar.Badge(moreDestination)
		assert.False(t, ok)
	})
	t.Run("shows OFF during daily downtime", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		setNow("2026-09-24T11:05:00Z")
		// when
		u.updateUpdateStatus(t.Context())
		// then
		assert.True(t, strings.HasPrefix(u.moreUpdateStatusItem.Supporting, "Off during daily downtime:"))
		xassert.Equal(t, widget.WarningImportance, u.moreUpdateStatusItem.SupportingImportance)
		xassert.Equal(t, "OFF", u.homeUpdateStatusItem.label.Text)
		importance, ok := u.navBar.Badge(moreDestination)
		assert.True(t, ok)
		xassert.Equal(t, widget.WarningImportance, importance)
		assert.True(t, u.IsOffline())
	})
	t.Run("shows warning badge during daily downtime after errors", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      app.SectionCharacterSkills,
			ErrorMessage: "error",
		})
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		setNow("2026-09-24T12:00:00Z")
		u.updateUpdateStatus(t.Context())
		setNow("2026-09-24T11:05:00Z")
		// when
		u.updateUpdateStatus(t.Context())
		// then
		importance, ok := u.navBar.Badge(moreDestination)
		assert.True(t, ok)
		xassert.Equal(t, widget.WarningImportance, importance)
	})
	t.Run("is online again after daily downtime", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		setNow("2026-09-24T11:05:00Z")
		u.updateUpdateStatus(t.Context())
		setNow("2026-09-24T12:00:00Z")
		// when
		u.updateUpdateStatus(t.Context())
		// then
		assert.False(t, u.IsOffline())
		_, ok := u.navBar.Badge(moreDestination)
		assert.False(t, ok)
	})
}

func TestMobileUI_OnUpdateMissingScope(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	t.Run("shows warning when characters are missing scopes", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		// when
		u.onUpdateMissingScope(2)
		// then
		assert.NotNil(t, u.moreManageCharactersItem.Trailing)
		importance, ok := u.navBar.Badge(moreDestination)
		assert.True(t, ok)
		xassert.Equal(t, widget.WarningImportance, importance)
	})
	t.Run("removes warning when no character is missing scopes", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.onUpdateMissingScope(2)
		// when
		u.onUpdateMissingScope(0)
		// then
		assert.Nil(t, u.moreManageCharactersItem.Trailing)
		_, ok := u.navBar.Badge(moreDestination)
		assert.False(t, ok)
	})
}

func TestMobileUI_UpdateCharacterCount(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := MakeFakeMobileUI(t, st, NewFakeApp(t))
	factory.CreateCharacter()
	factory.CreateCharacter()
	// when
	u.updateCharacterCount(t.Context())
	// then
	xassert.Equal(t, "2 characters", u.moreManageCharactersItem.Supporting)
}

func TestMobileUI_UpdateVersionHint(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)
	const url = "https://api.github.com/repos/ErikKalkoken/evebuddy/releases/latest"
	t.Run("shows hint when a newer version is available", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder("GET", url, httpmock.NewJsonResponderOrPanic(200, map[string]any{"tag_name": "v0.2.0"}))
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "0.1.0"})
		u := MakeFakeMobileUI(t, st, fa)
		// when
		u.updateVersionHint(t.Context())
		// then
		xassert.Equal(t, "Update available", u.moreAboutItem.Supporting)
		xassert.Equal(t, widget.HighImportance, u.moreAboutItem.SupportingImportance)
		assert.NotNil(t, u.moreAboutItem.Trailing)
		importance, ok := u.navBar.Badge(moreDestination)
		assert.True(t, ok)
		xassert.Equal(t, widget.HighImportance, importance)
	})
	t.Run("removes hint when the version is current", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder("GET", url, httpmock.NewJsonResponderOrPanic(200, map[string]any{"tag_name": "v0.2.0"}))
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "0.1.0"})
		u := MakeFakeMobileUI(t, st, fa)
		u.updateVersionHint(t.Context())
		fa.SetMetadata(fyne.AppMetadata{Version: "0.2.0"})
		// when
		u.updateVersionHint(t.Context())
		// then
		xassert.Equal(t, "", u.moreAboutItem.Supporting)
		xassert.Equal(t, widget.MediumImportance, u.moreAboutItem.SupportingImportance)
		assert.Nil(t, u.moreAboutItem.Trailing)
		_, ok := u.navBar.Badge(moreDestination)
		assert.False(t, ok)
	})
	t.Run("keeps hint when the version check fails", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder("GET", url, httpmock.NewJsonResponderOrPanic(200, map[string]any{"tag_name": "v0.2.0"}))
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "0.1.0"})
		u := MakeFakeMobileUI(t, st, fa)
		u.updateVersionHint(t.Context())
		httpmock.Reset()
		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(500, "error"))
		// when
		u.updateVersionHint(t.Context())
		// then
		xassert.Equal(t, "Update available", u.moreAboutItem.Supporting)
	})
}

func TestMobileUI_Navigation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	t.Run("back key returns to the previous page of the selected destination", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
		u.homeNav.Push(xwidget.NewAppBar("Dummy", widget.NewLabel("Dummy")))
		u.characterNav.Push(xwidget.NewAppBar("Dummy", widget.NewLabel("Dummy")))
		// when
		u.handleTypedKey(&fyne.KeyEvent{Name: mobile.KeyBack})
		// then
		assert.True(t, u.homeNav.IsRoot())
		assert.False(t, u.characterNav.IsRoot())
	})
	t.Run("ignores other keys", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
		u.homeNav.Push(xwidget.NewAppBar("Dummy", widget.NewLabel("Dummy")))
		// when
		u.handleTypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
		// then
		assert.False(t, u.homeNav.IsRoot())
	})
	t.Run("selecting the current destination again returns to its root page", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
		u.navBar.Enable(corporationDestination)
		for _, tc := range []struct {
			name string
			id   int
			nav  *xwidget.Navigator
		}{
			{"home", homeDestination, u.homeNav},
			{"character", characterDestination, u.characterNav},
			{"corporation", corporationDestination, u.corporationNav},
			{"more", moreDestination, u.moreNav},
		} {
			t.Run(tc.name, func(t *testing.T) {
				u.navBar.Select(tc.id)
				tc.nav.Push(xwidget.NewAppBar("Dummy", widget.NewLabel("Dummy")))
				// when
				u.navBar.Select(tc.id)
				// then
				assert.True(t, tc.nav.IsRoot())
			})
		}
	})
	t.Run("closes character mail when the reading pane is cleared", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
		u.navBar.Select(characterDestination)
		u.characterMailItem.OnTapped()
		mailPage := u.characterNav.Current()
		u.characterMails.MessagePane.OnSelected()
		// when
		u.characterMails.ReadingPane.OnCleared()
		// then
		assert.Same(t, mailPage, u.characterNav.Current())
	})
	t.Run("closes unified mail when the reading pane is cleared", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
		u.homeMailItem.OnTapped()
		mailPage := u.homeNav.Current()
		u.unifiedMails.MessagePane.OnSelected()
		// when
		u.unifiedMails.ReadingPane.OnCleared()
		// then
		assert.Same(t, mailPage, u.homeNav.Current())
	})
	t.Run("keeps other pages when the reading pane is cleared", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeMobileUI(t, st, NewFakeApp(t))
		u.handleCurrentCharacterExchanged(t.Context(), factory.CreateCharacter())
		u.homeMailItem.OnTapped()
		u.unifiedMails.MessagePane.OnSelected()
		u.homeNav.Pop()
		mailPage := u.homeNav.Current()
		// when
		u.unifiedMails.ReadingPane.OnCleared()
		// then
		assert.Same(t, mailPage, u.homeNav.Current())
	})
}
