// Package uiservices defines the services using by UI elements.
package uiservices

import (
	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatusservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/infowindow"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
)

type UIServices interface {
	Character() *characterservice.CharacterService
	Corporation() *corporationservice.CorporationService
	ESIStatus() *esistatusservice.ESIStatusService
	EVEImage() *eveimageservice.EveImageService
	EVEUniverse() *eveuniverseservice.EveUniverseService
	Janice() *janiceservice.JaniceService
	Settings() *settings.Settings
	Signals() *app.Signals
	StatusCache() *statuscacheservice.StatusCacheService

	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	InfoWindow() *infowindow.InfoWindow
	MainWindow() fyne.Window
}
