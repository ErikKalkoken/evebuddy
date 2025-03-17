package infowindow

import (
	"fmt"
	"log/slog"
	"maps"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/dustin/go-humanize"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	renderIconPixelSize = 256
	renderIconUnitSize  = 128
	logoZoomFactor      = 1.3
	zoomImagePixelSize  = 512
	infoWindowWidth     = 600
	infoWindowHeight    = 500
)

type InfoVariant uint

const (
	NotSupported InfoVariant = iota
	Alliance
	Character
	Corporation
	InventoryType
	Location
	SolarSystem
)

var eveEntityCategory2InfoVariant = map[app.EveEntityCategory]InfoVariant{
	app.EveEntityAlliance:      Alliance,
	app.EveEntityCharacter:     Character,
	app.EveEntityCorporation:   Corporation,
	app.EveEntitySolarSystem:   SolarSystem,
	app.EveEntityStation:       Location,
	app.EveEntityInventoryType: InventoryType,
}

func eveEntity2InfoVariant(ee *app.EveEntity) InfoVariant {
	v, ok := eveEntityCategory2InfoVariant[ee.Category]
	if !ok {
		return NotSupported
	}
	return v

}

func SupportedEveEntities() set.Set[app.EveEntityCategory] {
	return set.Collect(maps.Keys(eveEntityCategory2InfoVariant))

}

// InfoWindow represents a dedicated window for showing information similar to the in-game info windows.
type InfoWindow struct {
	cs                 *character.CharacterService
	currentCharacterID func() int32
	eus                *eveuniverse.EveUniverseService
	eis                app.EveImageService
	w                  fyne.Window // parent window, e.g. for displaying error dialogs
}

// New returns a configured InfoWindow.
func New(
	currentCharacterID func() int32,
	cs *character.CharacterService,
	eus *eveuniverse.EveUniverseService,
	eis app.EveImageService,
	w fyne.Window,
) InfoWindow {
	iw := InfoWindow{
		currentCharacterID: currentCharacterID,
		cs:                 cs,
		eus:                eus,
		eis:                eis,
		w:                  w,
	}
	return iw
}

func (iw *InfoWindow) SetWindow(w fyne.Window) {
	iw.w = w
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
	case InventoryType:
		showWindow("Information", func(w fyne.Window) fyne.CanvasObject {
			// TODO: Restructure, so that window is first drawn empty and content loaded in background (as other info windo)
			a, err := NewInventoryTypeArea(iw, int32(id), iw.currentCharacterID(), w)
			if err != nil {
				slog.Error("show type", "error", err)
				l := widget.NewLabel(fmt.Sprintf("ERROR: Can not create info window: %s", err))
				l.Importance = widget.DangerImportance
				return l
			}
			w.SetTitle(a.MakeTitle("Information"))
			return a.Content
		})
	case SolarSystem:
		showWindow("Solar System", func(w fyne.Window) fyne.CanvasObject {
			a := newSolarSystemArea(iw, int32(id), w)
			return a.Content
		})
	case Location:
		showWindow("Location", func(w fyne.Window) fyne.CanvasObject {
			a := newLocationArea(iw, id, w)
			return a.Content
		})
	default:
		iwidget.ShowErrorDialog(
			"Can't show info window for unknown category",
			fmt.Errorf("infowindow: undefined category"),
			iw.w,
		)
	}
}

// Show shows a new info window for an EveEntity.
func (iw InfoWindow) ShowEveEntity(ee *app.EveEntity) {
	iw.Show(eveEntity2InfoVariant(ee), int64(ee.ID))
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

func historyItem2EntityItem(hi app.MembershipHistoryItem) entityItem {
	var endDateStr string
	if !hi.EndDate.IsZero() {
		endDateStr = hi.EndDate.Format(app.DateFormat)
	} else {
		endDateStr = "this day"
	}
	var closed string
	if hi.IsDeleted {
		closed = " (closed)"
	}
	text := fmt.Sprintf(
		"%s%s   **%s** to **%s** (%s days)",
		hi.OrganizationName(),
		closed,
		hi.StartDate.Format(app.DateFormat),
		endDateStr,
		humanize.Comma(int64(hi.Days)),
	)
	return NewEntityItemFromEveEntity(hi.Organization, text)
}
