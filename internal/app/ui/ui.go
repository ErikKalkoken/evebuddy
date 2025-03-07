package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	"github.com/ErikKalkoken/evebuddy/internal/github"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// Base UI constants
const (
	DefaultIconPixelSize = 64
	DefaultIconUnitSize  = 32
	MyFloatFormat        = "#,###.##"
)

// update info
const (
	githubOwner        = "ErikKalkoken"
	githubRepo         = "evebuddy"
	fallbackWebsiteURL = "https://github.com/ErikKalkoken/evebuddy"
)

// BaseUI represents the core UI logic and is used by both the desktop and mobile UI.
type BaseUI struct {
	CacheService       app.CacheService
	CharacterService   *character.CharacterService
	ESIStatusService   app.ESIStatusService
	EveImageService    app.EveImageService
	EveUniverseService *eveuniverse.EveUniverseService
	StatusCacheService app.StatusCacheService
	// Paths to user data (for information only)
	DataPaths map[string]string
	// Run the app in offline mode
	IsOffline bool
	// Whether to disable update tickers (useful for debugging)
	IsUpdateTickerDisabled bool

	HideMailIndicator  func()
	OnAppFirstStarted  func()
	OnAppStopped       func()
	OnAppTerminated    func()
	OnInit             func(*app.Character)
	OnRefreshCharacter func(*app.Character)
	OnRefreshCross     func()
	OnRefreshStatus    func()
	OnSetCharacter     func(int32)
	OnShowAndRun       func()
	ShowMailIndicator  func()

	// need to be implemented for each platform
	ShowTypeInfoWindow     func(typeID, characterID int32, selectTab TypeWindowTab)
	ShowLocationInfoWindow func(int64)

	FyneApp fyne.App
	DeskApp desktop.App
	Window  fyne.Window

	Snackbar *iwidget.Snackbar

	AccountArea           *AccountArea
	AssetsArea            *AssetsArea
	AssetSearchArea       *AssetSearchArea
	AttributesArea        *Attributes
	BiographyArea         *BiographyArea
	ColoniesArea          *ColoniesArea
	ContractsArea         *ContractsArea
	ImplantsArea          *ImplantsArea
	JumpClonesArea        *JumpClonesArea
	LocationsArea         *LocationsArea
	MailArea              *MailArea
	NotificationsArea     *NotificationsArea
	OverviewArea          *OverviewArea
	PlanetArea            *PlanetArea
	SettingsArea          *SettingsArea
	ShipsArea             *ShipsArea
	SkillCatalogueArea    *SkillCatalogueArea
	SkillqueueArea        *SkillqueueArea
	TrainingArea          *TrainingArea
	WalletJournalArea     *WalletJournalArea
	WalletTransactionArea *WalletTransactionArea
	WealthArea            *WealthArea

	character    *app.Character
	statusWindow fyne.Window
	isMobile     bool
	wasStarted   atomic.Bool // whether the app has already been started at least once
	isForeground atomic.Bool // whether the app is currently shown in the foreground
}

// NewBaseUI constructs and returns a new BaseUI.
//
// Note:Types embedding BaseUI should define callbacks instead of overwriting methods.
func NewBaseUI(fyneApp fyne.App) *BaseUI {
	u := &BaseUI{
		FyneApp: fyneApp,
		ShowTypeInfoWindow: func(_, _ int32, _ TypeWindowTab) {
			panic("not implemented")
		},
		ShowLocationInfoWindow: func(_ int64) {
			panic("not implemented")
		},
		isMobile: fyne.CurrentDevice().IsMobile(),
	}
	u.Window = fyneApp.NewWindow(u.AppName())

	desk, ok := u.FyneApp.(desktop.App)
	if ok {
		u.DeskApp = desk
	}

	if u.IsDesktop() {
		iwidget.DefaultImageScaleMode = canvas.ImageScaleFastest
		appwidget.DefaultImageScaleMode = canvas.ImageScaleFastest
	}

	u.Snackbar = iwidget.NewSnackbar(u.Window)

	u.AccountArea = u.NewAccountArea()
	u.AssetsArea = u.NewAssetsArea()
	u.AssetSearchArea = u.NewAssetSearchArea()
	u.AttributesArea = u.NewAttributes()
	u.BiographyArea = u.NewBiographyArea()
	u.ColoniesArea = u.NewColoniesArea()
	u.ContractsArea = u.NewContractsArea()
	u.ImplantsArea = u.NewImplantsArea()
	u.JumpClonesArea = u.NewJumpClonesArea()
	u.LocationsArea = u.NewLocationsArea()
	u.MailArea = u.NewMailArea()
	u.NotificationsArea = u.NewNotificationsArea()
	u.OverviewArea = u.NewOverviewArea()
	u.PlanetArea = u.NewPlanetArea()
	u.SettingsArea = u.NewSettingsArea()
	u.ShipsArea = u.newShipArea()
	u.SkillCatalogueArea = u.NewSkillCatalogueArea()
	u.SkillqueueArea = u.NewSkillqueueArea()
	u.TrainingArea = u.NewTrainingArea()
	u.WalletJournalArea = u.NewWalletJournalArea()
	u.WalletTransactionArea = u.NewWalletTransactionArea()
	u.WealthArea = u.NewWealthArea()
	return u
}

func (u *BaseUI) AppName() string {
	info := u.FyneApp.Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
}

// Init initialized the app.
// It is meant for initialization logic that requires all services to be initialized and available.
// It should be called directly after the app was created and before the Fyne loop is started.
func (u *BaseUI) Init() {
	u.AccountArea.Refresh()
	var c *app.Character
	var err error
	ctx := context.Background()
	if cID := u.FyneApp.Preferences().Int(settingLastCharacterID); cID != 0 {
		c, err = u.CharacterService.GetCharacter(ctx, int32(cID))
		if err != nil {
			if !errors.Is(err, character.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c == nil {
		c, err = u.CharacterService.GetAnyCharacter(ctx)
		if err != nil {
			if !errors.Is(err, character.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c == nil {
		return
	}
	u.SetCharacter(c)
	if u.OnInit != nil {
		u.OnInit(c)
	}
}

func (u *BaseUI) IsDesktop() bool {
	return u.DeskApp != nil
}

func (u *BaseUI) IsMobile() bool {
	return u.isMobile
}

func (u *BaseUI) MakeWindowTitle(subTitle string) string {
	return fmt.Sprintf("%s - %s", subTitle, u.AppName())
}

// ShowAndRun shows the UI and runs the Fyne loop (blocking),
func (u *BaseUI) ShowAndRun() {
	// SetOnStarted is called on initial start,
	// but also when an app is coninued after it was temporarily stopped,
	// which can happen on mobile
	u.FyneApp.Lifecycle().SetOnStarted(func() {
		wasStarted := !u.wasStarted.CompareAndSwap(false, true)
		if wasStarted {
			slog.Info("App continued")
			return
		}
		// First app start
		slog.Info("App started")
		if u.IsOffline {
			slog.Info("Started in offline mode")
		}
		if u.IsUpdateTickerDisabled {
			slog.Info("Update ticker disabled")
		}
		go func() {
			time.Sleep(250 * time.Millisecond) // FIXME: Workaround for occasional progess bar panic
			u.RefreshCrossPages()
			if u.HasCharacter() {
				u.SetCharacter(u.character)
			} else {
				u.ResetCharacter()
			}
			u.RefreshStatus()
		}()
		u.Snackbar.Start()
		if !u.IsOffline && !u.IsUpdateTickerDisabled {
			u.isForeground.Store(true)
			go func() {
				u.startUpdateTickerGeneralSections()
				u.startUpdateTickerCharacters()
			}()
		}
		if u.OnAppFirstStarted != nil {
			u.OnAppFirstStarted()
		}
	})
	u.FyneApp.Lifecycle().SetOnEnteredForeground(func() {
		slog.Debug("Entered foreground")
		u.isForeground.Store(true)
		u.updateCharactersIfNeeded(context.Background())
		u.UpdateGeneralSectionsAndRefreshIfNeeded(false)
	})
	u.FyneApp.Lifecycle().SetOnExitedForeground(func() {
		slog.Debug("Exited foreground")
		u.isForeground.Store(false)
	})
	u.FyneApp.Lifecycle().SetOnStopped(func() {
		slog.Info("App stopped")
		if u.OnAppStopped != nil {
			u.OnAppStopped()
		}
	})
	if u.OnShowAndRun != nil {
		u.OnShowAndRun()
	}
	u.Window.ShowAndRun()
	slog.Info("App terminated")
	if u.OnAppTerminated != nil {
		u.OnAppTerminated()
	}
}

// CharacterID returns the ID of the current character or 0 if non it set.
func (u *BaseUI) CharacterID() int32 {
	if u.character == nil {
		return 0
	}
	return u.character.ID
}

func (u *BaseUI) CurrentCharacter() *app.Character {
	return u.character
}

func (u *BaseUI) HasCharacter() bool {
	return u.character != nil
}

func (u *BaseUI) LoadCharacter(id int32) error {
	c, err := u.CharacterService.GetCharacter(context.Background(), id)
	if err != nil {
		return fmt.Errorf("load character ID %d: %w", id, err)
	}
	u.SetCharacter(c)
	return nil
}

// RefreshStatus refreshed all status information pages.
func (u *BaseUI) RefreshStatus() {
	if u.OnRefreshStatus == nil {
		return
	}
	go u.OnRefreshStatus()
}

// RefreshCharacter refreshes all pages for the current character.
func (u *BaseUI) RefreshCharacter() {
	ff := map[string]func(){
		"assets":            u.AssetsArea.Redraw,
		"attributes":        u.AttributesArea.Refresh,
		"bio":               u.BiographyArea.Refresh,
		"contracts":         u.ContractsArea.Refresh,
		"implants":          u.ImplantsArea.Refresh,
		"jumpClones":        u.JumpClonesArea.Redraw,
		"mail":              u.MailArea.Redraw,
		"notifications":     u.NotificationsArea.Refresh,
		"planets":           u.PlanetArea.Refresh,
		"ships":             u.ShipsArea.Refresh,
		"skillCatalogue":    u.SkillCatalogueArea.Redraw,
		"skillqueue":        u.SkillqueueArea.Refresh,
		"walletJournal":     u.WalletJournalArea.Refresh,
		"walletTransaction": u.WalletTransactionArea.Refresh,
	}
	c := u.CurrentCharacter()
	if c != nil {
		slog.Debug("Refreshing character", "ID", c.EveCharacter.ID, "name", c.EveCharacter.Name)
	} else {
		slog.Debug("Refreshing without character")
	}
	if u.OnRefreshCharacter != nil {
		ff["OnRefreshCharacter"] = func() {
			u.OnRefreshCharacter(c)
		}
	}
	runFunctionsWithProgressModal("Loading character", ff, u.Window)
	if c != nil && !u.IsUpdateTickerDisabled {
		u.UpdateCharacterAndRefreshIfNeeded(context.Background(), c.ID, false)
	}
}

// RefreshCrossPages refreshed all pages that contain information about multiple characters.
func (u *BaseUI) RefreshCrossPages() {
	ff := map[string]func(){
		"assetSearch": u.AssetSearchArea.Refresh,
		"colony":      u.ColoniesArea.Refresh,
		"locations":   u.LocationsArea.Refresh,
		"overview":    u.OverviewArea.Refresh,
		"training":    u.TrainingArea.Refresh,
		"wealth":      u.WealthArea.Refresh,
	}
	if u.OnRefreshCross != nil {
		ff["onRefreshCross"] = u.OnRefreshCross
	}
	runFunctionsWithProgressModal("Updating characters", ff, u.Window)
}

func runFunctionsWithProgressModal(title string, ff map[string]func(), w fyne.Window) {
	m := kxmodal.NewProgress("Updating", title, func(p binding.Float) error {
		start := time.Now()
		myLog := slog.With("title", title)
		myLog.Debug("started")
		var wg sync.WaitGroup
		var completed atomic.Int64
		for name, f := range ff {
			wg.Add(1)
			go func() {
				defer wg.Done()
				start2 := time.Now()
				f()
				x := completed.Add(1)
				if err := p.Set(float64(x)); err != nil {
					myLog.Warn("failed set progress", "error", err)
				}
				myLog.Debug("part completed", "name", name, "duration", time.Since(start2).Milliseconds())
			}()
		}
		wg.Wait()
		myLog.Debug("completed", "duration", time.Since(start).Milliseconds())
		return nil
	}, float64(len(ff)), w)
	m.Start()
}

func (u *BaseUI) ResetCharacter() {
	u.character = nil
	u.FyneApp.Preferences().SetInt(settingLastCharacterID, 0)
	u.RefreshCharacter()
	u.RefreshStatus()
}

func (u *BaseUI) SetCharacter(c *app.Character) {
	u.character = c
	u.FyneApp.Preferences().SetInt(settingLastCharacterID, int(c.ID))
	u.RefreshCharacter()
	u.RefreshStatus()
	if u.OnSetCharacter != nil {
		u.OnSetCharacter(c.ID)
	}
}

func (u *BaseUI) SetAnyCharacter() error {
	c, err := u.CharacterService.GetAnyCharacter(context.Background())
	if errors.Is(err, character.ErrNotFound) {
		u.ResetCharacter()
		return nil
	} else if err != nil {
		return err
	}
	u.SetCharacter(c)
	return nil
}

func (u *BaseUI) MakeAboutPage() fyne.CanvasObject {
	v, err := github.NormalizeVersion(u.FyneApp.Metadata().Version)
	if err != nil {
		slog.Error("normalize local version", "error", err)
		v = "?"
	}
	local := widget.NewLabel(v)
	latest := widget.NewLabel("?")
	latest.Hide()
	spinner := widget.NewActivity()
	spinner.Start()
	go func() {
		var s string
		var i widget.Importance
		var isBold bool
		v, err := u.AvailableUpdate()
		if err != nil {
			slog.Error("fetch github version for about", "error", err)
			s = "ERROR"
			i = widget.DangerImportance
		} else if v.IsRemoteNewer {
			s = v.Latest
			isBold = true
		} else {
			s = v.Latest
		}
		latest.Text = s
		latest.TextStyle.Bold = isBold
		latest.Importance = i
		latest.Refresh()
		spinner.Hide()
		latest.Show()
	}()
	title := iwidget.NewLabelWithSize(u.AppName(), theme.SizeNameSubHeadingText)
	title.TextStyle.Bold = true
	c := container.New(
		layout.NewCustomPaddedVBoxLayout(0),
		title,
		container.New(layout.NewCustomPaddedVBoxLayout(0),
			container.NewHBox(widget.NewLabel("Latest version:"), layout.NewSpacer(), container.NewStack(spinner, latest)),
			container.NewHBox(widget.NewLabel("You have:"), layout.NewSpacer(), local),
		),
		container.NewHBox(
			widget.NewHyperlink("Website", u.WebsiteRootURL()),
			widget.NewHyperlink("Downloads", u.WebsiteRootURL().JoinPath("releases")),
		),
		widget.NewLabel("\"EVE\", \"EVE Online\", \"CCP\", \nand all related logos and images \nare trademarks or registered trademarks of CCP hf."),
		widget.NewLabel("(c) 2024-25 Erik Kalkoken"),
	)
	return c
}

func (u *BaseUI) ShowUpdateStatusWindow() {
	if u.statusWindow != nil {
		u.statusWindow.Show()
		return
	}
	w := u.FyneApp.NewWindow(u.MakeWindowTitle("Update Status"))
	a := u.NewUpdateStatusArea()
	a.Refresh()
	w.SetContent(a.Content)
	w.Resize(fyne.Size{Width: 1100, Height: 500})
	ctx, cancel := context.WithCancel(context.Background())
	a.StartTicker(ctx)
	w.SetOnClosed(func() {
		cancel()
		u.statusWindow = nil
	})
	u.statusWindow = w
	w.Show()
}

func (u *BaseUI) AvailableUpdate() (github.VersionInfo, error) {
	current := u.FyneApp.Metadata().Version
	v, err := github.AvailableUpdate(githubOwner, githubRepo, current)
	if err != nil {
		return github.VersionInfo{}, err
	}
	return v, nil
}

func (u *BaseUI) UpdateAvatar(id int32, setIcon func(fyne.Resource)) {
	r, err := u.EveImageService.CharacterPortrait(id, DefaultIconPixelSize)
	if err != nil {
		slog.Error("Failed to fetch character portrait", "characterID", id, "err", err)
		r = icon.Characterplaceholder64Jpeg
	}
	r2, err := fynetools.MakeAvatar(r)
	if err != nil {
		slog.Error("Failed to make avatar", "characterID", id, "err", err)
		r2 = icon.Characterplaceholder64Jpeg
	}
	setIcon(r2)
}

func (u *BaseUI) UpdateMailIndicator() {
	if u.ShowMailIndicator == nil || u.HideMailIndicator == nil {
		return
	}
	if !u.FyneApp.Preferences().BoolWithFallback(SettingSysTrayEnabled, SettingSysTrayEnabledDefault) {
		return
	}
	n, err := u.CharacterService.GetAllCharacterMailUnreadCount(context.Background())
	if err != nil {
		slog.Error("update mail indicator", "error", err)
		return
	}
	if n > 0 {
		u.ShowMailIndicator()
	} else {
		u.HideMailIndicator()
	}
}

func (u *BaseUI) WebsiteRootURL() *url.URL {
	s := u.FyneApp.Metadata().Custom["Website"]
	if s == "" {
		s = fallbackWebsiteURL
	}
	uri, err := url.Parse(s)
	if err != nil {
		slog.Error("parse main website URL")
		uri, _ = url.Parse(fallbackWebsiteURL)
	}
	return uri
}

func (u *BaseUI) MakeCharacterSwitchMenu(refresh func()) []*fyne.MenuItem {
	characterID := u.CharacterID()
	cc := u.StatusCacheService.ListCharacters()
	items := make([]*fyne.MenuItem, 0)
	if len(cc) == 0 {
		it := fyne.NewMenuItem("No characters", nil)
		it.Disabled = true
		items = append(items, it)
		return items
	}
	var wg sync.WaitGroup
	for _, c := range cc {
		it := fyne.NewMenuItem(c.Name, func() {
			err := u.LoadCharacter(c.ID)
			if err != nil {
				slog.Error("make character switch menu", "error", err)
				u.Snackbar.Show("ERROR: Failed to switch character")
			}
		})
		if c.ID == characterID {
			continue
		}
		it.Icon, _ = fynetools.MakeAvatar(icon.Characterplaceholder64Jpeg)
		wg.Add(1)
		go u.UpdateAvatar(c.ID, func(r fyne.Resource) {
			defer wg.Done()
			it.Icon = r
		})
		items = append(items, it)
	}
	go func() {
		wg.Wait()
		refresh()
	}()
	return items
}
