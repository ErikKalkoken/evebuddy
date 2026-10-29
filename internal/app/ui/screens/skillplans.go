package screens

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
)

type skillPlanRow struct {
	group     string
	character characterservice.SkillPlanCharacter
}

type SkillPlans struct {
	widget.BaseWidget
	u              baseUI
	plans          []app.SkillPlan
	selected       app.SkillPlan
	characters     []characterservice.SkillPlanCharacter
	rows           []skillPlanRow
	selectedID     int64
	planSelect     *widget.Select
	tagSelect      *widget.Select
	qualifiedOnly  *widget.Check
	search         *widget.Entry
	edit           *widget.Button
	remove         *widget.Button
	summary        *widget.Label
	detailTitle    *widget.Label
	footer         *widget.Label
	results        *widget.List
	details        *widget.List
	detailSkills   []app.SkillPlanSkillStatus
	detailKnown    bool
	hasCharacter   bool
	body           fyne.CanvasObject
	updateRun      xsync.LatestRun
	checkRun       xsync.LatestRun
	settingOptions bool
}

func NewSkillPlans(u baseUI) *SkillPlans {
	a := &SkillPlans{
		u:           u,
		summary:     ui.NewLabelWithWrapping("Select a plan to compare active skill levels across all characters."),
		detailTitle: ui.NewLabelWithWrapping("Plan requirements"),
		footer:      ui.NewLabelWithWrapping(""),
	}
	a.ExtendBaseWidget(a)
	a.planSelect = widget.NewSelect(nil, func(name string) {
		if a.settingOptions {
			return
		}
		for _, p := range a.plans {
			if p.Name == name {
				a.selectPlan(p)
				return
			}
		}
	})
	a.planSelect.PlaceHolder = "Select a skill plan"
	a.tagSelect = widget.NewSelect(nil, func(string) { a.filterRows() })
	a.tagSelect.PlaceHolder = "All account tags"
	a.qualifiedOnly = widget.NewCheck("Qualified only", func(bool) { a.filterRows() })
	a.search = widget.NewEntry()
	a.search.PlaceHolder = "Search characters or account tags"
	a.search.OnChanged = func(string) { a.filterRows() }
	if !u.IsMobile() {
		a.edit = widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() { a.showEditor(a.selected) })
		a.remove = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), a.deleteSelected)
		a.edit.Disable()
		a.remove.Disable()
	}
	a.results = widget.NewList(
		func() int { return len(a.rows) },
		func() fyne.CanvasObject {
			character := ui.NewEveEntityListItem(u.EVEImage().EveEntityLogoAsync)
			character.IsAvatar = true
			character.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(
				character,
				ui.NewLabelWithTruncation(""),
				ui.NewLabelWithTruncation(""),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rows) {
				return
			}
			r := a.rows[id]
			c := r.character
			objects := co.(*fyne.Container).Objects
			objects[0].(*ui.EveEntityListItem).Set2(c.Character.ID, c.Character.Name, app.EveEntityCharacter)
			objects[1].(*widget.Label).SetText("Account tag: " + r.group)
			status := fmt.Sprintf("Missing requirements (%d/%d met)", c.MetCount(), len(c.Skills))
			importance := widget.WarningImportance
			if !c.SkillsKnown {
				status = "Unknown - skills have not been synced"
			} else if c.Qualified() {
				status = fmt.Sprintf("Meets plan (%d/%d)", len(c.Skills), len(c.Skills))
				importance = widget.SuccessImportance
			}
			label := objects[2].(*widget.Label)
			label.Importance = importance
			label.SetText(status)
		},
	)
	a.results.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(a.rows) {
			return
		}
		c := a.rows[id].character
		a.selectedID = c.Character.ID
		a.showCharacter(&c)
	}
	a.details = widget.NewList(
		func() int { return len(a.detailSkills) },
		func() fyne.CanvasObject {
			name := ui.NewLabelWithTruncation("")
			level := widget.NewLabel("")
			level.Alignment = fyne.TextAlignTrailing
			status := ui.NewLabelWithTruncation("")
			// List measures the template before UpdateItem, including visible empty labels.
			if !a.hasCharacter {
				status.Hide()
			}
			return container.NewVBox(
				container.NewBorder(nil, nil, nil, level, name),
				status,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.detailSkills) {
				return
			}
			s := a.detailSkills[id]
			objects := co.(*fyne.Container).Objects
			header := objects[0].(*fyne.Container).Objects
			header[0].(*widget.Label).SetText(s.Requirement.Name)
			header[1].(*widget.Label).SetText(fmt.Sprintf("Required: %d", s.Requirement.Level))
			status := objects[1].(*widget.Label)
			if a.hasCharacter {
				var text string
				if !a.detailKnown {
					text = "Active: unknown"
				} else {
					text = fmt.Sprintf("Active: %d", s.ActiveLevel)
					if s.Met() {
						text += " | Met"
					} else {
						text += fmt.Sprintf(" | Missing %d level(s)", int64(s.Requirement.Level)-s.ActiveLevel)
					}
				}
				status.SetText(text)
				status.Show()
			} else {
				status.SetText("")
				status.Hide()
			}
		},
	)
	a.details.OnSelected = func(id widget.ListItemID) {
		a.details.UnselectAll()
		if id >= 0 && id < len(a.detailSkills) {
			u.InfoViewer().ShowType(a.detailSkills[id].Requirement.TypeID, a.selectedID)
		}
	}
	left := container.NewBorder(widget.NewLabel("Characters by account tag"), nil, nil, nil, a.results)
	right := container.NewBorder(a.detailTitle, nil, nil, nil, a.details)
	if u.IsMobile() {
		a.body = container.NewVSplit(left, right)
	} else {
		a.body = container.NewHSplit(left, right)
	}
	u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) { a.update(ctx) })
	u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) { a.update(ctx) })
	u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) { a.update(ctx) })
	u.Signals().TagsChanged.AddListener(func(ctx context.Context, _ struct{}) { a.update(ctx) })
	u.Signals().CharacterSectionUpdated.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if arg.Section == app.SectionCharacterSkills {
			a.update(ctx)
		}
	})
	return a
}

func (a *SkillPlans) CreateRenderer() fyne.WidgetRenderer {
	reset := widget.NewButtonWithIcon("Reset", theme.CancelIcon(), func() {
		a.tagSelect.ClearSelected()
		a.qualifiedOnly.SetChecked(false)
		a.search.SetText("")
	})
	var controls fyne.CanvasObject
	if a.u.IsMobile() {
		controls = container.NewVBox(a.planSelect, container.NewBorder(nil, nil, nil, reset, a.tagSelect), a.qualifiedOnly, a.search)
	} else {
		buttons := container.NewHBox(
			widget.NewButtonWithIcon("New / Import", theme.ContentAddIcon(), func() { a.showEditor(app.SkillPlan{}) }),
			a.edit, a.remove,
		)
		controls = container.NewVBox(
			container.NewBorder(nil, nil, nil, buttons, a.planSelect),
			container.NewBorder(nil, nil, container.NewHBox(a.tagSelect, a.qualifiedOnly, reset), nil, a.search),
		)
	}
	top := container.NewVBox(controls, a.summary)
	return widget.NewSimpleRenderer(container.NewBorder(top, a.footer, nil, nil, a.body))
}

func (a *SkillPlans) update(ctx context.Context) {
	isLatest := a.updateRun.Start()
	plans, err := a.u.Character().ListSkillPlans(ctx)
	if ctx.Err() != nil {
		return
	}
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		if err != nil {
			a.checkRun.Start()
			a.characters = nil
			a.filterRows()
			a.reportError("Could not load skill plans", err)
			return
		}
		a.plans = plans
		var options []string
		var selected app.SkillPlan
		for _, p := range plans {
			options = append(options, p.Name)
			if p.ID == a.selected.ID {
				selected = p
			}
		}
		if selected.ID == 0 && len(plans) > 0 {
			selected = plans[0]
		}
		a.settingOptions = true
		a.planSelect.SetOptions(options)
		if selected.ID != 0 {
			a.planSelect.SetSelected(selected.Name)
		} else {
			a.planSelect.ClearSelected()
		}
		a.settingOptions = false
		a.selectPlan(selected)
	})
}

func (a *SkillPlans) selectPlan(p app.SkillPlan) {
	a.selected = p
	a.characters = nil
	a.filterRows()
	isLatest := a.checkRun.Start()
	if p.ID == 0 {
		a.summary.Importance = widget.MediumImportance
		if a.u.IsMobile() {
			a.summary.SetText("No skill plans available. Skill plans are view-only on mobile.")
		} else {
			a.edit.Disable()
			a.remove.Disable()
			a.summary.SetText("No skill plans. Use New / Import to create a plan or paste a skill list.")
		}
		return
	}
	if !a.u.IsMobile() {
		a.edit.Enable()
		a.remove.Enable()
	}
	a.summary.Importance = widget.MediumImportance
	a.summary.SetText(fmt.Sprintf("%s: %d required skills. Checking characters...", p.Name, len(p.Requirements)))
	a.u.Go(func(ctx context.Context) {
		characters, err := a.u.Character().CheckSkillPlan(ctx, p)
		if ctx.Err() != nil {
			return
		}
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			if err != nil {
				a.reportError("Could not check skill plan", err)
				return
			}
			a.characters = characters
			a.summary.SetText(fmt.Sprintf("%s: %d required skills. Active levels from the last successful ESI sync; account groups use character tags.", p.Name, len(p.Requirements)))
			var tags []string
			for _, c := range characters {
				tags = append(tags, c.Tags...)
			}
			slices.Sort(tags)
			tags = slices.Compact(tags)
			a.tagSelect.SetOptions(tags)
			if a.tagSelect.Selected != "" && !slices.Contains(tags, a.tagSelect.Selected) {
				a.tagSelect.ClearSelected()
			}
			a.filterRows()
		})
	})
}

func (a *SkillPlans) filterRows() {
	a.rows = nil
	search := strings.ToLower(strings.TrimSpace(a.search.Text))
	qualified := 0
	for _, c := range a.characters {
		if c.Qualified() {
			qualified++
		}
		if a.qualifiedOnly.Checked && !c.Qualified() {
			continue
		}
		groups := c.Tags
		if len(groups) == 0 {
			groups = []string{"Untagged"}
		}
		for _, group := range groups {
			if a.tagSelect.Selected != "" && !slices.Contains(c.Tags, a.tagSelect.Selected) {
				continue
			}
			if a.tagSelect.Selected != "" && group != a.tagSelect.Selected {
				continue
			}
			if search != "" && !strings.Contains(strings.ToLower(c.Character.Name+" "+group), search) {
				continue
			}
			a.rows = append(a.rows, skillPlanRow{group: group, character: c})
		}
	}
	slices.SortStableFunc(a.rows, func(a, b skillPlanRow) int {
		return strings.Compare(a.group, b.group)
	})
	a.results.UnselectAll()
	a.results.Refresh()
	a.showCharacter(nil)
	for i, r := range a.rows {
		if r.character.Character.ID == a.selectedID {
			a.results.Select(i)
			break
		}
	}
	a.footer.SetText(fmt.Sprintf("%d/%d characters qualify | %d visible account-tag entries. Characters with multiple tags appear in each group.", qualified, len(a.characters), len(a.rows)))
}

func (a *SkillPlans) showCharacter(c *characterservice.SkillPlanCharacter) {
	a.hasCharacter = c != nil
	a.detailKnown = c != nil && c.SkillsKnown
	if c == nil {
		a.detailTitle.SetText("Plan requirements - select a character to see active levels")
		a.detailSkills = a.selected.Evaluate(nil)
	} else {
		title := c.Character.Name
		if c.SkillsKnown {
			title += "\nSkills synced: " + c.UpdatedAt.Local().Format("2006-01-02 15:04")
		} else {
			title += "\nUnknown: sync character skills first."
		}
		if c.UpdateError != "" {
			title += "\nLatest sync failed: " + c.UpdateError
		}
		a.detailTitle.SetText(title)
		a.detailSkills = c.Skills
	}
	a.details.Refresh()
}

func (a *SkillPlans) reportError(message string, err error) {
	slog.Error(message, "error", err)
	a.summary.Importance = widget.DangerImportance
	a.summary.SetText(message + ": " + a.u.ErrorDisplay(err))
}

func (a *SkillPlans) deleteSelected() {
	if a.u.IsMobile() {
		a.u.DisplaySnackbar("Skill plans are view-only on mobile.")
		return
	}
	p := a.selected
	ui.ShowConfirm("Delete skill plan", fmt.Sprintf("Delete %q? Character skills will not be changed.", p.Name), "Delete", func(ok bool) {
		if !ok {
			return
		}
		a.u.Go(func(ctx context.Context) {
			err := a.u.Character().DeleteSkillPlan(ctx, p.ID)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				fyne.Do(func() { a.reportError("Could not delete skill plan", err) })
				return
			}
			a.update(ctx)
		})
	}, a.u.MainWindow())
}

func (a *SkillPlans) showEditor(p app.SkillPlan) {
	if a.u.IsMobile() {
		a.u.DisplaySnackbar("Skill plans are view-only on mobile.")
		return
	}
	a.u.Go(func(ctx context.Context) {
		skills, err := a.u.EVEUniverse().ListSkills(ctx)
		if ctx.Err() != nil {
			return
		}
		fyne.Do(func() {
			if err != nil {
				a.reportError("Could not load skill catalogue", err)
				return
			}
			a.makeEditor(p, skills).Show()
		})
	})
}

func (a *SkillPlans) makeEditor(p app.SkillPlan, skills []*app.EveSkill) dialog.Dialog {
	name := widget.NewEntry()
	name.SetText(p.Name)
	name.PlaceHolder = "Plan name"
	text := widget.NewMultiLineEntry()
	text.Wrapping = fyne.TextWrapWord
	text.SetMinRowsVisible(6)
	text.PlaceHolder = "Navigation IV\nAfterburner 3"
	var lines []string
	for _, r := range p.Requirements {
		lines = append(lines, fmt.Sprintf("%s %d", r.Name, r.Level))
	}
	text.SetText(strings.Join(lines, "\n"))
	var names []string
	for _, s := range skills {
		names = append(names, s.Type.Name)
	}
	slices.Sort(names)
	skill := widget.NewSelectEntry(names)
	skill.PlaceHolder = "Choose or type a skill"
	level := widget.NewSelect([]string{"1", "2", "3", "4", "5"}, nil)
	level.SetSelected("1")
	add := widget.NewButton("Add skill", func() {
		line := skill.Text + " " + level.Selected
		if text.Text != "" {
			line = strings.TrimRight(text.Text, "\n") + "\n" + line
		}
		text.SetText(line)
	})
	add.Disable()
	skill.OnChanged = func(value string) {
		if strings.TrimSpace(value) == "" {
			add.Disable()
		} else {
			add.Enable()
		}
	}
	paste := widget.NewButtonWithIcon("Paste skill list", theme.ContentPasteIcon(), func() {
		text.SetText(fyne.CurrentApp().Clipboard().Content())
	})
	errorLabel := ui.NewLabelWithWrapping("")
	errorLabel.Importance = widget.DangerImportance
	var d dialog.Dialog
	var save *widget.Button
	save = widget.NewButton("Save plan", func() {
		planName, planText := name.Text, text.Text
		save.Disable()
		if !a.u.Go(func(ctx context.Context) {
			id, err := a.u.Character().SaveSkillPlan(ctx, p.ID, planName, planText)
			if ctx.Err() != nil {
				return
			}
			fyne.Do(func() {
				save.Enable()
				if err != nil {
					slog.Error("Could not save skill plan", "error", err)
					errorLabel.SetText(a.u.ErrorDisplay(err))
					return
				}
				d.Hide()
				a.selected = app.SkillPlan{ID: id}
				a.u.Go(a.update)
			})
		}) {
			save.Enable()
			errorLabel.SetText("The app is shutting down; the plan was not saved.")
		}
	})
	save.Importance = widget.HighImportance
	help := ui.NewLabelWithWrapping("Create a plan here or paste a skill list: one skill name and level (I-V or 1-5) per line. Repeated skills use the highest level. ESI does not expose saved in-game plans.")
	content := container.NewVBox(help, name, container.NewBorder(nil, nil, nil, container.NewHBox(level, add), skill), paste, text, errorLabel, save)
	d = dialog.NewCustom("Skill plan", "Close", container.NewVScroll(content), a.u.MainWindow())
	size := a.u.MainWindow().Canvas().Size()
	d.Resize(fyne.NewSize(min(650, size.Width*0.9), min(600, size.Height*0.9)))
	return d
}
