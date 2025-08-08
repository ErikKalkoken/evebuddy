// Package ui implements the graphical user interface of the app.
package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxdialog "github.com/ErikKalkoken/fyne-kx/dialog"
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"
	kxtheme "github.com/ErikKalkoken/fyne-kx/theme"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/maniartech/signals"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatusservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	"github.com/ErikKalkoken/evebuddy/internal/github"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

// update info
const (
	githubOwner        = "ErikKalkoken"
	githubRepo         = "evebuddy"
	fallbackWebsiteURL = "https://github.com/ErikKalkoken/evebuddy"
)

// daily downtime
const (
	downtimeStart    = "11:00"
	downtimeDuration = 15 * time.Minute
)

// ticker
const (
	characterSectionsUpdateTicker = 60 * time.Second
	generalSectionsUpdateTicker   = 300 * time.Second
)

// Default ScaleMode for images
var defaultImageScaleMode canvas.ImageScale

// services represents a wrapper for passing the main services to functions.
type services struct {
	cs  *characterservice.CharacterService
	eis *eveimageservice.EveImageService
	eus *eveuniverseservice.EveUniverseService
	rs  *corporationservice.CorporationService
	scs *statuscacheservice.StatusCacheService
}

// baseUI represents the core UI logic and is used by both the desktop and mobile UI.
type baseUI struct {
	// Callbacks
	clearCache                      func() // clear all caches
	disableMenuShortcuts            func()
	enableMenuShortcuts             func()
	hideMailIndicator               func()
	onAppFirstStarted               func()
	onAppStopped                    func()
	onAppTerminated                 func()
	onRefreshCross                  func()
	onSetCharacter                  func(int32)
	onSetCorporation                func(int32)
	onShowAndRun                    func()
	onUpdateCharacter               func(*app.Character)
	onUpdateCorporation             func(*app.Corporation)
	onUpdateCorporationWalletTotals func(balance string)
	onUpdateStatus                  func()
	showMailIndicator               func()
	showManageCharacters            func()

	// UI elements
	assets                  *assets
	characterAsset          *characterAssets
	characterAttributes     *characterAttributes
	characterAugmentations  *characterAugmentations
	characterBiography      *characterBiography
	characterCommunications *characterCommunications
	characterCorporation    *corporationSheet
	characterJumpClones     *characterJumpClones
	characterLocations      *characterLocations
	characterMail           *characterMails
	characterOverview       *characterOverview
	characterSheet          *characterSheet
	characterShips          *characterFlyableShips
	characterSkillCatalogue *characterSkillCatalogue
	characterSkillQueue     *characterSkillQueue
	characterWallet         *characterWallet
	clones                  *clones
	colonies                *colonies
	contracts               *contracts
	corporationMember       *corporationMember
	corporationSheet        *corporationSheet
	corporationWallets      map[app.Division]*corporationWallet
	gameSearch              *gameSearch
	industryJobs            *industryJobs
	slotsManufacturing      *industrySlots
	slotsReactions          *industrySlots
	slotsResearch           *industrySlots
	snackbar                *iwidget.Snackbar
	training                *training
	wealth                  *wealth

	// Signals
	characterSectionUpdated signals.Signal[app.CharacterUpdateSectionParams]
	characterChanged        signals.Signal[*app.Character]

	// Services
	cs       *characterservice.CharacterService
	eis      *eveimageservice.EveImageService
	ess      *esistatusservice.ESIStatusService
	eus      *eveuniverseservice.EveUniverseService
	js       *janiceservice.JaniceService
	memcache *memcache.Cache
	rs       *corporationservice.CorporationService
	scs      *statuscacheservice.StatusCacheService
	settings *settings.Settings

	// UI state
	app                fyne.App
	character          atomic.Pointer[app.Character]
	corporation        atomic.Pointer[app.Corporation]
	dataPaths          map[string]string      // Paths to user data
	isDesktop          bool                   // whether the app runs on a desktop. If false we assume it's on mobile.
	isForeground       atomic.Bool            // whether the app is currently shown in the foreground
	isOffline          bool                   // Run the app in offline mode
	isStartupCompleted atomic.Bool            // whether the app has completed startup (for testing)
	isUpdateDisabled   bool                   // Whether to disable update tickers (useful for debugging)
	wasStarted         atomic.Bool            // whether the app has already been started at least once
	window             fyne.Window            // main window
	windows            map[string]fyne.Window // child windows
}

type BaseUIParams struct {
	App                fyne.App
	CharacterService   *characterservice.CharacterService
	CorporationService *corporationservice.CorporationService
	ESIStatusService   *esistatusservice.ESIStatusService
	EveImageService    *eveimageservice.EveImageService
	EveUniverseService *eveuniverseservice.EveUniverseService
	JaniceService      *janiceservice.JaniceService
	MemCache           *memcache.Cache
	StatusCacheService *statuscacheservice.StatusCacheService
	// optional
	ClearCacheFunc   func()
	DataPaths        map[string]string
	IsDesktop        bool
	IsOffline        bool
	IsUpdateDisabled bool
}

// NewBaseUI constructs and returns a new BaseUI.
//
// Note:Types embedding BaseUI should define callbacks instead of overwriting methods.
func NewBaseUI(args BaseUIParams) *baseUI {
	u := &baseUI{
		app:                     args.App,
		characterChanged:        signals.New[*app.Character](),
		characterSectionUpdated: signals.New[app.CharacterUpdateSectionParams](),
		corporationWallets:      make(map[app.Division]*corporationWallet),
		cs:                      args.CharacterService,
		eis:                     args.EveImageService,
		ess:                     args.ESIStatusService,
		eus:                     args.EveUniverseService,
		isDesktop:               args.IsDesktop,
		isOffline:               args.IsOffline,
		isUpdateDisabled:        args.IsUpdateDisabled,
		js:                      args.JaniceService,
		memcache:                args.MemCache,
		rs:                      args.CorporationService,
		scs:                     args.StatusCacheService,
		settings:                settings.New(args.App.Preferences()),
		windows:                 make(map[string]fyne.Window),
	}
	u.window = u.app.NewWindow(u.appName())

	if args.ClearCacheFunc != nil {
		u.clearCache = args.ClearCacheFunc
	} else {
		u.clearCache = func() {}
	}

	if len(args.DataPaths) > 0 {
		u.dataPaths = args.DataPaths
	} else {
		u.dataPaths = make(map[string]string)
	}

	if u.isDesktop {
		iwidget.DefaultImageScaleMode = canvas.ImageScaleFastest
		defaultImageScaleMode = canvas.ImageScaleFastest
	}

	u.characterAsset = newCharacterAssets(u)
	u.characterAttributes = newCharacterAttributes(u)
	u.characterBiography = newCharacterBiography(u)
	u.characterCommunications = newCharacterCommunications(u)
	u.characterAugmentations = newCharacterAugmentations(u)
	u.characterJumpClones = newCharacterJumpClones(u)
	u.characterMail = newCharacterMails(u)
	u.characterSheet = newCharacterSheet(u)
	u.characterCorporation = newCorporationSheet(u, false)
	u.characterShips = newCharacterFlyableShips(u)
	u.characterSkillCatalogue = newCharacterSkillCatalogue(u)
	u.characterSkillQueue = newCharacterSkillQueue(u)
	u.characterWallet = newCharacterWallet(u)
	u.corporationSheet = newCorporationSheet(u, true)
	u.corporationMember = newCorporationMember(u)
	for _, d := range app.Divisions {
		u.corporationWallets[d] = newCorporationWallet(u, d)
	}
	u.contracts = newContracts(u)
	u.gameSearch = newGameSearch(u)
	u.industryJobs = newIndustryJobs(u)
	u.slotsManufacturing = newIndustrySlots(u, app.ManufacturingJob)
	u.slotsReactions = newIndustrySlots(u, app.ReactionJob)
	u.slotsResearch = newIndustrySlots(u, app.ScienceJob)
	u.assets = newAssets(u)
	u.characterOverview = newCharacterOverview(u)
	u.clones = newClones(u)
	u.colonies = newColonies(u)
	u.characterLocations = newCharacterLocations(u)
	u.training = newTraining(u)
	u.wealth = newWealth(u)
	u.snackbar = iwidget.NewSnackbar(u.window)
	u.MainWindow().SetMaster()

	// SetOnStarted is called on initial start,
	// but also when an app is continued after it was temporarily stopped,
	// which can happen on mobile
	u.app.Lifecycle().SetOnStarted(func() {
		wasStarted := !u.wasStarted.CompareAndSwap(false, true)
		if wasStarted {
			slog.Info("App continued")
			return
		}
		// First app start
		if u.isOffline {
			slog.Info("App started in offline mode")
		} else {
			slog.Info("App started")
		}
		u.setColorTheme(u.settings.ColorTheme())
		u.isForeground.Store(true)
		u.snackbar.Start()
		go func() {
			u.characterSkillQueue.start()
			u.initCharacter()
			u.initCorporation()
			u.updateHome()
			u.updateStatus()
			u.isStartupCompleted.Store(true)
			u.training.startUpdateTicker()
			u.characterJumpClones.startUpdateTicker()
			if !u.isOffline && !u.isUpdateDisabled {
				time.Sleep(5 * time.Second) // Workaround to prevent concurrent updates from happening at startup.
				u.startUpdateTickerGeneralSections()
				u.startUpdateTickerCharacters()
				u.startUpdateTickerCorporations()
			} else {
				slog.Info("Update ticker disabled")
			}
		}()
		if u.onAppFirstStarted != nil {
			u.onAppFirstStarted()
		}
	})
	u.app.Lifecycle().SetOnEnteredForeground(func() {
		slog.Debug("Entered foreground")
		u.isForeground.Store(true)
		if !u.isOffline && !u.isUpdateDisabled {
			u.updateCharactersIfNeeded(context.Background(), false)
			u.updateCorporationsIfNeeded(context.Background(), false)
			u.updateGeneralSectionsIfNeeded(context.Background(), false)
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

func (u *baseUI) services() services {
	return services{
		cs:  u.cs,
		eis: u.eis,
		eus: u.eus,
		rs:  u.rs,
		scs: u.scs,
	}
}

func (u *baseUI) App() fyne.App {
	return u.app
}

func (u *baseUI) ClearAllCaches() {
	u.clearCache()
}

func (u *baseUI) MainWindow() fyne.Window {
	return u.window
}

func (u *baseUI) IsDeveloperMode() bool {
	return u.settings.DeveloperMode()
}

func (u *baseUI) IsOffline() bool {
	return u.isOffline
}

func (u *baseUI) IsStartupCompleted() bool {
	return u.isStartupCompleted.Load()
}

// humanizeError returns user friendly representation of an error for display in the UI.
func (u *baseUI) humanizeError(err error) string {
	if err == nil {
		return "No error"
	}
	if u.settings.DeveloperMode() {
		return err.Error()
	}
	return err.Error()
	// return ihumanize.Error(err) TODO: Re-enable again when app is stable enough
}

//////////////////
// Current character

func (u *baseUI) initCharacter() {
	var c *app.Character
	var err error
	ctx := context.Background()
	if cID := u.settings.LastCharacterID(); cID != 0 {
		c, err = u.cs.GetCharacter(ctx, int32(cID))
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
		u.resetCharacter()
		return
	}
	u.setCharacter(c)
}

// currentCharacterID returns the ID of the current character or 0 if non is set.
func (u *baseUI) currentCharacterID() int32 {
	c := u.currentCharacter()
	if c == nil {
		return 0
	}
	return c.ID
}

func (u *baseUI) currentCharacter() *app.Character {
	return u.character.Load()
}

func (u *baseUI) hasCharacter() bool {
	return u.currentCharacter() != nil
}

func (u *baseUI) loadCharacter(id int32) error {
	c, err := u.cs.GetCharacter(context.Background(), id)
	if err != nil {
		return fmt.Errorf("load character ID %d: %w", id, err)
	}
	u.setCharacter(c)
	return nil
}

// reloadCurrentCharacter reloads the current character from storage.
func (u *baseUI) reloadCurrentCharacter() {
	id := u.currentCharacterID()
	if id == 0 {
		return
	}
	c, err := u.cs.GetCharacter(context.Background(), id)
	if err != nil {
		slog.Error("reload character", "characterID", id, "error", err)
	}
	u.character.Store(c)
}

func (u *baseUI) resetCharacter() {
	u.character.Store(nil)
	u.characterChanged.Emit(context.Background(), nil)
	u.settings.ResetLastCharacterID()
	u.updateCharacter()
	u.updateStatus()
}

func (u *baseUI) setCharacter(c *app.Character) {
	u.character.Store(c)
	u.characterChanged.Emit(context.Background(), c)
	u.settings.SetLastCharacterID(c.ID)
	u.updateCharacter()
	u.updateStatus()
	if u.onSetCharacter != nil {
		u.onSetCharacter(c.ID)
	}
}

func (u *baseUI) setAnyCharacter() error {
	c, err := u.cs.GetAnyCharacter(context.Background())
	if errors.Is(err, app.ErrNotFound) {
		u.resetCharacter()
		return nil
	} else if err != nil {
		return err
	}
	u.setCharacter(c)
	return nil
}

// updateCharacter updates all pages for the current character.
func (u *baseUI) updateCharacter() {
	c := u.currentCharacter()
	if c != nil {
		slog.Debug("Updating character", "ID", c.EveCharacter.ID, "name", c.EveCharacter.Name)
	} else {
		slog.Debug("Updating without character")
	}
	runFunctionsWithProgressModal("Loading character", u.defineCharacterUpdates(), func() {
		if u.onUpdateCharacter != nil {
			u.onUpdateCharacter(c)
		}
		if c != nil && !u.isUpdateDisabled {
			u.updateCharacterAndRefreshIfNeeded(context.Background(), c.ID, false)
		}
	},
		u.window,
	)
}

func (u *baseUI) defineCharacterUpdates() map[string]func() {
	ff := map[string]func(){
		"assets":               u.characterAsset.update,
		"attributes":           u.characterAttributes.update,
		"biography":            u.characterBiography.update,
		"implants":             u.characterAugmentations.update,
		"jumpClones":           u.characterJumpClones.update,
		"mail":                 u.characterMail.update,
		"notifications":        u.characterCommunications.update,
		"characterSheet":       u.characterSheet.update,
		"characterCorporation": u.characterCorporation.update,
		"ships":                u.characterShips.update,
		"skillCatalogue":       u.characterSkillCatalogue.update,
		// "skillqueue":           u.characterSkillQueue.update,
		"wallet": u.characterWallet.update,
	}
	return ff
}

func (u *baseUI) updateCharacterAvatar(id int32, setIcon func(fyne.Resource)) {
	r, err := u.eis.CharacterPortrait(id, app.IconPixelSize)
	if err != nil {
		slog.Error("Failed to fetch character portrait", "characterID", id, "err", err)
		r = icons.Characterplaceholder64Jpeg
	}
	r2, err := fynetools.MakeAvatar(r)
	if err != nil {
		slog.Error("Failed to make avatar", "characterID", id, "err", err)
		r2 = icons.Characterplaceholder64Jpeg
	}
	setIcon(r2)
}

//////////////////
// Current corporation

func (u *baseUI) initCorporation() {
	var c *app.Corporation
	var err error
	ctx := context.Background()
	if cID := u.settings.LastCorporationID(); cID != 0 {
		c, err = u.rs.GetCorporation(ctx, int32(cID))
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
		u.resetCorporation()
		return
	}
	u.setCorporation(c)
}

// currentCorporationID returns the ID of the current corporation or 0 if non is set.
func (u *baseUI) currentCorporationID() int32 {
	c := u.currentCorporation()
	if c == nil {
		return 0
	}
	return c.ID
}

func (u *baseUI) currentCorporation() *app.Corporation {
	return u.corporation.Load()
}

func (u *baseUI) hasCorporation() bool {
	return u.currentCorporation() != nil
}

func (u *baseUI) loadCorporation(id int32) error {
	c, err := u.rs.GetCorporation(context.Background(), id)
	if err != nil {
		return fmt.Errorf("load corporation ID %d: %w", id, err)
	}
	u.setCorporation(c)
	return nil
}

func (u *baseUI) resetCorporation() {
	u.corporation.Store(nil)
	u.settings.ResetLastCorporationID()
	u.updateCorporation()
	u.updateStatus()
}

func (u *baseUI) setCorporation(c *app.Corporation) {
	u.corporation.Store(c)
	u.settings.SetLastCorporationID(c.ID)
	u.updateCorporation()
	u.updateStatus()
	if u.onSetCorporation != nil {
		u.onSetCorporation(c.ID)
	}
}

func (u *baseUI) setAnyCorporation() error {
	c, err := u.rs.GetAnyCorporation(context.Background())
	if errors.Is(err, app.ErrNotFound) {
		u.resetCorporation()
		return nil
	} else if err != nil {
		return err
	}
	u.setCorporation(c)
	return nil
}

// updateCorporation updates all pages for the current corporation.
func (u *baseUI) updateCorporation() {
	c := u.currentCorporation()
	if c != nil {
		slog.Debug("Updating corporation", "ID", c.EveCorporation.ID, "name", c.EveCorporation.Name)
	} else {
		slog.Debug("Updating without corporation")
	}
	runFunctionsWithProgressModal("Loading corporation", u.defineCorporationUpdates(), func() {
		if c != nil && !u.isUpdateDisabled {
			u.updateCorporationAndRefreshIfNeeded(context.Background(), c.ID, false)
		}
		if u.onUpdateCorporation != nil {
			u.onUpdateCorporation(c)
		}
	},
		u.window,
	)
}

func (u *baseUI) defineCorporationUpdates() map[string]func() {
	ff := make(map[string]func())
	ff["corporationSheet"] = u.corporationSheet.update
	ff["corporationMember"] = u.corporationMember.update
	ff["corporationWalletTotal"] = u.updateCorporationWalletTotal
	for id, w := range u.corporationWallets {
		ff[fmt.Sprintf("walletJournal%d", id)] = w.update
	}
	return ff
}

func (u *baseUI) updateCorporationAvatar(id int32, setIcon func(fyne.Resource)) {
	r, err := u.eis.CorporationLogo(id, app.IconPixelSize)
	if err != nil {
		slog.Error("Failed to fetch corporation logo", "corporationID", id, "err", err)
		r = icons.Corporationplaceholder64Png
	}
	r2, err := fynetools.MakeAvatar(r)
	if err != nil {
		slog.Error("Failed to make avatar", "corporationID", id, "err", err)
		r2 = icons.Corporationplaceholder64Png
	}
	setIcon(r2)
}

//////////////////
// Home

// updateHome refreshed all pages that contain information about multiple characters.
func (u *baseUI) updateHome() {
	runFunctionsWithProgressModal("Updating home", u.defineHomeUpdates(), u.onRefreshCross, u.window)
}

func (u *baseUI) defineHomeUpdates() map[string]func() {
	ff := map[string]func(){
		"assetSearch":        u.assets.update,
		"contracts":          u.contracts.update,
		"cloneSearch":        u.clones.update,
		"colony":             u.colonies.update,
		"industryJobs":       u.industryJobs.update,
		"slotsManufacturing": u.slotsManufacturing.update,
		"slotsReactions":     u.slotsReactions.update,
		"slotsResearch":      u.slotsResearch.update,
		"locations":          u.characterLocations.update,
		"overview":           u.characterOverview.update,
		"training":           u.training.update,
		"wealth":             u.wealth.update,
	}
	return ff
}

// UpdateAll updates all UI elements. This method is usually only called from tests.
func (u *baseUI) UpdateAll() {
	updates := slices.Collect(xiter.Chain(maps.Values(u.defineCharacterUpdates()), maps.Values(u.defineHomeUpdates())))
	for _, f := range updates {
		f()
	}
}

func runFunctionsWithProgressModal(title string, ff map[string]func(), onSuccess func(), w fyne.Window) {
	fyne.Do(func() {
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
					fyne.Do(func() {
						if err := p.Set(float64(x)); err != nil {
							myLog.Warn("failed set progress", "error", err)
						}
					})
					myLog.Debug("part completed", "name", name, "duration", time.Since(start2).Milliseconds())
				}()
			}
			wg.Wait()
			myLog.Debug("completed", "duration", time.Since(start).Milliseconds())
			return nil
		}, float64(len(ff)), w)
		m.OnSuccess = onSuccess
		m.Start()
	})
}

// updateStatus refreshed all status information pages.
func (u *baseUI) updateStatus() {
	if u.onUpdateStatus == nil {
		return
	}
	u.onUpdateStatus()
}

func (u *baseUI) setColorTheme(s settings.ColorTheme) {
	switch s {
	case settings.Light:
		u.app.Settings().SetTheme(kxtheme.DefaultWithFixedVariant(theme.VariantLight))
	case settings.Dark:
		u.app.Settings().SetTheme(kxtheme.DefaultWithFixedVariant(theme.VariantDark))
	default:
		u.app.Settings().SetTheme(theme.DefaultTheme())
	}
}

func (u *baseUI) updateMailIndicator() {
	if u.showMailIndicator == nil || u.hideMailIndicator == nil {
		return
	}
	if !u.settings.SysTrayEnabled() {
		return
	}
	n, err := u.cs.GetAllMailUnreadCount(context.Background())
	if err != nil {
		slog.Error("update mail indicator", "error", err)
		return
	}
	if n > 0 {
		u.showMailIndicator()
	} else {
		u.hideMailIndicator()
	}
}

func (u *baseUI) makeCharacterSwitchMenu(refresh func()) []*fyne.MenuItem {
	cc := u.scs.ListCharacters()
	items := make([]*fyne.MenuItem, 0)
	if len(cc) == 0 {
		it := fyne.NewMenuItem("No characters", nil)
		it.Disabled = true
		return append(items, it)
	}
	it := fyne.NewMenuItem("Switch to...", nil)
	it.Disabled = true
	items = append(items, it)
	var wg sync.WaitGroup
	currentID := u.currentCharacterID()
	fallbackIcon, _ := fynetools.MakeAvatar(icons.Characterplaceholder64Jpeg)
	for _, c := range cc {
		it := fyne.NewMenuItem(c.Name, func() {
			err := u.loadCharacter(c.ID)
			if err != nil {
				slog.Error("make character switch menu", "error", err)
				u.snackbar.Show("ERROR: Failed to switch character")
			}
		})
		if c.ID == currentID {
			it.Icon = theme.AccountIcon()
			it.Disabled = true
		} else {
			it.Icon = fallbackIcon
			wg.Add(1)
			go u.updateCharacterAvatar(c.ID, func(r fyne.Resource) {
				defer wg.Done()
				fyne.Do(func() {
					it.Icon = r
				})
			})
		}
		items = append(items, it)
	}
	go func() {
		wg.Wait()
		fyne.Do(func() {
			refresh()
		})
	}()
	return items
}

func (u *baseUI) makeCorporationSwitchMenu(refresh func()) []*fyne.MenuItem {
	cc := u.scs.ListCorporations()
	items := make([]*fyne.MenuItem, 0)
	if len(cc) == 0 {
		it := fyne.NewMenuItem("No corporations", nil)
		it.Disabled = true
		return append(items, it)
	}
	it := fyne.NewMenuItem("Switch to...", nil)
	it.Disabled = true
	items = append(items, it)
	var wg sync.WaitGroup
	currentID := u.currentCorporationID()
	fallbackIcon, _ := fynetools.MakeAvatar(icons.Corporationplaceholder64Png)
	for _, c := range cc {
		it := fyne.NewMenuItem(c.Name, func() {
			err := u.loadCorporation(c.ID)
			if err != nil {
				slog.Error("make corporation switch menu", "error", err)
				u.snackbar.Show("ERROR: Failed to switch corporation")
			}
		})
		if c.ID == currentID {
			it.Icon = theme.AccountIcon()
			it.Disabled = true
		} else {
			it.Icon = fallbackIcon
			wg.Add(1)
			go u.updateCorporationAvatar(c.ID, func(r fyne.Resource) {
				defer wg.Done()
				fyne.Do(func() {
					it.Icon = r
				})
			})
		}
		items = append(items, it)
	}
	go func() {
		wg.Wait()
		fyne.Do(func() {
			refresh()
		})
	}()
	return items
}

func (u *baseUI) sendDesktopNotification(title, content string) {
	fyne.Do(func() {
		u.app.SendNotification(fyne.NewNotification(title, content))
	})
	slog.Info("desktop notification sent", "title", title, "content", content)
}

// update general sections

func (u *baseUI) startUpdateTickerGeneralSections() {
	ticker := time.NewTicker(generalSectionsUpdateTicker)
	go func() {
		for {
			ctx := context.Background()
			u.updateGeneralSectionsIfNeeded(ctx, false)
			<-ticker.C
		}
	}()
}

func (u *baseUI) updateGeneralSectionsIfNeeded(ctx context.Context, forceUpdate bool) {
	if !forceUpdate && !u.isDesktop && !u.isForeground.Load() {
		slog.Debug("Skipping general sections update while in background")
		return
	}
	if !forceUpdate && isNowDailyDowntime() {
		slog.Info("Skipping regular update of general sections during daily downtime")
		return
	}
	for _, s := range app.GeneralSections {
		go func() {
			u.updateGeneralSectionAndRefreshIfNeeded(ctx, s, forceUpdate)
		}()
	}
}

func (u *baseUI) updateGeneralSectionAndRefreshIfNeeded(ctx context.Context, section app.GeneralSection, forceUpdate bool) {
	hasChanged, err := u.eus.UpdateSection(ctx, section, forceUpdate)
	if err != nil {
		slog.Error("Failed to update general section", "section", section, "err", err)
		return
	}
	if !hasChanged && !forceUpdate {
		return
	}
	switch section {
	case app.SectionEveEntities:
		u.updateHome()
		u.updateCharacter()
	case app.SectionEveTypes:
		u.characterShips.update()
		u.characterSkillCatalogue.update()
	case app.SectionEveCharacters:
		u.reloadCurrentCharacter()
		u.characterOverview.update()
	case app.SectionEveCorporations:
		// TODO: Only update when shown entity changed
		u.characterCorporation.update()
		u.corporationSheet.update()
	case app.SectionEveMarketPrices:
		u.characterAsset.update()
		u.characterOverview.update()
		u.assets.update()
		u.reloadCurrentCharacter()
	default:
		slog.Warn(fmt.Sprintf("section not part of the update ticker refresh: %s", section))
	}
}

// update character sections

func (u *baseUI) startUpdateTickerCharacters() {
	ticker := time.NewTicker(characterSectionsUpdateTicker)
	go func() {
		for {
			ctx := context.Background()
			if err := u.updateCharactersIfNeeded(ctx, false); err != nil {
				slog.Error("Failed to update characters", "error", err)
			}
			if err := u.notifyCharactersIfNeeded(ctx); err != nil {
				slog.Error("Failed to notify characters", "error", err)
			}
			<-ticker.C
		}
	}()
}

func (u *baseUI) updateCharactersIfNeeded(ctx context.Context, forceUpdate bool) error {
	if !forceUpdate && isNowDailyDowntime() {
		slog.Info("Skipping regular update of characters during daily downtime")
		return nil
	}
	cc, err := u.cs.ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	for _, c := range cc {
		go u.updateCharacterAndRefreshIfNeeded(ctx, c.ID, forceUpdate)
	}
	slog.Debug("started update status characters")
	return nil
}

func (u *baseUI) notifyCharactersIfNeeded(ctx context.Context) error {
	cc, err := u.cs.ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	for _, c := range cc {
		go u.notifyExpiredExtractionsIfNeeded(ctx, c.ID)
		go u.notifyExpiredTrainingIfNeeded(ctx, c.ID)
	}
	slog.Debug("started notify characters")
	return nil
}

// updateCharacterAndRefreshIfNeeded runs update for all sections of a character if needed
// and refreshes the UI accordingly.
func (u *baseUI) updateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
	if u.isOffline {
		return
	}
	var sections []app.CharacterSection
	if !u.isDesktop && !u.isForeground.Load() {
		// only update what is needed for notifications on mobile when running in background to save battery
		if u.settings.NotifyCommunicationsEnabled() {
			sections = append(sections, app.SectionCharacterNotifications)
		}
		if u.settings.NotifyContractsEnabled() {
			sections = append(sections, app.SectionCharacterContracts)
		}
		if u.settings.NotifyMailsEnabled() {
			sections = append(sections, app.SectionCharacterMailLabels)
			sections = append(sections, app.SectionCharacterMailLists)
			sections = append(sections, app.SectionCharacterMails)
		}
		if u.settings.NotifyPIEnabled() {
			sections = append(sections, app.SectionCharacterPlanets)
		}
		if u.settings.NotifyTrainingEnabled() {
			sections = append(sections, app.SectionCharacterSkillqueue)
			sections = append(sections, app.SectionCharacterSkills)
		}
	} else {
		sections = app.CharacterSections
	}
	if len(sections) == 0 {
		return
	}
	slog.Debug("Starting to check character sections for update", "sections", sections)
	_, err := u.cs.GetValidCharacterToken(ctx, characterID)
	if err != nil {
		slog.Error("Failed to refresh token for update", "characterID", characterID, "error", err)
	}
	for _, s := range sections {
		go u.updateCharacterSectionAndRefreshIfNeeded(ctx, characterID, s, forceUpdate)
	}
}

// updateCharacterSectionAndRefreshIfNeeded runs update for a character section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on character sections needs to be included
// to make sure they are refreshed when data changes.
func (u *baseUI) updateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, s app.CharacterSection, forceUpdate bool) {
	updateArg := app.CharacterUpdateSectionParams{
		CharacterID:           characterID,
		Section:               s,
		ForceUpdate:           forceUpdate,
		MaxMails:              u.settings.MaxMails(),
		MaxWalletTransactions: u.settings.MaxWalletTransactions(),
	}
	hasChanged, err := u.cs.UpdateSectionIfNeeded(ctx, updateArg)
	if err != nil {
		slog.Error("Failed to update character section", "characterID", characterID, "section", s, "err", err)
		return
	}
	isShown := characterID == u.currentCharacterID()

	var corporationID int32
	character := u.currentCharacter()
	if character != nil {
		corporationID = character.EveCharacter.Corporation.ID
		ok, err := u.rs.HasCorporation(ctx, corporationID)
		if err != nil {
			slog.Error("Failed to check corporation exists", "error", err)
		}
		if !ok {
			corporationID = 0
		}
	}

	needsRefresh := hasChanged || forceUpdate
	if needsRefresh {
		u.characterSectionUpdated.Emit(ctx, updateArg)
	}

	switch s {
	case app.SectionCharacterAssets:
		if needsRefresh {
			u.assets.update()
			u.wealth.update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterAsset.update()
				u.characterSheet.update()
			}
		}
	case app.SectionCharacterAttributes:
		if isShown && needsRefresh {
			u.characterAttributes.update()
		}
	case app.SectionCharacterContracts:
		if needsRefresh {
			u.contracts.update()
		}
		if u.settings.NotifyContractsEnabled() {
			go func() {
				earliest := u.settings.NotifyContractsEarliest()
				if err := u.cs.NotifyUpdatedContracts(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
					slog.Error("notify contract update", "error", err)
				}
			}()
		}
	case app.SectionCharacterImplants:
		if isShown && needsRefresh {
			u.characterAugmentations.update()
		}
	case app.SectionCharacterJumpClones:
		if needsRefresh {
			u.characterOverview.update()
			u.clones.update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterJumpClones.update()
			}
		}
	case app.SectionCharacterIndustryJobs:
		if needsRefresh {
			u.industryJobs.update()
			u.slotsManufacturing.update()
			u.slotsReactions.update()
			u.slotsResearch.update()
		}
	case app.SectionCharacterLocation, app.SectionCharacterOnline, app.SectionCharacterShip:
		if needsRefresh {
			u.characterLocations.update()
			if isShown {
				u.reloadCurrentCharacter()
			}
		}
	case app.SectionCharacterMailLabels, app.SectionCharacterMailLists:
		if needsRefresh {
			u.characterOverview.update()
			if isShown {
				u.characterMail.update()
			}
		}
	case app.SectionCharacterMails:
		if needsRefresh {
			go u.characterOverview.update()
			go u.updateMailIndicator()
			if isShown {
				u.characterMail.update()
			}
		}
		if u.settings.NotifyMailsEnabled() {
			go func() {
				earliest := u.settings.NotifyMailsEarliest()
				if err := u.cs.NotifyMails(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
					slog.Error("notify mails", "characterID", characterID, "error", err)
				}
			}()
		}
	case app.SectionCharacterNotifications:
		if isShown && needsRefresh {
			u.characterCommunications.update()
		}
		if u.settings.NotifyCommunicationsEnabled() {
			go func() {
				earliest := u.settings.NotifyCommunicationsEarliest()
				typesEnabled := u.settings.NotificationTypesEnabled()
				err := u.cs.NotifyCommunications(
					ctx,
					characterID,
					earliest,
					typesEnabled,
					u.sendDesktopNotification,
				)
				if err != nil {
					slog.Error("notify communications", "characterID", characterID, "error", err)
				}
			}()
		}
	case app.SectionCharacterPlanets:
		if needsRefresh {
			u.colonies.update()
			u.notifyExpiredExtractionsIfNeeded(ctx, characterID)
		}
	case app.SectionCharacterRoles:
		if needsRefresh {
			if isShown {
				u.characterSheet.update()
			}
			if corporationID == 0 {
				return
			}
			if err := u.rs.RemoveSectionDataWhenPermissionLost(ctx, corporationID); err != nil {
				slog.Error("Failed to remove corp data after character role change", "characterID", characterID, "error", err)
			}
			u.updateCorporationAndRefreshIfNeeded(ctx, corporationID, true)
		}
	case app.SectionCharacterSkills:
		if needsRefresh {
			u.training.update()
			u.slotsManufacturing.update()
			u.slotsReactions.update()
			u.slotsResearch.update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterSkillCatalogue.update()
				u.characterShips.update()
			}
		}

	case app.SectionCharacterSkillqueue:
		if needsRefresh {
			u.training.update()
			u.notifyExpiredTrainingIfNeeded(ctx, characterID)
			// if isShown {
			// 	u.characterSkillQueue.update()
			// }
		}
		if u.settings.NotifyTrainingEnabled() {
			err := u.cs.EnableTrainingWatcher(ctx, characterID)
			if err != nil {
				slog.Error("Failed to enable training watcher", "characterID", characterID, "error", err)
			}
		}
	case app.SectionCharacterWalletBalance:
		if needsRefresh {
			u.characterOverview.update()
			u.wealth.update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterWallet.update()
			}
		}
	case app.SectionCharacterWalletJournal:
		if isShown && needsRefresh {
			u.characterWallet.journal.update()
		}
	case app.SectionCharacterWalletTransactions:
		if isShown && needsRefresh {
			u.characterWallet.transactions.update()
		}
	default:
		slog.Warn(fmt.Sprintf("section not part of the refresh ticker: %s", s))
	}
}

// update corporation sections

func (u *baseUI) startUpdateTickerCorporations() {
	ticker := time.NewTicker(characterSectionsUpdateTicker)
	ctx := context.Background()
	go func() {
		for {
			if err := u.updateCorporationsIfNeeded(ctx, false); err != nil {
				slog.Error("Failed to update corporations", "error", err)
			}
			<-ticker.C
		}
	}()
}

func (u *baseUI) updateCorporationsIfNeeded(ctx context.Context, forceUpdate bool) error {
	if !forceUpdate && isNowDailyDowntime() {
		slog.Info("Skipping regular update of corporations during daily downtime")
		return nil
	}
	removed, err := u.rs.RemoveStaleCorporations(ctx)
	if err != nil {
		return err
	}
	if removed {
		u.updateStatus()
	}
	all, err := u.rs.ListCorporationIDs(ctx)
	if err != nil {
		return err
	}
	for id := range all.All() {
		go u.updateCorporationAndRefreshIfNeeded(ctx, id, forceUpdate)
	}
	slog.Debug("started update status corporations")
	return nil
}

// updateCorporationAndRefreshIfNeeded runs update for all sections of a corporation if needed
// and refreshes the UI accordingly.
func (u *baseUI) updateCorporationAndRefreshIfNeeded(ctx context.Context, corporationID int32, forceUpdate bool) {
	if u.isOffline {
		return
	}
	if !u.isDesktop && !u.isForeground.Load() && !forceUpdate {
		// nothing to update
		return
	}
	sections := app.CorporationSections
	slog.Debug("Starting to check corporation sections for update", "sections", sections)
	for _, s := range sections {
		go u.updateCorporationSectionAndRefreshIfNeeded(ctx, corporationID, s, forceUpdate)
	}
}

// updateCorporationSectionAndRefreshIfNeeded runs update for a corporation section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on corporation sections needs to be included
// to make sure they are refreshed when data changes.
func (u *baseUI) updateCorporationSectionAndRefreshIfNeeded(ctx context.Context, corporationID int32, s app.CorporationSection, forceUpdate bool) {
	hasChanged, err := u.rs.UpdateSectionIfNeeded(
		ctx, app.CorporationUpdateSectionParams{
			CorporationID:         corporationID,
			ForceUpdate:           forceUpdate,
			MaxWalletTransactions: u.settings.MaxWalletTransactions(),
			Section:               s,
		})
	if err != nil {
		slog.Error("Failed to update corporation section", "corporationID", corporationID, "section", s, "err", err)
		return
	}
	if !hasChanged && !forceUpdate {
		return
	}
	isShown := corporationID == u.currentCorporationID()
	switch s {
	case app.SectionCorporationIndustryJobs:
		u.industryJobs.update()
		u.slotsManufacturing.update()
		u.slotsReactions.update()
		u.slotsResearch.update()
	case app.SectionCorporationMembers:
		if isShown {
			u.corporationMember.update()
		}
	case app.SectionCorporationDivisions:
		if isShown {
			for _, d := range app.Divisions {
				u.corporationWallets[d].updateName()
			}
		}
	case app.SectionCorporationWalletBalances:
		if isShown {
			for _, d := range app.Divisions {
				u.corporationWallets[d].updateBalance()
			}
			u.updateCorporationWalletTotal()
		}
	case app.SectionCorporationWalletJournal1:
		if isShown {
			u.corporationWallets[app.Division1].journal.update()
		}
	case app.SectionCorporationWalletJournal2:
		if isShown {
			u.corporationWallets[app.Division2].journal.update()
		}
	case app.SectionCorporationWalletJournal3:
		if isShown {
			u.corporationWallets[app.Division3].journal.update()
		}
	case app.SectionCorporationWalletJournal4:
		if isShown {
			u.corporationWallets[app.Division4].journal.update()
		}
	case app.SectionCorporationWalletJournal5:
		if isShown {
			u.corporationWallets[app.Division5].journal.update()
		}
	case app.SectionCorporationWalletJournal6:
		if isShown {
			u.corporationWallets[app.Division6].journal.update()
		}
	case app.SectionCorporationWalletJournal7:
		if isShown {
			u.corporationWallets[app.Division7].journal.update()
		}
	case app.SectionCorporationWalletTransactions1:
		if isShown {
			u.corporationWallets[app.Division1].transactions.update()
		}
	case app.SectionCorporationWalletTransactions2:
		if isShown {
			u.corporationWallets[app.Division2].transactions.update()
		}
	case app.SectionCorporationWalletTransactions3:
		if isShown {
			u.corporationWallets[app.Division3].transactions.update()
		}
	case app.SectionCorporationWalletTransactions4:
		if isShown {
			u.corporationWallets[app.Division4].transactions.update()
		}
	case app.SectionCorporationWalletTransactions5:
		if isShown {
			u.corporationWallets[app.Division5].transactions.update()
		}
	case app.SectionCorporationWalletTransactions6:
		if isShown {
			u.corporationWallets[app.Division6].transactions.update()
		}
	case app.SectionCorporationWalletTransactions7:
		if isShown {
			u.corporationWallets[app.Division7].transactions.update()
		}
	default:
		slog.Warn(fmt.Sprintf("section not part of the refresh ticker: %s", s))
	}
}

func (u *baseUI) updateCorporationWalletTotal() {
	if u.onUpdateCorporationWalletTotals == nil {
		return
	}
	s := func() string {
		corporationID := u.currentCorporationID()
		if corporationID == 0 {
			return ""
		}
		hasRole, err := u.rs.PermittedSection(context.Background(), corporationID, app.SectionCorporationWalletBalances)
		if err != nil {
			slog.Error("Failed to determine role for corporation wallet", "error", err)
			return ""
		}
		if !hasRole {
			return ""
		}
		b, err := u.rs.GetWalletBalancesTotal(context.Background(), corporationID)
		if err != nil {
			slog.Error("Failed to update wallet total", "corporationID", corporationID, "error", err)
			return ""
		}
		return b.StringFunc("", func(v float64) string {
			return humanize.Number(b.ValueOrZero(), 1)
		})
	}()
	u.onUpdateCorporationWalletTotals(s)
}

func (u *baseUI) notifyExpiredTrainingIfNeeded(ctx context.Context, characterID int32) {
	if u.settings.NotifyTrainingEnabled() {
		go func() {
			// TODO: earliest := calcNotifyEarliest(u.fyneApp.Preferences(), settingNotifyTrainingEarliest)
			err := u.cs.NotifyExpiredTraining(ctx, characterID, u.sendDesktopNotification)
			if err != nil {
				slog.Error("notify expired training", "error", err)
			}
		}()
	}
}

func (u *baseUI) notifyExpiredExtractionsIfNeeded(ctx context.Context, characterID int32) {
	if u.settings.NotifyPIEnabled() {
		go func() {
			earliest := u.settings.NotifyPIEarliest()
			err := u.cs.NotifyExpiredExtractions(ctx, characterID, earliest, u.sendDesktopNotification)
			if err != nil {
				slog.Error("notify expired extractions", "characterID", characterID, "error", err)
			}
		}()
	}
}

func (u *baseUI) availableUpdate() (github.VersionInfo, error) {
	current := u.app.Metadata().Version
	v, err := github.AvailableUpdate(githubOwner, githubRepo, current)
	if err != nil {
		return github.VersionInfo{}, err
	}
	return v, nil
}

func (u *baseUI) ShowInformationDialog(title, message string, parent fyne.Window) {
	d := dialog.NewInformation(title, message, parent)
	u.ModifyShortcutsForDialog(d, parent)
	d.Show()
}

func (u *baseUI) ShowConfirmDialog(title, message, confirm string, callback func(bool), parent fyne.Window) {
	d := dialog.NewConfirm(title, message, callback, parent)
	d.SetConfirmImportance(widget.DangerImportance)
	d.SetConfirmText(confirm)
	d.SetDismissText("Cancel")
	u.ModifyShortcutsForDialog(d, parent)
	d.Show()
}

func (u *baseUI) NewErrorDialog(message string, err error, parent fyne.Window) dialog.Dialog {
	text := widget.NewLabel(fmt.Sprintf("%s\n\n%s", message, u.humanizeError(err)))
	text.Wrapping = fyne.TextWrapWord
	text.Importance = widget.DangerImportance
	c := container.NewVScroll(text)
	c.SetMinSize(fyne.Size{Width: 400, Height: 100})
	d := dialog.NewCustom("Error", "OK", c, parent)
	u.ModifyShortcutsForDialog(d, parent)
	return d
}

func (u *baseUI) showErrorDialog(message string, err error, parent fyne.Window) {
	d := u.NewErrorDialog(message, err, parent)
	d.Show()
}

// ModifyShortcutsForDialog modifies the shortcuts for a dialog.
func (u *baseUI) ModifyShortcutsForDialog(d dialog.Dialog, w fyne.Window) {
	kxdialog.AddDialogKeyHandler(d, w)
	if u.disableMenuShortcuts != nil && u.enableMenuShortcuts != nil {
		u.disableMenuShortcuts()
		d.SetOnClosed(func() {
			u.enableMenuShortcuts()
		})
	}
}

func (u *baseUI) ShowLocationInfoWindow(id int64) {
	iw := newInfoWindow(u)
	iw.showLocation(id)
}

func (u *baseUI) ShowRaceInfoWindow(id int32) {
	iw := newInfoWindow(u)
	iw.showRace(id)
}

func (u *baseUI) ShowTypeInfoWindow(id int32) {
	iw := newInfoWindow(u)
	iw.Show(app.EveEntityInventoryType, id)
}

func (u *baseUI) ShowTypeInfoWindowWithCharacter(entityID, characterID int32) {
	iw := newInfoWindow(u)
	iw.showWithCharacterID(infoInventoryType, int64(entityID), characterID)
}

func (u *baseUI) ShowEveEntityInfoWindow(o *app.EveEntity) {
	iw := newInfoWindow(u)
	iw.showEveEntity(o)
}

func (u *baseUI) ShowInfoWindow(c app.EveEntityCategory, id int32) {
	iw := newInfoWindow(u)
	iw.Show(c, id)
}

func (u *baseUI) ShowSnackbar(text string) {
	u.snackbar.Show(text)
}

func (u *baseUI) websiteRootURL() *url.URL {
	s := u.app.Metadata().Custom["Website"]
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

func (u *baseUI) appName() string {
	info := u.app.Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
}

func (u *baseUI) makeAboutPage() fyne.CanvasObject {
	v, err := github.NormalizeVersion(u.app.Metadata().Version)
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
		v, err := u.availableUpdate()
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
		fyne.Do(func() {
			latest.Text = s
			latest.TextStyle.Bold = isBold
			latest.Importance = i
			latest.Refresh()
			spinner.Hide()
			latest.Show()
		})
	}()
	title := widget.NewLabel(u.appName())
	title.SizeName = theme.SizeNameSubHeadingText
	title.TextStyle.Bold = true
	c := container.New(
		layout.NewCustomPaddedVBoxLayout(0),
		title,
		container.New(layout.NewCustomPaddedVBoxLayout(0),
			container.NewHBox(widget.NewLabel("Latest version:"), layout.NewSpacer(), container.NewStack(spinner, latest)),
			container.NewHBox(widget.NewLabel("You have:"), layout.NewSpacer(), local),
		),
		container.NewHBox(
			widget.NewHyperlink("Website", u.websiteRootURL()),
			widget.NewHyperlink("Downloads", u.websiteRootURL().JoinPath("releases")),
		),
		widget.NewLabel("\"EVE\", \"EVE Online\", \"CCP\", \nand all related logos and images \nare trademarks or registered trademarks of CCP hf."),
		widget.NewLabel("(c) 2024-25 Erik Kalkoken"),
	)
	return c
}

// getOrCreateWindow returns a unique window as defined by the given id string
// and reports whether a new window was created or the window already exists.
func (u *baseUI) getOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool) {
	w, ok, f := u.getOrCreateWindowWithOnClosed(id, titles...)
	if f != nil {
		w.SetOnClosed(f)
	}
	return w, ok
}

// getOrCreateWindowWithOnClosed is like getOrCreateWindow,
// but returns an additional onClosed function which must be called when the window is closed.
// This allows constructing a custom onClosed callback for the window.
func (u *baseUI) getOrCreateWindowWithOnClosed(id string, titles ...string) (window fyne.Window, created bool, onClosed func()) {
	w, ok := u.windows[id]
	if ok {
		return w, false, nil
	}
	w = u.App().NewWindow(u.makeWindowTitle(titles...))
	u.windows[id] = w
	f := func() {
		delete(u.windows, id)
	}
	return w, true, f
}

func (u *baseUI) makeWindowTitle(parts ...string) string {
	if len(parts) == 0 {
		parts = append(parts, "PLACEHOLDER")
	}
	if !u.isDesktop {
		return parts[0]
	}
	parts = append(parts, u.appName())
	return strings.Join(parts, " - ")
}

func (u *baseUI) makeCopyToClipboardLabel(text string) *kxwidget.TappableLabel {
	return kxwidget.NewTappableLabel(text, func() {
		u.App().Clipboard().SetContent(text)
	})
}

// makeTopText makes the content for the top label of a gui element.
func (u *baseUI) makeTopText(characterID int32, hasData bool, err error, make func() (string, widget.Importance)) (string, widget.Importance) {
	if err != nil {
		return "ERROR: " + u.humanizeError(err), widget.DangerImportance
	}
	if characterID == 0 {
		return "No entity", widget.LowImportance
	}
	if !hasData {
		return "No data", widget.WarningImportance
	}
	if make == nil {
		return "", widget.MediumImportance
	}
	return make()
}

// isNowDailyDowntime reports whether the daily downtime is expected to happen currently.
func isNowDailyDowntime() bool {
	return isTimeWithinRange(downtimeStart, downtimeDuration, time.Now())
}

func isTimeWithinRange(start string, duration time.Duration, t time.Time) bool {
	t2, err := time.Parse("15:04", t.UTC().Format("15:04"))
	if err != nil {
		panic(err)
	}
	start2, err := time.Parse("15:04", start)
	if err != nil {
		panic(err)
	}
	end := start2.Add(duration)
	if t2.Before(start2) {
		return false
	}
	if t2.After(end) {
		return false
	}
	return true
}
