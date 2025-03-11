package infowindow

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"

	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	defaultIconPixelSize = 64
	defaultIconUnitSize  = 32
	logoZoomFactor       = 1.3
	zoomImagePixelSize   = 512
	infoWindowWidth      = 600
	infoWindowHeight     = 500
)

// InfoWindow represents a dedicated window for showing information similar to the in-game info windows.
type InfoWindow struct {
	eus *eveuniverse.EveUniverseService
	eis app.EveImageService
	sb  *iwidget.Snackbar
}

// New returns a configured InfoWindow.
func New(eus *eveuniverse.EveUniverseService, eis app.EveImageService, sb *iwidget.Snackbar) InfoWindow {
	w := InfoWindow{eus: eus, eis: eis, sb: sb}
	return w
}

// Show shows a new info window for an EveEntity.
func (iw InfoWindow) ShowEveEntity(o *app.EveEntity) {
	showWindow := func(category string, create func() fyne.CanvasObject) {
		w := fyne.CurrentApp().NewWindow(fmt.Sprintf("%s: Information", category))
		w.SetContent(create())
		w.Resize(fyne.Size{Width: infoWindowWidth, Height: infoWindowHeight})
		w.Show()
	}
	switch o.Category {
	case app.EveEntityAlliance:
		showWindow("Alliance", func() fyne.CanvasObject {
			a := newAllianceInfoArea(iw, o.ID)
			return a.Content
		})
	case app.EveEntityCharacter:
		showWindow("Character", func() fyne.CanvasObject {
			a := newCharacterInfoArea(iw, o.ID)
			return a.Content
		})
	case app.EveEntityCorporation:
		showWindow("Corporation", func() fyne.CanvasObject {
			a := newCorporationInfoArea(iw, o.ID)
			return a.Content
		})
	default:
		iw.sb.Show(fmt.Sprintf("Can't show info window for %s", o.Category))
	}
}

func showZoomWindow(title string, id int32, load func(int32, int) (fyne.Resource, error)) {
	w := fyne.CurrentApp().NewWindow(title)
	s := float32(zoomImagePixelSize) / w.Canvas().Scale()
	i := appwidget.NewImageResourceAsync(icon.QuestionmarkSvg, fyne.NewSquareSize(s), func() (fyne.Resource, error) {
		return load(id, zoomImagePixelSize)
	})
	p := theme.Padding()
	w.SetContent(container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), i))
	w.Show()
}
