package ui

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type characterLoyaltyPointsRow struct {
	characterID     int64
	corporationID   int64
	corporationName string
	factionID       int64
	factionName     string
	points          int64
	searchTarget    string
}

type characterLoyaltyPoints struct {
	widget.BaseWidget

	bottom        *widget.Label
	character     atomic.Pointer[app.Character]
	columnSorter  *iwidget.ColumnSorter[characterLoyaltyPointsRow]
	list          *widget.List
	rows          []characterLoyaltyPointsRow
	rowsFiltered  []characterLoyaltyPointsRow
	searchBox     *widget.Entry
	selectFaction *kxwidget.FilterChipSelect
	sortButton    *iwidget.SortButton
	top           *widget.Label
	u             *baseUI
}

const (
	characterLoyaltyPointsColCorporation = iota + 1
	characterLoyaltyPointsColPoints
)

func newCharacterLoyaltyPoints(u *baseUI) *characterLoyaltyPoints {
	columnSorter := iwidget.NewColumnSorter(iwidget.NewDataColumns([]iwidget.DataColumn[characterLoyaltyPointsRow]{{
		ID:    characterLoyaltyPointsColCorporation,
		Label: "Corporation",
		Sort: func(a, b characterLoyaltyPointsRow) int {
			return strings.Compare(a.corporationName, b.corporationName)
		},
	}, {
		ID:    characterLoyaltyPointsColPoints,
		Label: "Points",
		Sort: func(a, b characterLoyaltyPointsRow) int {
			return cmp.Compare(a.points, b.points)
		},
	}}),
		characterLoyaltyPointsColCorporation,
		iwidget.SortAsc,
	)
	a := &characterLoyaltyPoints{
		columnSorter: columnSorter,
		rows:         make([]characterLoyaltyPointsRow, 0),
		top:          widget.NewLabel(""),
		bottom:       widget.NewLabel(""),
		u:            u,
	}
	a.list = a.makeList()
	a.ExtendBaseWidget(a)

	a.searchBox = widget.NewEntry()
	a.searchBox.SetPlaceHolder("Search corporations")
	a.searchBox.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.searchBox.SetText("")
	})
	a.searchBox.OnChanged = func(s string) {
		if len(s) == 1 {
			return
		}
		a.filterRows()
		a.list.ScrollToTop()
	}
	a.selectFaction = kxwidget.NewFilterChipSelect("Faction", []string{}, func(string) {
		a.filterRows()
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRows()
	}, a.u.window)

	// signals
	a.u.currentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	},
	)
	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.characterID {
			return
		}
		if arg.section != app.SectionCharacterLoyaltyPoints {
			return
		}
		a.update(ctx)
	})
	return a
}

func (a *characterLoyaltyPoints) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(
			a.top,
			container.NewHBox(a.selectFaction, a.sortButton),
			a.searchBox,
		),
		a.bottom,
		nil,
		nil,
		a.list,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *characterLoyaltyPoints) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			icon1 := iwidget.NewImageFromResource(icons.Corporationplaceholder64Png,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			icon2 := iwidget.NewTappableIcon(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil)
			name := widget.NewLabel("Template")
			name.Truncation = fyne.TextTruncateEllipsis
			points := widget.NewLabel("Template")
			return container.NewBorder(
				nil,
				nil,
				icon1,
				container.NewHBox(points, icon2),
				name,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			box := co.(*fyne.Container).Objects
			name := box[0].(*widget.Label)
			icon1 := box[1].(*canvas.Image)
			a.u.eis.CorporationLogoAsync(r.corporationID, app.IconPixelSize, func(r fyne.Resource) {
				icon1.Resource = r
				icon1.Refresh()
			})
			name.SetText(r.corporationName)
			hbox := box[2].(*fyne.Container).Objects
			points := hbox[0].(*widget.Label)
			points.SetText(ihumanize.Comma(r.points))
			icon2 := hbox[1].(*iwidget.TappableIcon)
			icon2.OnTapped = func() {
				a.u.ShowInfoWindow(app.EveEntityCorporation, r.corporationID)
			}
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
	}
	return l
}

func (a *characterLoyaltyPoints) filterRows() {
	rows := slices.Clone(a.rows)
	total := len(rows)
	search := strings.ToLower(a.searchBox.Text)
	faction := a.selectFaction.Selected
	sortCol, dir, doSort := a.columnSorter.CalcSort(-1)

	go func() {
		if faction != "" {
			rows = slices.DeleteFunc(rows, func(r characterLoyaltyPointsRow) bool {
				return r.factionName != faction
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r characterLoyaltyPointsRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		factionOptions := xslices.Map(rows, func(r characterLoyaltyPointsRow) string {
			return r.factionName
		})
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		var bottom string
		if total > 0 {
			bottom = fmt.Sprintf("Showing %d / %d corporations", len(rows), total)
		} else {
			bottom = ""
		}
		fyne.Do(func() {
			a.bottom.SetText(bottom)
			a.selectFaction.SetOptions(factionOptions)
			a.rowsFiltered = rows
			a.list.Refresh()
		})
	}()
}

func (a *characterLoyaltyPoints) update(ctx context.Context) {
	clear := func() {
		fyne.Do(func() {
			clear(a.rowsFiltered)
			clear(a.rows)
			a.filterRows()
		})
	}
	showTop := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.top.Text = s
			a.top.Importance = i
			a.top.Refresh()
			a.top.Show()
		})
	}

	character := a.character.Load()
	if character == nil {
		clear()
		showTop("No character", widget.LowImportance)
		return
	}

	if !a.u.scs.HasCharacterSection(character.ID, app.SectionCharacterLoyaltyPoints) {
		clear()
		showTop("Loading data...", widget.WarningImportance)
		return
	}

	rows, err := a.fetchRows(ctx, character.ID)
	if err != nil {
		clear()
		showTop("ERROR: "+a.u.humanizeError(err), widget.DangerImportance)
		return
	}
	fyne.Do(func() {
		a.top.Hide()
		a.rows = rows
		a.filterRows()
	})
}

func (a *characterLoyaltyPoints) fetchRows(ctx context.Context, characterID int64) ([]characterLoyaltyPointsRow, error) {
	oo, err := a.u.cs.ListLoyaltyPointEntries(ctx, characterID)
	if err != nil {
		return nil, err
	}
	var rows []characterLoyaltyPointsRow
	for _, o := range oo {
		r := characterLoyaltyPointsRow{
			characterID:     characterID,
			corporationID:   o.Corporation.ID,
			corporationName: o.Corporation.Name,
			points:          o.LoyaltyPoints,
			searchTarget:    strings.ToLower(o.Corporation.Name),
		}
		if f, ok := o.Faction.Value(); ok {
			r.factionID = f.ID
			r.factionName = f.Name
		}
		rows = append(rows, r)
	}
	return rows, nil
}
