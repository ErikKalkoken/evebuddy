package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	updateStatusTicker = 3 * time.Second
)

type sectionCategory uint

const (
	sectionCharacter sectionCategory = iota + 1
	sectionCorporation
	sectionGeneral
	sectionHeader
)

// An entity which has update sections, e.g. a character
type sectionEntity struct {
	id       int32
	name     string
	category sectionCategory
	ss       app.StatusSummary
}

type detailsItem struct {
	label      string
	value      string
	importance widget.Importance
}

type updateStatus struct {
	widget.BaseWidget

	charactersTop     *widget.Label
	details           *updateStatusDetail
	detailsTop        *widget.Label
	entities          fyne.CanvasObject
	entityList        *widget.List
	nav               *iwidget.Navigator
	onEntitySelected  func(int)
	onSectionSelected func(int)
	sectionEntities   []sectionEntity
	sectionList       *widget.List
	sections          []app.SectionStatus
	sectionsTop       *widget.Label
	selectedEntityID  int
	selectedSectionID int
	top2              fyne.CanvasObject
	top3              fyne.CanvasObject
	u                 *baseUI
	updateAllSections *widget.Button
	updateSection     *widget.Button
}

func newUpdateStatus(u *baseUI) *updateStatus {
	a := &updateStatus{
		charactersTop:     makeTopLabel(),
		details:           newUpdateStatusDetail(),
		detailsTop:        makeTopLabel(),
		sectionEntities:   make([]sectionEntity, 0),
		sections:          make([]app.SectionStatus, 0),
		sectionsTop:       makeTopLabel(),
		selectedEntityID:  -1,
		selectedSectionID: -1,
		u:                 u,
	}
	a.ExtendBaseWidget(a)
	a.entityList = a.makeEntityList()
	a.entities = container.NewBorder(
		container.NewVBox(a.charactersTop, widget.NewSeparator()),
		nil,
		nil,
		nil,
		a.entityList,
	)

	a.sectionList = a.makeSectionList()
	a.updateAllSections = widget.NewButton("Force update all sections", nil)
	a.updateAllSections.Disable()
	a.updateSection = widget.NewButton("Force update section", nil)
	a.updateSection.Disable()

	a.top2 = container.NewVBox(a.sectionsTop, widget.NewSeparator())
	a.top3 = container.NewVBox(a.detailsTop, widget.NewSeparator())
	if !u.isDesktop {
		sections := container.NewBorder(a.top2, nil, nil, nil, a.sectionList)
		details := container.NewBorder(a.top3, nil, nil, nil, a.details)
		menu := iwidget.NewIconButtonWithMenu(
			theme.MoreVerticalIcon(),
			fyne.NewMenu("", fyne.NewMenuItem(a.updateAllSections.Text, a.makeUpdateAllAction())),
		)
		a.onEntitySelected = func(id int) {
			a.nav.Push(
				iwidget.NewAppBar("Sections", sections, menu),
			)
		}
		a.onSectionSelected = func(id int) {
			s := a.sections[id]
			menu := iwidget.NewIconButtonWithMenu(
				theme.MoreVerticalIcon(),
				fyne.NewMenu(
					"", fyne.NewMenuItem(a.updateSection.Text, a.makeUpdateSectionAction(s.EntityID, s.SectionID))),
			)
			a.nav.Push(
				iwidget.NewAppBar("Section Detail", details, menu),
			)
		}
	}
	return a
}

func (a *updateStatus) CreateRenderer() fyne.WidgetRenderer {
	updateMenu := fyne.NewMenu("",
		fyne.NewMenuItem("Update all characters", func() {
			a.u.updateCharactersIfNeeded(context.Background(), true)
		}),
		fyne.NewMenuItem("Update all corporations", func() {
			a.u.updateCorporationsIfNeeded(context.Background(), true)
		}),
		fyne.NewMenuItem("Update all general topics", func() {
			a.u.updateGeneralSectionsIfNeeded(context.Background(), true)
		}),
	)
	updateEntities := iwidget.NewContextMenuButton("Force update all entities", updateMenu)
	var c fyne.CanvasObject
	if !a.u.isDesktop {
		ab := iwidget.NewAppBar("Home", a.entities, iwidget.NewIconButtonWithMenu(
			theme.MoreVerticalIcon(),
			updateMenu,
		))
		a.nav = iwidget.NewNavigatorWithAppBar(ab)
		c = a.nav
	} else {
		sections := container.NewBorder(a.top2, a.updateAllSections, nil, nil, a.sectionList)
		details := container.NewBorder(a.top3, a.updateSection, nil, nil, a.details)
		vs := container.NewHSplit(sections, details)
		vs.SetOffset(0.5)
		hs := container.NewHSplit(container.NewBorder(nil, updateEntities, nil, nil, a.entities), vs)
		hs.SetOffset(0.33)
		c = hs
	}
	return widget.NewSimpleRenderer(c)
}

func (a *updateStatus) makeEntityList() *widget.List {
	list := widget.NewList(
		func() int {
			return len(a.sectionEntities)
		},
		func() fyne.CanvasObject {
			icon := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
			name := widget.NewLabel("Template")
			status := widget.NewLabel("Template")
			spinner := widget.NewActivity()
			spinner.Stop()
			row := container.NewHBox(icon, name, spinner, layout.NewSpacer(), status)
			return row
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.sectionEntities) {
				return
			}
			c := a.sectionEntities[id]
			row := co.(*fyne.Container).Objects
			name := row[1].(*widget.Label)
			icon := row[0].(*canvas.Image)
			spinner := row[2].(*widget.Activity)
			status := row[4].(*widget.Label)

			name.Text = c.name
			switch c.category {
			case sectionGeneral:
				name.TextStyle.Bold = false
				name.Refresh()
				icon.Resource = eveicon.FromName(eveicon.StarMap)
				icon.Refresh()
			case sectionCharacter, sectionCorporation:
				name.TextStyle.Bold = false
				name.Refresh()
				iwidget.RefreshImageAsync(icon, func() (fyne.Resource, error) {
					switch c.category {
					case sectionCharacter:
						return a.u.eis.CharacterPortrait(c.id, app.IconPixelSize)
					case sectionCorporation:
						return a.u.eis.CorporationLogo(c.id, app.IconPixelSize)
					}
					return icons.BlankSvg, nil
				})
			case sectionHeader:
				name.TextStyle.Bold = true
				name.Refresh()
				icon.Hide()
				spinner.Hide()
				status.Hide()
				return
			}

			if c.ss.IsRunning && !a.u.IsOffline() {
				spinner.Start()
				spinner.Show()
			} else {
				spinner.Stop()
				spinner.Hide()
			}

			t := c.ss.Display()
			i := c.ss.Status().ToImportance2()
			status.Text = t
			status.Importance = i
			status.Refresh()
			status.Show()

			icon.Show()
		})

	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.sectionEntities) || a.sectionEntities[id].category == sectionHeader {
			list.UnselectAll()
			return
		}

		a.selectedEntityID = id
		a.selectedSectionID = -1
		a.sectionList.UnselectAll()

		if !a.u.IsOffline() {
			a.updateAllSections.OnTapped = a.makeUpdateAllAction()
			a.updateAllSections.Enable()
		}
		if a.onEntitySelected != nil {
			a.onEntitySelected(id)
			list.UnselectAll()
		}
		a.refreshSections()
		a.refreshDetails()
	}
	return list
}

func (a *updateStatus) makeUpdateAllAction() func() {
	return func() {
		ctx := context.Background()
		c := a.sectionEntities[a.selectedEntityID]
		switch c.category {
		case sectionGeneral:
			a.u.updateGeneralSectionsIfNeeded(ctx, true)
		case sectionCharacter:
			a.u.updateCharacterAndRefreshIfNeeded(ctx, c.id, true)
		case sectionCorporation:
			a.u.updateCorporationAndRefreshIfNeeded(ctx, c.id, true)
		default:
			slog.Error("makeUpdateAllAction: Undefined category", "entity", c)
		}
	}
}

func (a *updateStatus) update() {
	entities, count := a.updateEntityList(a.u.services())

	fyne.Do(func() {
		a.sectionEntities = entities
		a.entityList.Refresh()
		a.refreshSections()
		a.charactersTop.SetText(fmt.Sprintf("Entities: %d", count))
		a.refreshDetails()
	})
}

func (*updateStatus) updateEntityList(s services) ([]sectionEntity, int) {
	var count int
	entities := make([]sectionEntity, 0)
	cc := s.scs.ListCharacters()
	if len(cc) > 0 {
		entities = append(entities, sectionEntity{category: sectionHeader, name: "Characters"})
		count += len(cc)
		for _, c := range cc {
			ss := s.scs.CharacterSectionSummary(c.ID)
			o := sectionEntity{
				category: sectionCharacter,
				id:       c.ID,
				name:     c.Name,
				ss:       ss,
			}
			entities = append(entities, o)
		}
	}
	rr := s.scs.ListCorporations()
	if len(rr) > 0 {
		entities = append(entities, sectionEntity{category: sectionHeader, name: "Corporations"})
		count += len(rr)
		for _, r := range rr {
			ss := s.scs.CorporationSectionSummary(r.ID)
			o := sectionEntity{
				category: sectionCorporation,
				id:       r.ID,
				name:     r.Name,
				ss:       ss,
			}
			entities = append(entities, o)
		}
	}
	entities = append(entities, sectionEntity{category: sectionHeader, name: "General"})
	ss := s.scs.GeneralSectionSummary()
	o := sectionEntity{
		category: sectionGeneral,
		id:       app.GeneralSectionEntityID,
		name:     app.GeneralSectionEntityName,
		ss:       ss,
	}
	entities = append(entities, o)
	count += 1
	return entities, count
}

func (a *updateStatus) makeSectionList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.sections)
		},
		func() fyne.CanvasObject {
			spinner := widget.NewActivity()
			spinner.Stop()
			return container.NewHBox(
				widget.NewLabel("Section name long"),
				spinner,
				layout.NewSpacer(),
				widget.NewLabel("Status XXXX"),
			)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.sections) {
				return
			}
			cs := a.sections[id]
			hbox := co.(*fyne.Container).Objects
			name := hbox[0].(*widget.Label)
			status := hbox[3].(*widget.Label)
			name.SetText(cs.SectionName)
			s, i := statusDisplay(cs)
			status.Text = s
			status.Importance = i
			status.Refresh()

			spinner := hbox[1].(*widget.Activity)
			if cs.IsRunning() && !a.u.IsOffline() {
				spinner.Start()
				spinner.Show()
			} else {
				spinner.Stop()
				spinner.Hide()
			}
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.sections) {
			l.UnselectAll()
			return
		}
		a.selectedSectionID = id
		a.refreshDetails()
		if a.onSectionSelected != nil {
			a.onSectionSelected(id)
			l.UnselectAll()
		}
	}
	return l
}

func (a *updateStatus) refreshSections() {
	if a.selectedEntityID == -1 || a.selectedEntityID >= len(a.sectionEntities) {
		return
	}
	se := a.sectionEntities[a.selectedEntityID]
	switch se.category {
	case sectionCharacter:
		a.sections = a.u.scs.ListCharacterSections(se.id)
	case sectionCorporation:
		a.sections = a.u.scs.ListCorporationSections(se.id)
	case sectionGeneral:
		a.sections = a.u.scs.ListGeneralSections()
	}
	a.sectionList.Refresh()
	a.sectionsTop.SetText(fmt.Sprintf("%s: Sections", se.name))
}

func (a *updateStatus) refreshDetails() {
	id := a.selectedSectionID
	if id == -1 || id >= len(a.sections) {
		a.details.Hide()
		a.updateSection.Disable()
		a.detailsTop.SetText("")
		return
	}
	ss := a.sections[id]
	if ss.EntityName != "" {
		a.detailsTop.SetText(fmt.Sprintf("%s: %s", ss.EntityName, ss.SectionName))
	} else {
		a.detailsTop.SetText("")
	}
	if !a.u.IsOffline() {
		a.updateSection.OnTapped = a.makeUpdateSectionAction(ss.EntityID, ss.SectionID)
		a.updateSection.Enable()
	}
	a.details.set(ss)
	a.details.Show()
}

func (a *updateStatus) makeUpdateSectionAction(entityID int32, sectionID string) func() {
	return func() {
		ctx := context.Background()
		c := a.sectionEntities[a.selectedEntityID]
		switch c.category {
		case sectionGeneral:
			go a.u.updateGeneralSectionAndRefreshIfNeeded(
				ctx, app.GeneralSection(sectionID), true,
			)
		case sectionCharacter:
			go a.u.updateCharacterSectionAndRefreshIfNeeded(
				ctx, entityID, app.CharacterSection(sectionID), true,
			)
		case sectionCorporation:
			go a.u.updateCorporationSectionAndRefreshIfNeeded(
				ctx, entityID, app.CorporationSection(sectionID), true,
			)
		default:
			slog.Error("makeUpdateAllAction: Undefined category", "entity", c)
		}
	}
}

func (a *updateStatus) startTicker(ctx context.Context) {
	ticker := time.NewTicker(updateStatusTicker)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				a.update()
				<-ticker.C
			}
		}
	}()
}

func statusDisplay(ss app.SectionStatus) (string, widget.Importance) {
	var s string
	var i widget.Importance
	if ss.HasError() {
		s = "ERROR"
		i = widget.DangerImportance
	} else if ss.IsMissing() {
		s = "Missing"
		i = widget.WarningImportance
	} else if ss.HasComment() {
		s = "Skipped"
		i = widget.MediumImportance
	} else if !ss.IsCurrent() {
		s = "Stale"
		i = widget.HighImportance
	} else {
		s = "OK"
		i = widget.SuccessImportance
	}
	return s, i
}

type updateStatusDetail struct {
	widget.BaseWidget

	completedAt *widget.Label
	issue       *widget.Label
	startedAt   *widget.Label
	status      *widget.Label
	timeout     *widget.Label
}

func newUpdateStatusDetail() *updateStatusDetail {
	makeLabel := func() *widget.Label {
		l := widget.NewLabel("")
		l.Wrapping = fyne.TextWrapWord
		return l
	}
	w := &updateStatusDetail{
		completedAt: makeLabel(),
		issue:       makeLabel(),
		startedAt:   makeLabel(),
		status:      makeLabel(),
		timeout:     makeLabel(),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *updateStatusDetail) set(ss app.SectionStatus) {
	w.status.Text, w.status.Importance = statusDisplay(ss)
	w.status.Refresh()

	var issue string
	var issueImportance widget.Importance
	if ss.ErrorMessage != "" {
		issue = ss.ErrorMessage
		issueImportance = widget.DangerImportance
	} else if ss.Comment != "" {
		issue = ss.Comment
	} else {
		issue = "-"
	}
	w.issue.Text, w.issue.Importance = issue, issueImportance
	w.issue.Refresh()

	w.completedAt.SetText(ihumanize.Time(ss.CompletedAt, "?"))
	w.startedAt.SetText(ihumanize.Time(ss.StartedAt, "-"))
	now := time.Now()
	w.timeout.SetText(humanize.RelTime(now.Add(ss.Timeout), now, "", ""))
}

func (w *updateStatusDetail) CreateRenderer() fyne.WidgetRenderer {
	layout := kxlayout.NewColumns(100)
	c := container.NewVBox(
		container.New(layout, widget.NewLabel("Status"), w.status),
		container.New(layout, widget.NewLabel("Issue"), w.issue),
		container.New(layout, widget.NewLabel("Started"), w.startedAt),
		container.New(layout, widget.NewLabel("Completed"), w.completedAt),
		container.New(layout, widget.NewLabel("Timeout"), w.timeout),
	)
	return widget.NewSimpleRenderer(c)
}
