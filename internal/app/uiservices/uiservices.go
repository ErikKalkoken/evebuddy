// Package uiservices defines a shared interface for UI services.
package uiservices

import (
	"context"

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
	"github.com/ErikKalkoken/evebuddy/internal/singleinstance"
)

type UIServices interface {
	Character() *characterservice.CharacterService
	Corporation() *corporationservice.CorporationService
	ESIStatus() *esistatusservice.ESIStatusService
	EVEImage() *eveimageservice.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	Janice() *janiceservice.JaniceService
	Settings() *settings.Settings
	Signals() *app.Signals
	StatusCache() *statuscacheservice.StatusCacheService

	CurrentCharacterID() int64
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	GetOrCreateWindowWithOnClosed(id string, titles ...string) (window fyne.Window, created bool, onClosed func())
	InfoWindow() *infowindow.InfoWindow
	IsUpdateDisabled() bool
	LoadCharacter(id int64) error
	MainWindow() fyne.Window
	OnShowCharacterFunc() func()
	ShowSnackbar(text string)
	SingleInstance() *singleinstance.Group
	UpdateMailIndicator(ctx context.Context)
}
