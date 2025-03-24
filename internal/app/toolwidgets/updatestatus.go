package toolwidgets

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
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	updateStatusTicker = 3 * time.Second
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
	importance widget.Importance
}

type UpdateStatus struct {
	widget.BaseWidget

	charactersTop     *widget.Label
	details           []detailsItem
	detailsButton     *widget.Button
	detailsList       *widget.List
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
	u                 app.UI
}

func NewUpdateStatus(u app.UI) *UpdateStatus {
	a := &UpdateStatus{
		charactersTop:     appwidget.MakeTopLabel(),
		details:           make([]detailsItem, 0),
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
	a.detailsList = a.makeDetails()

	a.top2 = container.NewVBox(a.sectionsTop, widget.NewSeparator())
	a.top3 = container.NewVBox(a.detailsTop, widget.NewSeparator())
	if u.IsMobile() {
		sections := container.NewBorder(a.top2, nil, nil, nil, a.sectionList)
		details := container.NewBorder(a.top3, nil, nil, nil, a.detailsList)
		menu := iwidget.NewIconButtonWithMenu(
			theme.MoreVerticalIcon(),
			fyne.NewMenu("", fyne.NewMenuItem(a.entitiesButton.Text, a.MakeUpdateAllAction())),
		)
		a.onEntitySelected = func(id int) {
			a.nav.Push(
				iwidget.NewAppBar("", sections, menu),
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
				iwidget.NewAppBar("", details, menu),
			)
		}
	} else {

	}

	return a
}

func (a *UpdateStatus) CreateRenderer() fyne.WidgetRenderer {
	var c fyne.CanvasObject
	if a.u.IsMobile() {
		ab := iwidget.NewAppBar("Home", a.entities)
		a.nav = iwidget.NewNavigator(ab)
		c = a.nav

	} else {
		sections := container.NewBorder(a.top2, a.entitiesButton, nil, nil, a.sectionList)
		details := container.NewBorder(a.top3, a.detailsButton, nil, nil, a.detailsList)
		vs := container.NewHSplit(sections, details)
		vs.SetOffset(0.5)
		hs := container.NewHSplit(a.entities, vs)
		hs.SetOffset(0.33)
		c = hs
	}
	return widget.NewSimpleRenderer(c)
}

func (a *UpdateStatus) makeEntityList() *widget.List {
	list := widget.NewList(
		func() int {
			return len(a.sectionEntities)
		},
		func() fyne.CanvasObject {
			icon := iwidget.NewImageFromResource(icons.QuestionmarkSvg, fyne.NewSquareSize(app.IconUnitSize))
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
			i := c.ss.Status().ToImportance2()
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

		a.entitiesButton.OnTapped = a.MakeUpdateAllAction()
		a.entitiesButton.Enable()
		if a.onEntitySelected != nil {
			a.onEntitySelected(id)
			list.UnselectAll()
		}
		a.refreshSections()
		a.refreshDetails()
	}
	return list
}

func (a *UpdateStatus) MakeUpdateAllAction() func() {
	return func() {
		c := a.sectionEntities[a.selectedEntityID]
		if c.IsGeneralSection() {
			a.u.UpdateGeneralSectionsAndRefreshIfNeeded(true)
		} else {
			a.u.UpdateCharacterAndRefreshIfNeeded(context.Background(), c.id, true)
		}
	}
}

func (a *UpdateStatus) Update() {
	if err := a.updateEntityList(); err != nil {
		slog.Warn("failed to refresh entity list for status window", "error", err)
	}
	a.refreshSections()
	a.refreshDetails()
	a.charactersTop.SetText(fmt.Sprintf("Entities: %d", len(a.sectionEntities)))
}

func (a *UpdateStatus) updateEntityList() error {
	entities := make([]sectionEntity, 0)
	cc := a.u.StatusCacheService().ListCharacters()
	for _, c := range cc {
		ss := a.u.StatusCacheService().CharacterSectionSummary(c.ID)
		o := sectionEntity{id: c.ID, name: c.Name, ss: ss}
		entities = append(entities, o)
	}
	ss := a.u.StatusCacheService().GeneralSectionSummary()
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

func (a *UpdateStatus) makeSectionList() *widget.List {
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
		a.refreshDetails()
		if a.onSectionSelected != nil {
			a.onSectionSelected(id)
			l.UnselectAll()
		}
	}
	return l
}

func (a *UpdateStatus) refreshSections() {
	if a.selectedEntityID == -1 || a.selectedEntityID >= len(a.sectionEntities) {
		return
	}
	se := a.sectionEntities[a.selectedEntityID]
	a.sections = a.u.StatusCacheService().SectionList(se.id)
	a.sectionList.Refresh()
	a.sectionsTop.SetText(fmt.Sprintf("%s: All sections", se.name))
}

func (a *UpdateStatus) makeDetails() *widget.List {
	rowLayout := kxlayout.NewColumns(100)
	var l *widget.List
	l = widget.NewList(
		func() int {
			return len(a.details)
		},
		func() fyne.CanvasObject {
			status := widget.NewLabel("")
			status.Wrapping = fyne.TextWrapWord
			label := widget.NewLabel("")
			return container.New(rowLayout, label, status)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.details) {
				return
			}
			item := a.details[id]
			border := co.(*fyne.Container).Objects
			label := border[0].(*widget.Label)
			status := border[1].(*widget.Label)
			label.SetText(item.label)
			status.Importance = item.importance
			v := item.value
			if v == "" {
				v = "?"
				status.Importance = widget.LowImportance
			}
			status.Text = v
			l.SetItemHeight(id, status.MinSize().Height)
			status.Refresh()
		},
	)
	l.OnSelected = func(_ widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *UpdateStatus) refreshDetails() {
	id := a.selectedSectionID
	if id == -1 || id >= len(a.sections) {
		a.details = []detailsItem{}
		a.detailsButton.Disable()
		a.detailsList.Refresh()
		a.detailsTop.SetText("")
		return
	}
	es := a.sections[id]
	statusValue, statusImportance := statusDisplay(es)
	var errorText string
	var errorImportance widget.Importance
	if es.ErrorMessage == "" {
		errorText = "-"
	} else {
		errorText = es.ErrorMessage
		errorImportance = widget.DangerImportance
	}
	completedAt := ihumanize.Time(es.CompletedAt, "?")
	startedAt := ihumanize.Time(es.StartedAt, "-")
	now := time.Now()
	timeout := humanize.RelTime(now.Add(es.Timeout), now, "", "")

	a.details = []detailsItem{
		{label: "Status", value: statusValue, importance: statusImportance},
		{label: "Error", value: errorText, importance: errorImportance},
		{label: "Started", value: startedAt},
		{label: "Completed", value: completedAt},
		{label: "Timeout", value: timeout},
	}
	if es.EntityName != "" {
		a.detailsTop.SetText(fmt.Sprintf("%s: Section %s", es.EntityName, es.SectionName))
	} else {
		a.detailsTop.SetText("")
	}
	a.detailsButton.OnTapped = a.makeDetailsAction(es.EntityID, es.SectionID)
	a.detailsButton.Enable()
	a.detailsList.Refresh()
}

func (a *UpdateStatus) makeDetailsAction(entityID int32, sectionID string) func() {
	return func() {
		if entityID == app.GeneralSectionEntityID {
			go a.u.UpdateGeneralSectionAndRefreshIfNeeded(
				context.TODO(), app.GeneralSection(sectionID), true)
		} else {
			go a.u.UpdateCharacterSectionAndRefreshIfNeeded(
				context.TODO(), entityID, app.CharacterSection(sectionID), true)
		}
	}
}

func (a *UpdateStatus) StartTicker(ctx context.Context) {
	ticker := time.NewTicker(updateStatusTicker)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				a.Update()
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
