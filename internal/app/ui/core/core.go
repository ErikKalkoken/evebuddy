// Package core provides the core components to construct the UI.
package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/mobile"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatusservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/assets"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/characters"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/clones"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/contracts"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/corporations"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/gamesearch"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/industry"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infoviewer"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/skills"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/wallets"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	"github.com/ErikKalkoken/evebuddy/internal/github"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xmaps"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// update info
const (
	githubOwner = "ErikKalkoken"
	githubRepo  = "evebuddy"
)

// ticker
const (
	refreshUITick           = 30 * time.Second
	characterUpdateTick     = 60 * time.Second
	corporationUpdateTick   = 60 * time.Second
	eveUniverseUpdateTick   = 300 * time.Second
	delayBeforeUpdateStatus = 3 * time.Second
)

// Default ScaleMode for images
var defaultImageScaleMode canvas.ImageScale

type UIParams struct {
	App         fyne.App
	Character   *characterservice.CharacterService
	Corporation *corporationservice.CorporationService
	ESIStatus   *esistatusservice.ESIStatusService
	EVEImage    ui.EVEImageService
	EVEUniverse *eveuniverseservice.EVEUniverseService
	Janice      *janiceservice.JaniceService
	StatusCache *statuscache.StatusCache
	Signals     *app.Signals
	Settings    *settings.Settings
	// optional
	ClearCacheFunc   func()
	ConcurrencyLimit int
	DataPaths        map[string]string
	IsFakeMobile     bool
	IsMobile         bool
	IsOfflineMode    bool
	IsUpdateDisabled bool
}

// baseUI represents the core UI logic and is used by both the desktop and mobile UI.
type baseUI struct {
	// Callbacks
	clearCache                      func() // clear all caches
	hideMailIndicator               func()
	onAppFirstStarted               func()
	onAppStopped                    func()
	onAppTerminated                 func()
	onSetCharacter                  func(*app.Character)
	onShowCharacter                 func()
	onSetCorporation                func(*app.Corporation)
	onShowAndRun                    func()
	onUpdateCorporationWalletTotals func(balance float64, ok bool)
	onUpdateMissingScope            func(characterCount int)
	onUpdateStatus                  func(ctx context.Context)
	showMailIndicator               func()
	showManageCharacters            func()

	// UI elements
	assetSearchAll          *assets.Search
	augmentations           *clones.Augmentations
	characterAssetBrowser   *assets.Browser
	characterAttributes     *skills.Attributes
	characterAugmentations  *clones.CharacterAugmentations
	characterBiography      *characters.Biography
	characterContacts       *characters.Contacts
	characterCommunications *characters.Communications
	characterCorporation    *corporations.CorporationSheet
	characterJumpClones     *clones.CharacterClones
	characterMails          *characters.Mails
	characterOverview       *characters.Overview
	characterSheet          *characters.CharacterSheet
	characterShips          *skills.FlyableShips
	characterSkillCatalogue *skills.Catalogue
	characterSkillQueue     *skills.Queue
	characterWallet         *wallets.CharacterWallet
	clones                  *clones.Clones
	colonies                *industry.Colonies
	contracts               *contracts.Contracts
	corporationAssetBrowser *assets.Browser
	corporationAssetSearch  *assets.Search
	corporationContracts    *contracts.Contracts
	corporationIndyJobs     *industry.Jobs
	corporationMember       *corporations.Members
	corporationSheet        *corporations.CorporationSheet
	corporationStructures   *corporations.Structures
	corporationWallets      map[app.Division]*wallets.CorporationWallet
	gameSearch              *gamesearch.GameSearch
	industryJobs            *industry.Jobs
	loyaltyPoints           *wallets.LoyaltyPoints
	marketOrdersBuy         *industry.MarketOrders
	marketOrdersSell        *industry.MarketOrders
	slotsManufacturing      *industry.Slots
	slotsReactions          *industry.Slots
	slotsResearch           *industry.Slots
	snackbar                *xwidget.Snackbar
	statusText              *statusText
	training                *skills.Training
	wealth                  *wallets.Wealth
	iw                      *infoviewer.InfoViewer

	// Services
	cs       *characterservice.CharacterService
	eis      ui.EVEImageService
	ess      *esistatusservice.ESIStatusService
	eus      *eveuniverseservice.EVEUniverseService
	js       *janiceservice.JaniceService
	rs       *corporationservice.CorporationService
	scs      *statuscache.StatusCache
	settings *settings.Settings

	// UI state & configuration
	app                            fyne.App
	avatarCache                    xsync.Map[int64, fyne.Resource]
	character                      atomic.Pointer[app.Character]
	characterAvatarPlaceholder64   fyne.Resource
	concurrencyLimit               int
	corporation                    atomic.Pointer[app.Corporation]
	corporationAvatarPlaceholder64 fyne.Resource
	dataPaths                      xmaps.OrderedMap[string, string] // Paths to user data
	defaultTheme                   fyne.Theme
	isDeveloperMode                atomic.Bool
	isFakeMobile                   bool        // Show mobile variant on a desktop (for development)
	isForeground                   atomic.Bool // whether the app is currently shown in the foreground
	isMobile                       bool        // whether Fyne has detected the app running on a mobile. Else we assume it's a desktop.
	isOffline                      atomic.Bool
	isOfflineMode                  bool
	isStartupCompleted             atomic.Bool // whether the app has completed startup (for testing)
	isUpdateDisabled               atomic.Bool // Whether to disable update tickers (useful for debugging)
	signals                        *app.Signals
	wasStarted                     atomic.Bool            // whether the app has already been started at least once
	window                         fyne.Window            // main window
	windows                        map[string]fyne.Window // child windows
}

// newBaseUI constructs and returns a new BaseUI.
//
// Note:Types embedding BaseUI should define callbacks instead of overwriting methods.
func newBaseUI(arg UIParams) *baseUI {
	if arg.Character == nil {
		panic("CharacterService missing")
	}
	if arg.Corporation == nil {
		panic("CorporationService missing")
	}
	if arg.ESIStatus == nil {
		panic("ESIStatusService missing")
	}
	if arg.EVEImage == nil {
		panic("EveImageService missing")
	}
	if arg.EVEUniverse == nil {
		panic("EveUniverseService missing")
	}
	if arg.Janice == nil {
		panic("JaniceService missing")
	}
	if arg.Settings == nil {
		panic("Settings missing")
	}
	if arg.StatusCache == nil {
		panic("StatusCacheService missing")
	}
	if arg.Signals == nil {
		panic("Signals missing")
	}
	characterAvatarPlaceholder64, _ := fynetools.MakeAvatar(icons.Characterplaceholder64Jpeg)
	corporationAvatarPlaceholder64, _ := fynetools.MakeAvatar(icons.Corporationplaceholder64Png)
	u := &baseUI{
		app:                            arg.App,
		concurrencyLimit:               -1, // Default is no limit
		corporationWallets:             make(map[app.Division]*wallets.CorporationWallet),
		cs:                             arg.Character,
		eis:                            arg.EVEImage,
		ess:                            arg.ESIStatus,
		eus:                            arg.EVEUniverse,
		isFakeMobile:                   arg.IsFakeMobile,
		isMobile:                       arg.IsMobile,
		isOfflineMode:                  arg.IsOfflineMode,
		js:                             arg.Janice,
		rs:                             arg.Corporation,
		scs:                            arg.StatusCache,
		settings:                       arg.Settings,
		signals:                        arg.Signals,
		statusText:                     newStatusText(),
		windows:                        make(map[string]fyne.Window),
		characterAvatarPlaceholder64:   characterAvatarPlaceholder64,
		corporationAvatarPlaceholder64: corporationAvatarPlaceholder64,
	}

	u.window = u.app.NewWindow(ui.Name())
	u.isUpdateDisabled.Store(arg.IsUpdateDisabled)

	if arg.ClearCacheFunc != nil {
		u.clearCache = arg.ClearCacheFunc
	} else {
		u.clearCache = func() {}
	}
	if arg.ConcurrencyLimit > 0 {
		u.concurrencyLimit = arg.ConcurrencyLimit
	}
	if len(arg.DataPaths) > 0 {
		u.dataPaths = arg.DataPaths
	} else {
		u.dataPaths = make(xmaps.OrderedMap[string, string])
	}

	if !u.isMobile {
		xwidget.DefaultImageScaleMode = canvas.ImageScaleFastest
		defaultImageScaleMode = canvas.ImageScaleFastest
	}

	// updateStatus refreshes all status pages and dynamic menus.
	updateStatus := func(ctx context.Context) {
		if u.onUpdateStatus == nil {
			return
		}
		u.onUpdateStatus(ctx)
	}

	// Signal logging and base listeners
	u.signals.CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		slog.Debug("Signal: CurrentCharacterExchanged", "characterID", c.IDOrZero())
		updateStatus(ctx)
	})
	u.signals.CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		slog.Debug("Signal: CharacterSectionChanged", "arg", arg)
		logErr := func(err error) {
			slog.Error("Failed to process CharacterSectionChanged", "arg", arg, "error", err)
		}
		isShown := arg.CharacterID == u.character.Load().IDOrZero()
		switch arg.Section {
		case app.SectionCharacterAssets:
			if isShown {
				u.ReloadCurrentCharacter(ctx)
			}
		case
			app.SectionCharacterJumpClones,
			app.SectionCharacterLocation,
			app.SectionCharacterOnline,
			app.SectionCharacterShip,
			app.SectionCharacterSkills,
			app.SectionCharacterWalletBalance:
			if isShown {
				u.ReloadCurrentCharacter(ctx)
			}
		case app.SectionCharacterMailHeaders:
			u.UpdateMailIndicator(ctx)
		case app.SectionCharacterRoles:
			updateStatus(ctx)
			character, err := u.cs.GetCharacter(ctx, arg.CharacterID)
			if err != nil {
				logErr(err)
				return
			}
			corporationID := character.EveCharacter.Corporation.ID
			ok, err := u.rs.HasCorporation(ctx, corporationID)
			if err != nil {
				logErr(err)
				return
			}
			if !ok {
				return
			}
			if err := u.rs.RemoveSectionDataWhenPermissionLost(ctx, corporationID); err != nil {
				logErr(err)
			}
			u.rs.UpdateCorporationAndRefreshIfNeeded(ctx, corporationID, true)
		}
	})
	u.signals.CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		updateStatus(ctx)
	})
	u.signals.CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		updateStatus(ctx)
	})

	u.signals.CorporationsChanged.AddListener(func(ctx context.Context, _ struct{}) {
		updateStatus(ctx)
	})
	u.signals.CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
		slog.Debug("Signal: CurrentCorporationExchanged", "corporationID", c.IDOrZero())
		updateStatus(ctx)
		u.updateCorporationWalletTotal(ctx)
	})
	u.signals.CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		slog.Debug("Signal: CorporationSectionChanged", "arg", arg)
		if u.CurrentCorporation().IDOrZero() != arg.CorporationID {
			return
		}
		if arg.Section == app.SectionCorporationWalletBalances {
			u.updateCorporationWalletTotal(ctx)
		}
	})

	u.signals.EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		slog.Debug("Signal: EveUniverseSectionChanged", "arg", arg)
		switch arg.Section {
		case app.SectionEveCharacters:
			if arg.Changed.Contains(u.character.Load().IDOrZero()) {
				u.ReloadCurrentCharacter(ctx)
			}
			characters := u.scs.ListCharacterIDs()
			if characters.ContainsAny(arg.Changed.All()) {
				updateStatus(ctx)
			}
		case app.SectionEveCorporations:
			corporationIDs, err := u.cs.ListCharacterCorporationIDs(ctx)
			if err != nil {
				slog.Error("Failed to update status", "arg", arg, "err", err)
				return
			}
			if arg.Changed.ContainsAny(corporationIDs.All()) {
				updateStatus(ctx)
			}
		case app.SectionEveMarketPrices:
			for _, c := range u.scs.ListCharacters() {
				_, err := u.cs.UpdateAssetTotalValue(ctx, c.ID)
				if err != nil {
					slog.Error("Failed to update asset value", "characterID", c.ID)
					continue
				}
			}
			u.ReloadCurrentCharacter(ctx)
		}
	})

	u.assetSearchAll = assets.NewSearchForAll(u)
	u.augmentations = clones.NewAugmentations(u)
	u.characterAssetBrowser = assets.NewCharacterBrowser(u)
	u.characterAttributes = skills.NewAttributes(u)
	u.characterAugmentations = clones.NewCharacterAugmentations(u)
	u.characterBiography = characters.NewBiography(u)
	u.characterContacts = characters.NewContacts(u)
	u.characterCommunications = characters.NewCommunications(u)
	u.characterCorporation = corporations.NewCorporationSheet(u, false)
	u.characterJumpClones = clones.NewCharacterClones(u)
	u.characterMails = characters.NewMails(u)
	u.characterOverview = characters.NewOverview(u)
	u.characterSheet = characters.NewCharacterSheet(u)
	u.characterShips = skills.NewFlyableShips(u)
	u.characterSkillCatalogue = skills.NewCatalogue(u)
	u.characterSkillQueue = skills.NewQueue(u)
	u.characterWallet = wallets.NewCharacterWallet(u)
	u.clones = clones.NewClones(u)
	u.colonies = industry.NewColonies(u)
	u.contracts = contracts.NewContractsForCharacters(u)
	u.corporationAssetBrowser = assets.NewCorporationBrowser(u)
	u.corporationAssetSearch = assets.NewSearchForCorporation(u)
	u.corporationContracts = contracts.NewContractsForCorporation(u)
	u.corporationIndyJobs = industry.NewJobsForCorporation(u)

	u.corporationMember = corporations.NewMembers(u)
	u.corporationStructures = corporations.NewStructures(u)
	u.corporationSheet = corporations.NewCorporationSheet(u, true)
	for _, d := range app.Divisions {
		u.corporationWallets[d] = wallets.NewCorporationWallet(u, d)
	}
	u.gameSearch = gamesearch.NewGameSearch(u)
	u.industryJobs = industry.NewJobsForOverview(u)
	u.loyaltyPoints = wallets.NewLoyaltyPoints(u)
	u.marketOrdersBuy = industry.NewMarketOrders(u, true)
	u.marketOrdersSell = industry.NewMarketOrders(u, false)
	u.slotsManufacturing = industry.NewSlots(u, app.ManufacturingJob)
	u.slotsReactions = industry.NewSlots(u, app.ReactionJob)
	u.slotsResearch = industry.NewSlots(u, app.ScienceJob)
	u.snackbar = xwidget.NewSnackbar(u.window)
	u.training = skills.NewTraining(u)
	u.wealth = wallets.NewWealth(u)

	u.iw = infoviewer.New(u)

	u.MainWindow().SetMaster()

	// SetOnStarted is called on initial start,
	// but also when an app is continued after it was temporarily stopped,
	// which happens regularly on mobile.
	u.app.Lifecycle().SetOnStarted(func() {
		u.Start()
	})
	u.app.Lifecycle().SetOnEnteredForeground(func() {
		slog.Debug("Entered foreground")
		u.isForeground.Store(true)
		ctx := context.Background()
		if u.isMobile {
			// When the app is restarted on mobile the UI must be
			// refreshed immediately to avoid showing stale data (e.g. timers) to users
			// and updates must be run at once
			go u.signals.RefreshTickerExpired.Emit(ctx, struct{}{})
			if !u.isOfflineMode && !u.isUpdateDisabled.Load() {
				go func() {
					time.Sleep(1 * time.Second) // allow app to fully load before updating
					go u.cs.UpdateCharactersIfNeeded(ctx, false)
					go u.rs.UpdateCorporationsIfNeeded(ctx, false)
					go u.eus.UpdateSectionsIfNeeded(ctx, false)
				}()
			}
		}
	})
	u.app.Lifecycle().SetOnExitedForeground(func() {
		slog.Debug("Exited foreground")
		u.isForeground.Store(false)
	})
	u.app.Lifecycle().SetOnStopped(func() {
		slog.Info("App stopped")
		if u.onAppStopped != nil {
			u.onAppStopped()
		}
	})
	return u
}

// Start starts the app and reports whether it was started.
func (u *baseUI) Start() bool {
	wasStarted := !u.wasStarted.CompareAndSwap(false, true)
	if wasStarted {
		slog.Info("App continued")
		return false
	}
	// First app start
	u.isDeveloperMode.Store(u.settings.DeveloperMode())
	u.defaultTheme = theme.Current()
	u.SetColorTheme(u.settings.ColorTheme())
	if u.isOfflineMode {
		slog.Info("App started in offline mode")
	} else {
		slog.Info("App started")
	}
	u.snackbar.Start()
	ctx := context.Background()
	go func() {
		var wg sync.WaitGroup
		wg.Go(func() {
			u.signals.AppInit.Emit(ctx, struct{}{})
		})
		wg.Go(func() {
			u.initCharacter(ctx)
		})
		wg.Go(func() {
			u.initCorporation(ctx)
		})
		wg.Wait()

		updateCharactersMissingScope := func(ctx context.Context) {
			cc, err := u.cs.CharactersWithMissingScopes(ctx)
			if err != nil {
				slog.Error("Failed to fetch characters with missing scopes", "error", err)
				return
			}
			if u.onUpdateMissingScope != nil {
				fyne.Do(func() {
					u.onUpdateMissingScope(len(cc))
				})
			}
		}
		u.signals.CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
			updateCharactersMissingScope(ctx)
		})
		u.signals.CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			updateCharactersMissingScope(ctx)
		})
		updateCharactersMissingScope(ctx)

		u.isStartupCompleted.Store(true)
		go func() {
			for range time.Tick(refreshUITick) {
				u.signals.RefreshTickerExpired.Emit(ctx, struct{}{})
			}
		}()
		if u.onAppFirstStarted != nil {
			u.onAppFirstStarted()
		}
		if !u.isOfflineMode && !u.isUpdateDisabled.Load() {
			time.Sleep(delayBeforeUpdateStatus) // allow app to fully load before updating
			slog.Info("Starting update ticker")
			u.eus.StartUpdateTicker(eveUniverseUpdateTick)
			u.cs.StartUpdateTickerCharacters(characterUpdateTick)
			u.rs.StartUpdateTickerCorporations(corporationUpdateTick)
		} else {
			slog.Info("Update ticker disabled")
		}
	}()
	return true
}

// ShowAndRun shows the UI and runs the Fyne loop (blocking),
func (u *baseUI) ShowAndRun() {
	if u.onShowAndRun != nil {
		u.onShowAndRun()
	}
	u.window.ShowAndRun()
	slog.Info("App terminated")
	if u.onAppTerminated != nil {
		u.onAppTerminated()
	}
}

//////////////////
// Services

func (u *baseUI) ClearAllCaches() {
	u.clearCache()
}

func (u *baseUI) Character() *characterservice.CharacterService {
	return u.cs
}

func (u *baseUI) Corporation() *corporationservice.CorporationService {
	return u.rs
}

func (u *baseUI) DataPaths() xmaps.OrderedMap[string, string] {
	return u.dataPaths
}

func (u *baseUI) EVEImage() ui.EVEImageService {
	return u.eis
}

func (u *baseUI) ESIStatus() *esistatusservice.ESIStatusService {
	return u.ess
}

func (u *baseUI) EVEUniverse() *eveuniverseservice.EVEUniverseService {
	return u.eus
}

// InfoViewer returns the info window.
func (u *baseUI) InfoViewer() ui.InfoViewer {
	return u.iw
}

func (u *baseUI) IsMobile() bool {
	return u.isMobile
}

func (u *baseUI) MainWindow() fyne.Window {
	return u.window
}

// MakeWindowTitle creates a standardized title for a window.
func (u *baseUI) MakeWindowTitle(parts ...string) string {
	if len(parts) == 0 {
		parts = append(parts, "PLACEHOLDER")
	}
	if u.isMobile {
		return parts[0]
	}
	parts = append(parts, ui.Name())
	return strings.Join(parts, " - ")
}

func (u *baseUI) SetDeveloperMode(b bool) {
	u.isDeveloperMode.Store(b)
}

func (u *baseUI) IsDeveloperMode() bool {
	return u.isDeveloperMode.Load()
}

func (u *baseUI) IsOffline() bool {
	return u.isOfflineMode || u.isOffline.Load()
}
func (u *baseUI) IsStartupCompleted() bool {
	return u.isStartupCompleted.Load()
}

func (u *baseUI) IsUpdateDisabled() bool {
	return u.isUpdateDisabled.Load()
}

func (u *baseUI) Janice() *janiceservice.JaniceService {
	return u.js
}

func (u *baseUI) Settings() *settings.Settings {
	return u.settings
}

func (u *baseUI) ShowCharacter(ctx context.Context, characterID int64) {
	character := u.character.Load()
	if u.onShowCharacter != nil {
		u.onShowCharacter()
	}
	if character.IDOrZero() != characterID {
		err := u.LoadCharacter(ctx, characterID)
		if err != nil {
			slog.Error("Failed to load character", "characterID", characterID, "error", err)
			u.ShowSnackbar(fmt.Sprintf("Failed to load character: %s", u.ErrorDisplay(err)))
			return
		}
	}
}

func (u *baseUI) ShowSnackbar(text string) {
	u.snackbar.Show(text)
}

func (u *baseUI) Signals() *app.Signals {
	return u.signals
}

func (u *baseUI) StatusCache() *statuscache.StatusCache {
	return u.scs
}

//////////////////
// Current character

func (u *baseUI) initCharacter(ctx context.Context) {
	var c *app.Character
	var err error
	if cID := u.settings.LastCharacterID(); cID != 0 {
		c, err = u.cs.GetCharacter(ctx, int64(cID))
		if err != nil {
			if !errors.Is(err, app.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c == nil {
		c, err = u.cs.GetAnyCharacter(ctx)
		if err != nil {
			if !errors.Is(err, app.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c == nil {
		u.ResetCharacter(ctx)
		return
	}
	u.SetCharacter(ctx, c)
}

// CurrentCharacter() returns the current character or nil when none is configured.
func (u *baseUI) CurrentCharacter() *app.Character {
	return u.character.Load()
}

func (u *baseUI) LoadCharacter(ctx context.Context, id int64) error {
	c, err := u.cs.GetCharacter(ctx, id)
	if err != nil {
		return fmt.Errorf("load character ID %d: %w", id, err)
	}
	u.SetCharacter(ctx, c)
	return nil
}

// ReloadCurrentCharacter reloads the current character from storage.
func (u *baseUI) ReloadCurrentCharacter(ctx context.Context) {
	id := u.character.Load().IDOrZero()
	if id == 0 {
		return
	}
	c, err := u.cs.GetCharacter(ctx, id)
	if err != nil {
		slog.Error("reload character", "characterID", id, "error", err)
	}
	u.character.Store(c)
}

func (u *baseUI) ResetCharacter(ctx context.Context) {
	u.character.Store(nil)
	go u.signals.CurrentCharacterExchanged.Emit(ctx, nil)
	u.settings.ResetLastCharacterID()
	// if u.onSetCharacter != nil {
	// 	u.onSetCharacter(nil)
	// }
}

func (u *baseUI) SetCharacter(ctx context.Context, c *app.Character) {
	u.character.Store(c)
	if u.onSetCharacter != nil {
		go u.onSetCharacter(c)
	}
	go u.signals.CurrentCharacterExchanged.Emit(ctx, c)
	u.settings.SetLastCharacterID(c.ID)
}

func (u *baseUI) SetAnyCharacter(ctx context.Context) error {
	c, err := u.cs.GetAnyCharacter(ctx)
	if errors.Is(err, app.ErrNotFound) {
		u.ResetCharacter(ctx)
		return nil
	} else if err != nil {
		return err
	}
	u.SetCharacter(ctx, c)
	return nil
}

//////////////////
// Current corporation

func (u *baseUI) initCorporation(ctx context.Context) {
	var c *app.Corporation
	var err error
	if cID := u.settings.LastCorporationID(); cID != 0 {
		c, err = u.rs.GetCorporation(ctx, int64(cID))
		if err != nil {
			if !errors.Is(err, app.ErrNotFound) {
				slog.Error("Failed to load corporation", "error", err)
			}
		}
	}
	if c == nil {
		c, err = u.rs.GetAnyCorporation(ctx)
		if err != nil {
			if !errors.Is(err, app.ErrNotFound) {
				slog.Error("Failed to load corporation", "error", err)
			}
		}
	}
	if c == nil {
		u.ResetCorporation(ctx)
		return
	}
	u.SetCorporation(ctx, c)
}

// CurrentCorporation returns the current corporation or nil if not set.
func (u *baseUI) CurrentCorporation() *app.Corporation {
	return u.corporation.Load()
}

func (u *baseUI) LoadCorporation(ctx context.Context, id int64) error {
	c, err := u.rs.GetCorporation(ctx, id)
	if err != nil {
		return fmt.Errorf("load corporation ID %d: %w", id, err)
	}
	u.SetCorporation(ctx, c)
	return nil
}

func (u *baseUI) ResetCorporation(ctx context.Context) {
	u.corporation.Store(nil)
	go u.signals.CurrentCorporationExchanged.Emit(ctx, nil)
	u.settings.ResetLastCorporationID()
	// if u.onSetCorporation != nil {
	// 	u.onSetCorporation(nil)
	// }
}

func (u *baseUI) SetCorporation(ctx context.Context, c *app.Corporation) {
	u.corporation.Store(c)
	if u.onSetCorporation != nil {
		go u.onSetCorporation(c)
	}
	go u.signals.CurrentCorporationExchanged.Emit(ctx, c)
	u.settings.SetLastCorporationID(c.ID)
}

func (u *baseUI) SetAnyCorporation(ctx context.Context) error {
	c, err := u.rs.GetAnyCorporation(ctx)
	if errors.Is(err, app.ErrNotFound) {
		u.ResetCorporation(ctx)
		return nil
	}
	if err != nil {
		return err
	}
	u.SetCorporation(ctx, c)
	return nil
}

//////////////////
// Home

func (u *baseUI) SetColorTheme(s settings.ColorTheme) {
	u.app.Settings().SetTheme(ui.New(u.defaultTheme, s))
}

func (u *baseUI) UpdateMailIndicator(ctx context.Context) {
	if u.showMailIndicator == nil || u.hideMailIndicator == nil {
		return
	}
	if !u.settings.SysTrayEnabled() {
		return
	}
	n, err := u.cs.GetAllMailUnreadCount(ctx)
	if err != nil {
		slog.Error("update mail indicator", "error", err)
		return
	}
	fyne.Do(func() {
		if n > 0 {
			u.showMailIndicator()
		} else {
			u.hideMailIndicator()
		}
	})
}

func (u *baseUI) ListCorporationsForSelection(ctx context.Context) ([]*app.EntityShort, error) {
	if u.settings.HideLimitedCorporations() {
		return u.rs.ListPrivilegedCorporations(ctx)
	}
	return u.cs.ListCharacterCorporations(ctx)
}

func (u *baseUI) updateCorporationWalletTotal(ctx context.Context) {
	v, ok := func() (float64, bool) {
		corporationID := u.CurrentCorporation().IDOrZero()
		if corporationID == 0 {
			return 0, false
		}
		hasRole, err := u.rs.PermittedSection(ctx, corporationID, app.SectionCorporationWalletBalances)
		if err != nil {
			slog.Error("Failed to determine role for corporation wallet", "error", err)
			return 0, false
		}
		if !hasRole {
			return 0, false
		}
		b, err := u.rs.GetWalletBalancesTotal(ctx, corporationID)
		if err != nil {
			slog.Error("Failed to update wallet total", "corporationID", corporationID, "error", err)
			return 0, false
		}
		return b.Value()
	}()
	fyne.Do(func() {
		u.onUpdateCorporationWalletTotals(v, ok)
	})
}

func (u *baseUI) availableUpdate(ctx context.Context) (github.VersionInfo, error) {
	current := u.app.Metadata().Version
	v, err := github.AvailableUpdate(ctx, githubOwner, githubRepo, current)
	if err != nil {
		return github.VersionInfo{}, err
	}
	return v, nil
}

// Avatars & switch menus

func (u *baseUI) setCharacterAvatarAsync(characterID int64, setIcon func(fyne.Resource)) {
	xwidget.LoadResourceAsyncWithCache(
		u.characterAvatarPlaceholder64,
		func() (fyne.Resource, bool) {
			return u.avatarCache.Load(characterID)
		},
		setIcon,
		func() (fyne.Resource, error) {
			r, err := u.eis.CharacterPortrait(characterID, ui.IconPixelSize)
			if err != nil {
				return nil, err
			}
			return fynetools.MakeAvatar(r)
		},
		func(r fyne.Resource) {
			u.avatarCache.Store(characterID, r)
		},
	)
}

func (u *baseUI) setCorporationAvatarAsync(corporationID int64, setIcon func(fyne.Resource)) {
	xwidget.LoadResourceAsyncWithCache(
		u.corporationAvatarPlaceholder64,
		func() (fyne.Resource, bool) {
			return u.avatarCache.Load(corporationID)
		},
		setIcon,
		func() (fyne.Resource, error) {
			r, err := u.eis.CorporationLogo(corporationID, ui.IconPixelSize)
			if err != nil {
				return nil, err
			}
			return fynetools.MakeAvatar(r)
		},
		func(r fyne.Resource) {
			u.avatarCache.Store(corporationID, r)
		},
	)
}

func (u *baseUI) setCharacterSwitchMenu(ctx context.Context, setItems func(items []*fyne.MenuItem), refresh func()) {
	cc := u.scs.ListCharacters()
	if len(cc) == 0 {
		it := fyne.NewMenuItem("No characters", nil)
		it.Disabled = true
		fyne.Do(func() {
			setItems(nil)
		})
		return
	}

	it := fyne.NewMenuItem("Switch to...", nil)
	it.Disabled = true
	items := []*fyne.MenuItem{it}
	currentID := u.character.Load().IDOrZero()
	for _, c := range cc {
		it := fyne.NewMenuItem(c.Name, func() {
			go func() {
				err := u.LoadCharacter(ctx, c.ID)
				if err != nil {
					slog.Error("make character switch menu", "error", err)
					u.snackbar.Show("ERROR: Failed to switch character")
				}
			}()
		})
		if c.ID == currentID {
			it.Icon = theme.NewThemedResource(icons.AccountCircleSvg)
			it.Disabled = true
		} else {
			it.Icon = u.characterAvatarPlaceholder64
			fyne.Do(func() {
				u.setCharacterAvatarAsync(c.ID, func(r fyne.Resource) {
					it.Icon = r
					refresh()
				})
			})
		}
		items = append(items, it)
	}
	fyne.Do(func() {
		setItems(items)
	})
}

func (u *baseUI) setCorporationSwitchMenu(ctx context.Context, setItems func(items []*fyne.MenuItem), refresh func()) {
	cc, err := u.ListCorporationsForSelection(ctx)
	if err != nil {
		slog.Error("Failed to fetch corporations", "error", err)
		fyne.Do(func() {
			setItems(nil)
		})
		return
	}

	if len(cc) == 0 {
		it := fyne.NewMenuItem("No corporations", nil)
		it.Disabled = true
		fyne.Do(func() {
			setItems(nil)
		})
		return
	}

	corporations := set.Collect(xiter.MapSlice(cc, func(x *app.EntityShort) int64 {
		return x.ID
	}))
	currentID := u.CurrentCorporation().IDOrZero()
	if currentID != 0 && !corporations.Contains(currentID) {
		u.SetAnyCorporation(ctx)
		return
	}

	it := fyne.NewMenuItem("Switch to...", nil)
	it.Disabled = true
	items := []*fyne.MenuItem{it}
	for _, c := range cc {
		it := fyne.NewMenuItem(c.Name, func() {
			go func() {
				err := u.LoadCorporation(ctx, c.ID)
				if err != nil {
					slog.Error("make corporation switch menu", "error", err)
					u.snackbar.Show("ERROR: Failed to switch corporation")
				}
			}()
		})
		if c.ID == currentID {
			it.Icon = theme.NewThemedResource(icons.StarCircleOutlineSvg)
			it.Disabled = true
		} else {
			fyne.Do(func() {
				u.setCorporationAvatarAsync(c.ID, func(r fyne.Resource) {
					it.Icon = r
					refresh()
				})
			})
		}
		items = append(items, it)
	}
	fyne.Do(func() {
		setItems(items)
	})
}

// ErrorDisplay returns a user friendly error message for an error.
// Or returns the full error when in developer mode.
func (u *baseUI) ErrorDisplay(err error) string {
	if u.isDeveloperMode.Load() {
		return err.Error()
	}
	return app.ErrorDisplay(err)
}

// Windows

// GetOrCreateWindow returns a unique window as defined by the given id string
// and reports whether a new window was created or the window already exists.
// When id is zero it will always create a new window.
func (u *baseUI) GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool) {
	w, ok, f := u.GetOrCreateWindowWithOnClosed(id, titles...)
	if f != nil {
		w.SetOnClosed(f)
	}
	return w, ok
}

// GetOrCreateWindowWithOnClosed is like GetOrCreateWindow,
// but returns an additional onClosed function which must be called when the window is closed.
// This allows constructing a custom onClosed callback for the window.
func (u *baseUI) GetOrCreateWindowWithOnClosed(id string, titles ...string) (window fyne.Window, created bool, onClosed func()) {
	if id != "" {
		if w, ok := u.windows[id]; ok {
			return w, false, nil
		}
	}

	w := u.app.NewWindow(u.MakeWindowTitle(titles...))

	if fyne.CurrentDevice().IsMobile() {
		w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
			if ev.Name != mobile.KeyBack {
				return
			}
			w.Close() // Back gesture closes window
		})
	}

	var f func()
	if id != "" {
		u.windows[id] = w
		f = func() {
			delete(u.windows, id)
		}
	} else {
		f = func() {}
	}

	return w, true, f
}

// statusText is a widget that can show/hide multiple status texts with a spinner.
type statusText struct {
	widget.BaseWidget

	label    *widget.Label
	messages map[string]string
	spinner  *widget.Activity
}

func newStatusText() *statusText {
	w := &statusText{
		messages: make(map[string]string),
		label:    widget.NewLabel(""),
		spinner:  widget.NewActivity(),
	}
	w.label.Hide()
	w.spinner.Hide()
	w.spinner.Stop()
	w.ExtendBaseWidget(w)
	return w
}

func (w *statusText) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewHBox(w.label, w.spinner)
	return widget.NewSimpleRenderer(c)
}

func (w *statusText) Set(key, text string) {
	w.messages[key] = text
	if w.label.Hidden {
		w.label.SetText(text)
		w.spinner.Start()
		w.label.Show()
		w.spinner.Show()
	}
}

func (w *statusText) Unset(key string) {
	delete(w.messages, key)
	if len(w.messages) == 0 {
		w.label.Hide()
		w.spinner.Hide()
		w.spinner.Stop()
	}
	var text string
	for _, s := range w.messages {
		text = s
		break
	}
	w.label.SetText(text)
}
