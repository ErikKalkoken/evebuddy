// Package ui implements the graphical user interface of the app.
package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

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
	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// update info
const (
	githubOwner        = "ErikKalkoken"
	githubRepo         = "evebuddy"
	fallbackWebsiteURL = "https://github.com/ErikKalkoken/evebuddy"
)

// width of common columns in data tables
const (
	columnWidthCharacter = 200
	columnWidthDateTime  = 150
	columnWidthLocation  = 350
	columnWidthRegion    = 150
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
	scs *statuscacheservice.StatusCacheService
}

// BaseUI represents the core UI logic and is used by both the desktop and mobile UI.
type BaseUI struct {
	DisableMenuShortcuts func()
	EnableMenuShortcuts  func()
	hideMailIndicator    func()
	ShowMailIndicator    func()

	onAppFirstStarted    func()
	onAppStopped         func()
	onAppTerminated      func()
	onRefreshCross       func()
	onSetCharacter       func(int32)
	onShowAndRun         func()
	onUpdateCharacter    func(*app.Character)
	onUpdateStatus       func()
	showManageCharacters func()

	characterAsset             *CharacterAssets
	characterAttributes        *CharacterAttributes
	characterBiography         *CharacterBiography
	characterCommunications    *CharacterCommunications
	characterImplants          *CharacterAugmentations
	characterJumpClones        *CharacterJumpClones
	characterMail              *CharacterMails
	characterSheet             *CharacterSheet
	characterShips             *CharacterFlyableShips
	characterSkillCatalogue    *CharacterSkillCatalogue
	characterSkillQueue        *CharacterSkillQueue
	characterWalletJournal     *CharacterWalletJournal
	characterWalletTransaction *CharacterWalletTransaction
	contracts                  *contracts
	gameSearch                 *GameSearch
	industryJobs               *industryJobs
	manageCharacters           *ManageCharacters
	overviewAssets             *overviewAssets
	overviewCharacters         *OverviewCharacters
	overviewClones             *overviewClones
	colonies                   *colonies
	overviewLocations          *OverviewLocations
	overviewTraining           *OverviewTraining
	overviewWealth             *OverviewWealth
	userSettings               *UserSettings

	app                fyne.App
	clearCache         func() // clear all caches
	cs                 *characterservice.CharacterService
	dataPaths          map[string]string // Paths to user data
	eis                *eveimageservice.EveImageService
	ess                *esistatusservice.ESIStatusService
	eus                *eveuniverseservice.EveUniverseService
	isDesktop          bool        // whether the app runs on a desktop. If false we assume it's on mobile.
	isForeground       atomic.Bool // whether the app is currently shown in the foreground
	isOffline          bool        // Run the app in offline mode
	isStartupCompleted atomic.Bool // whether the app has completed startup (for testing)
	isUpdateDisabled   bool        // Whether to disable update tickers (useful for debugging)
	js                 *janiceservice.JaniceService
	memcache           *memcache.Cache
	rs                 *corporationservice.CorporationService
	scs                *statuscacheservice.StatusCacheService
	settings           *settings.Settings
	snackbar           *iwidget.Snackbar
	statusWindow       fyne.Window
	wasStarted         atomic.Bool // whether the app has already been started at least once
	window             fyne.Window

	character atomic.Pointer[app.Character]
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
func NewBaseUI(args BaseUIParams) *BaseUI {
	u := &BaseUI{
		app:              args.App,
		cs:               args.CharacterService,
		eis:              args.EveImageService,
		ess:              args.ESIStatusService,
		eus:              args.EveUniverseService,
		isDesktop:        args.IsDesktop,
		isOffline:        args.IsOffline,
		isUpdateDisabled: args.IsUpdateDisabled,
		js:               args.JaniceService,
		memcache:         args.MemCache,
		rs:               args.CorporationService,
		scs:              args.StatusCacheService,
		settings:         settings.New(args.App.Preferences()),
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

	u.characterAsset = NewCharacterAssets(u)
	u.characterAttributes = NewCharacterAttributes(u)
	u.characterBiography = NewCharacterBiography(u)
	u.characterCommunications = NewCharacterCommunications(u)
	u.characterImplants = NewCharacterAugmentations(u)
	u.characterJumpClones = NewCharacterJumpClones(u)
	u.characterMail = NewCharacterMails(u)
	u.characterSheet = NewSheet(u)
	u.characterShips = NewCharacterFlyableShips(u)
	u.characterSkillCatalogue = NewCharacterSkillCatalogue(u)
	u.characterSkillQueue = NewCharacterSkillQueue(u)
	u.characterWalletJournal = NewCharacterWalletJournal(u)
	u.characterWalletTransaction = NewCharacterWalletTransaction(u)
	u.contracts = newContracts(u)
	u.gameSearch = NewGameSearch(u)
	u.industryJobs = NewIndustryJobs(u)
	u.manageCharacters = NewManageCharacters(u)
	u.overviewAssets = newOverviewAssets(u)
	u.overviewCharacters = NewOverviewCharacters(u)
	u.overviewClones = newOverviewClones(u)
	u.colonies = NewColonies(u)
	u.overviewLocations = NewOverviewLocations(u)
	u.overviewTraining = NewOverviewTraining(u)
	u.overviewWealth = NewOverviewWealth(u)
	u.snackbar = iwidget.NewSnackbar(u.window)
	u.userSettings = NewSettings(u)
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
		u.isForeground.Store(true)
		u.snackbar.Start()
		go func() {
			u.initCharacter()
			u.manageCharacters.update()
			u.updateCrossPages()
			u.updateStatus()
			u.isStartupCompleted.Store(true)
			u.characterJumpClones.StartUpdateTicker()
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
			u.updateCharactersIfNeeded(context.Background())
			u.updateGeneralSectionsAndRefreshIfNeeded(false)
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

func (u *BaseUI) initCharacter() {
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

// ShowAndRun shows the UI and runs the Fyne loop (blocking),
func (u *BaseUI) ShowAndRun() {
	if u.onShowAndRun != nil {
		u.onShowAndRun()
	}
	u.window.ShowAndRun()
	slog.Info("App terminated")
	if u.onAppTerminated != nil {
		u.onAppTerminated()
	}
}

func (u *BaseUI) services() services {
	return services{
		cs:  u.cs,
		eis: u.eis,
		eus: u.eus,
		scs: u.scs,
	}
}

func (u *BaseUI) App() fyne.App {
	return u.app
}

func (u *BaseUI) ClearAllCaches() {
	u.clearCache()
}

func (u *BaseUI) MainWindow() fyne.Window {
	return u.window
}

func (u *BaseUI) IsDeveloperMode() bool {
	return u.settings.DeveloperMode()
}

func (u *BaseUI) IsOffline() bool {
	return u.isOffline
}

func (u *BaseUI) IsStartupCompleted() bool {
	return u.isStartupCompleted.Load()
}

// humanizeError returns user friendly representation of an error for display in the UI.
func (u *BaseUI) humanizeError(err error) string {
	if err == nil {
		return "No error"
	}
	if u.settings.DeveloperMode() {
		return err.Error()
	}
	return err.Error()
	// return ihumanize.Error(err) TODO: Re-enable again when app is stable enough
}

func (u *BaseUI) MakeWindowTitle(subTitle string) string {
	if !u.isDesktop {
		return subTitle
	}
	return fmt.Sprintf("%s - %s", subTitle, u.appName())
}

// currentCharacterID returns the ID of the current character or 0 if non is set.
func (u *BaseUI) currentCharacterID() int32 {
	c := u.currentCharacter()
	if c == nil {
		return 0
	}
	return c.ID
}

func (u *BaseUI) currentCharacter() *app.Character {
	return u.character.Load()
}

func (u *BaseUI) hasCharacter() bool {
	return u.currentCharacter() != nil
}

func (u *BaseUI) loadCharacter(id int32) error {
	c, err := u.cs.GetCharacter(context.Background(), id)
	if err != nil {
		return fmt.Errorf("load character ID %d: %w", id, err)
	}
	u.setCharacter(c)
	return nil
}

// reloadCurrentCharacter reloads the current character from storage.
func (u *BaseUI) reloadCurrentCharacter() {
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

// updateStatus refreshed all status information pages.
func (u *BaseUI) updateStatus() {
	if u.onUpdateStatus == nil {
		return
	}
	u.onUpdateStatus()
}

// updateCharacter updates all pages for the current character.
func (u *BaseUI) updateCharacter() {
	c := u.currentCharacter()
	if c != nil {
		slog.Debug("Updating character", "ID", c.EveCharacter.ID, "name", c.EveCharacter.Name)
	} else {
		slog.Debug("Updating without character")
	}
	ff := map[string]func(){
		"assets":            u.characterAsset.update,
		"attributes":        u.characterAttributes.update,
		"biography":         u.characterBiography.update,
		"implants":          u.characterImplants.update,
		"jumpClones":        u.characterJumpClones.update,
		"mail":              u.characterMail.update,
		"notifications":     u.characterCommunications.update,
		"sheet":             u.characterSheet.update,
		"ships":             u.characterShips.update,
		"skillCatalogue":    u.characterSkillCatalogue.update,
		"skillqueue":        u.characterSkillQueue.update,
		"walletJournal":     u.characterWalletJournal.update,
		"walletTransaction": u.characterWalletTransaction.update,
	}
	runFunctionsWithProgressModal("Loading character", ff, func() {
		if u.onUpdateCharacter != nil {
			u.onUpdateCharacter(c)
		}
		if c != nil && !u.isUpdateDisabled {
			u.updateCharacterAndRefreshIfNeeded(context.Background(), c.ID, false)
		}
	}, u.window)
}

// updateCrossPages refreshed all pages that contain information about multiple characters.
func (u *BaseUI) updateCrossPages() {
	ff := map[string]func(){
		"assetSearch": u.overviewAssets.update,
		"contracts":   u.contracts.update,
		"cloneSearch": u.overviewClones.update,
		"colony":      u.colonies.update,
		"industryJob": u.industryJobs.update,
		"locations":   u.overviewLocations.update,
		"overview":    u.overviewCharacters.update,
		"training":    u.overviewTraining.update,
		"wealth":      u.overviewWealth.update,
	}
	runFunctionsWithProgressModal("Updating characters", ff, u.onRefreshCross, u.window)
}

// TODO: Replace with "infinite" variant, because progress can not be shown correctly.
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

func (u *BaseUI) resetCharacter() {
	u.character.Store(nil)
	u.settings.ResetLastCharacterID()
	u.updateCharacter()
	u.updateStatus()
}

func (u *BaseUI) setCharacter(c *app.Character) {
	u.character.Store(c)
	u.settings.SetLastCharacterID(c.ID)
	u.updateCharacter()
	u.updateStatus()
	if u.onSetCharacter != nil {
		u.onSetCharacter(c.ID)
	}
}

func (u *BaseUI) setAnyCharacter() error {
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

func (u *BaseUI) updateAvatar(id int32, setIcon func(fyne.Resource)) {
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

func (u *BaseUI) updateMailIndicator() {
	if u.ShowMailIndicator == nil || u.hideMailIndicator == nil {
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
		u.ShowMailIndicator()
	} else {
		u.hideMailIndicator()
	}
}

func (u *BaseUI) makeCharacterSwitchMenu(refresh func()) []*fyne.MenuItem {
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
			go u.updateAvatar(c.ID, func(r fyne.Resource) {
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

func (u *BaseUI) sendDesktopNotification(title, content string) {
	fyne.Do(func() {
		u.app.SendNotification(fyne.NewNotification(title, content))
	})
	slog.Info("desktop notification sent", "title", title, "content", content)
}

// update general sections

func (u *BaseUI) startUpdateTickerGeneralSections() {
	ticker := time.NewTicker(generalSectionsUpdateTicker)
	go func() {
		for {
			u.updateGeneralSectionsAndRefreshIfNeeded(false)
			<-ticker.C
		}
	}()
}

func (u *BaseUI) updateGeneralSectionsAndRefreshIfNeeded(forceUpdate bool) {
	if !forceUpdate && !u.isDesktop && !u.isForeground.Load() {
		slog.Debug("Skipping general sections update while in background")
		return
	}
	ctx := context.Background()
	for _, s := range app.GeneralSections {
		go func(s app.GeneralSection) {
			u.updateGeneralSectionAndRefreshIfNeeded(ctx, s, forceUpdate)
		}(s)
	}
}

func (u *BaseUI) updateGeneralSectionAndRefreshIfNeeded(ctx context.Context, section app.GeneralSection, forceUpdate bool) {
	hasChanged, err := u.eus.UpdateSection(ctx, section, forceUpdate)
	if err != nil {
		slog.Error("Failed to update general section", "section", section, "err", err)
		return
	}
	needsRefresh := hasChanged || forceUpdate
	switch section {
	case app.SectionEveTypes:
		if needsRefresh {
			u.characterShips.update()
			u.characterSkillCatalogue.update()
		}
	case app.SectionEveCharacters:
		if needsRefresh {
			u.reloadCurrentCharacter()
			u.overviewCharacters.update()
		}
	case app.SectionEveCorporations:
		// nothing to do
	case app.SectionEveMarketPrices:
		u.characterAsset.update()
		u.overviewCharacters.update()
		u.overviewAssets.update()
		u.reloadCurrentCharacter()
	default:
		slog.Warn(fmt.Sprintf("section not part of the update ticker refresh: %s", section))
	}
}

// update character sections

func (u *BaseUI) startUpdateTickerCharacters() {
	ticker := time.NewTicker(characterSectionsUpdateTicker)
	ctx := context.Background()
	go func() {
		for {
			if err := u.updateCharactersIfNeeded(ctx); err != nil {
				slog.Error("Failed to update characters", "error", err)
			}
			if err := u.notifyCharactersIfNeeded(ctx); err != nil {
				slog.Error("Failed to notify characters", "error", err)
			}
			<-ticker.C
		}
	}()
}

func (u *BaseUI) updateCharactersIfNeeded(ctx context.Context) error {
	cc, err := u.cs.ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	for _, c := range cc {
		go u.updateCharacterAndRefreshIfNeeded(ctx, c.ID, false)
	}
	slog.Debug("started update status characters")
	return nil
}

func (u *BaseUI) notifyCharactersIfNeeded(ctx context.Context) error {
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
func (u *BaseUI) updateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
	if u.isOffline {
		return
	}
	var sections []app.CharacterSection
	if !u.isDesktop && !u.isForeground.Load() {
		// only update what is needed for notifications on mobile when running in background to save battery
		if u.settings.NotifyCommunicationsEnabled() {
			sections = append(sections, app.SectionNotifications)
		}
		if u.settings.NotifyContractsEnabled() {
			sections = append(sections, app.SectionContracts)
		}
		if u.settings.NotifyMailsEnabled() {
			sections = append(sections, app.SectionMailLabels)
			sections = append(sections, app.SectionMailLists)
			sections = append(sections, app.SectionMails)
		}
		if u.settings.NotifyPIEnabled() {
			sections = append(sections, app.SectionPlanets)
		}
		if u.settings.NotifyTrainingEnabled() {
			sections = append(sections, app.SectionSkillqueue)
			sections = append(sections, app.SectionSkills)
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
func (u *BaseUI) updateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, s app.CharacterSection, forceUpdate bool) {
	hasChanged, err := u.cs.UpdateSectionIfNeeded(
		ctx, app.CharacterUpdateSectionParams{
			CharacterID:           characterID,
			Section:               s,
			ForceUpdate:           forceUpdate,
			MaxMails:              u.settings.MaxMails(),
			MaxWalletTransactions: u.settings.MaxWalletTransactions(),
		})
	if err != nil {
		slog.Error("Failed to update character section", "characterID", characterID, "section", s, "err", err)
		return
	}
	isShown := characterID == u.currentCharacterID()
	needsRefresh := hasChanged || forceUpdate
	switch s {
	case app.SectionAssets:
		if needsRefresh {
			u.overviewAssets.update()
			u.overviewWealth.update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterAsset.update()
				u.characterSheet.update()
			}
		}
	case app.SectionAttributes:
		if isShown && needsRefresh {
			u.characterAttributes.update()
		}
	case app.SectionContracts:
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
	case app.SectionImplants:
		if isShown && needsRefresh {
			u.characterImplants.update()
		}
	case app.SectionJumpClones:
		if needsRefresh {
			u.overviewCharacters.update()
			u.overviewClones.update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterJumpClones.update()
			}
		}
	case app.SectionIndustryJobs:
		if needsRefresh {
			u.industryJobs.update()
		}
	case app.SectionLocation, app.SectionOnline, app.SectionShip:
		if needsRefresh {
			u.overviewLocations.update()
			if isShown {
				u.reloadCurrentCharacter()
			}
		}
	case app.SectionPlanets:
		if needsRefresh {
			u.colonies.update()
			u.notifyExpiredExtractionsIfNeeded(ctx, characterID)
		}
	case app.SectionMailLabels, app.SectionMailLists:
		if needsRefresh {
			u.overviewCharacters.update()
			if isShown {
				u.characterMail.update()
			}
		}
	case app.SectionMails:
		if needsRefresh {
			go u.overviewCharacters.update()
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
	case app.SectionNotifications:
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
	case app.SectionRoles:
		// nothing to do
	case app.SectionSkills:
		if needsRefresh {
			u.overviewTraining.update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterSkillCatalogue.update()
				u.characterShips.update()
			}
		}

	case app.SectionSkillqueue:
		if u.settings.NotifyTrainingEnabled() {
			err := u.cs.EnableTrainingWatcher(ctx, characterID)
			if err != nil {
				slog.Error("Failed to enable training watcher", "characterID", characterID, "error", err)
			}
		}
		if isShown {
			u.characterSkillQueue.update()
		}
		if needsRefresh {
			u.overviewTraining.update()
			u.notifyExpiredTrainingIfNeeded(ctx, characterID)
		}
	case app.SectionWalletBalance:
		if needsRefresh {
			u.overviewCharacters.update()
			u.overviewWealth.update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterAsset.update()
			}
		}
	case app.SectionWalletJournal:
		if isShown && needsRefresh {
			u.characterWalletJournal.update()
		}
	case app.SectionWalletTransactions:
		if isShown && needsRefresh {
			u.characterWalletTransaction.update()
		}
	default:
		slog.Warn(fmt.Sprintf("section not part of the refresh ticker: %s", s))
	}
}

// update corporation sections

func (u *BaseUI) startUpdateTickerCorporations() {
	ticker := time.NewTicker(characterSectionsUpdateTicker)
	ctx := context.Background()
	go func() {
		for {
			if err := u.updateCorporationsIfNeeded(ctx); err != nil {
				slog.Error("Failed to update corporations", "error", err)
			}
			<-ticker.C
		}
	}()
}

func (u *BaseUI) updateCorporationsIfNeeded(ctx context.Context) error {
	ids, err := u.rs.ListCorporationIDs(ctx)
	if err != nil {
		return err
	}
	for id := range ids.All() {
		go u.updateCorporationAndRefreshIfNeeded(ctx, id, false)
	}
	slog.Debug("started update status corporations")
	return nil
}

// updateCorporationAndRefreshIfNeeded runs update for all sections of a corporation if needed
// and refreshes the UI accordingly.
func (u *BaseUI) updateCorporationAndRefreshIfNeeded(ctx context.Context, corporationID int32, forceUpdate bool) {
	if u.isOffline {
		return
	}
	var sections []app.CorporationSection
	if !u.isDesktop && !u.isForeground.Load() {
		// nothing to update
	} else {
		sections = app.CorporationSections
	}
	if len(sections) == 0 {
		return
	}
	slog.Debug("Starting to check corporation sections for update", "sections", sections)
	for _, s := range sections {
		_, err := u.cs.ValidCharacterTokenForCorporation(ctx, corporationID, s.Role())
		if err != nil {
			slog.Error("Skipping corporation section update due to missing valid token", "error", err)
			sections = slices.DeleteFunc(sections, func(x app.CorporationSection) bool {
				return x == s
			})
		}
	}
	for _, s := range sections {
		go u.updateCorporationSectionAndRefreshIfNeeded(ctx, corporationID, s, forceUpdate)
	}
}

// updateCorporationSectionAndRefreshIfNeeded runs update for a corporation section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on corporation sections needs to be included
// to make sure they are refreshed when data changes.
func (u *BaseUI) updateCorporationSectionAndRefreshIfNeeded(ctx context.Context, corporationID int32, s app.CorporationSection, forceUpdate bool) {
	hasChanged, err := u.rs.UpdateSectionIfNeeded(
		ctx, app.CorporationUpdateSectionParams{
			CorporationID: corporationID,
			Section:       s,
			ForceUpdate:   forceUpdate,
		})
	if err != nil {
		slog.Error("Failed to update corporation section", "corporationID", corporationID, "section", s, "err", err)
		return
	}
	needsRefresh := hasChanged || forceUpdate
	switch s {
	case app.SectionCorporationIndustryJobs:
		if needsRefresh {
			u.industryJobs.update()
		}
	default:
		slog.Warn(fmt.Sprintf("section not part of the refresh ticker: %s", s))
	}
}

func (u *BaseUI) notifyExpiredTrainingIfNeeded(ctx context.Context, characterID int32) {
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

func (u *BaseUI) notifyExpiredExtractionsIfNeeded(ctx context.Context, characterID int32) {
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

func (u *BaseUI) availableUpdate() (github.VersionInfo, error) {
	current := u.app.Metadata().Version
	v, err := github.AvailableUpdate(githubOwner, githubRepo, current)
	if err != nil {
		return github.VersionInfo{}, err
	}
	return v, nil
}

func (u *BaseUI) ShowInformationDialog(title, message string, parent fyne.Window) {
	d := dialog.NewInformation(title, message, parent)
	u.ModifyShortcutsForDialog(d, parent)
	d.Show()
}

func (u *BaseUI) ShowConfirmDialog(title, message, confirm string, callback func(bool), parent fyne.Window) {
	d := dialog.NewConfirm(title, message, callback, parent)
	d.SetConfirmImportance(widget.DangerImportance)
	d.SetConfirmText(confirm)
	d.SetDismissText("Cancel")
	u.ModifyShortcutsForDialog(d, parent)
	d.Show()
}

func (u *BaseUI) NewErrorDialog(message string, err error, parent fyne.Window) dialog.Dialog {
	text := widget.NewLabel(fmt.Sprintf("%s\n\n%s", message, u.humanizeError(err)))
	text.Wrapping = fyne.TextWrapWord
	text.Importance = widget.DangerImportance
	c := container.NewVScroll(text)
	c.SetMinSize(fyne.Size{Width: 400, Height: 100})
	d := dialog.NewCustom("Error", "OK", c, parent)
	u.ModifyShortcutsForDialog(d, parent)
	return d
}

func (u *BaseUI) ShowErrorDialog(message string, err error, parent fyne.Window) {
	d := u.NewErrorDialog(message, err, parent)
	d.Show()
}

// ModifyShortcutsForDialog modifies the shortcuts for a dialog.
func (u *BaseUI) ModifyShortcutsForDialog(d dialog.Dialog, w fyne.Window) {
	kxdialog.AddDialogKeyHandler(d, w)
	if u.DisableMenuShortcuts != nil && u.EnableMenuShortcuts != nil {
		u.DisableMenuShortcuts()
		d.SetOnClosed(func() {
			u.EnableMenuShortcuts()
		})
	}
}

func (u *BaseUI) showUpdateStatusWindow() {
	if u.statusWindow != nil {
		u.statusWindow.Show()
		return
	}
	w := u.app.NewWindow(u.MakeWindowTitle("Update Status"))
	a := newUpdateStatus(u)
	a.update()
	w.SetContent(a)
	w.Resize(fyne.Size{Width: 1100, Height: 500})
	ctx, cancel := context.WithCancel(context.Background())
	a.startTicker(ctx)
	w.SetOnClosed(func() {
		cancel()
		u.statusWindow = nil
	})
	u.statusWindow = w
	w.Show()
}

func (u *BaseUI) ShowLocationInfoWindow(id int64) {
	iw := newInfoWindow(u)
	iw.showLocation(id)
}

func (u *BaseUI) ShowRaceInfoWindow(id int32) {
	iw := newInfoWindow(u)
	iw.showRace(id)
}

func (u *BaseUI) ShowTypeInfoWindow(id int32) {
	iw := newInfoWindow(u)
	iw.Show(app.EveEntityInventoryType, id)
}

func (u *BaseUI) ShowEveEntityInfoWindow(o *app.EveEntity) {
	iw := newInfoWindow(u)
	iw.showEveEntity(o)
}

func (u *BaseUI) ShowInfoWindow(c app.EveEntityCategory, id int32) {
	iw := newInfoWindow(u)
	iw.Show(c, id)
}

func (u *BaseUI) ShowSnackbar(text string) {
	u.snackbar.Show(text)
}

func (u *BaseUI) websiteRootURL() *url.URL {
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

func (u *BaseUI) appName() string {
	info := u.app.Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
}

func (u *BaseUI) makeAboutPage() fyne.CanvasObject {
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
	title := iwidget.NewLabelWithSize(u.appName(), theme.SizeNameSubHeadingText)
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

func (u *BaseUI) makeDetailWindow(title, subTitle string, content fyne.CanvasObject) fyne.Window {
	w := u.App().NewWindow(u.MakeWindowTitle(title))
	t := iwidget.NewLabelWithSize(subTitle, theme.SizeNameSubHeadingText)
	top := container.NewVBox(t, widget.NewSeparator())
	vs := container.NewVScroll(content)
	vs.SetMinSize(fyne.NewSize(600, 500))
	c := container.NewBorder(
		top,
		nil,
		nil,
		nil,
		vs,
	)
	c.Refresh()
	w.SetContent(container.NewPadded(c))
	return w
}

func (u *BaseUI) makeCopyToClipboardLabel(text string) *kxwidget.TappableLabel {
	return kxwidget.NewTappableLabel(text, func() {
		u.App().Clipboard().SetContent(text)
	})
}

// makeTopText makes the content for the top label of a gui element.
func makeTopText(characterID int32, hasData bool, err error, make func() (string, widget.Importance)) (string, widget.Importance) {
	if err != nil {
		return "ERROR", widget.DangerImportance
	}
	if characterID == 0 {
		return "No character...", widget.LowImportance
	}
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	return make()
}
