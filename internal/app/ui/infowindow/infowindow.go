package infowindow

import (
	"fmt"
	"maps"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/dustin/go-humanize"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	defaultIconPixelSize = 64
	defaultIconUnitSize  = 32
	renderIconPixelSize  = 256
	renderIconUnitSize   = 128
	logoZoomFactor       = 1.3
	zoomImagePixelSize   = 512
	infoWindowWidth      = 600
	infoWindowHeight     = 500
	dateFormat           = "2006.01.02 15:04"
)

type InfoVariant uint

const (
	Unknown InfoVariant = iota
	Alliance
	Character
	Corporation
	Location
	SolarSystem
)

var eveEntityCategory2InfoVariant = map[app.EveEntityCategory]InfoVariant{
	app.EveEntityAlliance:    Alliance,
	app.EveEntityCharacter:   Character,
	app.EveEntityCorporation: Corporation,
	app.EveEntitySolarSystem: SolarSystem,
	app.EveEntityStation:     Location,
}

func eveEntity2InfoVariant(ee *app.EveEntity) InfoVariant {
	v, ok := eveEntityCategory2InfoVariant[ee.Category]
	if !ok {
		return Unknown
	}
	return v

}

func eveEntitySupportedCategories() []app.EveEntityCategory {
	return slices.Collect(maps.Keys(eveEntityCategory2InfoVariant))

}

// InfoWindow represents a dedicated window for showing information similar to the in-game info windows.
type InfoWindow struct {
	eus *eveuniverse.EveUniverseService
	eis app.EveImageService
	sb  *iwidget.Snackbar
}

// New returns a configured InfoWindow.
func New(eus *eveuniverse.EveUniverseService, eis app.EveImageService, sb *iwidget.Snackbar) InfoWindow {
	w := InfoWindow{
		eus: eus,
		eis: eis,
		sb:  sb,
	}
	return w
}

func (iw InfoWindow) Show(t InfoVariant, id int64) {
	switch t {
	case Alliance:
		showWindow("Alliance", func(w fyne.Window) fyne.CanvasObject {
			a := newAlliancArea(iw, int32(id), w)
			return a.Content
		})
	case Character:
		showWindow("Character", func(w fyne.Window) fyne.CanvasObject {
			a := newCharacterArea(iw, int32(id), w)
			return a.Content
		})
	case Corporation:
		showWindow("Corporation", func(w fyne.Window) fyne.CanvasObject {
			a := newCorporationArea(iw, int32(id), w)
			return a.Content
		})
	case SolarSystem:
		showWindow("Solar System", func(w fyne.Window) fyne.CanvasObject {
			a := newSolarSystemArea(iw, int32(id), w)
			return a.Content
		})
	case Location:
		iw.ShowLocation(id)
	default:
		iw.sb.Show("Can't show info window for unknown category")
	}
}

// Show shows a new info window for an EveEntity.
func (iw InfoWindow) ShowEveEntity(ee *app.EveEntity) {
	iw.Show(eveEntity2InfoVariant(ee), int64(ee.ID))
}

func (iw InfoWindow) ShowLocation(id int64) {
	showWindow("Location", func(w fyne.Window) fyne.CanvasObject {
		a := newLocationArea(iw, id, w)
		return a.Content
	})
}

func showWindow(category string, create func(w fyne.Window) fyne.CanvasObject) {
	w := fyne.CurrentApp().NewWindow(fmt.Sprintf("%s: Information", category))
	w.SetContent(create(w))
	w.Resize(fyne.Size{Width: infoWindowWidth, Height: infoWindowHeight})
	w.Show()
}

func (iw InfoWindow) showZoomWindow(title string, id int32, load func(int32, int) (fyne.Resource, error), w fyne.Window) {
	s := float32(zoomImagePixelSize) / w.Canvas().Scale()
	r, err := load(id, zoomImagePixelSize)
	if err != nil {
		iwidget.ShowErrorDialog("Failed to load image", err, w)
	}
	i := iwidget.NewImageFromResource(r, fyne.NewSquareSize(s))
	p := theme.Padding()
	w2 := fyne.CurrentApp().NewWindow(title)
	w2.SetContent(container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), i))
	w2.Show()
}

func historyItem2EntityItem(hi app.MembershipHistoryItem) EntityItem {
	var endDateStr string
	if !hi.EndDate.IsZero() {
		endDateStr = hi.EndDate.Format(dateFormat)
	} else {
		endDateStr = "this day"
	}
	var closed string
	if hi.IsDeleted {
		closed = " (closed)"
	}
	var text string
	if false && hi.IsOldest {
		text = fmt.Sprintf("Founded   **%s**", hi.StartDate.Format(dateFormat))
	} else {
		text = fmt.Sprintf(
			"%s%s   **%s** to **%s** (%s days)",
			hi.OrganizationName(),
			closed,
			hi.StartDate.Format(dateFormat),
			endDateStr,
			humanize.Comma(int64(hi.Days)),
		)
	}
	return NewEntityItemFromEveEntity(hi.Organization, text)
}
