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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infowindow"
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

// BaseUI represents the core UI logic and is used by both the desktop and mobile UI.
type BaseUI struct {
	DisableMenuShortcuts func()
	EnableMenuShortcuts  func()
	HideMailIndicator    func()
	ShowMailIndicator    func()

	onAppFirstStarted    func()
	onAppStopped         func()
	onAppTerminated      func()
	onInit               func(*app.Character)
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
	contractsActive            *Contracts
	contractsAll               *Contracts
	gameSearch                 *GameSearch
	industryJobsActive         *IndustryJobs
	industryJobsAll            *IndustryJobs
	manageCharacters           *ManageCharacters
	overviewAssets             *OverviewAssets
	overviewCharacters         *OverviewCharacters
	overviewClones             *OverviewClones
	colonies                   *Colonies
	overviewLocations          *OverviewLocations
	overviewTraining           *OverviewTraining
	overviewWealth             *OverviewWealth
	userSettings               *UserSettings

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

type BaseUIParams struct {
	App                fyne.App
	CharacterService   app.CharacterService
	ESIStatusService   app.ESIStatusService
	EveImageService    app.EveImageService
	EveUniverseService app.EveUniverseService
	MemCache           app.CacheService
	StatusCacheService app.StatusCacheService
	// optional
	ClearCacheFunc   func()
	IsOffline        bool
	IsUpdateDisabled bool
	DataPaths        map[string]string
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
		isMobile:         fyne.CurrentDevice().IsMobile(),
		isOffline:        args.IsOffline,
		isUpdateDisabled: args.IsUpdateDisabled,
		memcache:         args.MemCache,
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

	if u.IsDesktop() {
		iwidget.DefaultImageScaleMode = canvas.ImageScaleFastest
		appwidget.DefaultImageScaleMode = canvas.ImageScaleFastest
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
	u.contractsActive = NewContracts(u)
	u.contractsActive.ShowActiveOnly = true
	u.contractsAll = NewContracts(u)
	u.gameSearch = NewGameSearch(u)
	u.industryJobsActive = NewIndustryJobs(u)
	u.industryJobsActive.ShowActiveOnly = true
	u.industryJobsAll = NewIndustryJobs(u)
	u.manageCharacters = NewManageCharacters(u)
	u.overviewAssets = NewOverviewAssets(u)
	u.overviewCharacters = NewOverviewCharacters(u)
	u.overviewClones = NewOverviewClones(u)
	u.colonies = NewColonies(u)
	u.overviewLocations = NewOverviewLocations(u)
	u.overviewTraining = NewOverviewTraining(u)
	u.overviewWealth = NewOverviewWealth(u)
	u.snackbar = iwidget.NewSnackbar(u.window)
	u.userSettings = NewSettings(u)
	u.MainWindow().SetMaster()

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
			u.updateStatus()
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
		u.updateGeneralSectionsAndRefreshIfNeeded(false)
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

func (u *BaseUI) App() fyne.App {
	return u.app
}

func (u *BaseUI) ClearAllCaches() {
	u.clearCache()
}

func (u *BaseUI) CharacterService() app.CharacterService {
	return u.cs
}

func (u *BaseUI) ESIStatusService() app.ESIStatusService {
	return u.ess
}

func (u *BaseUI) EveImageService() app.EveImageService {
	return u.eis
}

func (u *BaseUI) EveUniverseService() app.EveUniverseService {
	return u.eus
}

func (u *BaseUI) MemCache() app.CacheService {
	return u.memcache
}

func (u *BaseUI) StatusCacheService() app.StatusCacheService {
	return u.scs
}

func (u *BaseUI) MainWindow() fyne.Window {
	return u.window
}

func (u *BaseUI) IsDeveloperMode() bool {
	return u.Settings().DeveloperMode()
}

func (u *BaseUI) IsOffline() bool {
	return u.isOffline
}

// Init initialized the app.
// It is meant for initialization logic that requires the UI to be fully created.
// It should be called directly after the UI was created and before the Fyne loop is started.
func (u *BaseUI) Init() {
	u.manageCharacters.Refresh()
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
func (u *BaseUI) ErrorDisplay(err error) string {
	if u.Settings().DeveloperMode() {
		return err.Error()
	}
	return err.Error()
	// return ihumanize.Error(err) TODO: Re-enable again when app is stable enough
}

func (u *BaseUI) IsDesktop() bool {
	_, ok := u.app.(desktop.App)
	return ok
}

func (u *BaseUI) IsMobile() bool {
	return u.isMobile
}

func (u *BaseUI) MakeWindowTitle(subTitle string) string {
	if u.IsMobile() {
		return subTitle
	}
	return fmt.Sprintf("%s - %s", subTitle, u.appName())
}

func (u *BaseUI) Settings() app.Settings {
	return u.settings
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

// CurrentCharacterID returns the ID of the current character or 0 if non it set.
func (u *BaseUI) CurrentCharacterID() int32 {
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

func (u *BaseUI) loadCharacter(id int32) error {
	c, err := u.CharacterService().GetCharacter(context.Background(), id)
	if err != nil {
		return fmt.Errorf("load character ID %d: %w", id, err)
	}
	u.setCharacter(c)
	return nil
}

// reloadCurrentCharacter reloads the current character from storage.
func (u *BaseUI) reloadCurrentCharacter() {
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

// updateStatus refreshed all status information pages.
func (u *BaseUI) updateStatus() {
	if u.onUpdateStatus == nil {
		return
	}
	go u.onUpdateStatus()
}

// updateCharacter updates all pages for the current character.
func (u *BaseUI) updateCharacter() {
	c := u.CurrentCharacter()
	if c != nil {
		slog.Debug("Updating character", "ID", c.EveCharacter.ID, "name", c.EveCharacter.Name)
	} else {
		slog.Debug("Updating without character")
	}
	ff := u.updateCharacterMap()
	if u.onUpdateCharacter != nil {
		ff["OnUpdateCharacter"] = func() {
			u.onUpdateCharacter(c)
		}
	}
	runFunctionsWithProgressModal("Loading character", ff, u.window)
	if c != nil && !u.isUpdateDisabled {
		u.updateCharacterAndRefreshIfNeeded(context.Background(), c.ID, false)
	}
}

func (u *BaseUI) updateCharacterMap() map[string]func() {
	ff := map[string]func(){
		"assets":            u.characterAsset.Update,
		"attributes":        u.characterAttributes.Update,
		"biography":         u.characterBiography.Update,
		"implants":          u.characterImplants.Update,
		"jumpClones":        u.characterJumpClones.Update,
		"mail":              u.characterMail.Update,
		"notifications":     u.characterCommunications.Update,
		"sheet":             u.characterSheet.Update,
		"ships":             u.characterShips.Update,
		"skillCatalogue":    u.characterSkillCatalogue.Update,
		"skillqueue":        u.characterSkillQueue.Update,
		"walletJournal":     u.characterWalletJournal.Update,
		"walletTransaction": u.characterWalletTransaction.Update,
	}
	return ff
}

// UpdateCrossPages refreshed all pages that contain information about multiple characters.
func (u *BaseUI) UpdateCrossPages() {
	runFunctionsWithProgressModal("Updating characters", u.updateCrossPagesMap(), u.window)
}

func (u *BaseUI) updateCrossPagesMap() map[string]func() {
	ff := map[string]func(){
		"assetSearch":       u.overviewAssets.Update,
		"contractsAll":      u.contractsAll.Update,
		"contractsActive":   u.contractsActive.Update,
		"cloneSeach":        u.overviewClones.Update,
		"colony":            u.colonies.Update,
		"industryJobAll":    u.industryJobsAll.Update,
		"industryJobActive": u.industryJobsActive.Update,
		"locations":         u.overviewLocations.Update,
		"overview":          u.overviewCharacters.Update,
		"training":          u.overviewTraining.Update,
		"wealth":            u.overviewWealth.Update,
	}
	if u.onRefreshCross != nil {
		ff["onRefreshCross"] = u.onRefreshCross
	}
	return ff
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

func (u *BaseUI) resetCharacter() {
	u.character = nil
	u.Settings().ResetLastCharacterID()
	u.updateCharacter()
	u.updateStatus()
}

func (u *BaseUI) setCharacter(c *app.Character) {
	u.character = c
	u.Settings().SetLastCharacterID(c.ID)
	u.updateCharacter()
	u.updateStatus()
	if u.onSetCharacter != nil {
		u.onSetCharacter(c.ID)
	}
}

func (u *BaseUI) setAnyCharacter() error {
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

func (u *BaseUI) updateAvatar(id int32, setIcon func(fyne.Resource)) {
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

func (u *BaseUI) UpdateMailIndicator() {
	if u.ShowMailIndicator == nil || u.HideMailIndicator == nil {
		return
	}
	if !u.Settings().SysTrayEnabled() {
		return
	}
	n, err := u.CharacterService().GetAllMailUnreadCount(context.Background())
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

func (u *BaseUI) makeCharacterSwitchMenu(refresh func()) []*fyne.MenuItem {
	cc := u.StatusCacheService().ListCharacters()
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
	currentID := u.CurrentCharacterID()
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
				it.Icon = r
			})
		}
		items = append(items, it)
	}
	go func() {
		wg.Wait()
		refresh()
	}()
	return items
}

func (u *BaseUI) sendDesktopNotification(title, content string) {
	u.app.SendNotification(fyne.NewNotification(title, content))
	slog.Info("desktop notification sent", "title", title, "content", content)
}

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
	if !forceUpdate && u.IsMobile() && !u.isForeground.Load() {
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
	hasChanged, err := u.EveUniverseService().UpdateSection(ctx, section, forceUpdate)
	if err != nil {
		slog.Error("Failed to update general section", "section", section, "err", err)
		return
	}
	needsRefresh := hasChanged || forceUpdate
	switch section {
	case app.SectionEveCategories:
		if needsRefresh {
			u.characterShips.Update()
			u.characterSkillCatalogue.Refresh()
		}
	case app.SectionEveCharacters:
		if needsRefresh {
			u.reloadCurrentCharacter()
			u.overviewCharacters.Update()
		}
	case app.SectionEveMarketPrices:
		u.characterAsset.Update()
		u.overviewCharacters.Update()
		u.overviewAssets.Update()
		u.reloadCurrentCharacter()
	default:
		slog.Warn(fmt.Sprintf("section not part of the update ticker refresh: %s", section))
	}
}

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
	cc, err := u.CharacterService().ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	for _, c := range cc {
		go u.updateCharacterAndRefreshIfNeeded(ctx, c.ID, false)
	}
	slog.Debug("started update status characters") // FIXME: Reset to DEBUG
	return nil
}

func (u *BaseUI) notifyCharactersIfNeeded(ctx context.Context) error {
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

// updateCharacterAndRefreshIfNeeded runs update for all sections of a character if needed
// and refreshes the UI accordingly.
func (u *BaseUI) updateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
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
		go u.updateCharacterSectionAndRefreshIfNeeded(ctx, characterID, s, forceUpdate)
	}
}

// updateCharacterSectionAndRefreshIfNeeded runs update for a character section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on character sections needs to be included
// to make sure they are refreshed when data changes.
func (u *BaseUI) updateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, s app.CharacterSection, forceUpdate bool) {
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
	switch s {
	case app.SectionAssets:
		if needsRefresh {
			u.overviewAssets.Update()
			u.overviewWealth.Update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterAsset.Update()
				u.characterSheet.Update()
			}
		}
	case app.SectionAttributes:
		if isShown && needsRefresh {
			u.characterAttributes.Update()
		}
	case app.SectionContracts:
		if needsRefresh {
			u.contractsActive.Update()
			u.contractsAll.Update()
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
		if needsRefresh {
			u.overviewCharacters.Update()
			u.overviewClones.Update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterJumpClones.Update()
			}
		}
	case app.SectionIndustryJobs:
		if needsRefresh {
			u.industryJobsAll.Update()
			u.industryJobsActive.Update()
		}
	case app.SectionLocation, app.SectionOnline, app.SectionShip:
		if needsRefresh {
			u.overviewLocations.Update()
			if isShown {
				u.reloadCurrentCharacter()
			}
		}
	case app.SectionPlanets:
		if needsRefresh {
			u.colonies.Update()
			u.notifyExpiredExtractionsIfNeeded(ctx, characterID)
		}
	case app.SectionMailLabels, app.SectionMailLists:
		if needsRefresh {
			u.overviewCharacters.Update()
			if isShown {
				u.characterMail.Update()
			}
		}
	case app.SectionMails:
		if needsRefresh {
			go u.overviewCharacters.Update()
			go u.UpdateMailIndicator()
			if isShown {
				u.characterMail.Update()
			}
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
		if needsRefresh {
			u.overviewTraining.Update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterSkillCatalogue.Refresh()
				u.characterShips.Update()
			}
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
			u.overviewTraining.Update()
			u.notifyExpiredTrainingIfneeded(ctx, characterID)
		}
	case app.SectionWalletBalance:
		if needsRefresh {
			u.overviewCharacters.Update()
			u.overviewWealth.Update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterAsset.Update()
			}
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

func (u *BaseUI) notifyExpiredTrainingIfneeded(ctx context.Context, characerID int32) {
	if u.Settings().NotifyTrainingEnabled() {
		go func() {
			// TODO: earliest := calcNotifyEarliest(u.fyneApp.Preferences(), settingNotifyTrainingEarliest)
			if err := u.CharacterService().NotifyExpiredTraining(ctx, characerID, u.sendDesktopNotification); err != nil {
				slog.Error("notify expired training", "error", err)
			}
		}()
	}
}

func (u *BaseUI) notifyExpiredExtractionsIfNeeded(ctx context.Context, characterID int32) {
	if u.Settings().NotifyPIEnabled() {
		go func() {
			earliest := u.Settings().NotifyPIEarliest()
			if err := u.CharacterService().NotifyExpiredExtractions(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
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
	text := widget.NewLabel(fmt.Sprintf("%s\n\n%s", message, u.ErrorDisplay(err)))
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
	a := NewUpdateStatus(u)
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

func (u *BaseUI) ShowLocationInfoWindow(id int64) {
	iw := infowindow.New(u)
	iw.ShowLocation(id)
}

func (u *BaseUI) ShowRaceInfoWindow(id int32) {
	iw := infowindow.New(u)
	iw.ShowRace(id)
}

func (u *BaseUI) ShowTypeInfoWindow(id int32) {
	iw := infowindow.New(u)
	iw.Show(app.EveEntityInventoryType, id)
}

func (u *BaseUI) ShowEveEntityInfoWindow(o *app.EveEntity) {
	iw := infowindow.New(u)
	iw.ShowEveEntity(o)
}

func (u *BaseUI) ShowInfoWindow(c app.EveEntityCategory, id int32) {
	iw := infowindow.New(u)
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

func (u *BaseUI) makeCopyToClipbardLabel(text string) *kxwidget.TappableLabel {
	return kxwidget.NewTappableLabel(text, func() {
		u.MainWindow().Clipboard().SetContent(text)
	})
}
