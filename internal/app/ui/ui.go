package ui

import (
	"context"
	"errors"
	"log/slog"

	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
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

	Character *app.Character
	FyneApp   fyne.App
	Window    fyne.Window

	refreshCharacter func()
}

func NewBaseUI(fyneApp fyne.App, refreshCharacter func()) *BaseUI {
	u := &BaseUI{
		FyneApp:          fyneApp,
		refreshCharacter: refreshCharacter,
	}
	u.Window = fyneApp.NewWindow(u.AppName())
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
	u.refreshCharacter()
	u.FyneApp.Preferences().SetInt(SettingLastCharacterID, int(c.ID))
}

func (u *BaseUI) ResetCharacter() {
	u.Character = nil
	u.FyneApp.Preferences().SetInt(SettingLastCharacterID, 0)
	u.refreshCharacter()
}
