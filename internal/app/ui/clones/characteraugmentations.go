package clones

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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type CharacterAugmentations struct {
	widget.BaseWidget

	character atomic.Pointer[app.Character]
	implants  []*app.CharacterImplant
	list      *widget.List
	top       *widget.Label
	s         baseUI
}

func NewCharacterAugmentations(s baseUI) *CharacterAugmentations {
	a := &CharacterAugmentations{
		top: ui.NewLabelWithWrapping(""),
		s:   s,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeImplantList()
	a.s.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.s.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDOrZero() != arg.CharacterID {
			return
		}
		if arg.Section == app.SectionCharacterImplants {
			a.update(ctx)
		}
	})
	return a
}

func (a *CharacterAugmentations) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.list)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterAugmentations) makeImplantList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.implants)
		},
		func() fyne.CanvasObject {
			return newCharacterAugmentationItem(
				a.s.EVEImage().InventoryTypeIconAsync,
				a.s.InfoWindow().ShowType,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.implants) {
				return
			}
			co.(*characterAugmentationItem).set(a.implants[id])
		})

	l.OnSelected = func(_ widget.ListItemID) {
		defer l.UnselectAll()
	}
	l.HideSeparators = true
	return l
}

func (a *CharacterAugmentations) update(ctx context.Context) {
	var err error
	var implants []*app.CharacterImplant
	characterID := a.character.Load().IDOrZero()
	hasData, err := a.s.Character().HasSection(ctx, characterID, app.SectionCharacterImplants)
	if err != nil {
		panic(err)
	}
	if hasData {
		implants2, err2 := a.s.Character().ListImplants(ctx, characterID)
		if err2 != nil {
			slog.Error("Failed to refresh implants UI", "err", err)
			err = err2
		} else {
			implants = implants2
		}
	}
	t, i := ui.MakeTopText(characterID, hasData, err, func() (string, widget.Importance) {
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

	iconInfo     *xwidget.TappableIcon
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
	iconMain := xwidget.NewImageFromResource(
		icons.BlankSvg,
		fyne.NewSquareSize(app.IconUnitSize*1.2),
	)
	iconInfo := xwidget.NewTappableIcon(
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
