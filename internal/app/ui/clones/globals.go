// Package clones provides widgets for building clone related UIs.
package clones

import (
	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

type baseUI interface {
	Character() *characterservice.CharacterService
	ErrorDisplay(err error) string
	EVEImage() ui.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	InfoViewer() ui.InfoViewer
	IsDeveloperMode() bool
	IsMobile() bool
	MainWindow() fyne.Window
	ShowSnackbar(text string)
	Signals() *app.Signals
	StatusCache() *statuscache.StatusCache
}

type loadFuncAsync func(int64, int, func(fyne.Resource))
