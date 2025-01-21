package ui

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
)

// Base UI constants
const (
	DefaultIconSize = 64
	MyFloatFormat   = "#,###.##"
)

type BaseUI struct {
	CacheService       app.CacheService
	CharacterService   *character.CharacterService
	ESIStatusService   app.ESIStatusService
	EveImageService    app.EveImageService
	EveUniverseService *eveuniverse.EveUniverseService
	StatusCacheService app.StatusCacheService
	// Run the app in offline mode
	IsOffline bool
	// Whether to disable update tickers (useful for debugging)
	IsUpdateTickerDisabled bool

	OnSetCharacter func(int32)

	Character *app.Character
	FyneApp   fyne.App
	Window    fyne.Window

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
	ShipsArea             *ShipsArea
	SkillCatalogueArea    *SkillCatalogueArea
	SkillqueueArea        *SkillqueueArea
	TrainingArea          *TrainingArea
	WalletJournalArea     *WalletJournalArea
	WalletTransactionArea *WalletTransactionArea
	WealthArea            *WealthArea
}

func NewBaseUI(fyneApp fyne.App) *BaseUI {
	u := &BaseUI{
		FyneApp: fyneApp,
	}
	u.Window = fyneApp.NewWindow(u.AppName())

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
	u.ShipsArea = u.newShipArea()
	u.SkillCatalogueArea = u.NewSkillCatalogueArea()
	u.SkillqueueArea = u.NewSkillqueueArea()
	u.TrainingArea = u.NewTrainingArea()
	u.WalletJournalArea = u.NewWalletJournalArea()
	u.WalletTransactionArea = u.NewWalletTransactionArea()
	u.WealthArea = u.NewWealthArea()

	return u
}

func (u BaseUI) AppName() string {
	info := u.FyneApp.Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
}

func (u *BaseUI) Init() {
	u.AccountArea.Refresh()
	var c *app.Character
	var err error
	ctx := context.Background()
	if cID := u.FyneApp.Preferences().Int(SettingLastCharacterID); cID != 0 {
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
	u.Character = c
}

// ShowAndRun shows the UI and runs it (blocking).
func (u *BaseUI) ShowAndRun() {
	u.FyneApp.Lifecycle().SetOnStarted(func() {
		slog.Info("App started")

		if u.IsOffline {
			slog.Info("Started in offline mode")
		}
		if u.IsUpdateTickerDisabled {
			slog.Info("Update ticker disabled")
		}
		go func() {
			u.RefreshCrossPages()
			if u.HasCharacter() {
				u.SetCharacter(u.Character)
			} else {
				u.ResetCharacter()
			}
		}()
		if !u.IsOffline && !u.IsUpdateTickerDisabled {
			go func() {
				u.startUpdateTickerGeneralSections()
				u.startUpdateTickerCharacters()
			}()
		}
	})
	u.FyneApp.Lifecycle().SetOnStopped(func() {
		slog.Info("App shut down complete")
	})

	u.Window.ShowAndRun()
}

// CharacterID returns the ID of the current character or 0 if non it set.
func (u *BaseUI) CharacterID() int32 {
	if u.Character == nil {
		return 0
	}
	return u.Character.ID
}

func (u *BaseUI) CurrentCharacter() *app.Character {
	return u.Character
}

func (u *BaseUI) HasCharacter() bool {
	return u.Character != nil
}

func (u *BaseUI) LoadCharacter(ctx context.Context, characterID int32) error {
	c, err := u.CharacterService.GetCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	u.SetCharacter(c)
	return nil
}

func (u *BaseUI) SetCharacter(c *app.Character) {
	u.Character = c
	u.RefreshCharacter()
	u.FyneApp.Preferences().SetInt(SettingLastCharacterID, int(c.ID))
	if u.OnSetCharacter != nil {
		u.OnSetCharacter(c.ID)
	}
}

func (u *BaseUI) ResetCharacter() {
	u.Character = nil
	u.FyneApp.Preferences().SetInt(SettingLastCharacterID, 0)
	u.RefreshCharacter()
}

func (u *BaseUI) SetAnyCharacter() error {
	c, err := u.CharacterService.GetAnyCharacter(context.TODO())
	if errors.Is(err, character.ErrNotFound) {
		u.ResetCharacter()
		return nil
	} else if err != nil {
		return err
	}
	u.SetCharacter(c)
	return nil
}

func (u *BaseUI) RefreshCharacter() {
	ff := map[string]func(){
		"assets":         u.AssetsArea.Redraw,
		"attributes":     u.AttributesArea.Refresh,
		"bio":            u.BiographyArea.Refresh,
		"contracts":      u.ContractsArea.Refresh,
		"implants":       u.ImplantsArea.Refresh,
		"jumpClones":     u.JumpClonesArea.Redraw,
		"mail":           u.MailArea.Redraw,
		"notifications":  u.NotificationsArea.Refresh,
		"planets":        u.PlanetArea.Refresh,
		"ships":          u.ShipsArea.Refresh,
		"skillCatalogue": u.SkillCatalogueArea.Redraw,
		"skillqueue":     u.SkillqueueArea.Refresh,
		// "toolbar":           u.toolbarArea.refresh,
		"walletJournal":     u.WalletJournalArea.Refresh,
		"walletTransaction": u.WalletTransactionArea.Refresh,
	}
	c := u.CurrentCharacter()
	// ff["toogleTabs"] = func() {
	// 	u.toogleTabs(c != nil)
	// }
	if c != nil {
		slog.Debug("Refreshing character", "ID", c.EveCharacter.ID, "name", c.EveCharacter.Name)
	}
	runFunctionsWithProgressModal("Loading character", ff, u.Window)
	if c != nil && !u.IsUpdateTickerDisabled {
		u.UpdateCharacterAndRefreshIfNeeded(context.TODO(), c.ID, false)
	}
	// go u.statusBarArea.refreshUpdateStatus()
}

// RefreshCrossPages refreshed all pages under the characters tab.
func (u *BaseUI) RefreshCrossPages() {
	ff := map[string]func(){
		"assetSearch": u.AssetSearchArea.Refresh,
		"colony":      u.ColoniesArea.Refresh,
		"locations":   u.LocationsArea.Refresh,
		"overview":    u.OverviewArea.Refresh,
		// "statusBar":   u.statusBarArea.refreshCharacterCount,
		// "toolbar":     u.toolbarArea.refresh,
		"training": u.TrainingArea.Refresh,
		"wealth":   u.WealthArea.Refresh,
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

func (ui *BaseUI) ShowLocationInfoWindow(int64) {

}

type TypeWindowTab uint

const (
	DescriptionTab TypeWindowTab = iota + 1
	RequirementsTab
)

func (ui BaseUI) ShowTypeInfoWindow(typeID, characterID int32, selectTab TypeWindowTab) {

}
