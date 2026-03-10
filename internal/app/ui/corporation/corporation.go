// Package corporation provides the widgets for building the corporation UI.
package corporation

import (
	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infowindow"
)

type ui interface {
	Character() *characterservice.CharacterService
	Corporation() *corporationservice.CorporationService
	ErrorDisplay(err error) string
	EVEImage() app.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	InfoWindow() *infowindow.InfoWindow
	IsDeveloperMode() bool
	IsMobile() bool
	MainWindow() fyne.Window
	Signals() *app.Signals
}
