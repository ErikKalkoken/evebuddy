package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
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

// characterAttributes shows the attributes for the current character.
type characterAttributes struct {
	widget.BaseWidget

	rows      []characterAttributeRow
	character atomic.Pointer[app.Character]
	list      *widget.List
	footer    *widget.Label
	u         *baseUI
}

func newCharacterAttributes(u *baseUI) *characterAttributes {
	a := &characterAttributes{
		footer: newLabelWithTruncation(),
		u:      u,
	}
	a.list = a.makeAttributeList()
	a.ExtendBaseWidget(a)
	a.u.signals.CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.signals.CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.CharacterID {
			return
		}
		if arg.Section == app.SectionCharacterAttributes {
			a.update(ctx)
		}
	})
	return a
}

func (a *characterAttributes) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.footer, nil, nil, nil, a.list)
	return widget.NewSimpleRenderer(c)
}

func (a *characterAttributes) makeAttributeList() *widget.List {
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
	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	l.HideSeparators = true
	return l
}

func (a *characterAttributes) update(ctx context.Context) {
	var err error
	var total int64
	var attributes []characterAttributeRow
	characterID := characterIDOrZero(a.character.Load())
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionCharacterAttributes)
	if hasData {
		total2, attributes2, err2 := a.fetchData(ctx, characterID)
		if err2 != nil {
			slog.Error("Failed to refresh attributes UI", "err", err)
			err = err2
		} else {
			attributes = attributes2
			total = total2
		}
	}
	t, i := a.u.makeTopText(characterID, hasData, err, func() (string, widget.Importance) {
		return fmt.Sprintf("Total points: %d", total), widget.MediumImportance
	})
	fyne.Do(func() {
		a.footer.Text, a.footer.Importance = t, i
		a.footer.Refresh()
	})
	fyne.Do(func() {
		a.rows = attributes
		a.list.Refresh()
	})
}

func (a *characterAttributes) fetchData(ctx context.Context, characterID int64) (int64, []characterAttributeRow, error) {
	attributes := make([]characterAttributeRow, 0, 6)
	if characterID == 0 {
		return 0, attributes, nil
	}
	ca, err := a.u.cs.GetAttributes(ctx, characterID)
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
		fyne.NewSquareSize(app.IconUnitSize),
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
