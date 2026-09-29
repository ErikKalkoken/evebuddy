// Package screens provides UI components to build the various screens
// for character and corporation features
package screens

import (
	"context"
	"image/color"

	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

type baseUI interface {
	Character() *characterservice.CharacterService
	Corporation() *corporationservice.CorporationService
	DestroyWindow(id string) bool
	DisplaySnackbar(text string)
	ErrorDisplay(err error) string
	EVEImage() ui.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	GetOrCreateWindowWithOnClosed(id string, titles ...string) (window fyne.Window, created bool, onClosed func())
	InfoViewer() ui.InfoViewer
	IsDeveloperMode() bool
	IsMobile() bool
	IsOffline() bool
	IsUpdateDisabled() bool
	MainWindow() fyne.Window
	MakeWindowTitle(parts ...string) string
	SetCharacterAvatarAsync(characterID int64, setIcon func(fyne.Resource))
	Settings() *settings.Settings
	ShowCharacter(ctx context.Context, characterID int64)
	ShowCorporation(ctx context.Context, corporationID int64)
	Signals() *app.Signals
	UpdateMailIndicator(ctx context.Context)
}

type loadFuncAsync func(int64, int, func(fyne.Resource))

// Fyne color for dark background
var colorDarkBackground = color.NRGBA{R: 0x17, G: 0x17, B: 0x18, A: 0xff}
