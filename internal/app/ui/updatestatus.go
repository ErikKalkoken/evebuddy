package ui

import (
	"context"
	"fmt"
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
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
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
	sectionCorpoation
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

func (se sectionEntity) isGeneralSection() bool {
	return se.category == sectionGeneral
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
	detailsButton     *widget.Button
	detailsTop        *widget.Label
	entities          fyne.CanvasObject
	entitiesButton    *widget.Button
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
	u                 *BaseUI
}

func newUpdateStatus(u *BaseUI) *updateStatus {
	a := &updateStatus{
		charactersTop:     appwidget.MakeTopLabel(),
		details:           newUpdateStatusDetail(),
		detailsTop:        appwidget.MakeTopLabel(),
		sectionEntities:   make([]sectionEntity, 0),
		sections:          make([]app.SectionStatus, 0),
		sectionsTop:       appwidget.MakeTopLabel(),
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
	a.entitiesButton = widget.NewButton("Force update all sections", nil)
	a.entitiesButton.Disable()
	a.detailsButton = widget.NewButton("Force update section", nil)
	a.detailsButton.Disable()

	a.top2 = container.NewVBox(a.sectionsTop, widget.NewSeparator())
	a.top3 = container.NewVBox(a.detailsTop, widget.NewSeparator())
	if u.IsMobile() {
		sections := container.NewBorder(a.top2, nil, nil, nil, a.sectionList)
		details := container.NewBorder(a.top3, nil, nil, nil, a.details)
		menu := iwidget.NewIconButtonWithMenu(
			theme.MoreVerticalIcon(),
			fyne.NewMenu("", fyne.NewMenuItem(a.entitiesButton.Text, a.makeUpdateAllAction())),
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
					"", fyne.NewMenuItem(a.detailsButton.Text, a.makeDetailsAction(s.EntityID, s.SectionID))),
			)
			a.nav.Push(
				iwidget.NewAppBar("Section Detail", details, menu),
			)
		}
	}
	return a
}

func (a *updateStatus) CreateRenderer() fyne.WidgetRenderer {
	var c fyne.CanvasObject
	if a.u.IsMobile() {
		ab := iwidget.NewAppBar("Home", a.entities)
		a.nav = iwidget.NewNavigatorWithAppBar(ab)
		c = a.nav
	} else {
		sections := container.NewBorder(a.top2, a.entitiesButton, nil, nil, a.sectionList)
		details := container.NewBorder(a.top3, a.detailsButton, nil, nil, a.details)
		vs := container.NewHSplit(sections, details)
		vs.SetOffset(0.5)
		hs := container.NewHSplit(a.entities, vs)
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
			case sectionCharacter, sectionCorpoation:
				name.TextStyle.Bold = false
				name.Refresh()
				iwidget.RefreshImageAsync(icon, func() (fyne.Resource, error) {
					switch c.category {
					case sectionCharacter:
						return a.u.eis.CharacterPortrait(c.id, app.IconPixelSize)
					case sectionCorpoation:
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
			a.entitiesButton.OnTapped = a.makeUpdateAllAction()
			a.entitiesButton.Enable()
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
		c := a.sectionEntities[a.selectedEntityID]
		if c.isGeneralSection() {
			a.u.updateGeneralSectionsAndRefreshIfNeeded(true)
		} else {
			a.u.updateCharacterAndRefreshIfNeeded(context.Background(), c.id, true)
		}
	}
}

func (a *updateStatus) update() {
	entities := a.updateEntityList(a.u.services())
	fyne.Do(func() {
		a.sectionEntities = entities
		a.entityList.Refresh()
		a.refreshSections()
		a.charactersTop.SetText(fmt.Sprintf("Entities: %d", len(a.sectionEntities)))
		a.refreshDetails()
	})
}

func (*updateStatus) updateEntityList(s services) []sectionEntity {
	entities := make([]sectionEntity, 0)
	entities = append(entities, sectionEntity{category: sectionHeader, name: "Characters"})
	cc := s.scs.ListCharacters()
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
	entities = append(entities, sectionEntity{category: sectionHeader, name: "Corporations"})
	rr := s.scs.ListCorporations()
	for _, r := range rr {
		ss := s.scs.CorporationSectionSummary(r.ID)
		o := sectionEntity{
			category: sectionCorpoation,
			id:       r.ID,
			name:     r.Name,
			ss:       ss,
		}
		entities = append(entities, o)
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
	return entities
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
		a.sections = a.u.scs.CharacterSectionList(se.id)
	case sectionCorpoation:
		a.sections = a.u.scs.CorporationSectionList(se.id)
	case sectionGeneral:
		a.sections = a.u.scs.GeneralSectionList()
	}
	a.sectionList.Refresh()
	a.sectionsTop.SetText(fmt.Sprintf("%s: Sections", se.name))
}

func (a *updateStatus) refreshDetails() {
	id := a.selectedSectionID
	if id == -1 || id >= len(a.sections) {
		a.details.Hide()
		a.detailsButton.Disable()
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
		a.detailsButton.OnTapped = a.makeDetailsAction(ss.EntityID, ss.SectionID)
		a.detailsButton.Enable()
	}
	a.details.set(ss)
	a.details.Show()
}

func (a *updateStatus) makeDetailsAction(entityID int32, sectionID string) func() {
	return func() {
		if entityID == app.GeneralSectionEntityID {
			go a.u.updateGeneralSectionAndRefreshIfNeeded(
				context.TODO(), app.GeneralSection(sectionID), true)
		} else {
			go a.u.updateCharacterSectionAndRefreshIfNeeded(
				context.TODO(), entityID, app.CharacterSection(sectionID), true)
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
