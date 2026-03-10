package wallet

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
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

type CharacterLoyaltyPoints struct {
	widget.BaseWidget

	footer        *widget.Label
	character     atomic.Pointer[app.Character]
	columnSorter  *xwidget.ColumnSorter[characterLoyaltyPointsRow]
	list          *widget.List
	rows          []characterLoyaltyPointsRow
	rowsFiltered  []characterLoyaltyPointsRow
	searchBox     *widget.Entry
	selectFaction *kxwidget.FilterChipSelect
	sortButton    *xwidget.SortButton
	u             baseUI
}

const (
	characterLoyaltyPointsColCorporation = iota + 1
	characterLoyaltyPointsColPoints
)

func NewCharacterLoyaltyPoints(u baseUI) *CharacterLoyaltyPoints {
	columnSorter := xwidget.NewColumnSorter(xwidget.NewDataColumns([]xwidget.DataColumn[characterLoyaltyPointsRow]{{
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
		xwidget.SortAsc,
	)
	a := &CharacterLoyaltyPoints{
		columnSorter: columnSorter,
		footer:       ui.NewLabelWithTruncation(""),
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
		a.filterRowsAsync()
		a.list.ScrollToTop()
	}
	a.selectFaction = kxwidget.NewFilterChipSelect("Faction", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync()
	}, a.u.MainWindow())

	// signals
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.Update(ctx)
	},
	)
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDOrZero() != arg.CharacterID {
			return
		}
		if arg.Section != app.SectionCharacterLoyaltyPoints {
			return
		}
		a.Update(ctx)
	})
	return a
}

func (a *CharacterLoyaltyPoints) CreateRenderer() fyne.WidgetRenderer {
	var topBox *fyne.Container
	if a.u.IsMobile() {
		topBox = container.NewVBox(
			container.NewHBox(a.selectFaction, a.sortButton),
			a.searchBox,
		)
	} else {
		topBox = container.NewBorder(
			nil,
			nil,
			container.NewHBox(a.selectFaction, a.sortButton),
			nil,
			a.searchBox,
		)
	}
	c := container.NewBorder(
		topBox,
		a.footer,
		nil,
		nil,
		a.list,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterLoyaltyPoints) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newLoyaltyPointsListItem(
				ui.LoadEveEntityIconFunc(a.u.EVEImage()),
				func(o *app.EveEntity) {
					a.u.InfoWindow().Show(o)
				})
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			o := &app.EveEntity{
				Category: app.EveEntityCorporation,
				ID:       r.corporationID,
				Name:     r.corporationName,
			}
			co.(*loyaltyPointsListItem).set(o, r.points, false)
		},
	)
	l.OnSelected = func(_ widget.ListItemID) {
		defer l.UnselectAll()
	}
	return l
}

func (a *CharacterLoyaltyPoints) filterRowsAsync() {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
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

		footer := fmt.Sprintf("Showing %d / %d corporations", len(rows), totalRows)

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectFaction.SetOptions(factionOptions)
			a.rowsFiltered = rows
			a.list.Refresh()
		})
	}()
}

func (a *CharacterLoyaltyPoints) Update(ctx context.Context) {
	reset := func() {
		fyne.Do(func() {
			a.rows = xslices.Reset(a.rows)
			a.filterRowsAsync()
		})
	}
	setFooter := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.footer.Text = s
			a.footer.Importance = i
			a.footer.Refresh()
		})
	}

	character := a.character.Load()
	if character == nil {
		reset()
		setFooter("No character", widget.LowImportance)
		return
	}

	if !a.u.StatusCache().HasCharacterSection(character.ID, app.SectionCharacterLoyaltyPoints) {
		reset()
		setFooter("Loading data...", widget.WarningImportance)
		return
	}

	rows, err := a.fetchRows(ctx, character.ID)
	if err != nil {
		reset()
		setFooter("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync()
	})
}

func (a *CharacterLoyaltyPoints) fetchRows(ctx context.Context, characterID int64) ([]characterLoyaltyPointsRow, error) {
	entries, err := a.u.Character().ListLoyaltyPointEntries(ctx, characterID)
	if err != nil {
		return nil, err
	}
	entries = slices.DeleteFunc(entries, func(x *app.CharacterLoyaltyPointEntry) bool {
		return x.LoyaltyPoints == 0
	})

	var rows []characterLoyaltyPointsRow
	for _, o := range entries {
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

type loyaltyPointsListItem struct {
	widget.BaseWidget

	icon     *xwidget.TappableIcon
	entity   *ui.EveEntityListItem
	points   *widget.Label
	loadIcon ui.EveEntityIconLoader
	showInfo func(*app.EveEntity)
}

func newLoyaltyPointsListItem(loadIcon ui.EveEntityIconLoader, showInfo func(*app.EveEntity)) *loyaltyPointsListItem {
	w := &loyaltyPointsListItem{
		entity:   ui.NewEveEntityListItem(loadIcon),
		icon:     xwidget.NewTappableIcon(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil),
		points:   widget.NewLabel(""),
		showInfo: showInfo,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *loyaltyPointsListItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(w.points, w.icon),
		w.entity,
	)
	return widget.NewSimpleRenderer(c)
}

func (w *loyaltyPointsListItem) set(o *app.EveEntity, points int64, isHeader bool) {
	w.entity.IsAvatar = !isHeader
	w.entity.Set(o)
	w.points.Text = ihumanize.Comma(points)
	w.points.TextStyle.Bold = isHeader
	w.points.Refresh()
	w.icon.OnTapped = func() {
		w.showInfo(o)
	}
}
