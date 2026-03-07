// Package ui implements the graphical user interface of the app.
package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetui"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterui"
	"github.com/ErikKalkoken/evebuddy/internal/app/clonesui"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationui"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatusservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/gamesearchui"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/industryui"
	"github.com/ErikKalkoken/evebuddy/internal/app/infowindow"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/skillui"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/walletui"
	"github.com/ErikKalkoken/evebuddy/internal/app/xtheme"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	"github.com/ErikKalkoken/evebuddy/internal/github"
	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
	"github.com/ErikKalkoken/evebuddy/internal/singleinstance"
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
	assetSearchAll          *assetui.AssetSearch
	augmentations           *clonesui.Augmentations
	characterAssetBrowser   *assetui.AssetBrowser
	characterAttributes     *skillui.Attributes
	characterAugmentations  *clonesui.CharacterAugmentations
	characterBiography      *characterui.Biography
	characterContacts       *characterui.Contacts
	characterCommunications *characterui.Communications
	characterCorporation    *corporationui.CorporationSheet
	characterJumpClones     *clonesui.CharacterClones
	characterMails          *characterui.Mails
	characterOverview       *characterui.Overview
	characterSheet          *characterui.CharacterSheet
	characterShips          *skillui.FlyableShips
	characterSkillCatalogue *skillui.Catalogue
	characterSkillQueue     *skillui.Queue
	characterWallet         *walletui.CharacterWallet
	clones                  *clonesui.Clones
	colonies                *industryui.Colonies
	contracts               *characterui.Contracts
	corporationAssetBrowser *assetui.AssetBrowser
	corporationAssetSearch  *assetui.AssetSearch
	corporationContracts    *characterui.Contracts
	corporationIndyJobs     *industryui.Jobs
	corporationMember       *corporationui.Members
	corporationSheet        *corporationui.CorporationSheet
	corporationStructures   *corporationui.Structures
	corporationWallets      map[app.Division]*walletui.CorporationWallet
	gameSearch              *gamesearchui.GameSearch
	industryJobs            *industryui.Jobs
	loyaltyPoints           *walletui.LoyaltyPoints
	marketOrdersBuy         *industryui.MarketOrders
	marketOrdersSell        *industryui.MarketOrders
	slotsManufacturing      *industryui.Slots
	slotsReactions          *industryui.Slots
	slotsResearch           *industryui.Slots
	snackbar                *xwidget.Snackbar
	statusText              *statusText
	training                *skillui.Training
	wealth                  *walletui.Wealth
	iw                      *infowindow.InfoWindow

	// Services
	cs       *characterservice.CharacterService
	eis      *eveimageservice.EVEImageService
	ess      *esistatusservice.ESIStatusService
	eus      *eveuniverseservice.EVEUniverseService
	js       *janiceservice.JaniceService
	rs       *corporationservice.CorporationService
	scs      *statuscacheservice.StatusCacheService
	settings *settings.Settings

	// UI state & configuration
	app                fyne.App
	character          atomic.Pointer[app.Character]
	concurrencyLimit   int
	corporation        atomic.Pointer[app.Corporation]
	dataPaths          xmaps.OrderedMap[string, string] // Paths to user data
	defaultTheme       fyne.Theme
	isFakeMobile       bool        // Show mobile variant on a desktop (for development)
	isForeground       atomic.Bool // whether the app is currently shown in the foreground
	isMobile           bool        // whether Fyne has detected the app running on a mobile. Else we assume it's a desktop.
	isOfflineMode      bool
	isStartupCompleted atomic.Bool // whether the app has completed startup (for testing)
	isUpdateDisabled   atomic.Bool // Whether to disable update tickers (useful for debugging)
	sig                *singleinstance.Group
	signals            *app.Signals
	wasStarted         atomic.Bool            // whether the app has already been started at least once
	window             fyne.Window            // main window
	windows            map[string]fyne.Window // child windows
}

type BaseUIParams struct {
	App         fyne.App
	Character   *characterservice.CharacterService
	Corporation *corporationservice.CorporationService
	ESIStatus   *esistatusservice.ESIStatusService
	EVEImage    *eveimageservice.EVEImageService
	EVEUniverse *eveuniverseservice.EVEUniverseService
	Janice      *janiceservice.JaniceService
	StatusCache *statuscacheservice.StatusCacheService
	Signals     *app.Signals
	Settings    *settings.Settings
	// optional
	ClearCacheFunc   func()
	ConcurrencyLimit int
	DataPaths        map[string]string
	IsFakeMobile     bool
	IsUpdateDisabled bool
	IsOfflineMode    bool
}

// NewBaseUI constructs and returns a new BaseUI.
//
// Note:Types embedding BaseUI should define callbacks instead of overwriting methods.
func NewBaseUI(arg BaseUIParams) *baseUI {
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
	u := &baseUI{
		app:                arg.App,
		concurrencyLimit:   -1, // Default is no limit
		corporationWallets: make(map[app.Division]*walletui.CorporationWallet),
		cs:                 arg.Character,
		eis:                arg.EVEImage,
		ess:                arg.ESIStatus,
		eus:                arg.EVEUniverse,
		isFakeMobile:       arg.IsFakeMobile,
		isOfflineMode:      arg.IsOfflineMode,
		js:                 arg.Janice,
		rs:                 arg.Corporation,
		scs:                arg.StatusCache,
		settings:           arg.Settings,
		sig:                singleinstance.NewGroup(),
		signals:            arg.Signals,
		statusText:         NewStatusText(),
		windows:            make(map[string]fyne.Window),
	}
	u.window = u.app.NewWindow(app.Name())
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

	if !app.IsMobile() {
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
		slog.Debug("Signal: CurrentCharacterExchanged", "characterID", c.IDorZero())
		updateStatus(ctx)
	})
	u.signals.CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		slog.Debug("Signal: CharacterSectionChanged", "arg", arg)
		logErr := func(err error) {
			slog.Error("Failed to process CharacterSectionChanged", "arg", arg, "error", err)
		}
		isShown := arg.CharacterID == u.CurrentCharacterID()
		switch arg.Section {
		case app.SectionCharacterAssets:
			if isShown {
				u.ReloadCurrentCharacter()
			}
		case
			app.SectionCharacterJumpClones,
			app.SectionCharacterLocation,
			app.SectionCharacterOnline,
			app.SectionCharacterShip,
			app.SectionCharacterSkills,
			app.SectionCharacterWalletBalance:
			if isShown {
				u.ReloadCurrentCharacter()
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
		slog.Debug("Signal: CurrentCorporationExchanged", "corporationID", c.IDorZero())
		updateStatus(ctx)
		u.updateCorporationWalletTotal(ctx)
	})
	u.signals.CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		slog.Debug("Signal: CorporationSectionChanged", "arg", arg)
		if u.CurrentCorporationID() != arg.CorporationID {
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
			if arg.Changed.Contains(u.CurrentCharacterID()) {
				u.ReloadCurrentCharacter()
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
			u.ReloadCurrentCharacter()
		}
	})

	u.assetSearchAll = assetui.NewSearchForAll(u)
	u.augmentations = clonesui.NewAugmentations(u)
	u.characterAssetBrowser = assetui.NewCharacterAssetBrowser(u)
	u.characterAttributes = skillui.NewAttributes(u)
	u.characterAugmentations = clonesui.NewCharacterAugmentations(u)
	u.characterBiography = characterui.NewBiography(u)
	u.characterContacts = characterui.NewContacts(u)
	u.characterCommunications = characterui.NewCommunications(u)
	u.characterCorporation = corporationui.NewCorporationSheet(u, false)
	u.characterJumpClones = clonesui.NewCharacterClones(u)
	u.characterMails = characterui.NewMails(u)
	u.characterOverview = characterui.NewOverview(u)
	u.characterSheet = characterui.NewCharacterSheet(u)
	u.characterShips = skillui.NewFlyableShips(u)
	u.characterSkillCatalogue = skillui.NewCatalogue(u)
	u.characterSkillQueue = skillui.NewQueue(u)
	u.characterWallet = walletui.NewCharacterWallet(u)
	u.clones = clonesui.NewClones(u)
	u.colonies = industryui.NewColonies(u)
	u.contracts = characterui.NewContractsForCharacters(u)
	u.corporationAssetBrowser = assetui.NewCorporationAssetBrowser(u)
	u.corporationAssetSearch = assetui.NewSearchForCorporation(u)
	u.corporationContracts = characterui.NewContractsForCorporation(u)
	u.corporationIndyJobs = industryui.NewJobsForCorporation(u)

	u.corporationMember = corporationui.NewMembers(u)
	u.corporationStructures = corporationui.NewStructures(u)
	u.corporationSheet = corporationui.NewCorporationSheet(u, true)
	for _, d := range app.Divisions {
		u.corporationWallets[d] = walletui.NewCorporationWallet(u, d)
	}
	u.gameSearch = gamesearchui.NewGameSearch(u)
	u.industryJobs = industryui.NewJobsForOverview(u)
	u.loyaltyPoints = walletui.NewLoyaltyPoints(u)
	u.marketOrdersBuy = industryui.NewMarketOrders(u, true)
	u.marketOrdersSell = industryui.NewMarketOrders(u, false)
	u.slotsManufacturing = industryui.NewSlots(u, app.ManufacturingJob)
	u.slotsReactions = industryui.NewSlots(u, app.ReactionJob)
	u.slotsResearch = industryui.NewSlots(u, app.ScienceJob)
	u.snackbar = xwidget.NewSnackbar(u.window)
	u.training = skillui.NewTraining(u)
	u.wealth = walletui.NewWealth(u)

	u.iw = infowindow.New(u)

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
		if app.IsMobile() {
			// When the app is restarted on mobile the UI must be
			// refreshed immediately to avoid showing stale data (e.g. timers) to users
			// and updates must be run at once
			go u.signals.RefreshTickerExpired.Emit(context.Background(), struct{}{})
			if !u.isOfflineMode && !u.isUpdateDisabled.Load() {
				go func() {
					time.Sleep(1 * time.Second) // allow app to fully load before updating
					go u.cs.UpdateCharactersIfNeeded(context.Background(), false)
					go u.rs.UpdateCorporationsIfNeeded(context.Background(), false)
					go u.eus.UpdateSectionsIfNeeded(context.Background(), false)
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
	app.SetDeveloperMode(u.settings.DeveloperMode())
	u.defaultTheme = theme.Current()
	u.SetColorTheme(u.settings.ColorTheme())
	if u.isOfflineMode {
		slog.Info("App started in offline mode")
	} else {
		slog.Info("App started")
	}
	u.snackbar.Start()
	go func() {
		ctx := context.Background()
		var wg sync.WaitGroup
		wg.Go(func() {
			u.initHome(ctx)
		})
		wg.Go(func() {
			u.initCharacter(ctx)
		})
		wg.Go(func() {
			u.initCorporation(ctx)
		})
		wg.Go(func() {
			u.gameSearch.Init(ctx)
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
				u.signals.RefreshTickerExpired.Emit(context.Background(), struct{}{})
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

func (u *baseUI) OnShowCharacterFunc() func() {
	return u.onShowCharacter
}

func (u *baseUI) ClearAllCaches() {
	u.clearCache()
}

func (u *baseUI) MainWindow() fyne.Window {
	return u.window
}

// InfoWindow returns the info window.
func (u *baseUI) InfoWindow() *infowindow.InfoWindow {
	return u.iw
}

func (u *baseUI) SingleInstance() *singleinstance.Group {
	return u.sig
}

func (u *baseUI) ShowSnackbar(text string) {
	u.snackbar.Show(text)
}

func (u *baseUI) IsOfflineMode() bool {
	return u.isOfflineMode
}
func (u *baseUI) IsStartupCompleted() bool {
	return u.isStartupCompleted.Load()
}

func (u *baseUI) IsUpdateDisabled() bool {
	return u.isUpdateDisabled.Load()
}

func (u *baseUI) DataPaths() xmaps.OrderedMap[string, string] {
	return u.dataPaths
}

func (u *baseUI) Character() *characterservice.CharacterService {
	return u.cs
}

func (u *baseUI) Corporation() *corporationservice.CorporationService {
	return u.rs
}

func (u *baseUI) EVEImage() *eveimageservice.EVEImageService {
	return u.eis
}

func (u *baseUI) ESIStatus() *esistatusservice.ESIStatusService {
	return u.ess
}

func (u *baseUI) EVEUniverse() *eveuniverseservice.EVEUniverseService {
	return u.eus
}

func (u *baseUI) Janice() *janiceservice.JaniceService {
	return u.js
}

func (u *baseUI) Settings() *settings.Settings {
	return u.settings
}

func (u *baseUI) Signals() *app.Signals {
	return u.signals
}

func (u *baseUI) StatusCache() *statuscacheservice.StatusCacheService {
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
		u.ResetCharacter()
		return
	}
	u.SetCharacter(c)
}

// CurrentCharacterID returns the ID of the current character or 0 if non is set.
func (u *baseUI) CurrentCharacterID() int64 {
	c := u.CurrentCharacter()
	if c == nil {
		return 0
	}
	return c.ID
}

func (u *baseUI) CurrentCharacter() *app.Character {
	return u.character.Load()
}

func (u *baseUI) HasCharacter() bool {
	return u.CurrentCharacter() != nil
}

func (u *baseUI) LoadCharacter(id int64) error {
	c, err := u.cs.GetCharacter(context.Background(), id)
	if err != nil {
		return fmt.Errorf("load character ID %d: %w", id, err)
	}
	u.SetCharacter(c)
	return nil
}

// ReloadCurrentCharacter reloads the current character from storage.
func (u *baseUI) ReloadCurrentCharacter() {
	id := u.CurrentCharacterID()
	if id == 0 {
		return
	}
	c, err := u.cs.GetCharacter(context.Background(), id)
	if err != nil {
		slog.Error("reload character", "characterID", id, "error", err)
	}
	u.character.Store(c)
}

func (u *baseUI) ResetCharacter() {
	u.character.Store(nil)
	u.signals.CurrentCharacterExchanged.Emit(context.Background(), nil)
	u.settings.ResetLastCharacterID()
	// if u.onSetCharacter != nil {
	// 	u.onSetCharacter(nil)
	// }
}

func (u *baseUI) SetCharacter(c *app.Character) {
	u.character.Store(c)
	if u.onSetCharacter != nil {
		go u.onSetCharacter(c)
	}
	u.signals.CurrentCharacterExchanged.Emit(context.Background(), c)
	u.settings.SetLastCharacterID(c.ID)
}

func (u *baseUI) SetAnyCharacter() error {
	c, err := u.cs.GetAnyCharacter(context.Background())
	if errors.Is(err, app.ErrNotFound) {
		u.ResetCharacter()
		return nil
	} else if err != nil {
		return err
	}
	u.SetCharacter(c)
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
		u.ResetCorporation()
		return
	}
	u.SetCorporation(c)
}

// CurrentCorporationID returns the ID of the current corporation or 0 if non is set.
func (u *baseUI) CurrentCorporationID() int64 {
	c := u.CurrentCorporation()
	if c == nil {
		return 0
	}
	return c.ID
}

func (u *baseUI) CurrentCorporation() *app.Corporation {
	return u.corporation.Load()
}

func (u *baseUI) HasCorporation() bool {
	return u.CurrentCorporation() != nil
}

func (u *baseUI) LoadCorporation(id int64) error {
	c, err := u.rs.GetCorporation(context.Background(), id)
	if err != nil {
		return fmt.Errorf("load corporation ID %d: %w", id, err)
	}
	u.SetCorporation(c)
	return nil
}

func (u *baseUI) ResetCorporation() {
	u.corporation.Store(nil)
	u.signals.CurrentCorporationExchanged.Emit(context.Background(), nil)
	u.settings.ResetLastCorporationID()
	// if u.onSetCorporation != nil {
	// 	u.onSetCorporation(nil)
	// }
}

func (u *baseUI) SetCorporation(c *app.Corporation) {
	u.corporation.Store(c)
	if u.onSetCorporation != nil {
		go u.onSetCorporation(c)
	}
	u.signals.CurrentCorporationExchanged.Emit(context.Background(), c)
	u.settings.SetLastCorporationID(c.ID)
}

func (u *baseUI) SetAnyCorporation() error {
	c, err := u.rs.GetAnyCorporation(context.Background())
	if errors.Is(err, app.ErrNotFound) {
		u.ResetCorporation()
		return nil
	}
	if err != nil {
		return err
	}
	u.SetCorporation(c)
	return nil
}

//////////////////
// Home

// initHome performs an initial load of all pages under the home tab.
func (u *baseUI) initHome(ctx context.Context) {
	ff := map[string]func(context.Context){
		"characterOverview":  u.characterOverview.Update,
		"assetSearchAll":     u.assetSearchAll.Update,
		"augmentations":      u.augmentations.Update,
		"contracts":          u.contracts.Update,
		"clones":             u.clones.Update,
		"colonies":           u.colonies.Update,
		"industryJobs":       u.industryJobs.Update,
		"loyaltyPoints":      u.loyaltyPoints.Update,
		"marketOrdersSell":   u.marketOrdersSell.Update,
		"marketOrdersBuy":    u.marketOrdersBuy.Update,
		"slotsManufacturing": u.slotsManufacturing.Update,
		"slotsReactions":     u.slotsReactions.Update,
		"slotsResearch":      u.slotsResearch.Update,
		"training":           u.training.Update,
		"wealth":             u.wealth.Update,
	}
	myLog := slog.With("title", "startup")
	myLog.Debug("started")
	g := new(errgroup.Group)
	g.SetLimit(u.concurrencyLimit)
	for name, f := range ff {
		g.Go(func() error {
			start2 := time.Now()
			f(ctx)
			myLog.Debug("part completed", "name", name, "duration", time.Since(start2).Milliseconds())
			return nil
		})
	}
	g.Wait()
}

func (u *baseUI) SetColorTheme(s settings.ColorTheme) {
	u.app.Settings().SetTheme(xtheme.New(u.defaultTheme, s))
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

func (u *baseUI) ListCorporationsForSelection() ([]*app.EntityShort, error) {
	if u.settings.HideLimitedCorporations() {
		return u.rs.ListPrivilegedCorporations(context.Background())
	}
	return u.cs.ListCharacterCorporations(context.Background())
}

func (u *baseUI) updateCorporationWalletTotal(ctx context.Context) {
	v, ok := func() (float64, bool) {
		corporationID := u.CurrentCorporationID()
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

var (
	avatarCache                       xsync.Map[int64, fyne.Resource]
	characterAvatarPlaceholder64, _   = fynetools.MakeAvatar(icons.Characterplaceholder64Jpeg)
	corporationAvatarPlaceholder64, _ = fynetools.MakeAvatar(icons.Corporationplaceholder64Png)
)

func (u *baseUI) setCharacterAvatar(characterID int64, setIcon func(fyne.Resource)) {
	xwidget.LoadResourceAsyncWithCache(
		characterAvatarPlaceholder64,
		func() (fyne.Resource, bool) {
			return avatarCache.Load(characterID)
		},
		setIcon,
		func() (fyne.Resource, error) {
			r, err := u.eis.CharacterPortrait(characterID, app.IconPixelSize)
			if err != nil {
				return nil, err
			}
			return fynetools.MakeAvatar(r)
		},
		func(r fyne.Resource) {
			avatarCache.Store(characterID, r)
		},
	)
}

func (u *baseUI) setCorporationAvatar(corporationID int64, setIcon func(fyne.Resource)) {
	xwidget.LoadResourceAsyncWithCache(
		corporationAvatarPlaceholder64,
		func() (fyne.Resource, bool) {
			return avatarCache.Load(corporationID)
		},
		setIcon,
		func() (fyne.Resource, error) {
			r, err := u.eis.CorporationLogo(corporationID, app.IconPixelSize)
			if err != nil {
				return nil, err
			}
			return fynetools.MakeAvatar(r)
		},
		func(r fyne.Resource) {
			avatarCache.Store(corporationID, r)
		},
	)
}

func (u *baseUI) makeCharacterSwitchMenu(refresh func()) []*fyne.MenuItem {
	cc := u.scs.ListCharacters()
	var items []*fyne.MenuItem
	if len(cc) == 0 {
		it := fyne.NewMenuItem("No characters", nil)
		it.Disabled = true
		return append(items, it)
	}
	it := fyne.NewMenuItem("Switch to...", nil)
	it.Disabled = true
	items = append(items, it)
	g := new(errgroup.Group)
	g.SetLimit(u.concurrencyLimit)
	currentID := u.CurrentCharacterID()
	for _, c := range cc {
		it := fyne.NewMenuItem(c.Name, func() {
			go func() {
				err := u.LoadCharacter(c.ID)
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
			it.Icon = characterAvatarPlaceholder64
			g.Go(func() error {
				fyne.Do(func() {
					u.setCharacterAvatar(c.ID, func(r fyne.Resource) {
						it.Icon = r
					})
				})
				return nil
			})
		}
		items = append(items, it)
	}
	go func() {
		g.Wait()
		fyne.Do(func() {
			refresh()
		})
	}()
	return items
}

func (u *baseUI) makeCorporationSwitchMenu(refresh func()) []*fyne.MenuItem {
	var items []*fyne.MenuItem
	cc, err := u.ListCorporationsForSelection()
	if err != nil {
		slog.Error("Failed to fetch corporations", "error", err)
		return items
	}
	if len(cc) == 0 {
		it := fyne.NewMenuItem("No corporations", nil)
		it.Disabled = true
		return append(items, it)
	}
	corporations := set.Collect(xiter.MapSlice(cc, func(x *app.EntityShort) int64 {
		return x.ID
	}))
	currentID := u.CurrentCorporationID()
	if currentID != 0 && !corporations.Contains(currentID) {
		go u.SetAnyCorporation()
	}
	it := fyne.NewMenuItem("Switch to...", nil)
	it.Disabled = true
	items = append(items, it)
	g := new(errgroup.Group)
	g.SetLimit(u.concurrencyLimit)
	for _, c := range cc {
		it := fyne.NewMenuItem(c.Name, func() {
			go func() {
				err := u.LoadCorporation(c.ID)
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
			g.Go(func() error {
				fyne.Do(func() {
					u.setCorporationAvatar(c.ID, func(r fyne.Resource) {
						it.Icon = r
					})
				})
				return nil
			})
		}
		items = append(items, it)
	}
	go func() {
		g.Wait()
		fyne.Do(func() {
			refresh()
		})
	}()
	return items
}

// Windows

// GetOrCreateWindow returns a unique window as defined by the given id string
// and reports whether a new window was created or the window already exists.
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
	w, ok := u.windows[id]
	if ok {
		return w, false, nil
	}
	w = u.app.NewWindow(app.MakeWindowTitle(titles...))
	u.windows[id] = w
	if fyne.CurrentDevice().IsMobile() {
		w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
			if ev.Name != mobile.KeyBack {
				return
			}
			// Back gesture does nothing
		})
	}
	f := func() {
		delete(u.windows, id)
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

func NewStatusText() *statusText {
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
