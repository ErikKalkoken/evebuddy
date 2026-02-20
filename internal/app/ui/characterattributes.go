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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type attribute struct {
	icon   fyne.Resource
	name   string
	points int64
}

func (a attribute) isText() bool {
	return a.points == 0
}

// characterAttributes shows the attributes for the current character.
type characterAttributes struct {
	widget.BaseWidget

	attributes []attribute
	character  atomic.Pointer[app.Character]
	list       *widget.List
	footer     *widget.Label
	u          *baseUI
}

func newCharacterAttributes(u *baseUI) *characterAttributes {
	a := &characterAttributes{
		attributes: make([]attribute, 0),
		footer:     newLabelWithTruncation(),
		u:          u,
	}
	a.list = a.makeAttributeList()
	a.ExtendBaseWidget(a)
	a.u.currentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.characterID {
			return
		}
		if arg.section == app.SectionCharacterAttributes {
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
			return len(a.attributes)
		},
		func() fyne.CanvasObject {
			icon := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
			return container.NewHBox(
				icon, widget.NewLabel("placeholder"), layout.NewSpacer(), widget.NewLabel("88 points"))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.attributes) {
				return
			}
			q := a.attributes[id]
			hbox := co.(*fyne.Container).Objects
			name := hbox[1].(*widget.Label)
			name.SetText(q.name)

			icon := hbox[0].(*canvas.Image)
			if q.isText() {
				icon.Hide()
			} else {
				icon.Show()
				icon.Resource = q.icon
				icon.Refresh()
			}

			points := hbox[3].(*widget.Label)
			if q.isText() {
				points.Hide()
			} else {
				points.Show()
				points.SetText(fmt.Sprintf("%d points", q.points))
			}
		})

	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	l.HideSeparators = true
	return l
}

func (a *characterAttributes) update(ctx context.Context) {
	var err error
	var total int64
	attributes := make([]attribute, 0)
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
		a.attributes = attributes
		a.list.Refresh()
	})
}

func (a *characterAttributes) fetchData(ctx context.Context, characterID int64) (int64, []attribute, error) {
	attributes := make([]attribute, 0, 6)
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
	attributes[0] = attribute{
		icon:   resPerception,
		name:   "Perception",
		points: ca.Perception,
	}
	attributes[1] = attribute{
		icon:   resMemory,
		name:   "Memory",
		points: ca.Memory,
	}
	attributes[2] = attribute{
		icon:   resWillpower,
		name:   "Willpower",
		points: ca.Willpower,
	}
	attributes[3] = attribute{
		icon:   resIntelligence,
		name:   "Intelligence",
		points: ca.Intelligence,
	}
	attributes[4] = attribute{
		icon:   resCharisma,
		name:   "Charisma",
		points: ca.Charisma,
	}
	attributes[5] = attribute{
		name: fmt.Sprintf("Bonus Remaps Available: %s", ca.BonusRemaps.StringFunc("?", func(v int64) string {
			return fmt.Sprint(v)
		})),
	}
	total := ca.Charisma + ca.Intelligence + ca.Memory + ca.Perception + ca.Willpower
	return total, attributes, nil
}
