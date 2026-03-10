package clones

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/xwindow"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func showCloneDetailWindow(u ui, r cloneRow, origin *app.EveSolarSystem, routePref app.EveRoutePreference) {
	if r.jc == nil {
		return
	}

	title := fmt.Sprintf("Clone #%d", r.jc.CloneID)
	w, ok := u.GetOrCreateWindow(fmt.Sprintf("clone-%d-%d", r.jc.Character.ID, r.jc.ID), title, r.jc.Character.Name)
	if !ok {
		w.Show()
		return
	}
	hasOrigin := origin != nil
	hasRoute := len(r.route) > 0

	var jumps string
	if !hasOrigin {
		jumps = "Not calculated"
	} else if !hasRoute {
		jumps = "No route found"
	} else {
		jumps = fmt.Sprintf("%s (%s)", r.jumps(), routePref.String())
	}

	location := xwidget.NewTappableRichText(r.jc.Location.DisplayRichText(), func() {
		u.InfoWindow().ShowLocation(r.jc.Location.ID)
	})
	character := makeLinkLabelWithWrap(r.jc.Character.Name, func() {
		u.InfoWindow().Show(app.EveEntityCharacter, r.jc.Character.ID)
	})

	var originInfo fyne.CanvasObject
	if hasOrigin {
		originInfo = xwidget.NewTappableRichText(origin.DisplayRichText(), func() {
			u.InfoWindow().Show(app.EveEntitySolarSystem, origin.ID)
		})
	} else {
		l := widget.NewLabel("No origin")
		l.Importance = widget.WarningImportance
		originInfo = l
	}

	col := kxlayout.NewColumns(80)
	top := container.New(
		layout.NewCustomPaddedVBoxLayout(0),
		container.New(
			col,
			widget.NewLabelWithStyle("Character", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			character,
		),
		container.New(
			col,
			widget.NewLabelWithStyle("Location", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			location,
		),
		container.New(
			col,
			widget.NewLabelWithStyle("Origin", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			originInfo,
		),
		container.New(
			col,
			widget.NewLabelWithStyle("Jumps", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(jumps),
		),
		container.New(
			col,
			widget.NewLabelWithStyle("Implants", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(fmt.Sprint(r.jc.ImplantsCount)),
		),
	)
	var implants fyne.CanvasObject
	if r.jc.ImplantsCount > 0 {
		implants = makeImplantsList(
			u.Character(),
			u.EVEImage(),
			u.InfoWindow().ShowType,
			r.jc.Character.ID,
			r.jc.CloneID,
			u.IsDeveloperMode(),
			w,
		)
	} else {
		l := widget.NewLabel("No implants")
		l.Importance = widget.LowImportance
		implants = l
	}

	var route fyne.CanvasObject
	if hasRoute {
		route = makeRoute(u, r)
	} else {
		l := widget.NewLabel("No data")
		l.Importance = widget.LowImportance
		route = l
	}
	tabs := container.NewAppTabs(
		container.NewTabItem("Implants", implants),
		container.NewTabItem("Route", route),
	)
	if r.jc.ImplantsCount == 0 {
		tabs.DisableIndex(0)
	}
	if !hasRoute {
		tabs.DisableIndex(1)
	} else {
		tabs.SelectIndex(1)
	}
	c := container.NewBorder(
		container.NewVBox(top, widget.NewSeparator()),
		nil,
		nil,
		nil,
		tabs,
	)
	xwindow.Set(xwindow.Params{
		Title:   title,
		Content: c,
		Window:  w,
	})
	w.Show()
}

func makeLinkLabelWithWrap(text string, action func()) *widget.Hyperlink {
	x := widget.NewHyperlink(text, nil)
	x.OnTapped = action
	x.Wrapping = fyne.TextWrapWord
	return x
}

type implantsCS interface {
	GetJumpClone(ctx context.Context, characterID int64, cloneID int64) (*app.CharacterJumpClone, error)
}

func makeImplantsList(cs implantsCS, eis awidget.EveEntityEIS, showTypeInfo func(int64), characterID, cloneID int64, IsDeveloperMode bool, w fyne.Window) *widget.List {
	var implants []*app.CharacterJumpCloneImplant
	list := widget.NewList(
		func() int {
			return len(implants)
		},
		func() fyne.CanvasObject {
			character := awidget.NewEveEntityListItem(awidget.LoadEveEntityIconFunc(eis))
			character.IsAvatar = true
			return character
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(implants) {
				return
			}
			r := implants[id]
			co.(*awidget.EveEntityListItem).Set(r.EveType.EveEntity())
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if id >= len(implants) {
			return
		}
		showTypeInfo(implants[id].EveType.ID)
	}

	go func() {
		clone, err := cs.GetJumpClone(context.Background(), characterID, cloneID)
		if err != nil {
			slog.Error("show clone", "error", err)
			xdialog.ShowErrorAndLog("failed to load clone", err, IsDeveloperMode, w)
			return
		}
		fyne.Do(func() {
			implants = clone.Implants
			list.Refresh()
		})
	}()
	return list
}

func makeRoute(u ui, r cloneRow) *widget.List {
	list := widget.NewList(
		func() int {
			return len(r.route)
		},
		func() fyne.CanvasObject {
			return newRouteListItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(r.route) {
				return
			}
			co.(*routeListItem).set(id, r.route[id])
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if id >= len(r.route) {
			return
		}
		s := r.route[id]
		u.InfoWindow().Show(app.EveEntitySolarSystem, s.ID)
	}
	return list
}

type routeListItem struct {
	widget.BaseWidget

	number *widget.Label
	system *xwidget.RichText
}

func newRouteListItem() *routeListItem {
	w := &routeListItem{
		number: widget.NewLabel(""),
		system: xwidget.NewRichText(),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *routeListItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.New(kxlayout.NewColumns(firstColumnWidth),
		w.number,
		w.system,
	)
	return widget.NewSimpleRenderer(c)
}

func (w *routeListItem) set(num int, s *app.EveSolarSystem) {
	w.number.SetText(fmt.Sprint(num))
	w.system.Set(s.DisplayRichTextWithRegion())
}
