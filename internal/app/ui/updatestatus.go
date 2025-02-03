package ui

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
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

type UpdateStatusArea struct {
	Content          fyne.CanvasObject
	OnEntitySelected func()

	entities          fyne.CanvasObject
	charactersTop     *widget.Label
	sectionEntities   []sectionEntity
	entityList        *widget.List
	sectionGrid       *widget.GridWrap
	sections          []app.SectionStatus
	sectionsTop       *widget.Label
	selectedEntityID  int
	selectedSectionID int
	u                 *BaseUI

	mu      sync.Mutex
	details *fyne.Container
}

func (u *BaseUI) NewUpdateStatusArea() *UpdateStatusArea {
	a := &UpdateStatusArea{
		sectionEntities:   make([]sectionEntity, 0),
		charactersTop:     widget.NewLabel(""),
		selectedEntityID:  -1,
		sections:          make([]app.SectionStatus, 0),
		sectionsTop:       widget.NewLabel(""),
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

	a.sectionGrid = a.makeSectionGrid()
	a.sectionsTop.TextStyle.Bold = true
	b := widget.NewButton("Force update all sections", func() {
		if a.selectedEntityID == -1 || a.selectedEntityID >= len(a.sectionEntities) {
			return
		}
		c := a.sectionEntities[a.selectedEntityID]
		if c.IsGeneralSection() {
			a.u.UpdateGeneralSectionsAndRefreshIfNeeded(true)
		} else {
			a.u.UpdateCharacterAndRefreshIfNeeded(context.TODO(), c.id, true)
		}
	})
	if a.u.IsOffline {
		b.Disable()
	}
	top2 := container.NewVBox(container.NewHBox(a.sectionsTop, layout.NewSpacer(), b), widget.NewSeparator())
	sections := container.NewBorder(top2, nil, nil, nil, a.sectionGrid)

	headline := widget.NewLabel("Section details")
	headline.TextStyle.Bold = true
	a.details = container.NewVBox()

	vs := container.NewBorder(nil, container.NewVBox(widget.NewSeparator(), a.details), nil, nil, sections)
	hs := container.NewHSplit(a.entities, vs)
	hs.SetOffset(0.33)

	if u.IsMobile() {
		a.Content = a.entities
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
		a.sectionGrid.UnselectAll()
		a.refreshDetailArea()
		if a.OnEntitySelected != nil {
			a.OnEntitySelected()
		}
	}
	return list
}

func (a *UpdateStatusArea) Refresh() {
	if err := a.refreshEntityList(); err != nil {
		slog.Warn("failed to refresh entity list for status window", "error", err)
	}
	a.refreshDetailArea()
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

func (a *UpdateStatusArea) makeSectionGrid() *widget.GridWrap {
	l := widget.NewGridWrap(
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
		a.setDetails()
	}
	return l
}

type formItems struct {
	label      string
	value      string
	wrap       bool
	importance widget.Importance
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

func (a *UpdateStatusArea) setDetails() {
	var d sectionStatusData
	ss, found, err := a.fetchSelectedEntityStatus()
	if err != nil {
		slog.Error("Failed to fetch selected entity status")
	} else if found {
		d.entityID = ss.EntityID
		d.sectionID = ss.SectionID
		d.sv, d.si = statusDisplay(ss)
		if ss.ErrorMessage == "" {
			d.errorText = "-"
		} else {
			d.errorText = ss.ErrorMessage
		}
		d.entityName = ss.EntityName
		d.sectionName = ss.SectionName
		d.completedAt = ihumanize.Time(ss.CompletedAt, "?")
		d.startedAt = ihumanize.Time(ss.StartedAt, "-")
		now := time.Now()
		d.timeout = humanize.RelTime(now.Add(ss.Timeout), now, "", "")
	}
	oo := a.makeDetailsContent(d)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.details.RemoveAll()
	for _, o := range oo {
		a.details.Add(o)
	}
}

func (a *UpdateStatusArea) fetchSelectedEntityStatus() (app.SectionStatus, bool, error) {
	id := a.selectedSectionID
	if id == -1 || id >= len(a.sections) {
		return app.SectionStatus{}, false, nil
	}
	return a.sections[id], true, nil
}

func (a *UpdateStatusArea) makeDetailsContent(d sectionStatusData) []fyne.CanvasObject {
	items := []formItems{
		{label: "Section", value: d.sectionName},
		{label: "Status", value: d.sv, importance: d.si},
		{label: "Error", value: d.errorText, wrap: true},
		{label: "Started", value: d.startedAt},
		{label: "Completed", value: d.completedAt},
		{label: "Timeout", value: d.timeout},
	}
	oo := make([]fyne.CanvasObject, 0)
	formLeading := makeForm(items[0:3])
	formTrailing := makeForm(items[3:])
	oo = append(oo, container.NewGridWithColumns(2, formLeading, formTrailing))
	b := widget.NewButton(fmt.Sprintf("Force update %s", d.sectionName), func() {
		if d.sectionID == "" {
			return
		}
		if d.IsGeneralSection() {
			go a.u.UpdateGeneralSectionAndRefreshIfNeeded(
				context.TODO(), app.GeneralSection(d.sectionID), true)
		} else {
			go a.u.UpdateCharacterSectionAndRefreshIfNeeded(
				context.TODO(), d.entityID, app.CharacterSection(d.sectionID), true)
		}
	})
	if a.u.IsOffline || d.sectionName == "" {
		b.Disable()
	}
	oo = append(oo, b)
	return oo
}

func makeForm(data []formItems) *widget.Form {
	form := widget.NewForm()
	for _, row := range data {
		c := widget.NewLabel(row.value)
		if row.wrap {
			c.Wrapping = fyne.TextWrapWord
		}
		if row.importance != 0 {
			c.Importance = row.importance
		}
		form.Append(row.label, c)
	}
	return form
}

func (a *UpdateStatusArea) refreshDetailArea() {
	if a.selectedEntityID == -1 || a.selectedEntityID >= len(a.sectionEntities) {
		return
	}
	se := a.sectionEntities[a.selectedEntityID]
	a.sections = a.u.StatusCacheService.SectionList(se.id)
	a.sectionGrid.Refresh()
	a.sectionsTop.SetText(fmt.Sprintf("Update status for %s", se.name))
	a.setDetails()
}

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
