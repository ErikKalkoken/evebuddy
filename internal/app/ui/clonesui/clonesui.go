// Package clonesui provides widgets for building clone related UIs.
package clonesui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infowindow"
)

type ui interface {
	Character() *characterservice.CharacterService
	ErrorDisplay(err error) string
	EVEImage() app.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	InfoWindow() *infowindow.InfoWindow
	IsDeveloperMode() bool
	IsMobile() bool
	MainWindow() fyne.Window
	ShowSnackbar(text string)
	Signals() *app.Signals
	StatusCache() *statuscache.StatusCache
}

type loadFuncAsync func(int64, int, func(fyne.Resource))

// TODO: Remove this helper

// makeTopText makes the content for the top label of a gui element.
func makeTopText(characterID int64, hasData bool, err error, create func() (string, widget.Importance)) (string, widget.Importance) {
	if err != nil {
		return "ERROR: " + app.ErrorDisplay(err), widget.DangerImportance
	}
	if characterID == 0 {
		return "No entity", widget.LowImportance
	}
	if !hasData {
		return "No data", widget.WarningImportance
	}
	if create == nil {
		return "", widget.MediumImportance
	}
	return create()
}
