// Package characterui provides widgets for building the character UI.
package characterui

import (
	"context"

	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infowindow"
	"github.com/ErikKalkoken/evebuddy/internal/singleinstance"
)

type ui interface {
	Character() *characterservice.CharacterService
	Corporation() *corporationservice.CorporationService
	CurrentCharacter() *app.Character
	ErrorDisplay(err error) string
	EVEImage() app.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	InfoWindow() *infowindow.InfoWindow
	IsDeveloperMode() bool
	IsMobile() bool
	IsOffline() bool
	IsUpdateDisabled() bool
	LoadCharacter(ctx context.Context, id int64) error
	MainWindow() fyne.Window
	OnShowCharacterFunc() func()
	Settings() *settings.Settings
	ShowSnackbar(text string)
	Signals() *app.Signals
	SingleInstance() *singleinstance.Group
	StatusCache() *statuscache.StatusCache
	UpdateMailIndicator(ctx context.Context)
}
