package ui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"log/slog"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/anthonynsimon/bild/effect"

	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
)

const (
	flyableCan    = "Can Fly"
	flyableCanNot = "Can Not Fly"
)

type CharacterFlyableShips struct {
	widget.BaseWidget

	flyableSelect   *widget.Select
	flyableSelected string
	grid            *widget.GridWrap
	groupSelect     *widget.Select
	groupSelected   string
	searchBox       *widget.Entry
	ships           []*app.CharacterShipAbility
	top             *widget.Label
	foundText       *widget.Label
	u               *BaseUI
}

func NewCharacterFlyableShips(u *BaseUI) *CharacterFlyableShips {
	a := &CharacterFlyableShips{
		ships:     make([]*app.CharacterShipAbility, 0),
		top:       widget.NewLabel(""),
		foundText: widget.NewLabel(""),
		u:         u,
	}
	a.ExtendBaseWidget(a)

	a.searchBox = widget.NewEntry()
	a.searchBox.SetPlaceHolder("Filter by ship name")
	a.searchBox.OnChanged = func(s string) {
		if len(s) == 1 {
			return
		}
		if err := a.updateEntries(); err != nil {
			a.u.ShowErrorDialog("Failed to update ships", err, a.u.MainWindow())
		}
		a.grid.Refresh()
		a.grid.ScrollToTop()
	}

	a.groupSelect = widget.NewSelect([]string{}, func(s string) {
		a.groupSelected = s
		if err := a.updateEntries(); err != nil {
			a.u.ShowErrorDialog("Failed to update ships", err, a.u.MainWindow())
		}
		a.grid.Refresh()
		a.grid.ScrollToTop()
	})
	a.groupSelect.PlaceHolder = "(Ship Class)"

	a.flyableSelect = widget.NewSelect([]string{}, func(s string) {
		a.flyableSelected = s
		if err := a.updateEntries(); err != nil {
			a.u.ShowErrorDialog("Failed to update ships", err, a.u.MainWindow())
		}
		a.grid.Refresh()
		a.grid.ScrollToTop()
	})
	a.flyableSelect.PlaceHolder = "(Flyable)"

	a.grid = a.makeShipsGrid()
	return a
}

func (a *CharacterFlyableShips) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewHBox(
		a.top,
		a.foundText,
		layout.NewSpacer(),
		widget.NewButton("Reset", func() {
			a.reset()
		}))
	topBox := container.NewVBox(
		top,
		container.NewBorder(
			nil,
			nil,
			nil,
			container.NewHBox(a.groupSelect, a.flyableSelect),
			a.searchBox),
	)
	c := container.NewBorder(topBox, nil, nil, nil, a.grid)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterFlyableShips) reset() {
	a.searchBox.SetText("")
	a.groupSelect.ClearSelected()
	a.flyableSelect.ClearSelected()
	a.foundText.Hide()
}

func (a *CharacterFlyableShips) makeShipsGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.ships)
		},
		func() fyne.CanvasObject {
			return NewShipItem(a.u.eis, a.u.memcache, icons.QuestionmarkSvg)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.ships) {
				return
			}
			o := a.ships[id]
			item := co.(*ShipItem)
			item.Set(o.Type.ID, o.Type.Name, o.CanFly)
		})
	g.OnSelected = func(id widget.GridWrapItemID) {
		defer g.UnselectAll()
		if id >= len(a.ships) {
			return
		}
		o := a.ships[id]
		a.u.ShowTypeInfoWindow(o.Type.ID)
	}
	return g
}

func (a *CharacterFlyableShips) update() {
	t, i, enabled, err := func() (string, widget.Importance, bool, error) {
		hasData := a.u.scs.GeneralSectionExists(app.SectionEveCategories)
		if !hasData {
			return "Waiting for universe data to be loaded...", widget.WarningImportance, false, nil
		}
		if err := a.updateEntries(); err != nil {
			return "", 0, false, err
		}
		return a.makeTopText()
	}()
	if err != nil {
		slog.Error("Failed to refresh ships UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.grid.Refresh()
		if enabled {
			a.searchBox.Enable()
		} else {
			a.searchBox.Disable()
		}
		a.reset()
	})
}

func (a *CharacterFlyableShips) updateEntries() error {
	characterID := a.u.currentCharacterID()
	if characterID == 0 {
		a.ships = make([]*app.CharacterShipAbility, 0)
		fyne.Do(func() {
			a.grid.Refresh()
			a.searchBox.SetText("")
			a.groupSelect.SetOptions([]string{})
			a.flyableSelect.SetOptions([]string{})
		})
		return nil
	}
	search := fmt.Sprintf("%%%s%%", a.searchBox.Text)
	oo, err := a.u.cs.ListShipsAbilities(context.Background(), characterID, search)
	if err != nil {
		return err
	}
	ships := make([]*app.CharacterShipAbility, 0)
	for _, o := range oo {
		isSelectedGroup := a.groupSelected == "" || o.Group.Name == a.groupSelected
		var isSelectedFlyable bool
		switch a.flyableSelected {
		case flyableCan:
			isSelectedFlyable = o.CanFly
		case flyableCanNot:
			isSelectedFlyable = !o.CanFly
		default:
			isSelectedFlyable = true
		}
		if isSelectedGroup && isSelectedFlyable {
			ships = append(ships, o)
		}
	}
	g := set.New[string]()
	f := set.New[string]()
	for _, o := range ships {
		g.Add(o.Group.Name)
		if o.CanFly {
			f.Add(flyableCan)
		} else {
			f.Add(flyableCanNot)
		}
	}
	groups := g.ToSlice()
	slices.Sort(groups)
	flyable := f.ToSlice()
	slices.Sort(flyable)
	a.ships = ships
	fyne.Do(func() {
		a.groupSelect.SetOptions(groups)
		a.flyableSelect.SetOptions(flyable)
		a.foundText.SetText(fmt.Sprintf("%d found", len(ships)))
		a.foundText.Show()
		a.grid.Refresh()
	})
	return nil
}

func (a *CharacterFlyableShips) makeTopText() (string, widget.Importance, bool, error) {
	if !a.u.hasCharacter() {
		return "No character", widget.LowImportance, false, nil
	}
	characterID := a.u.currentCharacterID()
	hasData := a.u.scs.CharacterSectionExists(characterID, app.SectionSkills)
	if !hasData {
		return "Waiting for skills to be loaded...", widget.WarningImportance, false, nil
	}
	oo, err := a.u.cs.ListShipsAbilities(context.Background(), characterID, "%%")
	if err != nil {
		return "", 0, false, err
	}
	c := 0
	for _, o := range oo {
		if o.CanFly {
			c++
		}
	}
	p := float32(c) / float32(len(oo)) * 100
	text := fmt.Sprintf("Can fly %d / %d ships (%.0f%%)", c, len(oo), p)
	return text, widget.MediumImportance, true, nil
}

// The ShipItem widget is used to render items on the type info window.
type ShipItem struct {
	widget.BaseWidget

	cache        *memcache.Cache
	fallbackIcon fyne.Resource
	image        *canvas.Image
	label        *widget.Label
	eis          *eveimageservice.EveImageService
}

func NewShipItem(eis *eveimageservice.EveImageService, cache *memcache.Cache, fallbackIcon fyne.Resource) *ShipItem {
	upLeft := image.Point{0, 0}
	lowRight := image.Point{128, 128}
	image := canvas.NewImageFromImage(image.NewRGBA(image.Rectangle{upLeft, lowRight}))
	image.FillMode = canvas.ImageFillContain
	image.ScaleMode = appwidget.DefaultImageScaleMode
	image.SetMinSize(fyne.NewSquareSize(128))
	w := &ShipItem{
		image:        image,
		label:        widget.NewLabel("First line\nSecond Line\nThird Line"),
		fallbackIcon: fallbackIcon,
		eis:          eis,
		cache:        cache,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *ShipItem) Set(typeID int32, label string, canFly bool) {
	w.label.Importance = widget.MediumImportance
	w.label.Text = label
	w.label.Wrapping = fyne.TextWrapWord
	var i widget.Importance
	if canFly {
		i = widget.MediumImportance
	} else {
		i = widget.LowImportance
	}
	w.label.Importance = i
	w.label.Refresh()
	go func() {
		// TODO: Move grayscale feature into general package
		key := fmt.Sprintf("ship-image-%d", typeID)
		var img *image.RGBA
		y, ok := w.cache.Get(key)
		if !ok {
			r, err := w.eis.InventoryTypeRender(typeID, 256)
			if err != nil {
				slog.Error("failed to fetch image for ship render", "error", err)
				return
			}
			j, _, err := image.Decode(bytes.NewReader(r.Content()))
			if err != nil {
				slog.Error("failed to decode image for ship render", "error", err)
				return
			}
			b := j.Bounds()
			img = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
			draw.Draw(img, img.Bounds(), j, b.Min, draw.Src)
			w.cache.Set(key, img, 3600*time.Second)
		} else {
			img = y.(*image.RGBA)
		}
		if !canFly {
			img = effect.Grayscale(img)
		}
		fyne.Do(func() {
			w.image.Image = img
			w.image.Refresh()
		})
	}()
}

func (w *ShipItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(container.NewPadded(w.image), w.label)
	return widget.NewSimpleRenderer(c)
}
