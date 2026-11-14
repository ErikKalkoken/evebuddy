package screens

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// showSkillSetDetailWindow shows how a character meets each requirement of a skill set.
func showSkillSetDetailWindow(u baseUI, s app.SkillSet, r skillSetRow) {
	title := s.Name
	w, created := u.GetOrCreateWindow(fmt.Sprintf("skillset-%d-%d", s.ID, r.characterID), title, r.characterName)
	if !created {
		w.Show()
		return
	}
	status := widget.NewLabel(r.status())
	status.Importance = r.statusImportance()
	form := widget.NewForm(
		widget.NewFormItem("Character", ui.MakeCharacterActionLabel(r.characterID, r.characterName, u.InfoViewer().Show)),
		widget.NewFormItem("Status", status),
		widget.NewFormItem("Skills", widget.NewLabel(r.skillsDisplay())),
	)
	form.Orientation = widget.Adaptive

	bold := fyne.TextStyle{Bold: true}
	header := makeSkillSetDetailRow(
		layout.NewSpacer(),
		widget.NewLabelWithStyle("Required", fyne.TextAlignLeading, bold),
		widget.NewLabelWithStyle("Current", fyne.TextAlignTrailing, bold),
	)

	skillsFiltered := r.skills
	list := widget.NewList(
		func() int {
			return len(skillsFiltered)
		},
		func() fyne.CanvasObject {
			return newSkillSetSkillItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(skillsFiltered) {
				return
			}
			co.(*skillSetSkillItem).set(skillsFiltered[id], r.skillsKnown)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		list.UnselectAll()
		if id >= 0 && id < len(skillsFiltered) {
			u.InfoViewer().ShowType(skillsFiltered[id].typeID, r.characterID)
		}
	}
	hintLabel := widget.NewLabel("All skills met")
	hintLabel.Importance = widget.LowImportance
	hintLabel.Hide()

	filterButton := xwidget.NewSegmentedButton([]string{"All", "Missing"}, func(index int) {
		if index == 0 {
			skillsFiltered = r.skills
		} else {
			skillsFiltered = slices.DeleteFunc(slices.Clone(r.skills), func(x skillSetSkill) bool {
				return x.met()
			})
		}
		list.Refresh()
		list.ScrollToOffset(0)
		if len(skillsFiltered) == 0 {
			hintLabel.Show()
		} else {
			hintLabel.Hide()
		}
	})
	if r.status() == skillSetsStatusMissing {
		filterButton.SetSelected(1)
	}

	filterBox := container.NewPadded(filterButton)
	// Missing has no meaning without skill data.
	if !r.skillsKnown {
		filterBox.Hide()
	}
	// Not MakeDetailWindow's image, whose full-height column would narrow the list.
	avatarImage := xwidget.NewTappableImage(icons.BlankSvg, func() {
		u.InfoViewer().Show(&app.EveEntity{ID: r.characterID, Category: app.EveEntityCharacter})
	})
	avatarImage.SetFillMode(canvas.ImageFillContain)
	avatarImage.SetMinSize(fyne.NewSquareSize(64))
	u.SetCharacterAvatarAsync(r.characterID, avatarImage.SetResource)
	top := container.NewBorder(nil, nil, nil, container.NewPadded(container.NewVBox(avatarImage)), form)
	// List scrolls on its own so the header stays visible.
	c := container.NewBorder(
		container.NewVBox(top, widget.NewSeparator(), filterBox, header),
		nil,
		nil,
		nil,
		container.NewStack(list, container.NewCenter(hintLabel)),
	)
	ui.MakeDetailWindow(ui.MakeDetailWindowParams{
		Content: c,
		Title:   title,
		Window:  w,
	})
	w.Show()
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

	current *widget.Label
	icon    *widget.Icon
	name    *widget.Label
}

func newSkillSetSkillItem() *skillSetSkillItem {
	w := &skillSetSkillItem{
		current: widget.NewLabelWithStyle("", fyne.TextAlignTrailing, fyne.TextStyle{}),
		icon:    widget.NewIcon(nil),
		name:    widget.NewLabel(""),
	}
	w.name.Truncation = fyne.TextTruncateEllipsis
	w.ExtendBaseWidget(w)
	return w
}

func (w *skillSetSkillItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(makeSkillSetDetailRow(container.NewCenter(w.icon), w.name, w.current))
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
	w.name.SetText(x.name + " " + ihumanize.RomanLetter(x.requiredLevel))
	w.icon.SetResource(icon)
	w.current.SetText(current)
}
