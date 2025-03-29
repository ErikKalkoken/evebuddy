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
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxdialog "github.com/ErikKalkoken/fyne-kx/dialog"
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterui"
	"github.com/ErikKalkoken/evebuddy/internal/app/collectionui"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/infowindow"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/toolui"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	"github.com/ErikKalkoken/evebuddy/internal/github"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// update info
const (
	githubOwner        = "ErikKalkoken"
	githubRepo         = "evebuddy"
	fallbackWebsiteURL = "https://github.com/ErikKalkoken/evebuddy"
)

// ticker
const (
	characterSectionsUpdateTicker = 60 * time.Second
	generalSectionsUpdateTicker   = 300 * time.Second
)

// UIBase represents the core UI logic and is used by both the desktop and mobile UI.
type UIBase struct {
	DisableMenuShortcuts func()
	EnableMenuShortcuts  func()
	HideMailIndicator    func()
	ShowMailIndicator    func()

	onAppFirstStarted func()
	onAppStopped      func()
	onAppTerminated   func()
	onInit            func(*app.Character)
	onRefreshCross    func()
	onSetCharacter    func(int32)
	onShowAndRun      func()
	onUpdateCharacter func(*app.Character)
	onUpdateStatus    func()

	allAssetSearch             *collectionui.AllAssetSearch
	characterAssets            *characterui.CharacterAssets
	characterAttributes        *characterui.CharacterAttributes
	characterCommunications    *characterui.CharacterCommunications
	characterContracts         *characterui.CharacterContracts
	characterImplants          *characterui.CharacterImplants
	characterJumpClones        *characterui.CharacterJumpClones
	characterMail              *characterui.CharacterMail
	characterOverview          *collectionui.CharacterOverview
	characterPlanets           *characterui.CharacterPlanets
	characterShips             *characterui.CharacterShips
	characterSkillCatalogue    *characterui.CharacterSkillCatalogue
	characterSkillQueue        *characterui.CharacterSkillQueue
	characterWalletJournal     *characterui.CharacterWalletJournal
	characterWalletTransaction *characterui.CharacterWalletTransaction
	colonyOverview             *collectionui.ColonyOverview
	cloneSearch                *collectionui.CloneSearch
	gameSearch                 *toolui.GameSearch
	locationOverview           *collectionui.LocationOverview
	managerCharacters          *toolui.ManageCharacters
	trainingOverview           *collectionui.TrainingOverview
	userSettings               *toolui.UserSettings
	wealthOverview             *collectionui.WealthOverview

	app              fyne.App
	character        *app.Character
	clearCache       func() // clear all caches
	cs               app.CharacterService
	dataPaths        map[string]string // Paths to user data
	eis              app.EveImageService
	ess              app.ESIStatusService
	eus              app.EveUniverseService
	isForeground     atomic.Bool // whether the app is currently shown in the foreground
	isMobile         bool
	isOffline        bool // Run the app in offline mode
	isUpdateDisabled bool // Whether to disable update tickers (useful for debugging)
	memcache         app.CacheService
	scs              app.StatusCacheService
	settings         app.Settings
	snackbar         *iwidget.Snackbar
	statusWindow     fyne.Window
	wasStarted       atomic.Bool // whether the app has already been started at least once
	window           fyne.Window
}

// NewBaseUI constructs and returns a new BaseUI.
//
// Note:Types embedding BaseUI should define callbacks instead of overwriting methods.
func NewBaseUI(
	app fyne.App,
	cs app.CharacterService,
	eis app.EveImageService,
	ess app.ESIStatusService,
	eus app.EveUniverseService,
	scs app.StatusCacheService,
	memCache app.CacheService,
	isOffline bool,
	isUpdateDisabled bool,
	dataPaths map[string]string,
	clearCache func(),
) *UIBase {
	u := &UIBase{
		app:              app,
		clearCache:       clearCache,
		cs:               cs,
		dataPaths:        dataPaths,
		eis:              eis,
		ess:              ess,
		eus:              eus,
		isMobile:         fyne.CurrentDevice().IsMobile(),
		isOffline:        isOffline,
		isUpdateDisabled: isUpdateDisabled,
		memcache:         memCache,
		scs:              scs,
		settings:         settings.New(app.Preferences()),
	}
	u.window = app.NewWindow(u.appName())

	if u.IsDesktop() {
		iwidget.DefaultImageScaleMode = canvas.ImageScaleFastest
		appwidget.DefaultImageScaleMode = canvas.ImageScaleFastest
	}

	u.snackbar = iwidget.NewSnackbar(u.window)

	u.allAssetSearch = collectionui.NewAssetSearch(u)
	u.characterAssets = characterui.NewCharacterAssets(u)
	u.characterAttributes = characterui.NewCharacterAttributes(u)
	u.characterCommunications = characterui.NewCharacterCommunications(u)
	u.characterContracts = characterui.NewCharacterContracts(u)
	u.characterImplants = characterui.NewCharacterImplants(u)
	u.characterJumpClones = characterui.NewCharacterJumpClones(u)
	u.characterMail = characterui.NewCharacterMail(u)
	u.characterOverview = collectionui.NewCharacterOverview(u)
	u.characterPlanets = characterui.NewCharacterPlanets(u)
	u.characterShips = characterui.NewCharacterShips(u)
	u.characterSkillCatalogue = characterui.NewCharacterSkillCatalogue(u)
	u.characterSkillQueue = characterui.NewCharacterSkillQueue(u)
	u.characterWalletJournal = characterui.NewCharacterWalletJournal(u)
	u.characterWalletTransaction = characterui.NewCharacterWalletTransaction(u)
	u.cloneSearch = collectionui.NewCloneSearch(u)
	u.colonyOverview = collectionui.NewColonyOverview(u)
	u.gameSearch = toolui.NewGameSearch(u)
	u.locationOverview = collectionui.NewLocationOverview(u)
	u.managerCharacters = toolui.NewManageCharacters(u)
	u.trainingOverview = collectionui.NewTrainingOverview(u)
	u.userSettings = toolui.NewSettings(u)
	u.wealthOverview = collectionui.NewWealthOverview(u)

	u.MainWindow().SetMaster()
	return u
}

func (u *UIBase) App() fyne.App {
	return u.app
}

func (u *UIBase) ClearAllCaches() {
	u.clearCache()
}

func (u *UIBase) CharacterService() app.CharacterService {
	return u.cs
}

func (u *UIBase) ESIStatusService() app.ESIStatusService {
	return u.ess
}

func (u *UIBase) EveImageService() app.EveImageService {
	return u.eis
}

func (u *UIBase) EveUniverseService() app.EveUniverseService {
	return u.eus
}

func (u *UIBase) MemCache() app.CacheService {
	return u.memcache
}

func (u *UIBase) StatusCacheService() app.StatusCacheService {
	return u.scs
}

func (u *UIBase) DataPaths() map[string]string {
	return u.dataPaths
}

func (u *UIBase) MainWindow() fyne.Window {
	return u.window
}

func (u *UIBase) IsDeveloperMode() bool {
	return u.Settings().DeveloperMode()
}

func (u *UIBase) IsOffline() bool {
	return u.isOffline
}

// Init initialized the app.
// It is meant for initialization logic that requires the UI to be fully created.
// It should be called directly after the UI was created and before the Fyne loop is started.
func (u *UIBase) Init() {
	u.managerCharacters.Refresh()
	var c *app.Character
	var err error
	ctx := context.Background()
	if cID := u.Settings().LastCharacterID(); cID != 0 {
		c, err = u.CharacterService().GetCharacter(ctx, int32(cID))
		if err != nil {
			if !errors.Is(err, app.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c == nil {
		c, err = u.CharacterService().GetAnyCharacter(ctx)
		if err != nil {
			if !errors.Is(err, app.ErrNotFound) {
				slog.Error("Failed to load character", "error", err)
			}
		}
	}
	if c == nil {
		return
	}
	u.setCharacter(c)
	if u.onInit != nil {
		u.onInit(c)
	}
}

// ErrorDisplay returns user friendly representation of an error for display in the UI.
func (u *UIBase) ErrorDisplay(err error) string {
	if u.Settings().DeveloperMode() {
		return err.Error()
	}
	return err.Error()
	// return ihumanize.Error(err) TODO: Re-enable again when app is stable enough
}

func (u *UIBase) IsDesktop() bool {
	_, ok := u.app.(desktop.App)
	return ok
}

func (u *UIBase) IsMobile() bool {
	return u.isMobile
}

func (u *UIBase) MakeWindowTitle(subTitle string) string {
	if u.IsMobile() {
		return subTitle
	}
	return fmt.Sprintf("%s - %s", subTitle, u.appName())
}

func (u *UIBase) Settings() app.Settings {
	return u.settings
}

// ShowAndRun shows the UI and runs the Fyne loop (blocking),
func (u *UIBase) ShowAndRun() {
	// SetOnStarted is called on initial start,
	// but also when an app is coninued after it was temporarily stopped,
	// which can happen on mobile
	u.app.Lifecycle().SetOnStarted(func() {
		wasStarted := !u.wasStarted.CompareAndSwap(false, true)
		if wasStarted {
			slog.Info("App continued")
			return
		}
		// First app start
		slog.Info("App started")
		if u.isOffline {
			slog.Info("Started in offline mode")
		}
		go func() {
			time.Sleep(250 * time.Millisecond) // FIXME: Workaround for occasional progess bar panic
			u.UpdateCrossPages()
			if u.HasCharacter() {
				u.setCharacter(u.character)
			} else {
				u.resetCharacter()
			}
			u.UpdateStatus()
		}()
		u.snackbar.Start()
		if !u.isOffline && !u.isUpdateDisabled {
			u.isForeground.Store(true)
			go func() {
				u.startUpdateTickerGeneralSections()
				u.startUpdateTickerCharacters()
			}()
		} else {
			slog.Info("Update ticker disabled")
		}
		go u.characterJumpClones.StartUpdateTicker()
		if u.onAppFirstStarted != nil {
			u.onAppFirstStarted()
		}
	})
	u.app.Lifecycle().SetOnEnteredForeground(func() {
		slog.Debug("Entered foreground")
		u.isForeground.Store(true)
		u.updateCharactersIfNeeded(context.Background())
		u.UpdateGeneralSectionsAndRefreshIfNeeded(false)
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
	if u.onShowAndRun != nil {
		u.onShowAndRun()
	}
	u.window.ShowAndRun()
	slog.Info("App terminated")
	if u.onAppTerminated != nil {
		u.onAppTerminated()
	}
}

// CurrentCharacterID returns the ID of the current character or 0 if non it set.
func (u *UIBase) CurrentCharacterID() int32 {
	if u.character == nil {
		return 0
	}
	return u.character.ID
}

func (u *UIBase) CurrentCharacter() *app.Character {
	return u.character
}

func (u *UIBase) HasCharacter() bool {
	return u.character != nil
}

func (u *UIBase) LoadCharacter(id int32) error {
	c, err := u.CharacterService().GetCharacter(context.Background(), id)
	if err != nil {
		return fmt.Errorf("load character ID %d: %w", id, err)
	}
	u.setCharacter(c)
	return nil
}

// reloadCurrentCharacter reloads the current character from storage.
func (u *UIBase) reloadCurrentCharacter() {
	id := u.CurrentCharacterID()
	if id == 0 {
		return
	}
	var err error
	u.character, err = u.CharacterService().GetCharacter(context.Background(), id)
	if err != nil {
		slog.Error("reload character", "characterID", id, "error", err)
	}
}

// UpdateStatus refreshed all status information pages.
func (u *UIBase) UpdateStatus() {
	if u.onUpdateStatus == nil {
		return
	}
	go u.onUpdateStatus()
}

// updateCharacter updates all pages for the current character.
func (u *UIBase) updateCharacter() {
	ff := map[string]func(){
		"assets":            u.characterAssets.Update,
		"attributes":        u.characterAttributes.Update,
		"contracts":         u.characterContracts.Update,
		"implants":          u.characterImplants.Update,
		"jumpClones":        u.characterJumpClones.Update,
		"mail":              u.characterMail.Update,
		"notifications":     u.characterCommunications.Update,
		"planets":           u.characterPlanets.Update,
		"ships":             u.characterShips.Update,
		"skillCatalogue":    u.characterSkillCatalogue.Update,
		"skillqueue":        u.characterSkillQueue.Update,
		"walletJournal":     u.characterWalletJournal.Update,
		"walletTransaction": u.characterWalletTransaction.Update,
	}
	c := u.CurrentCharacter()
	if c != nil {
		slog.Debug("Updating character", "ID", c.EveCharacter.ID, "name", c.EveCharacter.Name)
	} else {
		slog.Debug("Updating without character")
	}
	if u.onUpdateCharacter != nil {
		ff["OnUpdateCharacter"] = func() {
			u.onUpdateCharacter(c)
		}
	}
	runFunctionsWithProgressModal("Loading character", ff, u.window)
	if c != nil && !u.isUpdateDisabled {
		u.UpdateCharacterAndRefreshIfNeeded(context.Background(), c.ID, false)
	}
}

// UpdateCrossPages refreshed all pages that contain information about multiple characters.
func (u *UIBase) UpdateCrossPages() {
	ff := map[string]func(){
		"assetSearch": u.allAssetSearch.Update,
		"cloneSeach":  u.cloneSearch.Update,
		"colony":      u.colonyOverview.Update,
		"locations":   u.locationOverview.Update,
		"overview":    u.characterOverview.Update,
		"training":    u.trainingOverview.Update,
		"wealth":      u.wealthOverview.Update,
	}
	if u.onRefreshCross != nil {
		ff["onRefreshCross"] = u.onRefreshCross
	}
	runFunctionsWithProgressModal("Updating characters", ff, u.window)
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

func (u *UIBase) resetCharacter() {
	u.character = nil
	u.Settings().ResetLastCharacterID()
	u.updateCharacter()
	u.UpdateStatus()
}

func (u *UIBase) setCharacter(c *app.Character) {
	u.character = c
	u.Settings().SetLastCharacterID(c.ID)
	u.updateCharacter()
	u.UpdateStatus()
	if u.onSetCharacter != nil {
		u.onSetCharacter(c.ID)
	}
}

func (u *UIBase) SetAnyCharacter() error {
	c, err := u.CharacterService().GetAnyCharacter(context.Background())
	if errors.Is(err, app.ErrNotFound) {
		u.resetCharacter()
		return nil
	} else if err != nil {
		return err
	}
	u.setCharacter(c)
	return nil
}

func (u *UIBase) UpdateAvatar(id int32, setIcon func(fyne.Resource)) {
	r, err := u.EveImageService().CharacterPortrait(id, app.IconPixelSize)
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

func (u *UIBase) UpdateMailIndicator() {
	if u.ShowMailIndicator == nil || u.HideMailIndicator == nil {
		return
	}
	if !u.Settings().SysTrayEnabled() {
		return
	}
	n, err := u.CharacterService().GetAllCharacterMailUnreadCount(context.Background())
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

func (u *UIBase) MakeCharacterSwitchMenu(refresh func()) []*fyne.MenuItem {
	characterID := u.CurrentCharacterID()
	cc := u.StatusCacheService().ListCharacters()
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
				u.snackbar.Show("ERROR: Failed to switch character")
			}
		})
		if c.ID == characterID {
			continue
		}
		it.Icon, _ = fynetools.MakeAvatar(icons.Characterplaceholder64Jpeg)
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

func (u *UIBase) sendDesktopNotification(title, content string) {
	u.app.SendNotification(fyne.NewNotification(title, content))
	slog.Info("desktop notification sent", "title", title, "content", content)
}

func (u *UIBase) startUpdateTickerGeneralSections() {
	ticker := time.NewTicker(generalSectionsUpdateTicker)
	go func() {
		for {
			u.UpdateGeneralSectionsAndRefreshIfNeeded(false)
			<-ticker.C
		}
	}()
}

func (u *UIBase) UpdateGeneralSectionsAndRefreshIfNeeded(forceUpdate bool) {
	if !forceUpdate && u.IsMobile() && !u.isForeground.Load() {
		slog.Debug("Skipping general sections update while in background")
		return
	}
	ctx := context.Background()
	for _, s := range app.GeneralSections {
		go func(s app.GeneralSection) {
			u.UpdateGeneralSectionAndRefreshIfNeeded(ctx, s, forceUpdate)
		}(s)
	}
}

func (u *UIBase) UpdateGeneralSectionAndRefreshIfNeeded(ctx context.Context, section app.GeneralSection, forceUpdate bool) {
	hasChanged, err := u.EveUniverseService().UpdateSection(ctx, section, forceUpdate)
	if err != nil {
		slog.Error("Failed to update general section", "section", section, "err", err)
		return
	}
	switch section {
	case app.SectionEveCategories:
		if hasChanged {
			u.characterShips.Update()
			u.characterSkillCatalogue.Refresh()
		}
	case app.SectionEveCharacters, app.SectionEveMarketPrices:
		// nothing to refresh
	default:
		slog.Warn(fmt.Sprintf("section not part of the update ticker refresh: %s", section))
	}
}

func (u *UIBase) startUpdateTickerCharacters() {
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

func (u *UIBase) updateCharactersIfNeeded(ctx context.Context) error {
	cc, err := u.CharacterService().ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	for _, c := range cc {
		go u.UpdateCharacterAndRefreshIfNeeded(ctx, c.ID, false)
	}
	slog.Debug("started update status characters") // FIXME: Reset to DEBUG
	return nil
}

func (u *UIBase) notifyCharactersIfNeeded(ctx context.Context) error {
	cc, err := u.CharacterService().ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	for _, c := range cc {
		go u.notifyExpiredExtractionsIfNeeded(ctx, c.ID)
		go u.notifyExpiredTrainingIfneeded(ctx, c.ID)
	}
	slog.Debug("started notify characters")
	return nil
}

// UpdateCharacterAndRefreshIfNeeded runs update for all sections of a character if needed
// and refreshes the UI accordingly.
func (u *UIBase) UpdateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
	if u.isOffline {
		return
	}
	var sections []app.CharacterSection
	if u.IsMobile() && !u.isForeground.Load() {
		// only update what is needed for notifications on mobile when running in background to save battery
		if u.Settings().NotifyCommunicationsEnabled() {
			sections = append(sections, app.SectionNotifications)
		}
		if u.Settings().NotifyContractsEnabled() {
			sections = append(sections, app.SectionContracts)
		}
		if u.Settings().NotifyMailsEnabled() {
			sections = append(sections, app.SectionMailLabels)
			sections = append(sections, app.SectionMailLists)
			sections = append(sections, app.SectionMails)
		}
		if u.Settings().NotifyPIEnabled() {
			sections = append(sections, app.SectionPlanets)
		}
		if u.Settings().NotifyTrainingEnabled() {
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
	for _, s := range sections {
		s := s
		go u.UpdateCharacterSectionAndRefreshIfNeeded(ctx, characterID, s, forceUpdate)
	}
}

// UpdateCharacterSectionAndRefreshIfNeeded runs update for a character section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on character sections needs to be included
// to make sure they are refreshed when data changes.
func (u *UIBase) UpdateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, s app.CharacterSection, forceUpdate bool) {
	hasChanged, err := u.CharacterService().UpdateSectionIfNeeded(
		ctx, app.CharacterUpdateSectionParams{
			CharacterID:           characterID,
			Section:               s,
			ForceUpdate:           forceUpdate,
			MaxMails:              u.Settings().MaxMails(),
			MaxWalletTransactions: u.Settings().MaxWalletTransactions(),
		})
	if err != nil {
		slog.Error("Failed to update character section", "characterID", characterID, "section", s, "err", err)
		return
	}
	isShown := characterID == u.CurrentCharacterID()
	needsRefresh := hasChanged || forceUpdate
	if isShown && needsRefresh {
		u.reloadCurrentCharacter()
	}
	switch s {
	case app.SectionAssets:
		if needsRefresh {
			v, err := u.CharacterService().UpdateCharacterAssetTotalValue(ctx, characterID)
			if err != nil {
				slog.Error("update asset total value", "characterID", characterID, "err", err)
			}
			if isShown {
				u.character.AssetValue.Set(v)
			}
			u.allAssetSearch.Update()
			u.wealthOverview.Update()
		}
		if isShown && needsRefresh {
			u.characterAssets.Update()
		}
	case app.SectionAttributes:
		if isShown && needsRefresh {
			u.characterAttributes.Update()
		}
	case app.SectionContracts:
		if isShown && needsRefresh {
			u.characterContracts.Update()
		}
		if u.Settings().NotifyContractsEnabled() {
			go func() {
				earliest := u.Settings().NotifyContractsEarliest()
				if err := u.CharacterService().NotifyUpdatedContracts(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
					slog.Error("notify contract update", "error", err)
				}
			}()
		}
	case app.SectionImplants:
		if isShown && needsRefresh {
			u.characterImplants.Update()
		}
	case app.SectionJumpClones:
		if isShown && needsRefresh {
			u.characterJumpClones.Update()
		}
		if needsRefresh {
			u.characterOverview.Update()
			u.cloneSearch.Update()
		}
	case app.SectionLocation,
		app.SectionOnline,
		app.SectionShip:
		if needsRefresh {
			u.locationOverview.Update()
		}
	case app.SectionPlanets:
		if isShown && needsRefresh {
			u.characterPlanets.Update()
		}
		if needsRefresh {
			u.colonyOverview.Update()
			u.notifyExpiredExtractionsIfNeeded(ctx, characterID)
		}
	case app.SectionMailLabels,
		app.SectionMailLists:
		if isShown && needsRefresh {
			u.characterMail.Update()
		}
		if needsRefresh {
			u.characterOverview.Update()
		}
	case app.SectionMails:
		if isShown && needsRefresh {
			u.characterMail.Update()
		}
		if needsRefresh {
			go u.characterOverview.Update()
			go u.UpdateMailIndicator()
		}
		if u.Settings().NotifyMailsEnabled() {
			go func() {
				earliest := u.Settings().NotifyMailsEarliest()
				if err := u.CharacterService().NotifyMails(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
					slog.Error("notify mails", "characterID", characterID, "error", err)
				}
			}()
		}
	case app.SectionNotifications:
		if isShown && needsRefresh {
			u.characterCommunications.Update()
		}
		if u.Settings().NotifyCommunicationsEnabled() {
			go func() {
				earliest := u.Settings().NotifyCommunicationsEarliest()
				typesEnabled := u.Settings().NotificationTypesEnabled()
				if err := u.CharacterService().NotifyCommunications(ctx, characterID, earliest, typesEnabled, u.sendDesktopNotification); err != nil {
					slog.Error("notify communications", "characterID", characterID, "error", err)
				}
			}()
		}
	case app.SectionSkills:
		if isShown && needsRefresh {
			u.characterSkillCatalogue.Refresh()
			u.characterShips.Update()
			u.characterPlanets.Update()
		}
		if needsRefresh {
			u.trainingOverview.Update()
		}
	case app.SectionSkillqueue:
		if u.Settings().NotifyTrainingEnabled() {
			err := u.CharacterService().EnableTrainingWatcher(ctx, characterID)
			if err != nil {
				slog.Error("Failed to enable training watcher", "characterID", characterID, "error", err)
			}
		}
		if isShown {
			u.characterSkillQueue.Update()
		}
		if needsRefresh {
			u.trainingOverview.Update()
			u.notifyExpiredTrainingIfneeded(ctx, characterID)
		}
	case app.SectionWalletBalance:
		if needsRefresh {
			u.characterOverview.Update()
			u.wealthOverview.Update()
		}
	case app.SectionWalletJournal:
		if isShown && needsRefresh {
			u.characterWalletJournal.Update()
		}
	case app.SectionWalletTransactions:
		if isShown && needsRefresh {
			u.characterWalletTransaction.Update()
		}
	default:
		slog.Warn(fmt.Sprintf("section not part of the update ticker: %s", s))
	}
}

func (u *UIBase) notifyExpiredTrainingIfneeded(ctx context.Context, characerID int32) {
	if u.Settings().NotifyTrainingEnabled() {
		go func() {
			// TODO: earliest := calcNotifyEarliest(u.fyneApp.Preferences(), settingNotifyTrainingEarliest)
			if err := u.CharacterService().NotifyExpiredTraining(ctx, characerID, u.sendDesktopNotification); err != nil {
				slog.Error("notify expired training", "error", err)
			}
		}()
	}
}

func (u *UIBase) notifyExpiredExtractionsIfNeeded(ctx context.Context, characterID int32) {
	if u.Settings().NotifyPIEnabled() {
		go func() {
			earliest := u.Settings().NotifyPIEarliest()
			if err := u.CharacterService().NotifyExpiredExtractions(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
				slog.Error("notify expired extractions", "characterID", characterID, "error", err)
			}
		}()
	}
}

func (u *UIBase) AvailableUpdate() (github.VersionInfo, error) {
	current := u.app.Metadata().Version
	v, err := github.AvailableUpdate(githubOwner, githubRepo, current)
	if err != nil {
		return github.VersionInfo{}, err
	}
	return v, nil
}

func (u *UIBase) ShowInformationDialog(title, message string, parent fyne.Window) {
	d := dialog.NewInformation(title, message, parent)
	u.ModifyShortcutsForDialog(d, parent)
	d.Show()
}

func (u *UIBase) ShowConfirmDialog(title, message, confirm string, callback func(bool), parent fyne.Window) {
	d := dialog.NewConfirm(title, message, callback, parent)
	d.SetConfirmImportance(widget.DangerImportance)
	d.SetConfirmText(confirm)
	d.SetDismissText("Cancel")
	u.ModifyShortcutsForDialog(d, parent)
	d.Show()
}

func (u *UIBase) NewErrorDialog(message string, err error, parent fyne.Window) dialog.Dialog {
	text := widget.NewLabel(fmt.Sprintf("%s\n\n%s", message, u.ErrorDisplay(err)))
	text.Wrapping = fyne.TextWrapWord
	text.Importance = widget.DangerImportance
	c := container.NewVScroll(text)
	c.SetMinSize(fyne.Size{Width: 400, Height: 100})
	d := dialog.NewCustom("Error", "OK", c, parent)
	u.ModifyShortcutsForDialog(d, parent)
	return d
}

func (u *UIBase) ShowErrorDialog(message string, err error, parent fyne.Window) {
	d := u.NewErrorDialog(message, err, parent)
	d.Show()
}

// ModifyShortcutsForDialog modifies the shortcuts for a dialog.
func (u *UIBase) ModifyShortcutsForDialog(d dialog.Dialog, w fyne.Window) {
	kxdialog.AddDialogKeyHandler(d, w)
	if u.DisableMenuShortcuts != nil && u.EnableMenuShortcuts != nil {
		u.DisableMenuShortcuts()
		d.SetOnClosed(func() {
			u.EnableMenuShortcuts()
		})
	}
}

func (u *UIBase) ShowUpdateStatusWindow() {
	if u.statusWindow != nil {
		u.statusWindow.Show()
		return
	}
	w := u.app.NewWindow(u.MakeWindowTitle("Update Status"))
	a := toolui.NewUpdateStatus(u)
	a.Update()
	w.SetContent(a)
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

func (u *UIBase) ShowLocationInfoWindow(id int64) {
	iw := infowindow.New(u)
	iw.ShowLocation(id)
}

func (u *UIBase) ShowTypeInfoWindow(id int32) {
	iw := infowindow.New(u)
	iw.Show(app.EveEntityInventoryType, id)
}

func (u *UIBase) ShowEveEntityInfoWindow(o *app.EveEntity) {
	iw := infowindow.New(u)
	iw.ShowEveEntity(o)
}

func (u *UIBase) ShowInfoWindow(c app.EveEntityCategory, id int32) {
	iw := infowindow.New(u)
	iw.Show(c, id)
}

func (u *UIBase) ShowSnackbar(text string) {
	u.snackbar.Show(text)
}

func (u *UIBase) WebsiteRootURL() *url.URL {
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

func (u *UIBase) appName() string {
	info := u.app.Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
}

func (u *UIBase) makeAboutPage() fyne.CanvasObject {
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
			widget.NewHyperlink("Website", u.WebsiteRootURL()),
			widget.NewHyperlink("Downloads", u.WebsiteRootURL().JoinPath("releases")),
		),
		widget.NewLabel("\"EVE\", \"EVE Online\", \"CCP\", \nand all related logos and images \nare trademarks or registered trademarks of CCP hf."),
		widget.NewLabel("(c) 2024-25 Erik Kalkoken"),
	)
	return c
}
