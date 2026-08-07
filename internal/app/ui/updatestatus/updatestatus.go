// Package updatestatus shows the current update status in a window.
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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type baseUI interface {
	Character() *characterservice.CharacterService
	Corporation() *corporationservice.CorporationService
	ErrorDisplay(err error) string
	EVEImage() ui.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	GetOrCreateWindowWithOnClosed(id string, titles ...string) (window fyne.Window, created bool, onClosed func())
	IsMobile() bool
	IsOffline() bool
	Signals() *app.Signals
	StatusCache() *statuscache.StatusCache
}

// Show shows the update status window.
func Show(s baseUI) {
	w, ok, onClosed := s.GetOrCreateWindowWithOnClosed("statusWindow", "Update Status")
	if !ok {
		w.Show()
		return
	}
	a := newUpdateStatus(s, w)
	w.SetContent(a)
	w.Resize(fyne.Size{Width: 400, Height: 600})
	w.SetOnClosed(func() {
		a.stop()
		if onClosed != nil {
			onClosed()
		}
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

func (sc sectionCategory) name() string {
	switch sc {
	case sectionCharacter:
		return "Characters"
	case sectionCorporation:
		return "Corporations"
	case sectionGeneral:
		return "General"
	}
	return "?"
}

// An entity which has update sections, e.g. a character
type entity struct {
	id       int64
	name     string
	category sectionCategory
	ss       app.StatusSummary
}

type updateStatus struct {
	widget.BaseWidget

	currentEntityID   int
	currentSectionID  int
	currentSections   []app.CacheSectionStatus
	entities          []entity
	entityList        *widget.List
	entityMoreButton  *kxwidget.IconButton
	nav               *xwidget.Navigator
	sb                *xwidget.Snackbar
	sectionStatus     *sectionStatus
	sectionList       *widget.List
	sectionMoreButton *kxwidget.IconButton
	signalKey         string
	u                 baseUI
}

func newUpdateStatus(u baseUI, w fyne.Window) *updateStatus {
	a := &updateStatus{
		sectionStatus:    newSectionStatus(u.IsMobile()),
		sb:               xwidget.NewSnackbar(w.Canvas()),
		currentEntityID:  -1,
		currentSectionID: -1,
		signalKey:        u.Signals().UniqueKey(),
		u:                u,
	}
	a.ExtendBaseWidget(a)
	a.entityList = a.makeEntityList()

	a.sectionList = a.makeSectionList()
	a.entityMoreButton = kxwidget.NewIconButtonWithMenu(theme.MoreVerticalIcon(), fyne.NewMenu(""))
	a.entityMoreButton.Disable()
	a.sectionMoreButton = kxwidget.NewIconButtonWithMenu(theme.MoreVerticalIcon(), fyne.NewMenu(""))
	a.sectionMoreButton.Disable()

	menu := fyne.NewMenu("",
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
	ab := xwidget.NewAppBar("Update status", a.entityList, kxwidget.NewIconButtonWithMenu(
		theme.MoreVerticalIcon(), menu,
	))
	ab.HideBackground = !a.u.IsMobile()
	a.nav = xwidget.NewNavigator(ab)

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
	return widget.NewSimpleRenderer(a.nav)
}

func (a *updateStatus) makeEntityList() *widget.List {
	isOfflineMode := a.u.IsOffline()
	list := widget.NewList(
		func() int {
			return len(a.entities)
		},
		func() fyne.CanvasObject {
			return newEntityItem(
				isOfflineMode,
				a.u.EVEImage().CharacterPortraitAsync,
				a.u.EVEImage().CorporationLogoAsync,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.entities) {
				return
			}
			co.(*entityItem).set(a.entities[id])
		})

	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.entities) || a.entities[id].category == sectionHeader {
			list.UnselectAll()
			return
		}

		a.currentEntityID = id
		a.currentSectionID = -1
		a.sectionList.UnselectAll()

		if !a.u.IsOffline() {
			a.entityMoreButton.SetMenuItems(a.makeEntityMenuItems())
			a.entityMoreButton.Enable()
		}

		x := a.entities[id]
		subTitle := widget.NewLabel(x.category.name())
		subTitle.TextStyle.Bold = true
		c := container.NewBorder(subTitle, nil, nil, nil, a.sectionList)
		ab := xwidget.NewAppBar(x.name, c, a.entityMoreButton)
		ab.HideBackground = !a.u.IsMobile()
		a.nav.Push(ab)

		list.UnselectAll()
		a.refreshSections()
		a.refreshDetails()
	}
	return list
}

func (a *updateStatus) makeEntityMenuItems() []*fyne.MenuItem {
	action := func() {
		ctx := context.Background()
		c := a.entities[a.currentEntityID]
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
	item := fyne.NewMenuItem("Update all sections", action)
	return []*fyne.MenuItem{item}
}

func (a *updateStatus) update(ctx context.Context) {
	entities, _ := a.updateEntityList(ctx)

	fyne.Do(func() {
		a.entities = entities
		a.entityList.Refresh()
		a.refreshSections()
		a.refreshDetails()
	})
}

func (a *updateStatus) updateEntityList(_ context.Context) ([]entity, int) {
	var count int
	var entities []entity
	cc := a.u.StatusCache().ListCharacters()
	if len(cc) > 0 {
		entities = append(entities, entity{category: sectionHeader, name: "Characters"})
		count += len(cc)
		for _, c := range cc {
			ss := a.u.StatusCache().CharacterSectionSummary(c.ID)
			o := entity{
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
		entities = append(entities, entity{category: sectionHeader, name: "Corporations"})
		count += len(rr)
		for _, r := range rr {
			ss := a.u.StatusCache().CorporationSectionSummary(r.ID)
			o := entity{
				category: sectionCorporation,
				id:       r.ID,
				name:     r.Name,
				ss:       ss,
			}
			entities = append(entities, o)
		}
	}
	entities = append(entities, entity{category: sectionHeader, name: "General"})
	ss := a.u.StatusCache().EveUniverseSectionSummary()
	o := entity{
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
			return len(a.currentSections)
		},
		func() fyne.CanvasObject {
			return newSectionItem(isOfflineMode)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.currentSections) {
				return
			}
			co.(*sectionItem).set(a.currentSections[id])
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.currentSections) {
			l.UnselectAll()
			return
		}
		a.currentSectionID = id
		a.refreshDetails()
		x1 := a.entities[a.currentEntityID]
		x2 := a.currentSections[id]
		subTitle := widget.NewLabel(fmt.Sprintf("%s > %s", x1.category.name(), x2.EntityName))
		subTitle.TextStyle.Bold = true
		c := container.NewBorder(subTitle, nil, nil, nil, a.sectionStatus)
		ab := xwidget.NewAppBar(x2.SectionName, c, a.sectionMoreButton)
		ab.HideBackground = !a.u.IsMobile()
		a.nav.Push(ab)
		l.UnselectAll()
	}
	return l
}

func (a *updateStatus) refreshSections() {
	if a.currentEntityID == -1 || a.currentEntityID >= len(a.entities) {
		return
	}
	se := a.entities[a.currentEntityID]
	switch se.category {
	case sectionCharacter:
		a.currentSections = a.u.StatusCache().ListCharacterSections(se.id)
	case sectionCorporation:
		a.currentSections = a.u.StatusCache().ListCorporationSections(se.id)
	case sectionGeneral:
		a.currentSections = a.u.StatusCache().ListEveUniverseSections()
	}
	a.sectionList.Refresh()
}

func (a *updateStatus) refreshDetails() {
	id := a.currentSectionID
	if id == -1 || id >= len(a.currentSections) {
		a.sectionStatus.Hide()
		a.sectionMoreButton.Disable()
		return
	}
	ss := a.currentSections[id]
	if !a.u.IsOffline() {
		a.sectionMoreButton.SetMenuItems(a.makeSectionMenuItems(ss.EntityID, ss.SectionID))
		a.sectionMoreButton.Enable()
	}
	a.sectionStatus.set(ss)
	a.sectionStatus.Show()
}

func (a *updateStatus) makeSectionMenuItems(entityID int64, sectionID string) []*fyne.MenuItem {
	action := func() {
		ctx := context.Background()
		c := a.entities[a.currentEntityID]
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
	item := fyne.NewMenuItem("Update section", action)
	return []*fyne.MenuItem{item}
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
	icon := xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(ui.IconUnitSize))
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

func (w *entityItem) set(r entity) {
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
			w.loadCharacterIcon(r.id, ui.IconPixelSize, func(r fyne.Resource) {
				w.icon.Resource = r
				w.icon.Refresh()
			})
		case sectionCorporation:
			w.loadCorporationIcon(r.id, ui.IconPixelSize, func(r fyne.Resource) {
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

type sectionStatus struct {
	widget.BaseWidget

	completedAt *widget.Label
	issue       *widget.Label
	nextUpdate  *widget.Label
	startedAt   *widget.Label
	status      *widget.Label
	timeout     *widget.Label
	isMobile    bool
}

func newSectionStatus(isMobile bool) *sectionStatus {
	w := &sectionStatus{
		completedAt: ui.NewLabelWithWrapping(""),
		issue:       ui.NewLabelWithWrapping(""),
		nextUpdate:  ui.NewLabelWithWrapping(""),
		startedAt:   ui.NewLabelWithWrapping(""),
		status:      ui.NewLabelWithWrapping(""),
		timeout:     ui.NewLabelWithWrapping(""),
		isMobile:    isMobile,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *sectionStatus) CreateRenderer() fyne.WidgetRenderer {
	c := widget.NewForm(
		widget.NewFormItem("Status", w.status),
		widget.NewFormItem("Started", w.startedAt),
		widget.NewFormItem("Completed", w.completedAt),
		widget.NewFormItem("Timeout", w.timeout),
		widget.NewFormItem("Next update", w.nextUpdate),
		widget.NewFormItem("Issue", w.issue),
	)
	if w.isMobile {
		c.Orientation = widget.Adaptive
	}
	return widget.NewSimpleRenderer(container.NewVScroll(c))
}

func (w *sectionStatus) set(ss app.CacheSectionStatus) {
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
