package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

type skillRow struct {
	activeLevel  int64
	description  string
	groupID      int64
	groupName    string
	name         string
	searchTarget string
	trainedLevel int64
	typeID       int64
}

type characterSkillCatalogue struct {
	widget.BaseWidget

	character      atomic.Pointer[app.Character]
	footer         *widget.Label
	levelBlocked   *theme.ErrorThemedResource
	levelTrained   *theme.PrimaryThemedResource
	levelUnTrained *theme.DisabledResource
	rows           []skillRow
	rowsFiltered   []skillRow
	search         *widget.Entry
	selectGroup    *kxwidget.FilterChipSelect
	skills         fyne.CanvasObject
	top            *widget.Label
	u              *baseUI
}

func newCharacterSkillCatalogue(u *baseUI) *characterSkillCatalogue {
	a := &characterSkillCatalogue{
		footer:         newLabelWithTruncation(),
		levelBlocked:   theme.NewErrorThemedResource(theme.MediaStopIcon()),
		levelTrained:   theme.NewPrimaryThemedResource(theme.MediaStopIcon()),
		levelUnTrained: theme.NewDisabledResource(theme.MediaStopIcon()),
		rows:           make([]skillRow, 0),
		rowsFiltered:   make([]skillRow, 0),
		top:            newLabelWithWrapping(),
		search:         widget.NewEntry(),
		u:              u,
	}
	a.ExtendBaseWidget(a)
	a.skills = a.makeSkillsGrid()

	a.search.OnChanged = func(s string) {
		a.filterRowsAsync()
	}
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRowsAsync()
	})
	a.search.PlaceHolder = "Search skills"

	a.selectGroup = kxwidget.NewFilterChipSelect("Group", []string{}, func(string) {
		a.filterRowsAsync()
	})

	// signals
	a.u.currentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.characterID {
			return
		}
		if arg.section == app.SectionCharacterSkills {
			a.update(ctx)
		}
	})
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

func (a *characterSkillCatalogue) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(
			a.top,
			container.NewBorder(nil, nil, a.selectGroup, nil, a.search),
		),
		a.footer,
		nil,
		nil,
		a.skills,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *characterSkillCatalogue) makeSkillsGrid() fyne.CanvasObject {
	length := func() int {
		return len(a.rowsFiltered)
	}
	makeCreateItem := func(trunc fyne.TextTruncation) func() fyne.CanvasObject {
		return func() fyne.CanvasObject {
			title := ttwidget.NewLabel("Capital Shipboard Compression Technology")
			title.Truncation = trunc
			c := container.NewBorder(
				nil,
				nil,
				newSkillLevel(),
				nil,
				title,
			)
			return c
		}
	}
	updateItem := func(id int, co fyne.CanvasObject) {
		if id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		row := co.(*fyne.Container).Objects
		label := row[0].(*ttwidget.Label)
		label.SetText(r.name)
		label.SetToolTip(r.description)
		level := row[1].(*skillLevel)
		level.Set(r.activeLevel, r.trainedLevel, 0)
	}
	makeOnSelected := func(unselectAll func()) func(int) {
		return func(id int) {
			defer unselectAll()
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			a.u.ShowTypeInfoWindowWithCharacter(r.typeID, characterIDOrZero(a.character.Load()))
		}
	}
	return makeGridOrList(a.u.isMobile, length, makeCreateItem, updateItem, makeOnSelected)
}

func (a *characterSkillCatalogue) filterRowsAsync() {
	total := len(a.rows)
	rows := slices.Clone(a.rows)
	group := a.selectGroup.Selected
	search := strings.ToLower(a.search.Text)

	go func() {
		if group != "" {
			rows = slices.DeleteFunc(rows, func(r skillRow) bool {
				return r.groupName != group
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r skillRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}

		slices.SortFunc(rows, func(a, b skillRow) int {
			return strings.Compare(a.name, b.name)
		})

		groupOptions := xslices.Map(rows, func(r skillRow) string {
			return r.groupName
		})
		footer := fmt.Sprintf("Showing %d / %d skills", len(rows), total)

		fyne.Do(func() {
			a.footer.SetText(footer)
			a.selectGroup.SetOptions(groupOptions)
			a.rowsFiltered = rows
			a.skills.Refresh()
		})
	}()
}

func (a *characterSkillCatalogue) update(ctx context.Context) {
	setTop := func(t string, i widget.Importance) {
		fyne.Do(func() {
			a.top.Text = t
			a.top.Importance = i
			a.top.Refresh()
		})
	}

	clear := func() {
		fyne.Do(func() {
			clear(a.rows)
			clear(a.rowsFiltered)
		})
	}

	if !a.u.scs.HasGeneralSection(app.SectionEveTypes) {
		clear()
		setTop("No data yet", widget.WarningImportance)
		return
	}

	c := a.character.Load()
	if c == nil {
		clear()
		setTop("No character", widget.LowImportance)
		return
	}
	characterID := characterIDOrZero(c)
	if !a.u.scs.HasCharacterSection(characterID, app.SectionCharacterSkills) {
		clear()
		setTop("No data yet", widget.WarningImportance)
		return
	}
	oo, err := a.u.cs.ListSkills(ctx, characterID)
	if err != nil {
		slog.Error("Failed to refresh skill catalogue UI", "err", err)
		clear()
		setTop("ERROR: "+a.u.humanizeError(err), widget.DangerImportance)
		return
	}

	var rows []skillRow
	for _, o := range oo {
		rows = append(rows, skillRow{
			activeLevel:  o.ActiveSkillLevel,
			description:  o.Type.Description,
			groupID:      o.Type.Group.ID,
			groupName:    o.Type.Group.Name,
			name:         o.Type.Name,
			searchTarget: strings.ToLower(o.Type.Name),
			trainedLevel: o.TrainedSkillLevel,
			typeID:       o.Type.ID,
		})
	}

	top := fmt.Sprintf("%s Total SP (%s Unallocated)",
		c.TotalSP.StringFunc("?", func(v int64) string {
			return ihumanize.Comma(v)
		}), c.UnallocatedSP.StringFunc("?", func(v int64) string {
			return ihumanize.Comma(v)
		}),
	)
	setTop(top, widget.MediumImportance)

	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync()
	})
}

// makeGridOrList makes and returns a GridWrap on desktop and a List on mobile.
//
// This allows the grid items to render nicely as list on mobile and also enable truncation.
func makeGridOrList(isMobile bool, length func() int, makeCreateItem func(trunc fyne.TextTruncation) func() fyne.CanvasObject, updateItem func(id int, co fyne.CanvasObject), makeOnSelected func(unselectAll func()) func(int)) fyne.CanvasObject {
	var w fyne.CanvasObject
	if isMobile {
		w = widget.NewList(length, makeCreateItem(fyne.TextTruncateEllipsis), updateItem)
		l := w.(*widget.List)
		l.OnSelected = makeOnSelected(func() {
			l.UnselectAll()
		})
	} else {
		w = widget.NewGridWrap(length, makeCreateItem(fyne.TextTruncateOff), updateItem)
		g := w.(*widget.GridWrap)
		g.OnSelected = makeOnSelected(func() {
			g.UnselectAll()
		})
	}
	return w
}
