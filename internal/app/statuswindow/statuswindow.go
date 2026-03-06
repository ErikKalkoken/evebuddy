// Package statuswindow provides a window that shows the current update status.
package statuswindow

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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type EIS interface {
	CharacterPortraitAsync(id int64, size int, setter func(r fyne.Resource))
	CorporationLogoAsync(id int64, size int, setter func(r fyne.Resource))
}

type SCS interface {
	CharacterSectionSummary(characterID int64) app.StatusSummary
	CorporationSectionSummary(corporationID int64) app.StatusSummary
	EveUniverseSectionSummary() app.StatusSummary
	ListCharacters() []*app.EntityShort
	ListCharacterSections(characterID int64) []app.CacheSectionStatus
	ListCorporations() []*app.EntityShort
	ListCorporationSections(corporationID int64) []app.CacheSectionStatus
	ListEveUniverseSections() []app.CacheSectionStatus
}

type UIService interface {
	GetOrCreateWindowWithOnClosed(id string, titles ...string) (window fyne.Window, created bool, onClosed func())
	IsOffline() bool
	HumanizeError(err error) string
}

type Params struct {
	CharacterService   *characterservice.CharacterService     // TODO: Reduce to interface
	CorporationService *corporationservice.CorporationService // TODO: Reduce to interface
	EveImageService    EIS
	EveUniverseService *eveuniverseservice.EveUniverseService // TODO: Reduce to interface
	IsMobile           bool
	Signals            *app.Signals
	StatusCacheService SCS
	UIService          UIService
}

func Show(arg Params) {
	if arg.CharacterService == nil {
		panic("CharacterService missing")
	}
	if arg.CorporationService == nil {
		panic("CorporationService missing")
	}
	if arg.EveImageService == nil {
		panic("EveImageService missing")
	}
	if arg.EveUniverseService == nil {
		panic("EveUniverseService missing")
	}
	if arg.Signals == nil {
		panic("Signals missing")
	}
	if arg.StatusCacheService == nil {
		panic("StatusCacheService missing")
	}
	if arg.UIService == nil {
		panic("UIService missing")
	}
	w, ok, onClosed := arg.UIService.GetOrCreateWindowWithOnClosed("update-status", "Update Status")
	if !ok {
		w.Show()
		return
	}
	a := newStatusWindow(arg, w)
	w.SetContent(a)
	w.Resize(fyne.Size{Width: 1100, Height: 500})
	w.SetOnClosed(func() {
		a.stop()
		onClosed()
	})
	w.Show()
	go a.update(context.Background())
}

type sectionCategory uint

const (
	sectionCharacter sectionCategory = iota + 1
	sectionCorporation
	sectionGeneral
	sectionHeader
)

// An entity which has update sections, e.g. a character
type sectionEntity struct {
	id       int64
	name     string
	category sectionCategory
	ss       app.StatusSummary
}

type statusWindow struct {
	widget.BaseWidget

	charactersTop     *widget.Label
	details           *updateStatusDetail
	detailsTop        *widget.Label
	entities          fyne.CanvasObject
	entityList        *widget.List
	isMobile          bool
	nav               *iwidget.Navigator
	onEntitySelected  func(int)
	onSectionSelected func(int)
	sb                *iwidget.Snackbar
	sectionEntities   []sectionEntity
	sectionList       *widget.List
	sections          []app.CacheSectionStatus
	sectionsTop       *widget.Label
	selectedEntityID  int
	selectedSectionID int
	signalKey         string
	signals           *app.Signals
	top2              fyne.CanvasObject
	top3              fyne.CanvasObject
	updateAllSections *widget.Button
	updateSection     *widget.Button

	cs  *characterservice.CharacterService
	eis EIS
	eus *eveuniverseservice.EveUniverseService
	rs  *corporationservice.CorporationService
	scs SCS
	u   UIService
}

func newStatusWindow(arg Params, w fyne.Window) *statusWindow {
	newLabelWithWrapping := func() *widget.Label {
		l := widget.NewLabel("")
		l.Wrapping = fyne.TextWrapWord
		return l
	}
	a := &statusWindow{
		charactersTop:     newLabelWithWrapping(),
		details:           newUpdateStatusDetail(),
		detailsTop:        newLabelWithWrapping(),
		eis:               arg.EveImageService,
		eus:               arg.EveUniverseService,
		isMobile:          arg.IsMobile,
		sb:                iwidget.NewSnackbar(w),
		scs:               arg.StatusCacheService,
		sectionsTop:       newLabelWithWrapping(),
		selectedEntityID:  -1,
		selectedSectionID: -1,
		signalKey:         arg.Signals.UniqueKey(),
		signals:           arg.Signals,
		u:                 arg.UIService,
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
	if arg.IsMobile {
		sections := container.NewBorder(a.top2, nil, nil, nil, a.sectionList)
		details := container.NewBorder(a.top3, nil, nil, nil, a.details)
		menu := kxwidget.NewIconButtonWithMenu(
			theme.MoreVerticalIcon(),
			fyne.NewMenu("", fyne.NewMenuItem(a.updateAllSections.Text, a.makeUpdateAllAction())),
		)
		a.onEntitySelected = func(id int) {
			a.nav.Push(iwidget.NewAppBar("Sections", sections, menu))
		}
		a.onSectionSelected = func(id int) {
			s := a.sections[id]
			menu := kxwidget.NewIconButtonWithMenu(
				theme.MoreVerticalIcon(),
				fyne.NewMenu(
					"", fyne.NewMenuItem(a.updateSection.Text, a.makeUpdateSectionAction(s.EntityID, s.SectionID))),
			)
			a.nav.Push(iwidget.NewAppBar("Section Detail", details, menu))
		}
	}

	// Signals
	a.sb.Start()
	a.signals.CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	}, a.signalKey)
	a.signals.CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	}, a.signalKey)
	a.signals.CharacterSectionUpdated.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		a.update(ctx)
	}, a.signalKey)
	a.signals.CorporationSectionUpdated.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		a.update(ctx)
	}, a.signalKey)
	a.signals.EveUniverseSectionUpdated.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		a.update(ctx)
	}, a.signalKey)
	return a
}

func (a *statusWindow) stop() {
	a.sb.Stop()
	a.signals.CharacterAdded.RemoveListener(a.signalKey)
	a.signals.CharacterRemoved.RemoveListener(a.signalKey)
	a.signals.CharacterSectionUpdated.RemoveListener(a.signalKey)
	a.signals.CorporationSectionUpdated.RemoveListener(a.signalKey)
	a.signals.EveUniverseSectionUpdated.RemoveListener(a.signalKey)
}

func (a *statusWindow) CreateRenderer() fyne.WidgetRenderer {
	updateMenu := fyne.NewMenu("",
		fyne.NewMenuItem("Update all characters", func() {
			go func() {
				err := a.cs.UpdateCharactersIfNeeded(context.Background(), true)
				if err != nil {
					slog.Error("update status", "error", err)
					a.sb.Show("Error: " + a.u.HumanizeError(err))
				}
			}()
		}),
		fyne.NewMenuItem("Update all corporations", func() {
			go func() {
				err := a.rs.UpdateCorporationsIfNeeded(context.Background(), true)
				if err != nil {
					slog.Error("update status", "error", err)
					a.sb.Show("Error: " + a.u.HumanizeError(err))
				}
			}()

		}),
		fyne.NewMenuItem("Update all general topics", func() {
			go a.eus.UpdateSectionsIfNeeded(context.Background(), true)
		}),
	)
	updateEntities := iwidget.NewContextMenuButton("Force update all entities", updateMenu)
	var c fyne.CanvasObject
	if a.isMobile {
		ab := iwidget.NewAppBar("Home", a.entities, kxwidget.NewIconButtonWithMenu(
			theme.MoreVerticalIcon(),
			updateMenu,
		))
		a.nav = iwidget.NewNavigator(ab)
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

func (a *statusWindow) makeEntityList() *widget.List {
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
				switch c.category {
				case sectionCharacter:
					a.eis.CharacterPortraitAsync(c.id, app.IconPixelSize, func(r fyne.Resource) {
						icon.Resource = r
						icon.Refresh()
					})
				case sectionCorporation:
					a.eis.CorporationLogoAsync(c.id, app.IconPixelSize, func(r fyne.Resource) {
						icon.Resource = r
						icon.Refresh()
					})
				default:
					icon.Resource = icons.BlankSvg
					icon.Refresh()
				}
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

func (a *statusWindow) makeUpdateAllAction() func() {
	return func() {
		ctx := context.Background()
		c := a.sectionEntities[a.selectedEntityID]
		switch c.category {
		case sectionGeneral:
			go a.eus.UpdateSectionsIfNeeded(ctx, true)
		case sectionCharacter:
			go a.cs.UpdateCharacterAndRefreshIfNeeded(ctx, c.id, true)
		case sectionCorporation:
			go a.rs.UpdateCorporationAndRefreshIfNeeded(ctx, c.id, true)
		default:
			panic(fmt.Sprintf("makeUpdateAllAction: Undefined category: %v", c.category))
		}
	}
}

func (a *statusWindow) update(ctx context.Context) {
	entities, count := a.updateEntityList(ctx)

	fyne.Do(func() {
		a.sectionEntities = entities
		a.entityList.Refresh()
		a.refreshSections()
		a.charactersTop.SetText(fmt.Sprintf("Entities: %d", count))
		a.refreshDetails()
	})
}

func (a *statusWindow) updateEntityList(_ context.Context) ([]sectionEntity, int) {
	var count int
	var entities []sectionEntity
	cc := a.scs.ListCharacters()
	if len(cc) > 0 {
		entities = append(entities, sectionEntity{category: sectionHeader, name: "Characters"})
		count += len(cc)
		for _, c := range cc {
			ss := a.scs.CharacterSectionSummary(c.ID)
			o := sectionEntity{
				category: sectionCharacter,
				id:       c.ID,
				name:     c.Name,
				ss:       ss,
			}
			entities = append(entities, o)
		}
	}
	rr := a.scs.ListCorporations()
	if len(rr) > 0 {
		entities = append(entities, sectionEntity{category: sectionHeader, name: "Corporations"})
		count += len(rr)
		for _, r := range rr {
			ss := a.scs.CorporationSectionSummary(r.ID)
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
	ss := a.scs.EveUniverseSectionSummary()
	o := sectionEntity{
		category: sectionGeneral,
		id:       app.EveUniverseSectionEntityID,
		name:     app.EveUniverseSectionEntityName,
		ss:       ss,
	}
	entities = append(entities, o)
	count += 1
	return entities, count
}

func (a *statusWindow) makeSectionList() *widget.List {
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
			s, i := cs.Display()
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

func (a *statusWindow) refreshSections() {
	if a.selectedEntityID == -1 || a.selectedEntityID >= len(a.sectionEntities) {
		return
	}
	se := a.sectionEntities[a.selectedEntityID]
	switch se.category {
	case sectionCharacter:
		a.sections = a.scs.ListCharacterSections(se.id)
	case sectionCorporation:
		a.sections = a.scs.ListCorporationSections(se.id)
	case sectionGeneral:
		a.sections = a.scs.ListEveUniverseSections()
	}
	a.sectionList.Refresh()
	a.sectionsTop.SetText(fmt.Sprintf("%s: Sections", se.name))
}

func (a *statusWindow) refreshDetails() {
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

func (a *statusWindow) makeUpdateSectionAction(entityID int64, sectionID string) func() {
	return func() {
		ctx := context.Background()
		c := a.sectionEntities[a.selectedEntityID]
		switch c.category {
		case sectionGeneral:
			go a.eus.UpdateSectionAndRefreshIfNeeded(
				ctx, app.EveUniverseSection(sectionID), true,
			)
		case sectionCharacter:
			go a.cs.UpdateCharacterSectionAndRefreshIfNeeded(
				ctx, entityID, app.CharacterSection(sectionID), true,
			)
		case sectionCorporation:
			go a.rs.UpdateSectionAndRefreshIfNeeded(
				ctx, entityID, app.CorporationSection(sectionID), true,
			)
		default:
			slog.Error("makeUpdateAllAction: Undefined category", "entity", c)
		}
	}
}

type updateStatusDetail struct {
	widget.BaseWidget

	completedAt *widget.Label
	issue       *widget.Label
	nextUpdate  *widget.Label
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
		nextUpdate:  makeLabel(),
		startedAt:   makeLabel(),
		status:      makeLabel(),
		timeout:     makeLabel(),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *updateStatusDetail) set(ss app.CacheSectionStatus) {
	w.status.Text, w.status.Importance = ss.Display()
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

	w.completedAt.SetText(ihumanize.TimeWithFallback(ss.CompletedAt, "?"))
	w.startedAt.SetText(ihumanize.TimeWithFallback(ss.StartedAt, "-"))
	now := time.Now()
	w.timeout.SetText(humanize.RelTime(now.Add(ss.Timeout), now, "", ""))
	w.nextUpdate.SetText(humanize.RelTime(now, ss.CompletedAt.Add(ss.Timeout), "", ""))
}

func (w *updateStatusDetail) CreateRenderer() fyne.WidgetRenderer {
	layout := kxlayout.NewColumns(100)
	c := container.NewVScroll(container.NewVBox(
		container.New(layout, widget.NewLabel("Status"), w.status),
		container.New(layout, widget.NewLabel("Started"), w.startedAt),
		container.New(layout, widget.NewLabel("Completed"), w.completedAt),
		container.New(layout, widget.NewLabel("Timeout"), w.timeout),
		container.New(layout, widget.NewLabel("Next update"), w.nextUpdate),
		container.New(layout, widget.NewLabel("Issue"), w.issue),
	))
	return widget.NewSimpleRenderer(c)
}
