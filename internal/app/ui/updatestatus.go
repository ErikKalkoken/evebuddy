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
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

const (
	statusAreaTicker = 3 * time.Second
)

// An entity which has update sections, e.g. a character
type sectionEntity struct {
	id   int32
	name string
	ss   app.StatusSummary
}

func (se sectionEntity) IsGeneralSection() bool {
	return se.id == app.GeneralSectionEntityID
}

type detailsItem struct {
	label      string
	value      string
	wrap       bool
	importance widget.Importance
}

type UpdateStatusArea struct {
	Content           fyne.CanvasObject
	OnEntitySelected  func(int)
	OnSectionSelected func(int)

	charactersTop     *widget.Label
	details           []detailsItem
	detailsButton     *widget.Button
	detailsList       *widget.List
	detailsTop        *widget.Label
	entities          fyne.CanvasObject
	entitiesButton    *widget.Button
	entityList        *widget.List
	nav               *widgets.Navigator
	sectionEntities   []sectionEntity
	sectionList       *widget.List
	sections          []app.SectionStatus
	sectionsTop       *widget.Label
	selectedEntityID  int
	selectedSectionID int
	u                 *BaseUI
}

func (u *BaseUI) NewUpdateStatusArea() *UpdateStatusArea {
	a := &UpdateStatusArea{
		charactersTop:     widget.NewLabel(""),
		details:           make([]detailsItem, 0),
		detailsTop:        makeTopLabel(),
		sectionEntities:   make([]sectionEntity, 0),
		sections:          make([]app.SectionStatus, 0),
		sectionsTop:       makeTopLabel(),
		selectedEntityID:  -1,
		selectedSectionID: -1,
		u:                 u,
	}
	a.entityList = a.makeEntityList()
	a.charactersTop.TextStyle.Bold = true
	a.entities = container.NewBorder(
		container.NewVBox(a.charactersTop, widget.NewSeparator()),
		nil,
		nil,
		nil,
		a.entityList,
	)

	a.sectionList = a.makeSectionList()
	a.sectionsTop.TextStyle.Bold = true
	a.entitiesButton = widget.NewButton("Force update all sections", nil)
	a.entitiesButton.Disable()
	top2 := container.NewVBox(a.sectionsTop, widget.NewSeparator())
	sections := container.NewBorder(top2, a.entitiesButton, nil, nil, a.sectionList)

	headline := widget.NewLabel("Section details")
	headline.TextStyle.Bold = true
	a.detailsList = a.makeDetails()

	top3 := container.NewVBox(a.detailsTop, widget.NewSeparator())
	a.detailsButton = widget.NewButton("Force update section", nil)
	a.detailsButton.Disable()
	details := container.NewBorder(top3, a.detailsButton, nil, nil, a.detailsList)
	vs := container.NewHSplit(sections, details)
	vs.SetOffset(0.5)
	hs := container.NewHSplit(a.entities, vs)
	hs.SetOffset(0.33)

	if u.IsMobile() {
		ab := widgets.NewAppBar("Update Status", a.entities)
		a.nav = widgets.NewNavigator(ab)
		a.Content = a.nav
		a.OnEntitySelected = func(id int) {
			entity := a.sectionEntities[id]
			a.nav.Push(
				widgets.NewAppBar(entity.name, a.sectionList),
			)
		}
		a.OnSectionSelected = func(id int) {
			s := a.sections[id]
			a.nav.Push(
				widgets.NewAppBar(fmt.Sprintf("%s: %s", s.EntityName, s.SectionName), a.detailsList),
			)
		}
	} else {
		a.Content = hs
	}
	return a
}

func (a *UpdateStatusArea) makeEntityList() *widget.List {
	list := widget.NewList(
		func() int {
			return len(a.sectionEntities)
		},
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromResource(IconQuestionmarkSvg)
			icon.FillMode = canvas.ImageFillContain
			icon.SetMinSize(fyne.Size{Width: DefaultIconUnitSize, Height: DefaultIconUnitSize})
			name := widget.NewLabel("Template")
			status := widget.NewLabel("Template")
			pb := widget.NewActivity()
			pb.Stop()
			row := container.NewHBox(icon, name, pb, layout.NewSpacer(), status)
			return row
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.sectionEntities) {
				return
			}
			c := a.sectionEntities[id]
			row := co.(*fyne.Container).Objects
			name := row[1].(*widget.Label)
			name.SetText(c.name)

			icon := row[0].(*canvas.Image)
			if c.IsGeneralSection() {
				icon.Resource = eveicon.GetResourceByName(eveicon.StarMap)
				icon.Refresh()
			} else {
				go a.u.UpdateAvatar(c.id, func(r fyne.Resource) {
					icon.Resource = r
					icon.Refresh()
				})
			}

			pb := row[2].(*widget.Activity)
			if c.ss.IsRunning {
				pb.Start()
				pb.Show()
			} else {
				pb.Stop()
				pb.Hide()
			}

			status := row[4].(*widget.Label)
			t := c.ss.Display()
			i := c.ss.Status().ToImportance()
			status.Text = t
			status.Importance = i
			status.Refresh()
		})

	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.sectionEntities) {
			list.UnselectAll()
			return
		}
		a.selectedEntityID = id
		a.selectedSectionID = -1
		a.sectionList.UnselectAll()

		a.entitiesButton.OnTapped = func() {
			c := a.sectionEntities[a.selectedEntityID]
			if c.IsGeneralSection() {
				a.u.UpdateGeneralSectionsAndRefreshIfNeeded(true)
			} else {
				a.u.UpdateCharacterAndRefreshIfNeeded(context.TODO(), c.id, true)
			}
		}
		a.entitiesButton.Enable()
		if a.OnEntitySelected != nil {
			a.OnEntitySelected(id)
			list.UnselectAll()
		}
		a.updateSections()
		a.updateDetails()
	}
	return list
}

func (a *UpdateStatusArea) Refresh() {
	if err := a.refreshEntityList(); err != nil {
		slog.Warn("failed to refresh entity list for status window", "error", err)
	}
	a.updateSections()
	a.updateDetails()
	a.charactersTop.SetText(fmt.Sprintf("Entities: %d", len(a.sectionEntities)))
}

func (a *UpdateStatusArea) refreshEntityList() error {
	entities := make([]sectionEntity, 0)
	cc := a.u.StatusCacheService.ListCharacters()
	for _, c := range cc {
		ss := a.u.StatusCacheService.CharacterSectionSummary(c.ID)
		o := sectionEntity{id: c.ID, name: c.Name, ss: ss}
		entities = append(entities, o)
	}
	ss := a.u.StatusCacheService.GeneralSectionSummary()
	o := sectionEntity{
		id:   app.GeneralSectionEntityID,
		name: app.GeneralSectionEntityName,
		ss:   ss,
	}
	entities = append(entities, o)
	a.sectionEntities = entities
	a.entityList.Refresh()
	return nil
}

func (a *UpdateStatusArea) makeSectionList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.sections)
		},
		func() fyne.CanvasObject {
			pb := widget.NewActivity()
			pb.Stop()
			return container.NewHBox(
				widget.NewLabel("Section name long"),
				pb,
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

			pb := hbox[1].(*widget.Activity)
			if cs.IsRunning() {
				pb.Start()
				pb.Show()
			} else {
				pb.Stop()
				pb.Hide()
			}
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.sections) {
			l.UnselectAll()
			return
		}
		a.selectedSectionID = id
		a.updateDetails()
		if a.OnSectionSelected != nil {
			a.OnSectionSelected(id)
			l.UnselectAll()
		}
	}
	return l
}

func (a *UpdateStatusArea) updateSections() {
	if a.selectedEntityID == -1 || a.selectedEntityID >= len(a.sectionEntities) {
		return
	}
	se := a.sectionEntities[a.selectedEntityID]
	a.sections = a.u.StatusCacheService.SectionList(se.id)
	a.sectionList.Refresh()
	a.sectionsTop.SetText(fmt.Sprintf("Update status for %s", se.name))
}

type sectionStatusData struct {
	entityID    int32
	entityName  string
	errorText   string
	sectionName string
	completedAt string
	startedAt   string
	sectionID   string
	sv          string
	si          widget.Importance
	timeout     string
}

func (x sectionStatusData) IsGeneralSection() bool {
	return x.entityID == app.GeneralSectionEntityID
}

func (a *UpdateStatusArea) makeDetails() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.details)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewLabel(""), layout.NewSpacer(), widget.NewLabel(""))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.details) {
				return
			}
			item := a.details[id]
			hbox := co.(*fyne.Container).Objects
			label := hbox[0].(*widget.Label)
			status := hbox[2].(*widget.Label)
			label.SetText(item.label)
			status.Importance = item.importance
			v := item.value
			if v == "" {
				v = "?"
				status.Importance = widget.LowImportance
			}
			status.Text = v
			status.Refresh()
		},
	)
	l.OnSelected = func(_ widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *UpdateStatusArea) updateDetails() {
	var d sectionStatusData
	es, found, err := a.fetchSelectedEntityStatus()
	if err != nil {
		slog.Error("Failed to fetch selected entity status")
	} else if found {
		d.entityID = es.EntityID
		d.sectionID = es.SectionID
		d.sv, d.si = statusDisplay(es)
		if es.ErrorMessage == "" {
			d.errorText = "-"
		} else {
			d.errorText = es.ErrorMessage
		}
		d.entityName = es.EntityName
		d.sectionName = es.SectionName
		d.completedAt = ihumanize.Time(es.CompletedAt, "?")
		d.startedAt = ihumanize.Time(es.StartedAt, "-")
		now := time.Now()
		d.timeout = humanize.RelTime(now.Add(es.Timeout), now, "", "")
	}
	a.details = []detailsItem{
		{label: "Section", value: d.sectionName},
		{label: "Status", value: d.sv, importance: d.si},
		{label: "Error", value: d.errorText, wrap: true},
		{label: "Started", value: d.startedAt},
		{label: "Completed", value: d.completedAt},
		{label: "Timeout", value: d.timeout},
	}
	a.detailsList.Refresh()
	if d.entityName != "" {
		a.detailsTop.SetText(fmt.Sprintf("%s: %s", d.entityName, d.sectionName))
	} else {
		a.detailsTop.SetText("")
	}
	a.detailsButton.OnTapped = func() {
		if d.IsGeneralSection() {
			go a.u.UpdateGeneralSectionAndRefreshIfNeeded(
				context.TODO(), app.GeneralSection(d.sectionID), true)
		} else {
			go a.u.UpdateCharacterSectionAndRefreshIfNeeded(
				context.TODO(), d.entityID, app.CharacterSection(d.sectionID), true)
		}
	}
	if d.sectionID != "" || d.entityID != 0 {
		a.detailsButton.Enable()
	} else {
		a.detailsButton.Disable()
	}
}

func (a *UpdateStatusArea) fetchSelectedEntityStatus() (app.SectionStatus, bool, error) {
	id := a.selectedSectionID
	if id == -1 || id >= len(a.sections) {
		return app.SectionStatus{}, false, nil
	}
	return a.sections[id], true, nil
}

// func (a *UpdateStatusArea) makeDetailsContent(d sectionStatusData) []fyne.CanvasObject {
// 	if a.u.IsOffline || d.sectionName == "" {
// 		b.Disable()
// 	}
// }

func (a *UpdateStatusArea) StartTicker(ctx context.Context) {
	ticker := time.NewTicker(statusAreaTicker)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				a.Refresh()
				<-ticker.C
			}
		}
	}()
}

func statusDisplay(cs app.SectionStatus) (string, widget.Importance) {
	var s string
	var i widget.Importance
	if !cs.IsOK() {
		s = "ERROR"
		i = widget.DangerImportance
	} else if cs.IsMissing() {
		s = "Missing"
		i = widget.WarningImportance
	} else if !cs.IsCurrent() {
		s = "Stale"
		i = widget.HighImportance
	} else {
		s = "OK"
		i = widget.SuccessImportance
	}
	return s, i
}
