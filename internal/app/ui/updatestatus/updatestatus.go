// Package updatestatus provides a window that shows the current update status.
package updatestatus

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
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/icons"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type baseUI interface {
	Character() *characterservice.CharacterService
	Corporation() *corporationservice.CorporationService
	ErrorDisplay(err error) string
	EVEImage() app.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	GetOrCreateWindowWithOnClosed(id string, titles ...string) (window fyne.Window, created bool, onClosed func())
	IsMobile() bool
	IsOffline() bool
	Signals() *app.Signals
	StatusCache() *statuscache.StatusCache
}

// Show shows the status window.
func Show(s baseUI) {
	w, ok, onClosed := s.GetOrCreateWindowWithOnClosed("statusWindow", "Update Status")
	if !ok {
		w.Show()
		return
	}
	a := newUpdateStatus(s, w)
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

type updateStatus struct {
	widget.BaseWidget

	charactersTop     *widget.Label
	details           *sectionDetails
	detailsTop        *widget.Label
	entities          fyne.CanvasObject
	entityList        *widget.List
	nav               *xwidget.Navigator
	onEntitySelected  func(int)
	onSectionSelected func(int)
	sb                *xwidget.Snackbar
	sectionEntities   []sectionEntity
	sectionList       *widget.List
	sections          []app.CacheSectionStatus
	sectionsTop       *widget.Label
	selectedEntityID  int
	selectedSectionID int
	signalKey         string
	top2              fyne.CanvasObject
	top3              fyne.CanvasObject
	updateAllSections *widget.Button
	updateSection     *widget.Button
	u                 baseUI
}

func newUpdateStatus(s baseUI, w fyne.Window) *updateStatus {
	a := &updateStatus{
		charactersTop:     awidget.NewLabelWithWrapping(""),
		details:           newSectionDetails(),
		detailsTop:        awidget.NewLabelWithWrapping(""),
		sb:                xwidget.NewSnackbar(w),
		sectionsTop:       awidget.NewLabelWithWrapping(""),
		selectedEntityID:  -1,
		selectedSectionID: -1,
		signalKey:         s.Signals().UniqueKey(),
		u:                 s,
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
	if a.u.IsMobile() {
		sections := container.NewBorder(a.top2, nil, nil, nil, a.sectionList)
		details := container.NewBorder(a.top3, nil, nil, nil, a.details)
		menu := kxwidget.NewIconButtonWithMenu(
			theme.MoreVerticalIcon(),
			fyne.NewMenu("", fyne.NewMenuItem(a.updateAllSections.Text, a.makeUpdateAllAction())),
		)
		a.onEntitySelected = func(_ int) {
			a.nav.Push(xwidget.NewAppBar("Sections", sections, menu))
		}
		a.onSectionSelected = func(id int) {
			s := a.sections[id]
			menu := kxwidget.NewIconButtonWithMenu(
				theme.MoreVerticalIcon(),
				fyne.NewMenu(
					"", fyne.NewMenuItem(a.updateSection.Text, a.makeUpdateSectionAction(s.EntityID, s.SectionID))),
			)
			a.nav.Push(xwidget.NewAppBar("Section Detail", details, menu))
		}
	}

	// Signals
	a.sb.Start()
	a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	}, a.signalKey)
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	}, a.signalKey)
	a.u.Signals().CharacterSectionUpdated.AddListener(func(ctx context.Context, _ app.CharacterSectionUpdated) {
		a.update(ctx)
	}, a.signalKey)
	a.u.Signals().CorporationSectionUpdated.AddListener(func(ctx context.Context, _ app.CorporationSectionUpdated) {
		a.update(ctx)
	}, a.signalKey)
	a.u.Signals().EveUniverseSectionUpdated.AddListener(func(ctx context.Context, _ app.EveUniverseSectionUpdated) {
		a.update(ctx)
	}, a.signalKey)
	return a
}

func (a *updateStatus) stop() {
	a.sb.Stop()
	a.u.Signals().CharacterAdded.RemoveListener(a.signalKey)
	a.u.Signals().CharacterRemoved.RemoveListener(a.signalKey)
	a.u.Signals().CharacterSectionUpdated.RemoveListener(a.signalKey)
	a.u.Signals().CorporationSectionUpdated.RemoveListener(a.signalKey)
	a.u.Signals().EveUniverseSectionUpdated.RemoveListener(a.signalKey)
}

func (a *updateStatus) CreateRenderer() fyne.WidgetRenderer {
	updateMenu := fyne.NewMenu("",
		fyne.NewMenuItem("Update all characters", func() {
			go func() {
				err := a.u.Character().UpdateCharactersIfNeeded(context.Background(), true)
				if err != nil {
					slog.Error("update status", "error", err)
					a.sb.Show("Error: " + a.u.ErrorDisplay(err))
				}
			}()
		}),
		fyne.NewMenuItem("Update all corporations", func() {
			go func() {
				err := a.u.Corporation().UpdateCorporationsIfNeeded(context.Background(), true)
				if err != nil {
					slog.Error("update status", "error", err)
					a.sb.Show("Error: " + a.u.ErrorDisplay(err))
				}
			}()

		}),
		fyne.NewMenuItem("Update all general topics", func() {
			go a.u.EVEUniverse().UpdateSectionsIfNeeded(context.Background(), true)
		}),
	)
	updateEntities := xwidget.NewContextMenuButton("Force update all entities", updateMenu)
	var c fyne.CanvasObject
	if a.u.IsMobile() {
		ab := xwidget.NewAppBar("Home", a.entities, kxwidget.NewIconButtonWithMenu(
			theme.MoreVerticalIcon(),
			updateMenu,
		))
		a.nav = xwidget.NewNavigator(ab)
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
	isOfflineMode := a.u.IsOffline()
	list := widget.NewList(
		func() int {
			return len(a.sectionEntities)
		},
		func() fyne.CanvasObject {
			return newEntityItem(
				isOfflineMode,
				a.u.EVEImage().CharacterPortraitAsync,
				a.u.EVEImage().CorporationLogoAsync,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.sectionEntities) {
				return
			}
			co.(*entityItem).set(a.sectionEntities[id])
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
			go a.u.EVEUniverse().UpdateSectionsIfNeeded(ctx, true)
		case sectionCharacter:
			go a.u.Character().UpdateCharacterAndRefreshIfNeeded(ctx, c.id, true)
		case sectionCorporation:
			go a.u.Corporation().UpdateCorporationAndRefreshIfNeeded(ctx, c.id, true)
		default:
			panic(fmt.Sprintf("makeUpdateAllAction: Undefined category: %v", c.category))
		}
	}
}

func (a *updateStatus) update(ctx context.Context) {
	entities, count := a.updateEntityList(ctx)

	fyne.Do(func() {
		a.sectionEntities = entities
		a.entityList.Refresh()
		a.refreshSections()
		a.charactersTop.SetText(fmt.Sprintf("Entities: %d", count))
		a.refreshDetails()
	})
}

func (a *updateStatus) updateEntityList(_ context.Context) ([]sectionEntity, int) {
	var count int
	var entities []sectionEntity
	cc := a.u.StatusCache().ListCharacters()
	if len(cc) > 0 {
		entities = append(entities, sectionEntity{category: sectionHeader, name: "Characters"})
		count += len(cc)
		for _, c := range cc {
			ss := a.u.StatusCache().CharacterSectionSummary(c.ID)
			o := sectionEntity{
				category: sectionCharacter,
				id:       c.ID,
				name:     c.Name,
				ss:       ss,
			}
			entities = append(entities, o)
		}
	}
	rr := a.u.StatusCache().ListCorporations()
	if len(rr) > 0 {
		entities = append(entities, sectionEntity{category: sectionHeader, name: "Corporations"})
		count += len(rr)
		for _, r := range rr {
			ss := a.u.StatusCache().CorporationSectionSummary(r.ID)
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
	ss := a.u.StatusCache().EveUniverseSectionSummary()
	o := sectionEntity{
		category: sectionGeneral,
		id:       app.EveUniverseSectionEntityID,
		name:     app.EveUniverseSectionEntityName,
		ss:       ss,
	}
	entities = append(entities, o)
	count++
	return entities, count
}

func (a *updateStatus) makeSectionList() *widget.List {
	isOfflineMode := a.u.IsOffline()
	l := widget.NewList(
		func() int {
			return len(a.sections)
		},
		func() fyne.CanvasObject {
			return newSectionItem(isOfflineMode)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.sections) {
				return
			}
			co.(*sectionItem).set(a.sections[id])
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
		a.sections = a.u.StatusCache().ListCharacterSections(se.id)
	case sectionCorporation:
		a.sections = a.u.StatusCache().ListCorporationSections(se.id)
	case sectionGeneral:
		a.sections = a.u.StatusCache().ListEveUniverseSections()
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

func (a *updateStatus) makeUpdateSectionAction(entityID int64, sectionID string) func() {
	return func() {
		ctx := context.Background()
		c := a.sectionEntities[a.selectedEntityID]
		switch c.category {
		case sectionGeneral:
			go a.u.EVEUniverse().UpdateSectionAndRefreshIfNeeded(
				ctx, app.EveUniverseSection(sectionID), true,
			)
		case sectionCharacter:
			go a.u.Character().UpdateCharacterSectionAndRefreshIfNeeded(
				ctx, entityID, app.CharacterSection(sectionID), true,
			)
		case sectionCorporation:
			go a.u.Corporation().UpdateSectionAndRefreshIfNeeded(
				ctx, entityID, app.CorporationSection(sectionID), true,
			)
		default:
			slog.Error("makeUpdateAllAction: Undefined category", "entity", c)
		}
	}
}

type loadFuncAsync func(int64, int, func(fyne.Resource))

type entityItem struct {
	widget.BaseWidget

	icon                *canvas.Image
	isOfflineMode       bool
	loadCharacterIcon   loadFuncAsync
	loadCorporationIcon loadFuncAsync
	name                *widget.Label
	spinner             *widget.Activity
	status              *widget.Label
	thief               *xwidget.HooverThief
}

func newEntityItem(isOfflineMode bool, loadCharacterIcon, loadCorporationIcon loadFuncAsync) *entityItem {
	icon := xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
	name := widget.NewLabel("Template")
	status := widget.NewLabel("Template")
	spinner := widget.NewActivity()
	w := &entityItem{
		icon:                icon,
		isOfflineMode:       isOfflineMode,
		loadCharacterIcon:   loadCharacterIcon,
		loadCorporationIcon: loadCorporationIcon,
		name:                name,
		spinner:             spinner,
		status:              status,
		thief:               xwidget.NewHooverThief(),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *entityItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(
		container.NewHBox(w.icon, w.name, w.spinner, layout.NewSpacer(), w.status),
		w.thief,
	)
	return widget.NewSimpleRenderer(c)
}

func (w *entityItem) set(r sectionEntity) {
	w.name.Text = r.name

	switch r.category {
	case sectionGeneral:
		w.name.TextStyle.Bold = false
		w.name.Refresh()
		w.icon.Resource = eveicon.FromName(eveicon.StarMap)
		w.icon.Refresh()
	case sectionCharacter, sectionCorporation:
		w.name.TextStyle.Bold = false
		w.name.Refresh()
		switch r.category {
		case sectionCharacter:
			w.loadCharacterIcon(r.id, app.IconPixelSize, func(r fyne.Resource) {
				w.icon.Resource = r
				w.icon.Refresh()
			})
		case sectionCorporation:
			w.loadCorporationIcon(r.id, app.IconPixelSize, func(r fyne.Resource) {
				w.icon.Resource = r
				w.icon.Refresh()
			})
		default:
			w.icon.Resource = icons.BlankSvg
			w.icon.Refresh()
		}
	case sectionHeader:
		w.name.TextStyle.Bold = true
		w.name.Refresh()
		w.icon.Hide()
		w.spinner.Hide()
		w.status.Hide()
		w.thief.Show()
		return
	}

	w.thief.Hide()
	if r.ss.IsRunning && !w.isOfflineMode {
		w.spinner.Start()
		w.spinner.Show()
	} else {
		w.spinner.Stop()
		w.spinner.Hide()
	}

	w.icon.Show()

	t := r.ss.Display()
	i := r.ss.Status().ToImportance2()
	w.status.Text = t
	w.status.Importance = i
	w.status.Refresh()
	w.status.Show()
}

type sectionItem struct {
	widget.BaseWidget

	isOfflineMode bool
	name          *widget.Label
	spinner       *widget.Activity
	status        *widget.Label
}

func newSectionItem(isOfflineMode bool) *sectionItem {
	name := widget.NewLabel("")
	status := widget.NewLabel("")
	spinner := widget.NewActivity()
	w := &sectionItem{
		name:          name,
		spinner:       spinner,
		status:        status,
		isOfflineMode: isOfflineMode,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *sectionItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewHBox(
		w.name,
		w.spinner,
		layout.NewSpacer(),
		w.status,
	)
	return widget.NewSimpleRenderer(c)
}

func (w *sectionItem) set(r app.CacheSectionStatus) {
	w.name.SetText(r.SectionName)
	s, i := r.Display()
	w.status.Text = s
	w.status.Importance = i
	w.status.Refresh()

	if r.IsRunning() && !w.isOfflineMode {
		w.spinner.Start()
		w.spinner.Show()
	} else {
		w.spinner.Stop()
		w.spinner.Hide()
	}
}

type sectionDetails struct {
	widget.BaseWidget

	completedAt *widget.Label
	issue       *widget.Label
	nextUpdate  *widget.Label
	startedAt   *widget.Label
	status      *widget.Label
	timeout     *widget.Label
}

func newSectionDetails() *sectionDetails {
	w := &sectionDetails{
		completedAt: awidget.NewLabelWithWrapping(""),
		issue:       awidget.NewLabelWithWrapping(""),
		nextUpdate:  awidget.NewLabelWithWrapping(""),
		startedAt:   awidget.NewLabelWithWrapping(""),
		status:      awidget.NewLabelWithWrapping(""),
		timeout:     awidget.NewLabelWithWrapping(""),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *sectionDetails) CreateRenderer() fyne.WidgetRenderer {
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

func (w *sectionDetails) set(ss app.CacheSectionStatus) {
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
