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
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infowindow"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	"github.com/ErikKalkoken/evebuddy/internal/github"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/set"
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
	notifyEarliestFallback        = 24 * time.Hour
)

// BaseUI represents the core UI logic and is used by both the desktop and mobile UI.
type BaseUI struct {
	// Clears all caches
	ClearCache func()

	DisableMenuShortcuts func()
	EnableMenuShortcuts  func()
	HideMailIndicator    func()
	OnAppFirstStarted    func()
	OnAppStopped         func()
	OnAppTerminated      func()
	OnInit               func(*app.Character)
	OnUpdateCharacter    func(*app.Character)
	OnRefreshCross       func()
	OnUpdateStatus       func()
	OnSetCharacter       func(int32)
	OnShowAndRun         func()
	ShowMailIndicator    func()

	ManagerCharacters          *ManageCharacters
	AllAssetSearch             *AllAssetSearch
	CharacterAssets            *CharacterAssets
	CharacterAttributes        *CharacterAttributes
	CharacterCommunications    *CharacterCommunications
	CharacterContracts         *CharacterContracts
	CharacterImplants          *CharacterImplants
	CharacterJumpClones        *CharacterJumpClones
	CharacterMail              *CharacterMail
	CharacterOverview          *CharacterOverview
	CharacterPlanets           *CharacterPlanets
	CharacterShips             *CharacterShips
	CharacterSkillCatalogue    *CharacterSkillCatalogue
	CharacterSkillQueue        *CharacterSkillQueue
	CharacterWalletJournal     *CharacterWalletJournal
	CharacterWalletTransaction *CharacterWalletTransaction
	ColonyOverview             *ColonyOverview
	LocationOverview           *LocationOverview
	GameSearch                 *GameSearch
	Settings                   *Settings
	TrainingOverview           *TrainingOverview
	WealthOverview             *WealthOverview

	app              fyne.App
	character        *app.Character
	cs               app.CharacterService
	dataPaths        map[string]string // Paths to user data
	eis              app.EveImageService
	ess              app.ESIStatusService
	eus              app.EveUniverseService
	infoWindow       infowindow.InfoWindow
	isForeground     atomic.Bool // whether the app is currently shown in the foreground
	isMobile         bool
	isOffline        bool // Run the app in offline mode
	isUpdateDisabled bool // Whether to disable update tickers (useful for debugging)
	memcache         app.CacheService
	scs              app.StatusCacheService
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
) *BaseUI {
	u := &BaseUI{
		dataPaths:        dataPaths,
		app:              app,
		cs:               cs,
		eis:              eis,
		ess:              ess,
		eus:              eus,
		isMobile:         fyne.CurrentDevice().IsMobile(),
		isOffline:        isOffline,
		isUpdateDisabled: isUpdateDisabled,
		memcache:         memCache,
		scs:              scs,
	}
	u.window = app.NewWindow(u.AppName())

	if u.IsDesktop() {
		iwidget.DefaultImageScaleMode = canvas.ImageScaleFastest
		appwidget.DefaultImageScaleMode = canvas.ImageScaleFastest
	}

	u.snackbar = iwidget.NewSnackbar(u.window)
	u.infoWindow = infowindow.New(u, u.window)

	u.ManagerCharacters = NewManageCharacters(u)
	u.CharacterAssets = NewCharacterAssets(u)
	u.AllAssetSearch = NewAssetSearch(u)
	u.CharacterAttributes = NewCharacterAttributes(u)
	u.ColonyOverview = NewColonies(u)
	u.CharacterContracts = NewCharacterContracts(u)
	u.CharacterImplants = NewCharacterImplants(u)
	u.CharacterJumpClones = NewCharacterJumpClones(u)
	u.LocationOverview = NewLocations(u)
	u.CharacterMail = NewCharacterMail(u)
	u.CharacterCommunications = NewCharacterCommunications(u)
	u.CharacterOverview = NewCharacterOverview(u)
	u.CharacterPlanets = NewCharacterPlanets(u)
	u.GameSearch = NewGameSearch(u)
	u.Settings = NewSettings(u)
	u.CharacterShips = NewCharacterShips(u)
	u.CharacterSkillCatalogue = NewCharacterSkillCatalogue(u)
	u.CharacterSkillQueue = NewCharacterSkillQueue(u)
	u.TrainingOverview = NewTrainingOverview(u)
	u.CharacterWalletJournal = NewCharacterWalletJournal(u)
	u.CharacterWalletTransaction = NewCharacterWalletTransaction(u)
	u.WealthOverview = NewWealthOverview(u)
	return u
}

func (u *BaseUI) ClearAllCaches() {
	if u.ClearCache != nil {
		u.ClearCache()
	}
}

func (u *BaseUI) App() fyne.App {
	return u.app
}

func (u *BaseUI) AppName() string {
	info := u.app.Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
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

func (u *BaseUI) DataPaths() map[string]string {
	return u.dataPaths
}

func (u *BaseUI) MainWindow() fyne.Window {
	return u.window
}

func (u *BaseUI) IsDeveloperMode() bool {
	return u.app.Preferences().Bool(settingDeveloperMode)
}

func (u *BaseUI) IsOffline() bool {
	return u.isOffline
}

// Init initialized the app.
// It is meant for initialization logic that requires all services to be initialized and available.
// It should be called directly after the app was created and before the Fyne loop is started.
func (u *BaseUI) Init() {
	u.ManagerCharacters.Refresh()
	var c *app.Character
	var err error
	ctx := context.Background()
	if cID := u.app.Preferences().Int(settingLastCharacterID); cID != 0 {
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
	if u.OnInit != nil {
		u.OnInit(c)
	}
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
	return fmt.Sprintf("%s - %s", subTitle, u.AppName())
}

// ShowAndRun shows the UI and runs the Fyne loop (blocking),
func (u *BaseUI) ShowAndRun() {
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
			u.RefreshCrossPages()
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
		go u.CharacterJumpClones.StartUpdateTicker()
		if u.OnAppFirstStarted != nil {
			u.OnAppFirstStarted()
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
		if u.OnAppStopped != nil {
			u.OnAppStopped()
		}
	})
	if u.OnShowAndRun != nil {
		u.OnShowAndRun()
	}
	u.window.ShowAndRun()
	slog.Info("App terminated")
	if u.OnAppTerminated != nil {
		u.OnAppTerminated()
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

func (u *BaseUI) LoadCharacter(id int32) error {
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

// UpdateStatus refreshed all status information pages.
func (u *BaseUI) UpdateStatus() {
	if u.OnUpdateStatus == nil {
		return
	}
	go u.OnUpdateStatus()
}

// updateCharacter updates all pages for the current character.
func (u *BaseUI) updateCharacter() {
	ff := map[string]func(){
		"assets":            u.CharacterAssets.Update,
		"attributes":        u.CharacterAttributes.Update,
		"contracts":         u.CharacterContracts.Update,
		"implants":          u.CharacterImplants.Update,
		"jumpClones":        u.CharacterJumpClones.Update,
		"mail":              u.CharacterMail.Update,
		"notifications":     u.CharacterCommunications.Update,
		"planets":           u.CharacterPlanets.Update,
		"ships":             u.CharacterShips.Update,
		"skillCatalogue":    u.CharacterSkillCatalogue.Update,
		"skillqueue":        u.CharacterSkillQueue.Update,
		"walletJournal":     u.CharacterWalletJournal.Update,
		"walletTransaction": u.CharacterWalletTransaction.Update,
	}
	c := u.CurrentCharacter()
	if c != nil {
		slog.Debug("Updating character", "ID", c.EveCharacter.ID, "name", c.EveCharacter.Name)
	} else {
		slog.Debug("Updating without character")
	}
	if u.OnUpdateCharacter != nil {
		ff["OnUpdateCharacter"] = func() {
			u.OnUpdateCharacter(c)
		}
	}
	runFunctionsWithProgressModal("Loading character", ff, u.window)
	if c != nil && !u.isUpdateDisabled {
		u.UpdateCharacterAndRefreshIfNeeded(context.Background(), c.ID, false)
	}
}

// RefreshCrossPages refreshed all pages that contain information about multiple characters.
func (u *BaseUI) RefreshCrossPages() {
	ff := map[string]func(){
		"assetSearch": u.AllAssetSearch.Update,
		"colony":      u.ColonyOverview.Update,
		"locations":   u.LocationOverview.Update,
		"overview":    u.CharacterOverview.Update,
		"training":    u.TrainingOverview.Update,
		"wealth":      u.WealthOverview.Update,
	}
	if u.OnRefreshCross != nil {
		ff["onRefreshCross"] = u.OnRefreshCross
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

func (u *BaseUI) resetCharacter() {
	u.character = nil
	u.app.Preferences().SetInt(settingLastCharacterID, 0)
	u.updateCharacter()
	u.UpdateStatus()
}

func (u *BaseUI) setCharacter(c *app.Character) {
	u.character = c
	u.app.Preferences().SetInt(settingLastCharacterID, int(c.ID))
	u.updateCharacter()
	u.UpdateStatus()
	if u.OnSetCharacter != nil {
		u.OnSetCharacter(c.ID)
	}
}

func (u *BaseUI) SetAnyCharacter() error {
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

func (u *BaseUI) UpdateAvatar(id int32, setIcon func(fyne.Resource)) {
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
	if !u.app.Preferences().BoolWithFallback(SettingSysTrayEnabled, SettingSysTrayEnabledDefault) {
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

func (u *BaseUI) MakeCharacterSwitchMenu(refresh func()) []*fyne.MenuItem {
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

func (u *BaseUI) sendDesktopNotification(title, content string) {
	u.app.SendNotification(fyne.NewNotification(title, content))
	slog.Info("desktop notification sent", "title", title, "content", content)
}

func (u *BaseUI) startUpdateTickerGeneralSections() {
	ticker := time.NewTicker(generalSectionsUpdateTicker)
	go func() {
		for {
			u.UpdateGeneralSectionsAndRefreshIfNeeded(false)
			<-ticker.C
		}
	}()
}

func (u *BaseUI) UpdateGeneralSectionsAndRefreshIfNeeded(forceUpdate bool) {
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

func (u *BaseUI) UpdateGeneralSectionAndRefreshIfNeeded(ctx context.Context, section app.GeneralSection, forceUpdate bool) {
	hasChanged, err := u.EveUniverseService().UpdateSection(ctx, section, forceUpdate)
	if err != nil {
		slog.Error("Failed to update general section", "section", section, "err", err)
		return
	}
	switch section {
	case app.SectionEveCategories:
		if hasChanged {
			u.CharacterShips.Update()
			u.CharacterSkillCatalogue.Refresh()
		}
	case app.SectionEveCharacters, app.SectionEveMarketPrices:
		// nothing to refresh
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
		go u.UpdateCharacterAndRefreshIfNeeded(ctx, c.ID, false)
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

// UpdateCharacterAndRefreshIfNeeded runs update for all sections of a character if needed
// and refreshes the UI accordingly.
func (u *BaseUI) UpdateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
	if u.isOffline {
		return
	}
	var sections []app.CharacterSection
	if u.IsMobile() && !u.isForeground.Load() {
		// only update what is needed for notifications on mobile when running in background to save battery
		if u.app.Preferences().BoolWithFallback(settingNotifyCommunicationsEnabled, settingNotifyCommunicationsEnabledDefault) {
			sections = append(sections, app.SectionNotifications)
		}
		if u.app.Preferences().BoolWithFallback(settingNotifyContractsEnabled, settingNotifyContractsEnabledDefault) {
			sections = append(sections, app.SectionContracts)
		}
		if u.app.Preferences().BoolWithFallback(settingNotifyMailsEnabled, settingNotifyMailsEnabledDefault) {
			sections = append(sections, app.SectionMailLabels)
			sections = append(sections, app.SectionMailLists)
			sections = append(sections, app.SectionMails)
		}
		if u.app.Preferences().BoolWithFallback(settingNotifyPIEnabled, settingNotifyPIEnabledDefault) {
			sections = append(sections, app.SectionPlanets)
		}
		if u.app.Preferences().BoolWithFallback(settingNotifyTrainingEnabled, settingNotifyTrainingEnabledDefault) {
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
func (u *BaseUI) UpdateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, s app.CharacterSection, forceUpdate bool) {
	hasChanged, err := u.CharacterService().UpdateSectionIfNeeded(
		ctx, app.CharacterUpdateSectionParams{
			CharacterID:           characterID,
			Section:               s,
			ForceUpdate:           forceUpdate,
			MaxMails:              u.app.Preferences().IntWithFallback(settingMaxMails, settingMaxMailsDefault),
			MaxWalletTransactions: u.app.Preferences().IntWithFallback(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault),
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
			u.AllAssetSearch.Update()
			u.WealthOverview.Update()
		}
		if isShown && needsRefresh {
			u.CharacterAssets.Update()
		}
	case app.SectionAttributes:
		if isShown && needsRefresh {
			u.CharacterAttributes.Update()
		}
	case app.SectionContracts:
		if isShown && needsRefresh {
			u.CharacterContracts.Update()
		}
		if u.app.Preferences().BoolWithFallback(settingNotifyContractsEnabled, settingNotifyCommunicationsEnabledDefault) {
			go func() {
				earliest := calcNotifyEarliest(u.app.Preferences(), settingNotifyContractsEarliest)
				if err := u.CharacterService().NotifyUpdatedContracts(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
					slog.Error("notify contract update", "error", err)
				}
			}()
		}
	case app.SectionImplants:
		if isShown && needsRefresh {
			u.CharacterImplants.Update()
		}
	case app.SectionJumpClones:
		if isShown && needsRefresh {
			u.CharacterJumpClones.Update()
		}
		if needsRefresh {
			u.CharacterOverview.Update()
		}
	case app.SectionLocation,
		app.SectionOnline,
		app.SectionShip:
		if needsRefresh {
			u.LocationOverview.Update()
		}
	case app.SectionPlanets:
		if isShown && needsRefresh {
			u.CharacterPlanets.Update()
		}
		if needsRefresh {
			u.ColonyOverview.Update()
			u.notifyExpiredExtractionsIfNeeded(ctx, characterID)
		}
	case app.SectionMailLabels,
		app.SectionMailLists:
		if isShown && needsRefresh {
			u.CharacterMail.Update()
		}
		if needsRefresh {
			u.CharacterOverview.Update()
		}
	case app.SectionMails:
		if isShown && needsRefresh {
			u.CharacterMail.Update()
		}
		if needsRefresh {
			go u.CharacterOverview.Update()
			go u.UpdateMailIndicator()
		}
		if u.app.Preferences().BoolWithFallback(settingNotifyMailsEnabled, settingNotifyMailsEnabledDefault) {
			go func() {
				earliest := calcNotifyEarliest(u.app.Preferences(), settingNotifyMailsEarliest)
				if err := u.CharacterService().NotifyMails(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
					slog.Error("notify mails", "characterID", characterID, "error", err)
				}
			}()
		}
	case app.SectionNotifications:
		if isShown && needsRefresh {
			u.CharacterCommunications.Update()
		}
		if u.app.Preferences().BoolWithFallback(settingNotifyCommunicationsEnabled, settingNotifyCommunicationsEnabledDefault) {
			go func() {
				earliest := calcNotifyEarliest(u.app.Preferences(), settingNotifyCommunicationsEarliest)
				typesEnabled := set.NewFromSlice(u.app.Preferences().StringList(settingNotificationsTypesEnabled))
				if err := u.CharacterService().NotifyCommunications(ctx, characterID, earliest, typesEnabled, u.sendDesktopNotification); err != nil {
					slog.Error("notify communications", "characterID", characterID, "error", err)
				}
			}()
		}
	case app.SectionSkills:
		if isShown && needsRefresh {
			u.CharacterSkillCatalogue.Refresh()
			u.CharacterShips.Update()
			u.CharacterPlanets.Update()
		}
		if needsRefresh {
			u.TrainingOverview.Update()
		}
	case app.SectionSkillqueue:
		if u.app.Preferences().BoolWithFallback(settingNotifyTrainingEnabled, settingNotifyTrainingEnabledDefault) {
			err := u.CharacterService().EnableTrainingWatcher(ctx, characterID)
			if err != nil {
				slog.Error("Failed to enable training watcher", "characterID", characterID, "error", err)
			}
		}
		if isShown {
			u.CharacterSkillQueue.Update()
		}
		if needsRefresh {
			u.TrainingOverview.Update()
			u.notifyExpiredTrainingIfneeded(ctx, characterID)
		}
	case app.SectionWalletBalance:
		if needsRefresh {
			u.CharacterOverview.Update()
			u.WealthOverview.Update()
		}
	case app.SectionWalletJournal:
		if isShown && needsRefresh {
			u.CharacterWalletJournal.Update()
		}
	case app.SectionWalletTransactions:
		if isShown && needsRefresh {
			u.CharacterWalletTransaction.Update()
		}
	default:
		slog.Warn(fmt.Sprintf("section not part of the update ticker: %s", s))
	}
}

// calcNotifyEarliest returns the earliest time for a class of notifications.
// Might return a zero time in some circumstances.
func calcNotifyEarliest(pref fyne.Preferences, settingEarliest string) time.Time {
	earliest, err := time.Parse(time.RFC3339, pref.String(settingEarliest))
	if err != nil {
		// Recording the earliest when enabling a switch was added later for mails and communications
		// This workaround avoids a potential notification spam from older items.
		earliest = time.Now().UTC().Add(-notifyEarliestFallback)
		pref.SetString(settingEarliest, earliest.Format(time.RFC3339))
	}
	timeoutDays := pref.IntWithFallback(settingNotifyTimeoutHours, settingNotifyTimeoutHoursDefault)
	var timeout time.Time
	if timeoutDays > 0 {
		timeout = time.Now().UTC().Add(-time.Duration(timeoutDays) * time.Hour)
	}
	if earliest.After(timeout) {
		return earliest
	}
	return timeout
}

func (u *BaseUI) notifyExpiredTrainingIfneeded(ctx context.Context, characerID int32) {
	if u.app.Preferences().BoolWithFallback(settingNotifyTrainingEnabled, settingNotifyTrainingEnabledDefault) {
		go func() {
			// earliest := calcNotifyEarliest(u.fyneApp.Preferences(), settingNotifyTrainingEarliest)
			if err := u.CharacterService().NotifyExpiredTraining(ctx, characerID, u.sendDesktopNotification); err != nil {
				slog.Error("notify expired training", "error", err)
			}
		}()
	}
}

func (u *BaseUI) notifyExpiredExtractionsIfNeeded(ctx context.Context, characterID int32) {
	if u.app.Preferences().BoolWithFallback(settingNotifyPIEnabled, settingNotifyPIEnabledDefault) {
		go func() {
			earliest := calcNotifyEarliest(u.app.Preferences(), settingNotifyPIEarliest)
			if err := u.CharacterService().NotifyExpiredExtractions(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
				slog.Error("notify expired extractions", "characterID", characterID, "error", err)
			}
		}()
	}
}

func (u *BaseUI) MakeAboutPage() fyne.CanvasObject {
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

func (u *BaseUI) AvailableUpdate() (github.VersionInfo, error) {
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
	text := widget.NewLabel(fmt.Sprintf("%s\n\n%s", message, humanize.Error(err)))
	text.Wrapping = fyne.TextWrapWord
	text.Importance = widget.DangerImportance
	x := container.NewVScroll(text)
	x.SetMinSize(fyne.Size{Width: 400, Height: 100})
	d := dialog.NewCustom("Error", "OK", x, parent)
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

func (u *BaseUI) ShowUpdateStatusWindow() {
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
	u.infoWindow.ShowLocation(id)
}

func (u *BaseUI) ShowTypeInfoWindow(id int32) {
	u.infoWindow.Show(app.EveEntityInventoryType, id)
}

func (u *BaseUI) ShowEveEntityInfoWindow(o *app.EveEntity) {
	u.infoWindow.ShowEveEntity(o)
}

func (u *BaseUI) ShowInfoWindow(c app.EveEntityCategory, id int32) {
	u.infoWindow.Show(c, id)
}

func (u *BaseUI) ShowSnackbar(text string) {
	u.snackbar.Show(text)
}

func (u *BaseUI) WebsiteRootURL() *url.URL {
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
