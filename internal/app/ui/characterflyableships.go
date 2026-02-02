package ui

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
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
)

const (
	flyableCan    = "Can Fly"
	flyableCanNot = "Can Not Fly"
)

const (
	flyableColType = iota
	flyableColGroup
)

type flyableShipRow struct {
	canFly      bool
	characterID int32
	groupID     int32
	groupName   string
	searchText  string
	typeID      int32
	typeName    string
}

type characterFlyableShips struct {
	widget.BaseWidget

	bottom        *widget.Label
	character     atomic.Pointer[app.Character]
	columnSorter  *iwidget.ColumnSorter
	grid          *widget.GridWrap
	rows          []flyableShipRow
	rowsFiltered  []flyableShipRow
	search        *widget.Entry
	selectFlyable *kxwidget.FilterChipSelect
	selectGroup   *kxwidget.FilterChipSelect
	sortButton    *iwidget.SortButton
	top           *widget.Label
	u             *baseUI
}

func newCharacterFlyableShips(u *baseUI) *characterFlyableShips {
	columnSorter := iwidget.NewColumnSorter(iwidget.NewDataColumns([]iwidget.DataColumn{
		{
			Col:   flyableColType,
			Label: "Type",
		},
		{
			Col:   flyableColGroup,
			Label: "Class",
		},
	}),
		flyableColType,
		iwidget.SortAsc,
	)
	a := &characterFlyableShips{
		columnSorter: columnSorter,
		bottom:       widget.NewLabel(""),
		rows:         make([]flyableShipRow, 0),
		rowsFiltered: make([]flyableShipRow, 0),
		top:          makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	a.search = widget.NewEntry()
	a.search.SetPlaceHolder("Search type and class names")
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRows(-1)
	})
	a.search.OnChanged = func(s string) {
		a.filterRows(-1)
	}

	a.selectGroup = kxwidget.NewFilterChipSelectWithSearch("Class", []string{}, func(s string) {
		a.filterRows(-1)
	}, a.u.window)

	a.selectFlyable = kxwidget.NewFilterChipSelect("Flyable", []string{}, func(s string) {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRows(-1)
	}, a.u.window)
	a.grid = a.makeShipsGrid()

	// Signals
	a.u.currentCharacterExchanged.AddListener(
		func(ctx context.Context, c *app.Character) {
			a.character.Store(c)
			a.update(ctx)
		},
	)
	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.characterID {
			return
		}
		if arg.section == app.SectionCharacterSkills {
			a.update(ctx)
		}
	},
	)
	a.u.generalSectionChanged.AddListener(func(ctx context.Context, arg generalSectionUpdated) {
		characterID := characterIDOrZero(a.character.Load())
		if characterID == 0 {
			return
		}
		if arg.section == app.SectionEveTypes {
			a.update(ctx)
		}
	})
	return a
}

func (a *characterFlyableShips) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(
			a.top,
			a.search,
			container.NewHScroll(container.NewHBox(a.selectGroup, a.selectFlyable, a.sortButton)),
		),
		a.bottom,
		nil,
		nil,
		a.grid,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *characterFlyableShips) makeShipsGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newShipItem(a.u.eis)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			o := a.rowsFiltered[id]
			item := co.(*shipItem)
			item.Set(o.typeID, o.typeName, o.canFly)
		})
	g.OnSelected = func(id widget.GridWrapItemID) {
		defer g.UnselectAll()
		if id >= len(a.rowsFiltered) {
			return
		}
		o := a.rowsFiltered[id]
		a.u.ShowTypeInfoWindowWithCharacter(o.typeID, characterIDOrZero(a.character.Load()))
	}
	return g
}

func (a *characterFlyableShips) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	total := len(rows)
	group := a.selectGroup.Selected
	flyable := a.selectFlyable.Selected
	search := strings.ToLower(a.search.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

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
		var bottom string
		if total > 0 {
			bottom = fmt.Sprintf("Showing %d / %d ships", len(rows), total)
		} else {
			bottom = ""
		}
		if doSort {
			slices.SortFunc(rows, func(a, b flyableShipRow) int {
				var x int
				switch sortCol {
				case flyableColType:
					x = strings.Compare(a.typeName, b.typeName)
				case flyableColGroup:
					x = strings.Compare(a.groupName, b.groupName)
				}
				if dir == iwidget.SortAsc {
					return x
				} else {
					return -1 * x
				}
			})
		}

		fyne.Do(func() {
			a.selectGroup.SetOptions(groupOptions)
			a.selectFlyable.SetOptions(flyableOptions)
			a.rowsFiltered = rows
			a.bottom.SetText(bottom)
			a.grid.Refresh()
			a.grid.ScrollToTop()
		})
	}()

	// if characterID == 0 {
	// 	fyne.Do(func() {
	// 		a.ships = make([]*app.CharacterShipAbility, 0)
	// 		a.grid.Refresh()
	// 		a.searchBox.SetText("")
	// 		a.groupSelect.SetOptions([]string{})
	// 		a.flyableSelect.SetOptions([]string{})
	// 	})
	// 	return nil
	// }
	// search := fmt.Sprintf("%%%s%%", a.searchBox.Text)

	// ships := make([]*app.CharacterShipAbility, 0)
	// for _, o := range oo {
	// 	isSelectedGroup := a.groupSelected == "" || o.Group.Name == a.groupSelected
	// 	var isSelectedFlyable bool
	// 	switch a.flyableSelected {
	// 	case flyableCan:
	// 		isSelectedFlyable = o.CanFly
	// 	case flyableCanNot:
	// 		isSelectedFlyable = !o.CanFly
	// 	default:
	// 		isSelectedFlyable = true
	// 	}
	// 	if isSelectedGroup && isSelectedFlyable {
	// 		ships = append(ships, o)
	// 	}
	// }
	// g := set.Of[string]()
	// f := set.Of[string]()
	// for _, o := range ships {
	// 	g.Add(o.Group.Name)
	// 	if o.CanFly {
	// 		f.Add(flyableCan)
	// 	} else {
	// 		f.Add(flyableCanNot)
	// 	}
	// }
	// groups := slices.Collect(g.All())
	// slices.Sort(groups)
	// flyable := slices.Collect(f.All())
	// slices.Sort(flyable)
	// fyne.Do(func() {
	// 	a.groupSelect.SetOptions(groups)
	// 	a.flyableSelect.SetOptions(flyable)
	// 	a.foundText.SetText(fmt.Sprintf("%d found", len(ships)))
	// 	a.foundText.Show()
	// 	a.grid.Refresh()
	// })
	// return nil
}

func (a *characterFlyableShips) update(ctx context.Context) {
	clear := func() {
		fyne.Do(func() {
			a.rows = make([]flyableShipRow, 0)
			a.search.Disable()
			a.search.SetText("")
			a.selectGroup.SetOptions([]string{})
			a.selectFlyable.SetOptions([]string{})
			a.filterRows(-1)
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
		setTop(a.u.humanizeError(err), widget.DangerImportance)
	}

	ok1, err := a.u.eus.HasSection(ctx, app.SectionEveTypes)
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

	characterID := characterIDOrZero(a.character.Load())
	if characterID == 0 {
		clear()
		setTop("No character", widget.LowImportance)
		return
	}

	exists, err := a.u.cs.HasSection(ctx, characterID, app.SectionCharacterSkills)
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

	oo, err := a.u.cs.ListShipsAbilities(ctx, characterID)
	if err != nil {
		clear()
		reportError(err)
		return
	}
	rows := make([]flyableShipRow, 0)
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
		a.filterRows(-1)
	})
}

// assetIconCache caches the images for asset icons.
var shipImageCache xsync.Map[string, *image.RGBA]

type shipItemEIS interface {
	InventoryTypeRender(id int32, size int) (fyne.Resource, error)
}

// The shipItem widget is used to render items on the type info window.
type shipItem struct {
	widget.BaseWidget

	eis   shipItemEIS
	image *canvas.Image
	label *widget.Label
}

func newShipItem(eis shipItemEIS) *shipItem {
	upLeft := image.Point{0, 0}
	lowRight := image.Point{128, 128}
	image := canvas.NewImageFromImage(image.NewRGBA(image.Rectangle{upLeft, lowRight}))
	image.FillMode = canvas.ImageFillContain
	image.ScaleMode = defaultImageScaleMode
	image.CornerRadius = theme.InputRadiusSize()
	image.SetMinSize(fyne.NewSquareSize(128))
	w := &shipItem{
		image: image,
		label: widget.NewLabel("First line\nSecond Line\nThird Line"),
		eis:   eis,
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

	// TODO: Move grayscale feature into general package

	key := fmt.Sprintf("%d-%v", typeID, canFly)
	img, ok := shipImageCache.Load(key)
	if ok {
		w.image.Image = img
		w.image.Refresh()
		return
	}
	go func() {
		j, err := func() (image.Image, error) {
			r, err := w.eis.InventoryTypeRender(typeID, 256)
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
		shipImageCache.Store(key, img)

		fyne.Do(func() {
			w.image.Resource = nil
			w.image.Image = img
			w.image.Refresh()
		})
	}()
}

func (w *shipItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(container.NewPadded(w.image), w.label)
	return widget.NewSimpleRenderer(c)
}
