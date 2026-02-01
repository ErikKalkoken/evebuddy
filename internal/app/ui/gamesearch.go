package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
)

const (
	maxSearchResults = 500 // max results returned from the server
)

type gameSearch struct {
	widget.BaseWidget

	categories          *kxwidget.FilterChipGroup
	defaultCategories   []string
	entry               *widget.Entry
	indicator           *widget.ProgressBarInfinite
	mu                  sync.RWMutex
	recent              *widget.List
	recentItems         []*app.EveEntity
	recentPage          *fyne.Container
	resultCount         *widget.Label
	results             *iwidget.Tree[resultNode]
	resultsPage         *fyne.Container
	searchOptions       *widget.Accordion
	strict              *kxwidget.Switch
	supportedCategories set.Set[app.EveEntityCategory]
	u                   *baseUI
	w                   fyne.Window
}

func newGameSearch(u *baseUI) *gameSearch {
	a := &gameSearch{
		defaultCategories:   makeOptions(),
		entry:               widget.NewEntry(),
		indicator:           widget.NewProgressBarInfinite(),
		recentItems:         make([]*app.EveEntity, 0),
		resultCount:         widget.NewLabel(""),
		supportedCategories: infoWindowSupportedEveEntities(),
		u:                   u,
		w:                   u.MainWindow(),
	}
	a.ExtendBaseWidget(a)

	a.categories = kxwidget.NewFilterChipGroup(a.defaultCategories, func(s []string) {
		a.updateSearchOptionsTitle()
	})
	a.categories.Selected = a.defaultCategories

	a.strict = kxwidget.NewSwitch(func(on bool) {
		a.updateSearchOptionsTitle()
	})
	a.strict.On = false

	a.resultCount = widget.NewLabel("")
	a.resultCount.Hide()
	a.results = a.makeResults()
	a.entry.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.reset()
	})
	a.entry.PlaceHolder = "Search New Eden"
	a.entry.OnSubmitted = func(s string) {
		go a.doSearch2(s)
	}
	a.indicator.Hide()

	a.searchOptions = widget.NewAccordion(
		widget.NewAccordionItem(
			"",
			container.NewVBox(
				a.categories,
				container.New(
					layout.NewCustomPaddedHBoxLayout(0),
					a.strict,
					kxwidget.NewTappableLabel("Strict search", func() {
						a.strict.SetOn(!a.strict.On)
					})),
				widget.NewButton("Reset", a.resetOptions),
			),
		),
	)
	a.updateSearchOptionsTitle()

	a.recent = a.makeRecentSelected()

	a.resultsPage = container.NewBorder(
		container.NewVBox(
			a.indicator,
			a.resultCount,
		),
		nil,
		nil,
		nil,
		a.results,
	)
	clearRecent := widget.NewHyperlink("Clear", nil)
	clearRecent.OnTapped = func() {
		a.setRecentItems(make([]*app.EveEntity, 0))
		a.storeRecentItems()
	}
	a.recentPage = container.NewBorder(
		container.NewHBox(widget.NewLabel("Recent searches"), layout.NewSpacer(), clearRecent),
		nil,
		nil,
		nil,
		a.recent,
	)
	a.init()
	return a
}

func (a *gameSearch) init() {
	ids := a.u.settings.RecentSearches()
	if len(ids) == 0 {
		return
	}
	ee, err := a.u.eus.ListEntitiesForIDs(context.Background(), ids)
	if errors.Is(err, app.ErrNotFound) {
		ee = make([]*app.EveEntity, 0)
		fyne.Do(func() {
			a.setRecentItems(ee)
		})
		return
	}
	if err != nil {
		slog.Error("failed to load recent items from settings", "error", err)
		return
	}
	fyne.Do(func() {
		a.setRecentItems(ee)
		a.showRecent()
	})
}

func (a *gameSearch) resetOptions() {
	a.categories.SetSelected(a.defaultCategories)
	a.strict.SetOn(false)
	a.updateSearchOptionsTitle()
}

func (a *gameSearch) updateSearchOptionsTitle() {
	isDefault := func() bool {
		if a.strict.On {
			return false
		}
		if !set.Of(a.categories.Selected...).Equal(set.Of(a.defaultCategories...)) {
			return false
		}
		return true
	}()
	s := "Search options"
	if !isDefault {
		s += " (changed)"
	}
	a.searchOptions.Items[0].Title = s
	a.searchOptions.Refresh()
}

func (a *gameSearch) doSearch(s string) {
	a.entry.SetText(s)
	go a.doSearch2(s)
}

func (a *gameSearch) toogleOptions(enabled bool) {
	if enabled {
		a.searchOptions.Open(0)
	} else {
		a.searchOptions.Close(0)
	}
	a.searchOptions.Refresh()
}

func (a *gameSearch) setRecentItems(ee []*app.EveEntity) {
	a.mu.Lock()
	a.recentItems = ee
	a.mu.Unlock()
	a.recent.Refresh()
}

func (a *gameSearch) storeRecentItems() {
	ids := xslices.Map(a.recentItems, func(x *app.EveEntity) int32 {
		return x.ID
	})
	a.u.settings.SetRecentSearches(ids)
}

func (a *gameSearch) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(
			a.entry,
			a.searchOptions,
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewStack(a.recentPage, a.resultsPage),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *gameSearch) focus() {
	a.w.Canvas().Focus(a.entry)
}

func (a *gameSearch) reset() {
	a.entry.SetText("")
	a.clearResults()
}

func (a *gameSearch) setWindow(w fyne.Window) {
	a.w = w
}

func (a *gameSearch) makeResults() *iwidget.Tree[resultNode] {
	t := iwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			if isBranch {
				return widget.NewLabel("Template")
			}
			return newSearchResult(a.u.eis, a.u.eus, a.supportedCategories)
		},
		func(n *resultNode, isBranch bool, co fyne.CanvasObject) {
			if isBranch {
				co.(*widget.Label).SetText(n.String())
				return
			}
			co.(*searchResult).set(n.ee)
		},
	)
	t.OnSelectedNode = func(n *resultNode) {
		defer t.UnselectAll()
		if n.isCategory() {
			t.ToggleBranchNode(n)
			return
		}
		a.showSupportedResult(n.ee)
		a.mu.Lock()
		a.recentItems = slices.DeleteFunc(a.recentItems, func(a *app.EveEntity) bool {
			return a.ID == n.ee.ID
		})
		a.recentItems = slices.Insert(a.recentItems, 0, n.ee)
		a.storeRecentItems()
		a.mu.Unlock()
		a.recent.Refresh()
	}
	return t
}

func (a *gameSearch) showSupportedResult(o *app.EveEntity) {
	if !a.supportedCategories.Contains(o.Category) {
		return
	}
	a.u.ShowEveEntityInfoWindow(o)
}

func (a *gameSearch) makeRecentSelected() *widget.List {
	l := widget.NewList(
		func() int {
			a.mu.RLock()
			defer a.mu.RUnlock()
			return len(a.recentItems)
		},
		func() fyne.CanvasObject {
			return newSearchResult(a.u.eis, a.u.eus, infoWindowSupportedEveEntities())
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			a.mu.RLock()
			defer a.mu.RUnlock()
			if id >= len(a.recentItems) {
				return
			}
			it := a.recentItems[id]
			co.(*searchResult).set(it)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		a.mu.RLock()
		defer l.UnselectAll()
		if id >= len(a.recentItems) {
			a.mu.RUnlock()
			return
		}
		it := a.recentItems[id]
		a.mu.RUnlock()
		a.showSupportedResult(it)
	}
	return l
}

func (a *gameSearch) clearResults() {
	fyne.Do(func() {
		a.results.Clear()
		a.resultCount.Hide()
		a.showRecent()
	})
}

func (a *gameSearch) showRecent() {
	a.resultsPage.Hide()
	a.recentPage.Show()
}

func (a *gameSearch) doSearch2(search string) {
	if a.u.IsOffline() {
		fyne.Do(func() {
			a.u.ShowInformationDialog(
				"Offline",
				"Can't search when offline",
				a.w,
			)
		})
		return
	}
	a.clearResults()
	if search == "" {
		return
	}
	fyne.Do(func() {
		a.recentPage.Hide()
		a.resultsPage.Show()
		a.indicator.Show()
		a.indicator.Start()
	})
	defer func() {
		fyne.Do(func() {
			a.indicator.Stop()
			a.indicator.Hide()
		})
	}()
	categories := xslices.Map(a.categories.Selected, func(o string) app.SearchCategory {
		return option2searchCategory(o)
	})
	results, total, err := a.u.cs.SearchESI(
		context.Background(),
		search,
		categories,
		a.strict.On,
	)
	if err != nil {
		fyne.Do(func() {
			a.u.showErrorDialog("Search failed", err, a.u.MainWindow())
		})
		return
	}
	fyne.Do(func() {
		if total == maxSearchResults {
			a.resultCount.Importance = widget.WarningImportance
			a.resultCount.Wrapping = fyne.TextWrapWord
			a.resultCount.Text = fmt.Sprintf(
				"Search for \"%s\" exceeded the server limit of 500 results "+
					"and may not contain the items you are looking for.",
				search,
			)
		} else {
			a.resultCount.Importance = widget.MediumImportance
			a.resultCount.Text = fmt.Sprintf("%d Results", total)
		}
		a.resultCount.Refresh()
		a.resultCount.Show()

	})
	if total == 0 {
		return
	}
	var td iwidget.TreeData[resultNode]
	var categoriesFound int
	for _, c := range categories {
		items, ok := results[c]
		if !ok {
			continue
		}
		categoriesFound++
		itemsCount := len(items)
		category := &resultNode{category: c, count: itemsCount}
		err := td.Add(nil, category, itemsCount > 0)
		if err != nil {
			slog.Error("game search: adding node", "node", category)
			continue
		}
		for _, o := range items {
			entity := &resultNode{ee: o}
			td.Add(category, entity, false)
		}
	}
	fyne.Do(func() {
		a.results.Set(td)
		if categoriesFound == 1 {
			a.results.OpenAllBranches()
		}
	})
}

type searchResultEIS interface {
	AllianceLogo(int32, int) (fyne.Resource, error)
	CharacterPortrait(int32, int) (fyne.Resource, error)
	CorporationLogo(int32, int) (fyne.Resource, error)
	FactionLogo(int32, int) (fyne.Resource, error)
	InventoryTypeIcon(int32, int) (fyne.Resource, error)
	InventoryTypeSKIN(int32, int) (fyne.Resource, error)
	InventoryTypeBPO(int32, int) (fyne.Resource, error)
}

type searchResultEUS interface {
	GetOrCreateTypeESI(context.Context, int32) (*app.EveType, error)
}

var searchResultResourceCache xsync.Map[int32, fyne.Resource]

type searchResult struct {
	widget.BaseWidget

	eis                 searchResultEIS
	eus                 searchResultEUS
	image               *canvas.Image
	name                *widget.Label
	supportedCategories set.Set[app.EveEntityCategory]
}

func newSearchResult(eis searchResultEIS, eus searchResultEUS, supportedCategories set.Set[app.EveEntityCategory]) *searchResult {
	w := &searchResult{
		supportedCategories: supportedCategories,
		name:                widget.NewLabel(""),
		image: iwidget.NewImageFromResource(
			icons.BlankSvg,
			fyne.NewSquareSize(app.IconUnitSize),
		),
		eis: eis,
		eus: eus,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *searchResult) set(o *app.EveEntity) {
	var i widget.Importance
	if !w.supportedCategories.Contains(o.Category) {
		i = widget.LowImportance
	}
	w.name.Importance = i
	w.name.Text = o.Name
	w.name.Refresh()

	iwidget.LoadResourceAsyncWithCache(
		icons.BlankSvg,
		func() (fyne.Resource, bool) {
			return searchResultResourceCache.Load(o.ID)
		},
		func(r fyne.Resource) {
			w.image.Resource = r
			w.image.Refresh()

		},
		func() (fyne.Resource, error) {
			switch o.Category {
			case app.EveEntityInventoryType:
				et, err := w.eus.GetOrCreateTypeESI(context.Background(), o.ID)
				if err != nil {
					return nil, err
				}
				switch et.Group.Category.ID {
				case app.EveCategorySKINs:
					return w.eis.InventoryTypeSKIN(et.ID, app.IconPixelSize)
				case app.EveCategoryBlueprint:
					return w.eis.InventoryTypeBPO(et.ID, app.IconPixelSize)
				default:
					return w.eis.InventoryTypeIcon(et.ID, app.IconPixelSize)
				}
			default:
				return entityIcon(w.eis, o, app.IconPixelSize, icons.BlankSvg)
			}
		},
		func(r fyne.Resource) {
			searchResultResourceCache.Store(o.ID, r)
		},
	)
}

func (w *searchResult) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, container.NewPadded(w.image), nil, w.name)
	return widget.NewSimpleRenderer(c)
}

var searchCategory2optionMap = map[app.SearchCategory]string{
	app.SearchAgent:         "Agents",
	app.SearchAlliance:      "Alliances",
	app.SearchCharacter:     "Characters",
	app.SearchConstellation: "Constellations",
	app.SearchCorporation:   "Corporations",
	app.SearchFaction:       "Factions",
	app.SearchRegion:        "Regions",
	app.SearchSolarSystem:   "Systems",
	app.SearchStation:       "Stations",
	app.SearchType:          "Types",
}

func searchCategory2option(c app.SearchCategory) string {
	x, ok := searchCategory2optionMap[c]
	if !ok {
		panic(fmt.Sprintf("searchCategory2option: \"%s\" not found", c))
	}
	return x
}

func option2searchCategory(o string) app.SearchCategory {
	for k, v := range searchCategory2optionMap {
		if v == o {
			return k
		}
	}
	panic(fmt.Sprintf("option2searchCategory: \"%s\" not found", o))
}

func makeOptions() []string {
	options := xslices.Map(app.SearchCategories(), func(c app.SearchCategory) string {
		return searchCategory2option(c)
	})
	slices.Sort(options)
	return options
}

type resultNode struct {
	category app.SearchCategory
	count    int
	ee       *app.EveEntity
}

func (sn resultNode) isCategory() bool {
	return sn.ee == nil
}

func (sn resultNode) String() string {
	if sn.isCategory() {
		return fmt.Sprintf("%s (%d)", sn.category.String(), sn.count)
	}
	return sn.ee.Name
}
