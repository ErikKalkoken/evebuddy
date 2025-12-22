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
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/anthonynsimon/bild/effect"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
)

const (
	flyableCan    = "Can Fly"
	flyableCanNot = "Can Not Fly"
)

type characterFlyableShips struct {
	widget.BaseWidget

	character       *app.Character
	flyableSelect   *kxwidget.FilterChipSelect
	flyableSelected string
	grid            *widget.GridWrap
	groupSelect     *kxwidget.FilterChipSelect
	groupSelected   string
	searchBox       *widget.Entry
	ships           []*app.CharacterShipAbility
	top             *widget.Label
	foundText       *widget.Label
	u               *baseUI
}

func newCharacterFlyableShips(u *baseUI) *characterFlyableShips {
	a := &characterFlyableShips{
		ships:     make([]*app.CharacterShipAbility, 0),
		top:       widget.NewLabel(""),
		foundText: widget.NewLabel(""),
		u:         u,
	}
	a.ExtendBaseWidget(a)

	a.searchBox = widget.NewEntry()
	a.searchBox.SetPlaceHolder("Search ships")
	a.searchBox.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.searchBox.SetText("")
	})
	a.searchBox.OnChanged = func(s string) {
		if len(s) == 1 {
			return
		}
		if err := a.updateEntries(); err != nil {
			a.u.showErrorDialog("Failed to update ships", err, a.u.MainWindow())
		}
		a.grid.Refresh()
		a.grid.ScrollToTop()
	}

	a.groupSelect = kxwidget.NewFilterChipSelectWithSearch("Class", []string{}, func(s string) {
		a.groupSelected = s
		if err := a.updateEntries(); err != nil {
			a.u.showErrorDialog("Failed to update ships", err, a.u.MainWindow())
		}
		a.grid.Refresh()
		a.grid.ScrollToTop()
	}, a.u.window)

	a.flyableSelect = kxwidget.NewFilterChipSelect("Flyable", []string{}, func(s string) {
		a.flyableSelected = s
		if err := a.updateEntries(); err != nil {
			a.u.showErrorDialog("Failed to update ships", err, a.u.MainWindow())
		}
		a.grid.Refresh()
		a.grid.ScrollToTop()
	})

	a.grid = a.makeShipsGrid()

	a.u.currentCharacterExchanged.AddListener(
		func(_ context.Context, c *app.Character) {
			a.character = c
		},
	)
	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character) != arg.characterID {
			return
		}
		if arg.section == app.SectionCharacterSkills {
			a.update()
		}
	},
	)
	a.u.generalSectionChanged.AddListener(func(_ context.Context, arg generalSectionUpdated) {
		characterID := characterIDOrZero(a.character)
		if characterID == 0 {
			return
		}
		if arg.section == app.SectionEveTypes {
			a.update()
		}
	})
	return a
}

func (a *characterFlyableShips) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewHBox(a.top, layout.NewSpacer(), a.foundText)
	c := container.NewBorder(
		container.NewVBox(
			top,
			a.searchBox,
			container.NewHScroll(container.NewHBox(a.groupSelect, a.flyableSelect, widget.NewButton("Reset", func() {
				a.reset()
			}))),
		),
		nil,
		nil,
		nil,
		a.grid,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *characterFlyableShips) reset() {
	a.searchBox.SetText("")
	a.groupSelect.ClearSelected()
	a.flyableSelect.ClearSelected()
}

func (a *characterFlyableShips) makeShipsGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.ships)
		},
		func() fyne.CanvasObject {
			return newShipItem(a.u.eis, a.u.memcache, icons.QuestionmarkSvg)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.ships) {
				return
			}
			o := a.ships[id]
			item := co.(*shipItem)
			item.Set(o.Type.ID, o.Type.Name, o.CanFly)
		})
	g.OnSelected = func(id widget.GridWrapItemID) {
		defer g.UnselectAll()
		if id >= len(a.ships) {
			return
		}
		o := a.ships[id]
		a.u.ShowTypeInfoWindowWithCharacter(o.Type.ID, characterIDOrZero(a.character))
	}
	return g
}

func (a *characterFlyableShips) update() {
	t, i, enabled, err := func() (string, widget.Importance, bool, error) {
		hasData := a.u.scs.HasGeneralSection(app.SectionEveTypes)
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

func (a *characterFlyableShips) updateEntries() error {
	characterID := characterIDOrZero(a.character)
	if characterID == 0 {
		fyne.Do(func() {
			a.ships = make([]*app.CharacterShipAbility, 0)
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
	g := set.Of[string]()
	f := set.Of[string]()
	for _, o := range ships {
		g.Add(o.Group.Name)
		if o.CanFly {
			f.Add(flyableCan)
		} else {
			f.Add(flyableCanNot)
		}
	}
	groups := slices.Collect(g.All())
	slices.Sort(groups)
	flyable := slices.Collect(f.All())
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

func (a *characterFlyableShips) makeTopText() (string, widget.Importance, bool, error) {
	if a.character == nil {
		return "No character", widget.LowImportance, false, nil
	}
	characterID := characterIDOrZero(a.character)
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionCharacterSkills)
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

// The shipItem widget is used to render items on the type info window.
type shipItem struct {
	widget.BaseWidget

	cache        *memcache.Cache
	fallbackIcon fyne.Resource
	image        *canvas.Image
	label        *widget.Label
	eis          app.EveImageService
}

func newShipItem(eis app.EveImageService, cache *memcache.Cache, fallbackIcon fyne.Resource) *shipItem {
	upLeft := image.Point{0, 0}
	lowRight := image.Point{128, 128}
	image := canvas.NewImageFromImage(image.NewRGBA(image.Rectangle{upLeft, lowRight}))
	image.FillMode = canvas.ImageFillContain
	image.ScaleMode = defaultImageScaleMode
	image.SetMinSize(fyne.NewSquareSize(128))
	w := &shipItem{
		image:        image,
		label:        widget.NewLabel("First line\nSecond Line\nThird Line"),
		fallbackIcon: fallbackIcon,
		eis:          eis,
		cache:        cache,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *shipItem) Set(typeID int32, label string, canFly bool) {
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

func (w *shipItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(container.NewPadded(w.image), w.label)
	return widget.NewSimpleRenderer(c)
}
