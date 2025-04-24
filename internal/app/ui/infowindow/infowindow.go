package infowindow

import (
	"log/slog"
	"maps"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	renderIconPixelSize = 256
	renderIconUnitSize  = 128
	logoZoomFactor      = 1.3
	zoomImagePixelSize  = 512
	infoWindowWidth     = 600
	infoWindowHeight    = 600
)

type UI interface {
	App() fyne.App
	CharacterService() app.CharacterService
	CurrentCharacterID() int32
	ErrorDisplay(err error) string
	EveImageService() app.EveImageService
	EveUniverseService() app.EveUniverseService
	IsDeveloperMode() bool
	IsOffline() bool
	MainWindow() fyne.Window
	MakeWindowTitle(subTitle string) string
	ShowErrorDialog(message string, err error, parent fyne.Window)
	ShowInformationDialog(title, message string, parent fyne.Window)
}

// InfoWindow represents a dedicated window for showing information similar to the in-game info windows.
type InfoWindow struct {
	u   UI
	w   fyne.Window
	nav *iwidget.Navigator
}

// New returns a configured InfoWindow.
func New(u UI) *InfoWindow {
	iw := &InfoWindow{
		u: u,
		w: u.MainWindow(),
	}
	return iw
}

// Show shows a new info window for an EveEntity.
func (iw *InfoWindow) ShowEveEntity(ee *app.EveEntity) {
	iw.show(eveEntity2InfoVariant(ee), int64(ee.ID))
}

// Show shows a new info window for an EveEntity.
func (iw *InfoWindow) Show(c app.EveEntityCategory, id int32) {
	iw.show(eveEntity2InfoVariant(&app.EveEntity{Category: c}), int64(id))
}

func (iw *InfoWindow) ShowLocation(id int64) {
	iw.show(infoLocation, id)
}

func (iw *InfoWindow) ShowRace(id int32) {
	iw.show(infoRace, int64(id))
}

func (iw *InfoWindow) show(t infoVariant, id int64) {
	if iw.u.IsOffline() {
		iw.u.ShowInformationDialog(
			"Offline",
			"Can't show info window when offline",
			iw.w,
		)
		return
	}
	var title string
	var page fyne.CanvasObject
	switch t {
	case infoAlliance:
		title = "Alliance"
		page = newAllianceInfo(iw, int32(id))
	case infoCharacter:
		title = "Character"
		page = newCharacterInfo(iw, int32(id))
	case infoConstellation:
		title = "Constellation"
		page = newConstellationInfo(iw, int32(id))
	case infoCorporation:
		title = "Corporation"
		page = newCorporationInfo(iw, int32(id))
	case infoInventoryType:
		// TODO: Restructure, so that window is first drawn empty and content loaded in background (as other info windo)
		a, err := NewInventoryTypeInfo(iw, int32(id), iw.u.CurrentCharacterID())
		if err != nil {
			iw.u.ShowInformationDialog("ERROR", "Something whent wrong when trying to show info for type", iw.w)
			slog.Error("show type", "error", err)
			return
		}
		title = a.title()
		page = a
	case infoRace:
		title = "Race"
		page = newRaceInfo(iw, int32(id))
	case infoRegion:
		title = "Region"
		page = newRegionInfo(iw, int32(id))
	case infoSolarSystem:
		title = "Solar System"
		page = newSolarSystemInfo(iw, int32(id))
	case infoLocation:
		title = "Location"
		page = newLocationInfo(iw, id)
	default:
		iw.u.ShowInformationDialog(
			"Warning",
			"Can't show info window for unknown category",
			iw.w,
		)
		return
	}
	ab := iwidget.NewAppBar(title+": Information", page)
	if iw.nav == nil {
		w := iw.u.App().NewWindow(iw.u.MakeWindowTitle("Information"))
		iw.w = w
		iw.nav = iwidget.NewNavigatorWithAppBar(ab)
		w.SetContent(iw.nav)
		w.Resize(fyne.NewSize(infoWindowWidth, infoWindowHeight))
		w.Show()
	} else {
		iw.nav.Push(ab)
	}
}

func (iw *InfoWindow) showZoomWindow(title string, id int32, load func(int32, int) (fyne.Resource, error), w fyne.Window) {
	s := float32(zoomImagePixelSize) / w.Canvas().Scale()
	r, err := load(id, zoomImagePixelSize)
	if err != nil {
		iw.u.ShowErrorDialog("Failed to load image", err, w)
	}
	i := iwidget.NewImageFromResource(r, fyne.NewSquareSize(s))
	p := theme.Padding()
	w2 := iw.u.App().NewWindow(iw.u.MakeWindowTitle(title))
	w2.SetContent(container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), i))
	w2.Show()
}

type infoVariant uint

const (
	infoNotSupported infoVariant = iota
	infoAlliance
	infoCharacter
	infoConstellation
	infoCorporation
	infoInventoryType
	infoLocation
	infoRegion
	infoRace
	infoSolarSystem
)

var eveEntityCategory2InfoVariant = map[app.EveEntityCategory]infoVariant{
	app.EveEntityAlliance:      infoAlliance,
	app.EveEntityCharacter:     infoCharacter,
	app.EveEntityConstellation: infoConstellation,
	app.EveEntityCorporation:   infoCorporation,
	app.EveEntityRegion:        infoRegion,
	app.EveEntitySolarSystem:   infoSolarSystem,
	app.EveEntityStation:       infoLocation,
	app.EveEntityInventoryType: infoInventoryType,
}

func eveEntity2InfoVariant(ee *app.EveEntity) infoVariant {
	v, ok := eveEntityCategory2InfoVariant[ee.Category]
	if !ok {
		return infoNotSupported
	}
	return v

}

func SupportedEveEntities() set.Set[app.EveEntityCategory] {
	return set.Collect(maps.Keys(eveEntityCategory2InfoVariant))

}
