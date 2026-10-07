package screens

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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	flyableCan    = "Can Fly"
	flyableCanNot = "Can Not Fly"
)

// Names of the flyable ships filters, used as labels on desktop and as option names on mobile.
const (
	flyableShipsFilterClass   = "Class"
	flyableShipsFilterFlyable = "Flyable"
)

// flyableShipsFilter is the selected value of each flyable ships filter. Empty means not filtered.
type flyableShipsFilter struct {
	class   string
	flyable string
}

// match reports whether row r passes all filters.
func (f flyableShipsFilter) match(r flyableShipRow) bool {
	switch {
	case f.class != "" && r.groupName != f.class,
		f.flyable == flyableCan && !r.canFly,
		f.flyable == flyableCanNot && r.canFly:
		return false
	}
	return true
}

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

	character     atomic.Pointer[app.Character]
	columnSorter  *xwidget.ColumnSorter[flyableShipRow]
	filterChip    *xwidget.FilterChipCompact // only on mobile
	filterRun     latestRun
	footer        *widget.Label
	grid          *widget.GridWrap
	imageCache    xsync.Map[string, *image.RGBA]
	rows          []flyableShipRow
	rowsFiltered  []flyableShipRow
	searchEntry   *xwidget.SearchEntry
	selectFlyable *kxwidget.FilterChipSelect // select chips only on desktop
	selectGroup   *kxwidget.FilterChipSelect
	sortChip      *kxwidget.SortChip
	top           *widget.Label
	u             baseUI
}

func NewFlyableShips(u baseUI) *FlyableShips {
	columnSorter := xwidget.NewColumnSorter(xwidget.NewDataColumns([]xwidget.DataColumn[flyableShipRow]{{
		Label: "Type",
		Sort: func(a, b flyableShipRow) int {
			return strings.Compare(a.typeName, b.typeName)
		},
	}, {
		Label: "Class",
		Sort: func(a, b flyableShipRow) int {
			return strings.Compare(a.groupName, b.groupName)
		},
	}}),
		"Type",
		xwidget.SortAsc,
	)
	a := &FlyableShips{
		columnSorter: columnSorter,
		footer:       ui.NewLabelWithTruncation(""),
		top:          ui.NewLabelWithWrapping(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	placeholder := "Search type and class names"
	if a.u.IsMobile() {
		placeholder = "Search" // shares the row with the chips
	}
	a.searchEntry = xwidget.NewSearchEntry(placeholder, func(_ string) {
		a.filterRowsAsync()
	})

	if a.u.IsMobile() {
		a.filterChip = xwidget.NewFilterChipCompact(nil, func(map[string]string) {
			a.filterRowsAsync()
		})
	} else {
		a.selectGroup = kxwidget.NewFilterChipSelectWithSearch(flyableShipsFilterClass, []string{}, func(_ string) {
			a.filterRowsAsync()
		}, a.u.MainWindow())
		a.selectFlyable = kxwidget.NewFilterChipSelect(flyableShipsFilterFlyable, []string{}, func(_ string) {
			a.filterRowsAsync()
		})
	}
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync()
	})
	a.grid = a.makeShipsGrid()

	// Signals
	a.u.Signals().CurrentCharacterExchanged.AddListener(
		func(ctx context.Context, c *app.Character) {
			a.character.Store(c)
			fyne.Do(func() {
				a.searchEntry.ClearSilent()
				if a.filterChip != nil {
					a.filterChip.ResetSilent()
					return
				}
				clearSelectsSilent(a.selectFlyable, a.selectGroup)
			})
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
	topBox := container.NewVBox(a.top)
	if a.u.IsMobile() {
		topBox.Add(container.NewBorder(nil, nil, nil, container.NewHBox(a.filterChip, a.sortChip), a.searchEntry))
	} else {
		buttons := container.NewHBox(a.selectGroup, a.selectFlyable, a.sortChip)
		topBox.Add(container.NewBorder(nil, nil, buttons, nil, a.searchEntry))
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
		a.u.InfoViewer().ShowType(o.typeID, a.character.Load().IDOrZero())
	}
	return g
}

// currentFilter returns the selected filters: from the compact chip on mobile
// and from the filter chips on desktop.
func (a *FlyableShips) currentFilter() flyableShipsFilter {
	if a.filterChip != nil {
		s := a.filterChip.Selected()
		return flyableShipsFilter{
			class:   s[flyableShipsFilterClass],
			flyable: s[flyableShipsFilterFlyable],
		}
	}
	return flyableShipsFilter{
		class:   a.selectGroup.Selected,
		flyable: a.selectFlyable.Selected,
	}
}

func (a *FlyableShips) filterRowsAsync() {
	isLatest := a.filterRun.start()
	rows := slices.Clone(a.rows)
	total := len(rows)
	filter := a.currentFilter()
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort("")

	runAsync(func() {
		rows = slices.DeleteFunc(rows, func(r flyableShipRow) bool {
			return !filter.match(r)
		})
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
			if !isLatest() {
				return
			}
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			if a.filterChip != nil {
				a.filterChip.SetOptions(
					xwidget.NewFilterOptionMultiChoiceWithSearch(flyableShipsFilterClass, groupOptions),
					xwidget.NewFilterOptionMultiChoice(flyableShipsFilterFlyable, flyableOptions),
				)
			} else {
				a.selectGroup.SetOptions(groupOptions)
				a.selectFlyable.SetOptions(flyableOptions)
			}
			a.rowsFiltered = rows
			a.grid.Refresh()
			a.grid.ScrollToTop()
		})
	})
}

func (a *FlyableShips) update(ctx context.Context) {
	reset := func() {
		fyne.Do(func() {
			xslices.Clear(&a.rows)
			a.searchEntry.Disable()
			a.searchEntry.SetText("")
			if a.filterChip != nil {
				a.filterChip.SetOptions()
			} else {
				a.selectGroup.SetOptions([]string{})
				a.selectFlyable.SetOptions([]string{})
			}
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
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to update data for flyable ships UI", "error", err)
		setTop(a.u.ErrorDisplay(err), widget.DangerImportance)
	}

	ok1, err := a.u.EVEUniverse().HasSection(ctx, app.SectionEveTypes)
	if err != nil {
		reset()
		reportError(err)
		return
	}
	if !ok1 {
		reset()
		setTop("Waiting for universe data to be loaded...", widget.WarningImportance)
		return
	}

	characterID := a.character.Load().IDOrZero()
	if characterID == 0 {
		reset()
		setTop("No character", widget.LowImportance)
		return
	}

	exists, err := a.u.Character().HasSection(ctx, characterID, app.SectionCharacterSkills)
	if err != nil {
		reset()
		reportError(err)
		return
	}
	if !exists {
		reset()
		setTop("Waiting for character data to be loaded...", widget.WarningImportance)
		return
	}

	oo, err := a.u.Character().ListShipsAbilities(ctx, characterID)
	if err != nil {
		reset()
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
		a.searchEntry.Enable()
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
	runAsync(func() {
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
	})
}

func (w *ShipItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(container.NewPadded(w.image), w.label)
	return widget.NewSimpleRenderer(c)
}
