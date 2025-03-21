package infowindow

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
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

type UI interface {
	CharacterService() app.CharacterService
	CurrentCharacterID() int32
	EveImageService() app.EveImageService
	EveUniverseService() app.EveUniverseService
	IsDeveloperMode() bool
	IsOffline() bool
	ShowErrorDialog(message string, err error, parent fyne.Window)
	ShowInformationDialog(title, message string, parent fyne.Window)
}

// InfoWindow represents a dedicated window for showing information similar to the in-game info windows.
type InfoWindow struct {
	u UI
	w fyne.Window // parent window, e.g. for displaying error dialogs
}

// New returns a configured InfoWindow.
func New(u UI, w fyne.Window) InfoWindow {
	iw := InfoWindow{u: u, w: w}
	return iw
}

func (iw *InfoWindow) SetWindow(w fyne.Window) {
	iw.w = w
}

// Show shows a new info window for an EveEntity.
func (iw InfoWindow) ShowEveEntity(ee *app.EveEntity) {
	iw.show(eveEntity2InfoVariant(ee), int64(ee.ID))
}

// Show shows a new info window for an EveEntity.
func (iw InfoWindow) Show(c app.EveEntityCategory, id int32) {
	iw.show(eveEntity2InfoVariant(&app.EveEntity{Category: c}), int64(id))
}

func (iw InfoWindow) ShowLocation(id int64) {
	iw.show(infoLocation, id)
}

func (iw InfoWindow) show(t infoVariant, id int64) {
	if iw.u.IsOffline() {
		iw.u.ShowInformationDialog(
			"Offline",
			"Can't show info window when offline",
			iw.w,
		)
		return
	}
	switch t {
	case infoAlliance:
		showWindow("Alliance", func(w fyne.Window) fyne.CanvasObject {
			return newAllianceInfo(iw, int32(id), w)
		})
	case infoCharacter:
		showWindow("Character", func(w fyne.Window) fyne.CanvasObject {
			return newCharacterInfo(iw, int32(id), w)
		})
	case infoConstellation:
		showWindow("Constellation", func(w fyne.Window) fyne.CanvasObject {
			return newConstellationInfo(iw, int32(id), w)
		})
	case infoCorporation:
		showWindow("Corporation", func(w fyne.Window) fyne.CanvasObject {
			return newCorporationInfo(iw, int32(id), w)
		})
	case infoInventoryType:
		showWindow("Information", func(w fyne.Window) fyne.CanvasObject {
			// TODO: Restructure, so that window is first drawn empty and content loaded in background (as other info windo)
			a, err := NewInventoryTypeInfo(iw, int32(id), iw.u.CurrentCharacterID(), w)
			if err != nil {
				slog.Error("show type", "error", err)
				l := widget.NewLabel(fmt.Sprintf("ERROR: Can not create info window: %s", err))
				l.Importance = widget.DangerImportance
				return l
			}
			w.SetTitle(a.MakeTitle("Information"))
			return a
		})
	case infoRegion:
		showWindow("Region", func(w fyne.Window) fyne.CanvasObject {
			return newRegionInfo(iw, int32(id), w)
		})
	case infoSolarSystem:
		showWindow("Solar System", func(w fyne.Window) fyne.CanvasObject {
			return newSolarSystemInfo(iw, int32(id), w)
		})
	case infoLocation:
		showWindow("Location", func(w fyne.Window) fyne.CanvasObject {
			return newLocationInfo(iw, id, w)
		})
	default:
		iw.u.ShowInformationDialog(
			"Warning",
			"Can't show info window for unknown category",
			iw.w,
		)
	}
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
		iw.u.ShowErrorDialog("Failed to load image", err, w)
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
	return NewEntityItemFromEveEntityWithText(hi.Organization, text)
}
