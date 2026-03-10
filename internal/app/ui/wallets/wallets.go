// Package wallets provides widgets for building wallet UIs.
package wallets

import (
	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

type baseUI interface {
	Character() *characterservice.CharacterService
	Corporation() *corporationservice.CorporationService
	ErrorDisplay(err error) string
	EVEImage() ui.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	InfoWindow() ui.InfoWindow
	IsDeveloperMode() bool
	IsMobile() bool
	MainWindow() fyne.Window
	Signals() *app.Signals
	StatusCache() *statuscache.StatusCache
}
