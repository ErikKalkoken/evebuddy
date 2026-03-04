package ui

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type characterAugmentations struct {
	widget.BaseWidget

	character atomic.Pointer[app.Character]
	implants  []*app.CharacterImplant
	list      *widget.List
	top       *widget.Label
	u         *baseUI
}

func newCharacterAugmentations(u *baseUI) *characterAugmentations {
	a := &characterAugmentations{
		implants: make([]*app.CharacterImplant, 0),
		top:      newLabelWithWrapping(),
		u:        u,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeImplantList()
	a.u.currentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.characterID {
			return
		}
		if arg.section == app.SectionCharacterImplants {
			a.update(ctx)
		}
	})
	return a
}

func (a *characterAugmentations) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.list)
	return widget.NewSimpleRenderer(c)
}

func (a *characterAugmentations) makeImplantList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.implants)
		},
		func() fyne.CanvasObject {
			return newCharacterAugmentationItem(
				a.u.eis.InventoryTypeIconAsync,
				a.u.ShowTypeInfoWindowWithCharacter,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.implants) {
				return
			}
			co.(*characterAugmentationItem).set(a.implants[id])
		})

	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
	}
	l.HideSeparators = true
	return l
}

func (a *characterAugmentations) update(ctx context.Context) {
	var err error
	var implants []*app.CharacterImplant
	characterID := characterIDOrZero(a.character.Load())
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionCharacterImplants)
	if hasData {
		implants2, err2 := a.u.cs.ListImplants(ctx, characterID)
		if err2 != nil {
			slog.Error("Failed to refresh implants UI", "err", err)
			err = err2
		} else {
			implants = implants2
		}
	}
	t, i := a.u.makeTopText(characterID, hasData, err, func() (string, widget.Importance) {
		return fmt.Sprintf("%d implants", len(implants)), widget.MediumImportance
	})
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.implants = implants
		a.list.Refresh()
	})
}

type characterAugmentationItem struct {
	widget.BaseWidget

	iconInfo     *iwidget.TappableIcon
	iconMain     *canvas.Image
	loadTypeIcon loadFuncAsync
	name         *ttwidget.Label
	showType     func(int64, int64)
	slot         *widget.Label
}

func newCharacterAugmentationItem(
	loadTypeIcon loadFuncAsync,
	showType func(int64, int64),
) *characterAugmentationItem {
	iconMain := iwidget.NewImageFromResource(
		icons.BlankSvg,
		fyne.NewSquareSize(app.IconUnitSize*1.2),
	)
	iconInfo := iwidget.NewTappableIcon(
		theme.NewThemedResource(icons.InformationSlabCircleSvg),
		nil,
	)
	iconInfo.SetToolTip("Show information")
	name := ttwidget.NewLabel("placeholder")
	name.Truncation = fyne.TextTruncateEllipsis
	slot := widget.NewLabel("placeholder")
	slot.Truncation = fyne.TextTruncateEllipsis
	w := &characterAugmentationItem{
		iconInfo:     iconInfo,
		iconMain:     iconMain,
		loadTypeIcon: loadTypeIcon,
		name:         name,
		showType:     showType,
		slot:         slot,
	}
	w.ExtendBaseWidget(w)

	return w
}

func (w *characterAugmentationItem) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewBorder(
		nil,
		nil,
		w.iconMain,
		w.iconInfo,
		container.New(
			layout.NewCustomPaddedVBoxLayout(0),
			container.New(layout.NewCustomPaddedLayout(0, -p, 0, 0), w.name),
			container.New(layout.NewCustomPaddedLayout(-p, 0, 0, 0), w.slot),
		),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *characterAugmentationItem) set(o *app.CharacterImplant) {
	w.name.SetText(o.EveType.Name)
	w.name.SetToolTip(o.EveType.Description)
	w.slot.SetText(fmt.Sprintf("Slot %d", o.SlotNum))
	w.loadTypeIcon(o.EveType.ID, app.IconPixelSize, func(r fyne.Resource) {
		w.iconMain.Resource = r
		w.iconMain.Refresh()
	})
	w.iconInfo.OnTapped = func() {
		w.showType(o.EveType.ID, o.CharacterID)
	}

}
