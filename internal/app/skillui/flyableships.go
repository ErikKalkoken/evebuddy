package skillui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/anthonynsimon/bild/effect"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	flyableCan    = "Can Fly"
	flyableCanNot = "Can Not Fly"
)

const (
	flyableColType = iota + 1
	flyableColGroup
)

type flyableShipRow struct {
	canFly      bool
	characterID int64
	groupID     int64
	groupName   string
	searchText  string
	typeID      int64
	typeName    string
}

type FlyableShips struct {
	widget.BaseWidget

	footer        *widget.Label
	character     atomic.Pointer[app.Character]
	columnSorter  *xwidget.ColumnSorter[flyableShipRow]
	grid          *widget.GridWrap
	rows          []flyableShipRow
	rowsFiltered  []flyableShipRow
	search        *widget.Entry
	selectFlyable *kxwidget.FilterChipSelect
	selectGroup   *kxwidget.FilterChipSelect
	sortButton    *xwidget.SortButton
	top           *widget.Label
	u             uiServices
	imageCache    xsync.Map[string, *image.RGBA]
}

func NewFlyableShips(u uiServices) *FlyableShips {
	columnSorter := xwidget.NewColumnSorter(xwidget.NewDataColumns([]xwidget.DataColumn[flyableShipRow]{{
		ID:    flyableColType,
		Label: "Type",
		Sort: func(a, b flyableShipRow) int {
			return strings.Compare(a.typeName, b.typeName)
		},
	}, {
		ID:    flyableColGroup,
		Label: "Class",
		Sort: func(a, b flyableShipRow) int {
			return strings.Compare(a.groupName, b.groupName)
		},
	}}),
		flyableColType,
		xwidget.SortAsc,
	)
	a := &FlyableShips{
		columnSorter: columnSorter,
		footer:       awidget.NewLabelWithTruncation(""),
		top:          awidget.NewLabelWithWrapping(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	a.search = widget.NewEntry()
	a.search.SetPlaceHolder("Search type and class names")
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRowsAsync()
	})
	a.search.OnChanged = func(s string) {
		a.filterRowsAsync()
	}

	a.selectGroup = kxwidget.NewFilterChipSelectWithSearch("Class", []string{}, func(s string) {
		a.filterRowsAsync()
	}, a.u.MainWindow())

	a.selectFlyable = kxwidget.NewFilterChipSelect("Flyable", []string{}, func(s string) {
		a.filterRowsAsync()
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync()
	}, a.u.MainWindow())
	a.grid = a.makeShipsGrid()

	// Signals
	a.u.Signals().CurrentCharacterExchanged.AddListener(
		func(ctx context.Context, c *app.Character) {
			a.character.Store(c)
			a.update(ctx)
		},
	)
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDOrZero() != arg.CharacterID {
			return
		}
		if arg.Section == app.SectionCharacterSkills {
			a.update(ctx)
		}
	},
	)
	a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		characterID := a.character.Load().IDOrZero()
		if characterID == 0 {
			return
		}
		if arg.Section == app.SectionEveTypes {
			a.update(ctx)
		}
	})
	return a
}

func (a *FlyableShips) CreateRenderer() fyne.WidgetRenderer {
	buttons := container.NewHBox(a.selectGroup, a.selectFlyable, a.sortButton)
	topBox := container.NewVBox(a.top)
	if a.u.IsMobile() {
		topBox.Add(a.search)
		topBox.Add(container.NewHScroll(buttons))
	} else {
		topBox.Add(container.NewBorder(nil, nil, buttons, nil, a.search))
	}
	c := container.NewBorder(
		topBox,
		a.footer,
		nil,
		nil,
		a.grid,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *FlyableShips) makeShipsGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return NewShipItem(
				a.u.EVEImage().InventoryTypeRender,
				a.imageCache.Load,
				a.imageCache.Store,
			)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			o := a.rowsFiltered[id]
			item := co.(*ShipItem)
			item.Set(o.typeID, o.typeName, o.canFly)
		})
	g.OnSelected = func(id widget.GridWrapItemID) {
		defer g.UnselectAll()
		if id >= len(a.rowsFiltered) {
			return
		}
		o := a.rowsFiltered[id]
		a.u.InfoWindow().ShowTypeWithCharacter(o.typeID, a.character.Load().IDOrZero())
	}
	return g
}

func (a *FlyableShips) filterRowsAsync() {
	rows := slices.Clone(a.rows)
	total := len(rows)
	group := a.selectGroup.Selected
	flyable := a.selectFlyable.Selected
	search := strings.ToLower(a.search.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(-1)

	go func() {
		if group != "" {
			rows = slices.DeleteFunc(rows, func(r flyableShipRow) bool {
				return r.groupName != group
			})
		}
		if flyable != "" {
			rows = slices.DeleteFunc(rows, func(r flyableShipRow) bool {
				switch flyable {
				case flyableCan:
					return !r.canFly
				case flyableCanNot:
					return r.canFly
				}
				return false
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r flyableShipRow) bool {
				return !strings.Contains(r.searchText, search)
			})
		}
		groupOptions := xslices.Map(rows, func(r flyableShipRow) string {
			return r.groupName
		})
		flyableOptions := xslices.Map(rows, func(r flyableShipRow) string {
			if r.canFly {
				return flyableCan
			}
			return flyableCanNot
		})
		footer := fmt.Sprintf("Showing %d / %d ships", len(rows), total)
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectGroup.SetOptions(groupOptions)
			a.selectFlyable.SetOptions(flyableOptions)
			a.rowsFiltered = rows
			a.grid.Refresh()
			a.grid.ScrollToTop()
		})
	}()
}

func (a *FlyableShips) update(ctx context.Context) {
	clear := func() {
		fyne.Do(func() {
			a.rows = xslices.Reset(a.rows)
			a.search.Disable()
			a.search.SetText("")
			a.selectGroup.SetOptions([]string{})
			a.selectFlyable.SetOptions([]string{})
			a.filterRowsAsync()
		})
	}
	setTop := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.top.Text = s
			a.top.Importance = i
			a.top.Refresh()
		})
	}
	reportError := func(err error) {
		slog.Error("Failed to update data for flyable ships UI", "error", err)
		setTop(a.u.ErrorDisplay(err), widget.DangerImportance)
	}

	ok1, err := a.u.EVEUniverse().HasSection(ctx, app.SectionEveTypes)
	if err != nil {
		clear()
		reportError(err)
		return
	}
	if !ok1 {
		clear()
		setTop("Waiting for universe data to be loaded...", widget.WarningImportance)
		return
	}

	characterID := a.character.Load().IDOrZero()
	if characterID == 0 {
		clear()
		setTop("No character", widget.LowImportance)
		return
	}

	exists, err := a.u.Character().HasSection(ctx, characterID, app.SectionCharacterSkills)
	if err != nil {
		clear()
		reportError(err)
		return
	}
	if !exists {
		clear()
		setTop("Waiting for character data to be loaded...", widget.WarningImportance)
		return
	}

	oo, err := a.u.Character().ListShipsAbilities(ctx, characterID)
	if err != nil {
		clear()
		reportError(err)
		return
	}
	var rows []flyableShipRow
	for _, o := range oo {
		rows = append(rows, flyableShipRow{
			canFly:      o.CanFly,
			characterID: characterID,
			groupID:     o.Group.ID,
			groupName:   o.Group.Name,
			searchText:  strings.ToLower(fmt.Sprintf("%s|%s", o.Type.Name, o.Group.Name)),
			typeID:      o.Type.ID,
			typeName:    o.Type.Name,
		})
	}

	k := 0
	for _, o := range oo {
		if o.CanFly {
			k++
		}
	}
	p := float32(k) / float32(len(oo)) * 100
	text := fmt.Sprintf("Can fly %d / %d ships (%.0f%%)", k, len(oo), p)
	setTop(text, widget.MediumImportance)

	fyne.Do(func() {
		a.rows = rows
		a.search.Enable()
		a.filterRowsAsync()
	})
}

// The ShipItem widget is used to render items on the type info window.
type ShipItem struct {
	widget.BaseWidget

	image      *canvas.Image
	label      *widget.Label
	renderType func(int64, int) (fyne.Resource, error)
	cacheLoad  func(string) (*image.RGBA, bool)
	cacheStore func(string, *image.RGBA)
}

func NewShipItem(
	renderType func(int64, int) (fyne.Resource, error),
	cacheLoad func(string) (*image.RGBA, bool),
	cacheStore func(string, *image.RGBA),
) *ShipItem {
	upLeft := image.Point{0, 0}
	lowRight := image.Point{128, 128}
	image := canvas.NewImageFromImage(image.NewRGBA(image.Rectangle{upLeft, lowRight}))
	image.FillMode = canvas.ImageFillContain
	// image.ScaleMode = defaultImageScaleMode  // FIXME
	image.CornerRadius = theme.InputRadiusSize()
	image.SetMinSize(fyne.NewSquareSize(128))
	w := &ShipItem{
		image:      image,
		label:      widget.NewLabel("First line\nSecond Line\nThird Line"),
		renderType: renderType,
		cacheLoad:  cacheLoad,
		cacheStore: cacheStore,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *ShipItem) Set(typeID int64, label string, canFly bool) {
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

	// TODO: Move grayscale feature into general package

	key := fmt.Sprintf("%d-%v", typeID, canFly)
	img, ok := w.cacheLoad(key)
	if ok {
		w.image.Image = img
		w.image.Refresh()
		return
	}
	go func() {
		j, err := func() (image.Image, error) {
			r, err := w.renderType(typeID, 256)
			if err != nil {
				return nil, err
			}
			j, _, err := image.Decode(bytes.NewReader(r.Content()))
			if err != nil {
				return nil, err
			}
			return j, nil
		}()
		if err != nil {
			slog.Error("shipItem: image render", "error", err)
			fyne.Do(func() {
				w.image.Image = nil
				w.image.Resource = theme.BrokenImageIcon()
				w.image.Refresh()
			})
			return
		}

		b := j.Bounds()
		img = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(img, img.Bounds(), j, b.Min, draw.Src)
		if !canFly {
			img = effect.Grayscale(img)
		}
		w.cacheStore(key, img)

		fyne.Do(func() {
			w.image.Resource = nil
			w.image.Image = img
			w.image.Refresh()
		})
	}()
}

func (w *ShipItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(container.NewPadded(w.image), w.label)
	return widget.NewSimpleRenderer(c)
}
