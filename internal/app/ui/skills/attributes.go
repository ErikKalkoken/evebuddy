package skills

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type characterAttributeRow struct {
	icon   fyne.Resource
	name   string
	points int64
}

func (a characterAttributeRow) isComment() bool {
	return a.points == 0
}

// Attributes shows the attributes for the current character.
type Attributes struct {
	widget.BaseWidget

	rows      []characterAttributeRow
	character atomic.Pointer[app.Character]
	list      *widget.List
	top       *widget.Label
	u         baseUI
}

func NewAttributes(s baseUI) *Attributes {
	a := &Attributes{
		top: ui.NewLabelWithTruncation(""),
		u:   s,
	}
	a.list = a.makeAttributeList()
	a.ExtendBaseWidget(a)
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDOrZero() != arg.CharacterID {
			return
		}
		if arg.Section == app.SectionCharacterAttributes {
			a.update(ctx)
		}
	})
	return a
}

func (a *Attributes) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.list)
	return widget.NewSimpleRenderer(c)
}

func (a *Attributes) makeAttributeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rows)
		},
		func() fyne.CanvasObject {
			return newCharacterAttributeItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rows) {
				return
			}
			co.(*characterAttributeItem).set(a.rows[id])
		},
	)
	l.OnSelected = func(_ widget.ListItemID) {
		l.UnselectAll()
	}
	l.HideSeparators = true
	return l
}

func (a *Attributes) update(ctx context.Context) {
	reset := func() {
		fyne.Do(func() {
			a.rows = xslices.Reset(a.rows)
			a.list.Refresh()
		})
	}
	setTop := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.top.Text, a.top.Importance = s, i
			a.top.Refresh()
		})
	}
	characterID := a.character.Load().IDOrZero()
	if characterID == 0 {
		reset()
		setTop("No character", widget.LowImportance)
		return
	}

	hasData, err := a.u.Character().HasSection(ctx, characterID, app.SectionCharacterAttributes)
	if err != nil {
		reset()
		setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	if !hasData {
		reset()
		setTop("No data", widget.WarningImportance)
		return
	}

	total, attributes, err := a.fetchData(ctx, characterID)
	if err != nil {
		reset()
		setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	setTop(fmt.Sprintf("Total points: %d", total), widget.MediumImportance)
	fyne.Do(func() {
		a.rows = attributes
		a.list.Refresh()
	})
}

func (a *Attributes) fetchData(ctx context.Context, characterID int64) (int64, []characterAttributeRow, error) {
	attributes := make([]characterAttributeRow, 0, 6)
	if characterID == 0 {
		return 0, attributes, nil
	}
	ca, err := a.u.Character().GetAttributes(ctx, characterID)
	if errors.Is(err, app.ErrNotFound) {
		return 0, attributes, nil
	} else if err != nil {
		return 0, attributes, err
	}
	resPerception := eveicon.FromName(eveicon.Perception)
	resMemory := eveicon.FromName(eveicon.Memory)
	resWillpower := eveicon.FromName(eveicon.Willpower)
	resIntelligence := eveicon.FromName(eveicon.Intelligence)
	resCharisma := eveicon.FromName(eveicon.Charisma)
	attributes = attributes[:6]
	attributes[0] = characterAttributeRow{
		icon:   resPerception,
		name:   "Perception",
		points: ca.Perception,
	}
	attributes[1] = characterAttributeRow{
		icon:   resMemory,
		name:   "Memory",
		points: ca.Memory,
	}
	attributes[2] = characterAttributeRow{
		icon:   resWillpower,
		name:   "Willpower",
		points: ca.Willpower,
	}
	attributes[3] = characterAttributeRow{
		icon:   resIntelligence,
		name:   "Intelligence",
		points: ca.Intelligence,
	}
	attributes[4] = characterAttributeRow{
		icon:   resCharisma,
		name:   "Charisma",
		points: ca.Charisma,
	}
	attributes[5] = characterAttributeRow{
		name: fmt.Sprintf("Bonus Remaps Available: %s", ca.BonusRemaps.StringFunc("?", func(v int64) string {
			return fmt.Sprint(v)
		})),
	}
	total := ca.Charisma + ca.Intelligence + ca.Memory + ca.Perception + ca.Willpower
	return total, attributes, nil
}

type characterAttributeItem struct {
	widget.BaseWidget

	icon   *canvas.Image
	name   *widget.Label
	points *widget.Label
}

func newCharacterAttributeItem() *characterAttributeItem {
	icon := xwidget.NewImageFromResource(
		icons.BlankSvg,
		fyne.NewSquareSize(ui.IconUnitSize),
	)
	w := &characterAttributeItem{
		icon:   icon,
		name:   widget.NewLabel(""),
		points: widget.NewLabel(""),
	}
	w.name.Truncation = fyne.TextTruncateEllipsis
	w.ExtendBaseWidget(w)

	return w
}

func (w *characterAttributeItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		nil,
		nil,
		w.icon,
		w.points,
		w.name,
	)
	return widget.NewSimpleRenderer(c)
}

func (w *characterAttributeItem) set(r characterAttributeRow) {
	w.name.SetText(r.name)

	if r.isComment() {
		w.icon.Hide()
		w.points.Hide()
		return
	}
	w.icon.Show()
	w.icon.Resource = r.icon
	w.icon.Refresh()
	w.points.SetText(fmt.Sprintf("%d points", r.points))
	w.points.Show()
}
