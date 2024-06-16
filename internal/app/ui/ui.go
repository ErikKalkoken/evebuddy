// Package ui contains the code for rendering the UI.
package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
)

// UI constants
const (
	myDateTime      = "2006.01.02 15:04"
	defaultIconSize = 32
	myFloatFormat   = "#,###.##"
)

// The ui is the root object of the UI and contains all UI areas.
//
// Each UI area holds a pointer of the ui instance, so that areas can
// call methods on other UI areas and access shared variables in the UI.
type ui struct {
	fyneApp               fyne.App
	assetsArea            *assetsArea
	assetSearchArea       *assetSearchArea
	attributesArea        *attributesArea
	biographyArea         *biographyArea
	character             *app.Character
	isDebug               bool
	implantsArea          *implantsArea
	jumpClonesArea        *jumpClonesArea
	mailArea              *mailArea
	mailTab               *container.TabItem
	overviewArea          *overviewArea
	overviewTab           *container.TabItem
	statusBarArea         *statusBarArea
	sv                    *Service
	skillCatalogueArea    *skillCatalogueArea
	skillqueueArea        *skillqueueArea
	skillqueueTab         *container.TabItem
	shipsArea             *shipsArea
	statusWindow          fyne.Window
	toolbarArea           *toolbarArea
	tabs                  *container.AppTabs
	walletJournalArea     *walletJournalArea
	walletTransactionArea *walletTransactionArea
	wealthArea            *wealthArea
	window                fyne.Window
}

// NewUI build the UI and returns it.
func NewUI(sv *Service, isDebug bool) *ui {
	fyneApp := fyneapp.New()
	w := fyneApp.NewWindow(appName(fyneApp))
	u := &ui{
		fyneApp: fyneApp,
		isDebug: isDebug,
		sv:      sv,
		window:  w,
	}

	u.assetsArea = u.newAssetsArea()
	assetsTab := container.NewTabItemWithIcon("Assets",
		theme.NewThemedResource(resourceInventory2Svg), container.NewAppTabs(
			container.NewTabItem("Assets", u.assetsArea.content),
		))

	u.attributesArea = u.newAttributesArena()
	u.biographyArea = u.newBiographyArea()
	u.jumpClonesArea = u.NewJumpClonesArea()
	u.implantsArea = u.newImplantsArea()
	characterTab := container.NewTabItemWithIcon("Character",
		theme.NewThemedResource(resourcePortraitSvg), container.NewAppTabs(
			container.NewTabItem("Augmentations", u.implantsArea.content),
			container.NewTabItem("Jump Clones", u.jumpClonesArea.content),
			container.NewTabItem("Attributes", u.attributesArea.content),
			container.NewTabItem("Biography", u.biographyArea.content),
		))

	u.mailArea = u.newMailArea()
	u.mailTab = container.NewTabItemWithIcon("Mail",
		theme.MailComposeIcon(), container.NewAppTabs(
			container.NewTabItem("Mail", u.mailArea.content),
			// container.NewTabItem("Notifications", widget.NewLabel("PLACEHOLDER")),
		))

	u.overviewArea = u.newOverviewArea()
	u.assetSearchArea = u.newAssetSearchArea()
	u.wealthArea = u.newWealthArea()
	u.overviewTab = container.NewTabItemWithIcon("Characters",
		theme.NewThemedResource(resourceGroupSvg), container.NewAppTabs(
			container.NewTabItem("Overview", u.overviewArea.content),
			container.NewTabItem("Assets", u.assetSearchArea.content),
			container.NewTabItem("Wealth", u.wealthArea.content),
		))

	u.skillqueueArea = u.newSkillqueueArea()
	u.skillCatalogueArea = u.newSkillCatalogueArea()
	u.shipsArea = u.newShipArea()
	u.skillqueueTab = container.NewTabItemWithIcon("Skills",
		theme.NewThemedResource(resourceSchoolSvg), container.NewAppTabs(
			container.NewTabItem("Training Queue", u.skillqueueArea.content),
			container.NewTabItem("Skill Catalogue", u.skillCatalogueArea.content),
			container.NewTabItem("Ships", u.shipsArea.content),
		))

	u.walletJournalArea = u.newWalletJournalArea()
	u.walletTransactionArea = u.newWalletTransactionArea()
	walletTab := container.NewTabItemWithIcon("Wallet",
		theme.NewThemedResource(resourceAttachmoneySvg), container.NewAppTabs(
			container.NewTabItem("Transactions", u.walletJournalArea.content),
			container.NewTabItem("Market Transactions", u.walletTransactionArea.content),
		))

	u.statusBarArea = u.newStatusBarArea()
	u.toolbarArea = u.newToolbarArea()

	u.tabs = container.NewAppTabs(assetsTab, characterTab, u.mailTab, u.skillqueueTab, walletTab, u.overviewTab)
	u.tabs.SetTabLocation(container.TabLocationLeading)

	// for experiments
	// btn := widget.NewButton("Show experiment", func() {
	// 	err := u.sv.EveUniverse.UpdateEveShipSkills(context.TODO())
	// 	if err != nil {
	// 		slog.Error("UpdUpdateEveShipSkills failed", "err", err)
	// 	}
	// })
	// x := widgets.NewTappableImage(resourceSkinicon64pxPng, func() {
	// 	fmt.Println("Tapped")
	// })

	mainContent := container.NewBorder(u.toolbarArea.content, u.statusBarArea.content, nil, nil, u.tabs)
	w.SetContent(mainContent)
	w.SetMaster()

	var c *app.Character
	cID, ok, err := sv.Dictionary.Int(app.SettingLastCharacterID)
	if err == nil && ok {
		c, err = sv.Character.GetCharacter(context.Background(), int32(cID))
		if err != nil {
			if !errors.Is(err, character.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c != nil {
		u.setCharacter(c)
	} else {
		u.resetCharacter()
	}
	keyW := "window-width"
	width, ok, err := u.sv.Dictionary.Float32(keyW)
	if err != nil || !ok {
		width = 1000
	}
	keyH := "window-height"
	height, ok, err := u.sv.Dictionary.Float32(keyH)
	if err != nil || !ok {
		width = 600
	}
	w.Resize(fyne.NewSize(width, height))

	keyTabsMainID := "tabs-main-id"
	index, ok, err := u.sv.Dictionary.Int(keyTabsMainID)
	if err == nil && ok {
		u.tabs.SelectIndex(index)
	}
	makeSubTabsKey := func(i int) string {
		return fmt.Sprintf("tabs-sub%d-id", i)
	}
	for i, o := range u.tabs.Items {
		tabs, ok := o.Content.(*container.AppTabs)
		if !ok {
			continue
		}
		key := makeSubTabsKey(i)
		index, ok, err := u.sv.Dictionary.Int(key)
		if err == nil && ok {
			tabs.SelectIndex(index)
		}
	}
	w.SetOnClosed(func() {
		s := w.Canvas().Size()
		u.sv.Dictionary.SetFloat32(keyW, s.Width)
		u.sv.Dictionary.SetFloat32(keyH, s.Height)
		index := u.tabs.SelectedIndex()
		u.sv.Dictionary.SetInt(keyTabsMainID, index)
		for i, o := range u.tabs.Items {
			tabs, ok := o.Content.(*container.AppTabs)
			if !ok {
				continue
			}
			key := makeSubTabsKey(i)
			index := tabs.SelectedIndex()
			u.sv.Dictionary.SetInt(key, index)
		}
	})

	name, ok, err := u.sv.Dictionary.String(app.SettingTheme)
	if err != nil || !ok {
		name = app.ThemeAuto
	}
	u.setTheme(name)
	return u
}

func (u *ui) setTheme(name string) {
	switch name {
	case app.ThemeAuto:
		switch u.fyneApp.Settings().ThemeVariant() {
		case 0:
			u.fyneApp.Settings().SetTheme(theme.DarkTheme())
		default:
			u.fyneApp.Settings().SetTheme(theme.LightTheme())
		}
	case app.ThemeLight:
		u.fyneApp.Settings().SetTheme(theme.LightTheme())
	case app.ThemeDark:
		u.fyneApp.Settings().SetTheme(theme.DarkTheme())
	}
}

// ShowAndRun shows the UI and runs it (blocking).
func (u *ui) ShowAndRun() {
	go func() {
		// Workaround to mitigate a bug that causes the window to sometimes render
		// only in parts and freeze. The issue is known to happen on Linux desktops.
		if runtime.GOOS == "linux" {
			time.Sleep(1000 * time.Millisecond)
			s := u.window.Canvas().Size()
			u.window.Resize(fyne.NewSize(s.Width-0.2, s.Height-0.2))
			u.window.Resize(fyne.NewSize(s.Width, s.Height))
		}
		u.statusBarArea.StartUpdateTicker()
		u.startUpdateTickerGeneralSections()
		u.startUpdateTickerCharacters()
	}()
	u.refreshOverview()
	u.window.ShowAndRun()
}

// characterID returns the ID of the current character or 0 if non it set.
func (u *ui) characterID() int32 {
	if u.character == nil {
		return 0
	}
	return u.character.ID
}

func (u *ui) currentCharacter() *app.Character {
	return u.character
}

func (u *ui) hasCharacter() bool {
	return u.character != nil
}

func (u *ui) loadCharacter(ctx context.Context, characterID int32) error {
	c, err := u.sv.Character.GetCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	u.setCharacter(c)
	return nil
}

func (u *ui) setCharacter(c *app.Character) {
	u.character = c
	err := u.sv.Dictionary.SetInt(app.SettingLastCharacterID, int(c.ID))
	if err != nil {
		slog.Error("Failed to update last character setting", "characterID", c.ID)
	}
	u.refreshCharacter()
}

func (u *ui) refreshCharacter() {
	u.assetsArea.redraw()
	u.assetSearchArea.refresh()
	u.attributesArea.refresh()
	u.biographyArea.refresh()
	u.jumpClonesArea.redraw()
	u.implantsArea.refresh()
	u.mailArea.redraw()
	u.shipsArea.refresh()
	u.skillqueueArea.refresh()
	u.skillCatalogueArea.redraw()
	u.toolbarArea.refresh()
	u.walletJournalArea.refresh()
	u.walletTransactionArea.refresh()
	u.wealthArea.refresh()
	c := u.currentCharacter()
	if c != nil {
		for i := range u.tabs.Items {
			u.tabs.EnableIndex(i)
		}
		subTabs := u.overviewTab.Content.(*container.AppTabs)
		for i := range subTabs.Items {
			subTabs.EnableIndex(i)
		}
		u.updateCharacterAndRefreshIfNeeded(context.Background(), c.ID, false)
	} else {
		for i := range u.tabs.Items {
			u.tabs.DisableIndex(i)
		}
		u.tabs.Select(u.overviewTab)
		subTabs := u.overviewTab.Content.(*container.AppTabs)
		for i := range subTabs.Items {
			subTabs.DisableIndex(i)
		}
	}
	go u.statusBarArea.characterUpdateStatusArea.refresh()
	u.window.Content().Refresh()
}

func (u *ui) setAnyCharacter() error {
	c, err := u.sv.Character.GetAnyCharacter(context.Background())
	if errors.Is(err, character.ErrNotFound) {
		u.resetCharacter()
		return nil
	} else if err != nil {
		return err
	}
	u.setCharacter(c)
	return nil
}

func (u *ui) refreshOverview() {
	u.overviewArea.refresh()
}

func (u *ui) resetCharacter() {
	u.character = nil
	err := u.sv.Dictionary.Delete(app.SettingLastCharacterID)
	if err != nil {
		slog.Error("Failed to delete last character setting")
	}
	u.refreshCharacter()
}

func (u *ui) showErrorDialog(message string, err error) {
	text := widget.NewLabel(fmt.Sprintf("%s\n\n%s", message, humanize.Error(err)))
	text.Wrapping = fyne.TextWrapWord
	text.Importance = widget.DangerImportance
	x := container.NewVScroll(text)
	x.SetMinSize(fyne.Size{Width: 400, Height: 100})
	d := dialog.NewCustom("Error", "OK", x, u.window)
	d.Show()
}

func appName(a fyne.App) string {
	info := a.Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
}
