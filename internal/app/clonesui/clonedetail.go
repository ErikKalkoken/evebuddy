package clonesui

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
	"github.com/ErikKalkoken/evebuddy/internal/app/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/app/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/app/xwindow"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func showCloneDetailWindow(u ui, jc *app.CharacterJumpClone2) {
	if jc == nil {
		return
	}
	title := fmt.Sprintf("Clone #%d", jc.CloneID)
	w, ok := u.GetOrCreateWindow(fmt.Sprintf("clone-%d-%d", jc.Character.ID, jc.ID), title, jc.Character.Name)
	if !ok {
		w.Show()
		return
	}
	location := makeLocationLabel(jc.Location.ToShort(), u.InfoWindow().ShowLocation)
	character := makeLinkLabelWithWrap(jc.Character.Name, func() {
		u.InfoWindow().Show(app.EveEntityCharacter, jc.Character.ID)
	})
	col := kxlayout.NewColumns(80)
	top := container.New(
		layout.NewCustomPaddedVBoxLayout(0),
		container.New(
			col,
			widget.NewLabelWithStyle("Location", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			location,
		),
		container.New(
			col,
			widget.NewLabelWithStyle("Character", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			character,
		),
	)
	implants := makeImplantsList(
		u.Character(),
		u.EVEImage(),
		u.InfoWindow().ShowType,
		jc.Character.ID,
		jc.CloneID,
		u.IsDeveloperMode(),
		w,
	)
	c := container.NewBorder(
		container.NewVBox(top, widget.NewSeparator()),
		nil,
		nil,
		nil,
		container.NewAppTabs(
			container.NewTabItem("Implants", implants),
		),
	)
	xwindow.Set(xwindow.Params{
		Title:   title,
		Content: c,
		Window:  w,
	})
	w.Show()
}

type implantsCS interface {
	GetJumpClone(ctx context.Context, characterID int64, cloneID int64) (*app.CharacterJumpClone, error)
}

type implantsEIS interface {
	InventoryTypeIconAsync(id int64, size int, setter func(r fyne.Resource))
}

func makeImplantsList(cs implantsCS, eis implantsEIS, showTypeInfo func(int64), characterID, cloneID int64, IsDeveloperMode bool, w fyne.Window) *widget.List {
	var implants []*app.CharacterJumpCloneImplant
	list := widget.NewList(
		func() int {
			return len(implants)
		},
		func() fyne.CanvasObject {
			return awidget.NewEntityListItem(false, eis.InventoryTypeIconAsync)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(implants) {
				return
			}
			r := implants[id]
			co.(*awidget.EntityListItem).Set(r.EveType.ID, r.EveType.Name)
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

func showRouteWindow(u ui, origin *app.EveSolarSystem, routePref string, r cloneRow) {
	if r.jc == nil {
		return
	}
	title := fmt.Sprintf("Route: %s -> %s", origin.Name, r.jc.Location.SolarSystemName())
	w, ok := u.GetOrCreateWindow(fmt.Sprintf("route-%s", r.id()), title, r.jc.Character.Name)
	if !ok {
		w.Show()
		return
	}
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

	var fromText []widget.RichTextSegment
	if origin != nil {
		fromText = origin.DisplayRichTextWithRegion()
	}
	from := xwidget.NewTappableRichText(fromText, func() {
		if origin != nil {
			u.InfoWindow().Show(app.EveEntitySolarSystem, origin.ID)
		}
	})
	from.Wrapping = fyne.TextWrapWord

	var toText []widget.RichTextSegment
	if v, ok := r.jc.Location.SolarSystem.Value(); ok {
		toText = v.DisplayRichTextWithRegion()
	}
	to := xwidget.NewTappableRichText(toText, func() {
		if v, ok := r.jc.Location.SolarSystem.Value(); ok {
			u.InfoWindow().Show(app.EveEntitySolarSystem, v.ID)
		}
	})
	to.Wrapping = fyne.TextWrapWord

	jumps := widget.NewLabel(fmt.Sprintf("%s (%s)", r.jumps(), routePref))
	col := kxlayout.NewColumns(firstColumnWidth)
	top := container.New(
		layout.NewCustomPaddedVBoxLayout(0),
		container.New(
			col,
			widget.NewLabelWithStyle("From", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			from,
		),
		container.New(
			col,
			widget.NewLabelWithStyle("To", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			to,
		),
		container.New(
			col,
			widget.NewLabelWithStyle("Jumps", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			jumps,
		),
	)
	c := container.NewBorder(
		container.NewVBox(top, widget.NewSeparator()),
		nil,
		nil,
		nil,
		list,
	)
	xwindow.Set(xwindow.Params{
		Title:   title,
		Content: c,
		Window:  w,
	})
	w.Show()
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
