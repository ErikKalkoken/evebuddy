// Package characters provides widgets for building the character UI.
package characters

import (
	"context"

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
	ErrorDisplay(err error) string
	EVEImage() ui.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	InfoViewer() ui.InfoViewer
	IsDeveloperMode() bool
	IsMobile() bool
	IsOffline() bool
	IsUpdateDisabled() bool
	MainWindow() fyne.Window
	MakeWindowTitle(parts ...string) string
	Settings() *settings.Settings
	ShowCharacter(ctx context.Context, characterID int64)
	ShowSnackbar(text string)
	Signals() *app.Signals
	UpdateMailIndicator(ctx context.Context)
}

type loadFuncAsync func(int64, int, func(fyne.Resource))
