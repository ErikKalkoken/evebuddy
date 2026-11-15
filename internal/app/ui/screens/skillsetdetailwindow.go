package screens

import (
	"context"
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/maniartech/signals"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// showSkillSetDetailWindow shows how a character meets each requirement of a skill set.
// The window is kept up to date through dataChanged.
func showSkillSetDetailWindow(u baseUI, s app.SkillSet, r skillSetRow, dataChanged *signals.SyncSignal[skillSetsData]) {
	w, created, onClosed := u.GetOrCreateWindowWithOnClosed(fmt.Sprintf("skillset-%d-%d", s.ID, r.characterID), "Skill Set", s.Name, r.characterName)
	if !created {
		w.Show()
		return
	}
	a := newSkillSetDetail(u, s.ID, r, dataChanged)
	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		a.stop()
	})
	ui.MakeDetailWindow(ui.MakeDetailWindowParams{
		Content: a,
		Title:   s.Name,
		Window:  w,
	})
	w.Show()
}

// skillSetDetail shows how a character meets each requirement of a skill set.
type skillSetDetail struct {
	widget.BaseWidget

	characterID    int64
	dataChanged    *signals.SyncSignal[skillSetsData]
	filterBox      *fyne.Container
	filterButton   *xwidget.SegmentedButton
	hintLabel      *widget.Label
	issueLabel     *widget.Label
	list           *widget.List
	row            skillSetRow
	setID          int64
	signalKey      string
	skillsFiltered []skillSetSkill
	skillsLabel    *widget.Label
	statusLabel    *widget.Label
	u              baseUI
}

func newSkillSetDetail(u baseUI, setID int64, r skillSetRow, dataChanged *signals.SyncSignal[skillSetsData]) *skillSetDetail {
	a := &skillSetDetail{
		characterID: r.characterID,
		dataChanged: dataChanged,
		hintLabel:   widget.NewLabel("All skills met"),
		issueLabel:  ui.NewLabelWithWrapping(""),
		setID:       setID,
		signalKey:   u.Signals().UniqueKey(),
		skillsLabel: widget.NewLabel(""),
		statusLabel: widget.NewLabel(""),
		u:           u,
	}
	a.ExtendBaseWidget(a)
	a.issueLabel.Importance = widget.DangerImportance
	a.issueLabel.Hide()
	a.hintLabel.Importance = widget.LowImportance
	a.hintLabel.Hide()

	a.list = widget.NewList(
		func() int {
			return len(a.skillsFiltered)
		},
		func() fyne.CanvasObject {
			return newSkillSetSkillItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.skillsFiltered) {
				return
			}
			co.(*skillSetSkillItem).set(a.skillsFiltered[id], a.row.skillsKnown)
		},
	)
	a.list.OnSelected = func(id widget.ListItemID) {
		a.list.UnselectAll()
		if id >= 0 && id < len(a.skillsFiltered) {
			u.InfoViewer().ShowType(a.skillsFiltered[id].typeID, a.characterID)
		}
	}

	a.filterButton = xwidget.NewSegmentedButton([]string{"All", "Missing"}, func(int) {
		a.filterSkills()
		a.list.ScrollToOffset(0)
	})
	a.filterBox = container.NewPadded(a.filterButton)

	a.set(r)
	if r.status() == skillSetsStatusMissing {
		a.filterButton.SetSelected(1)
	}
	dataChanged.AddListener(func(_ context.Context, d skillSetsData) {
		a.update(d)
	}, a.signalKey)
	return a
}

func (a *skillSetDetail) CreateRenderer() fyne.WidgetRenderer {
	character := ui.MakeCharacterActionLabel(
		a.row.characterID,
		a.row.characterName,
		a.u.InfoViewer().Show,
	)
	form := widget.NewForm(
		widget.NewFormItem("Character", character),
		widget.NewFormItem("Status", a.statusLabel),
		widget.NewFormItem("Skills", a.skillsLabel),
	)
	form.Orientation = widget.Adaptive

	bold := fyne.TextStyle{Bold: true}
	header := makeSkillSetDetailRow(
		layout.NewSpacer(),
		widget.NewLabelWithStyle("Required", fyne.TextAlignLeading, bold),
		widget.NewLabelWithStyle("Current", fyne.TextAlignTrailing, bold),
	)

	// Not MakeDetailWindow's image, whose full-height column would narrow the list.
	avatarImage := xwidget.NewTappableImage(icons.BlankSvg, func() {
		a.u.InfoViewer().Show(&app.EveEntity{ID: a.characterID, Category: app.EveEntityCharacter})
	})
	avatarImage.SetFillMode(canvas.ImageFillContain)
	avatarImage.SetMinSize(fyne.NewSquareSize(64))
	a.u.SetCharacterAvatarAsync(a.characterID, avatarImage.SetResource)
	top := container.NewBorder(nil, nil, nil, container.NewPadded(container.NewVBox(avatarImage)), form)
	// List scrolls on its own so the header stays visible.
	c := container.NewBorder(
		container.NewVBox(a.issueLabel, top, widget.NewSeparator(), a.filterBox, header),
		nil,
		nil,
		nil,
		container.NewStack(a.list, container.NewCenter(a.hintLabel)),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *skillSetDetail) stop() {
	a.dataChanged.RemoveListener(a.signalKey)
}

// update shows the skill set and character from d, or an issue when one is gone.
func (a *skillSetDetail) update(d skillSetsData) {
	i := slices.IndexFunc(d.sets, func(s app.SkillSet) bool {
		return s.ID == a.setID
	})
	if i == -1 {
		a.setIssue("Skill set has been deleted")
		return
	}
	j := slices.IndexFunc(d.characters, func(c skillSetCharacter) bool {
		return c.id == a.characterID
	})
	if j == -1 {
		a.setIssue("Character has been removed")
		return
	}
	a.set(newSkillSetRow(d.characters[j], d.sets[i]))
}

// set shows r and keeps the current filter.
func (a *skillSetDetail) set(r skillSetRow) {
	a.row = r
	a.issueLabel.Hide()
	a.statusLabel.Text = r.status()
	a.statusLabel.Importance = r.statusImportance()
	a.statusLabel.Refresh()
	a.skillsLabel.SetText(r.skillsDisplay())
	// Missing has no meaning without skill data.
	if r.skillsKnown {
		a.filterBox.Show()
	} else {
		a.filterBox.Hide()
	}
	a.filterSkills()
}

// setIssue shows s instead of the skills.
func (a *skillSetDetail) setIssue(s string) {
	a.row = skillSetRow{characterID: a.row.characterID, characterName: a.row.characterName}
	a.issueLabel.SetText(s)
	a.issueLabel.Show()
	a.statusLabel.SetText("")
	a.skillsLabel.SetText("")
	a.filterBox.Hide()
	a.filterSkills()
}

// filterSkills applies the selected filter to the skills of the current row.
func (a *skillSetDetail) filterSkills() {
	if a.filterButton.Selected == 0 {
		a.skillsFiltered = a.row.skills
	} else {
		a.skillsFiltered = slices.DeleteFunc(slices.Clone(a.row.skills), func(x skillSetSkill) bool {
			return x.met()
		})
	}
	a.list.Refresh()
	if len(a.skillsFiltered) == 0 && len(a.row.skills) > 0 {
		a.hintLabel.Show()
	} else {
		a.hintLabel.Hide()
	}
}

// makeSkillSetDetailRow returns a row of the skill set detail window.
// Fixed widths keep header and rows aligned.
func makeSkillSetDetailRow(icon, skill, current fyne.CanvasObject) *fyne.Container {
	return container.NewBorder(
		nil,
		nil,
		container.NewStack(xwidget.NewSpacer(fyne.NewSize(theme.IconInlineSize(), 0)), icon),
		container.NewStack(xwidget.NewSpacer(fyne.NewSize(60, 0)), current),
		skill,
	)
}

// skillSetSkillItem is a skill row in the skill set detail window.
type skillSetSkillItem struct {
	widget.BaseWidget

	currentLabel *widget.Label
	nameLabel    *widget.Label
	statusIcon   *widget.Icon
}

func newSkillSetSkillItem() *skillSetSkillItem {
	w := &skillSetSkillItem{
		currentLabel: widget.NewLabelWithStyle("", fyne.TextAlignTrailing, fyne.TextStyle{}),
		nameLabel:    widget.NewLabel(""),
		statusIcon:   widget.NewIcon(nil),
	}
	w.nameLabel.Truncation = fyne.TextTruncateEllipsis
	w.ExtendBaseWidget(w)
	return w
}

func (w *skillSetSkillItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(makeSkillSetDetailRow(container.NewCenter(w.statusIcon), w.nameLabel, w.currentLabel))
}

func (w *skillSetSkillItem) set(x skillSetSkill, skillsKnown bool) {
	var icon fyne.Resource
	var current string
	switch {
	case !skillsKnown:
		icon = theme.QuestionIcon()
		current = "?"
	case x.met():
		icon = theme.NewSuccessThemedResource(theme.ConfirmIcon())
		current = ihumanize.RomanLetter(x.activeLevel)
	case x.activeLevel == 0:
		icon = theme.NewErrorThemedResource(theme.CancelIcon())
		current = "–"
	default:
		icon = theme.NewErrorThemedResource(theme.CancelIcon())
		current = ihumanize.RomanLetter(x.activeLevel)
	}
	w.nameLabel.SetText(x.name + " " + ihumanize.RomanLetter(x.requiredLevel))
	w.statusIcon.SetResource(icon)
	w.currentLabel.SetText(current)
}
