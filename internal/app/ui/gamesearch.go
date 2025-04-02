package ui

import (
	"context"
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

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infowindow"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

const (
	maxSearchResults = 500 // max results returned from the server
)

type SearchResult struct {
	widget.BaseWidget
	eis                 app.EveImageService
	eus                 app.EveUniverseService
	name                *widget.Label
	image               *canvas.Image
	supportedCategories set.Set[app.EveEntityCategory]
}

func NewSearchResult(
	eis app.EveImageService,
	eus app.EveUniverseService,
	supportedCategories set.Set[app.EveEntityCategory]) *SearchResult {
	w := &SearchResult{
		eis:                 eis,
		eus:                 eus,
		supportedCategories: supportedCategories,
		name:                widget.NewLabel(""),
		image:               iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *SearchResult) Set(o *app.EveEntity) {
	w.name.Text = o.Name
	var i widget.Importance
	if !w.supportedCategories.Contains(o.Category) {
		i = widget.LowImportance
	}
	w.name.Importance = i
	w.name.Refresh()
	imageCategory := o.Category.ToEveImage()
	if imageCategory == "" {
		w.image.Resource = icons.BlankSvg
		w.image.Refresh()
		return
	}
	go func() {
		ctx := context.Background()
		res, err := func() (fyne.Resource, error) {
			switch o.Category {
			case app.EveEntityInventoryType:
				et, err := w.eus.GetOrCreateTypeESI(ctx, o.ID)
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
				return w.eis.EntityIcon(o.ID, imageCategory, app.IconPixelSize)
			}
		}()
		if err != nil {
			res = theme.BrokenImageIcon()
			slog.Error("failed to load w.image", "error", err)
		}
		w.image.Resource = res
		w.image.Refresh()
	}()
}

func (w *SearchResult) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, container.NewPadded(w.image), nil, w.name)
	return widget.NewSimpleRenderer(c)
}

type GameSearch struct {
	widget.BaseWidget

	categories          *iwidget.FilterChipGroup
	defaultCategories   []string
	entry               *widget.Entry
	indicator           *widget.ProgressBarInfinite
	recent              *widget.List
	recentPage          *fyne.Container
	resultCount         *widget.Label
	results             *iwidget.Tree[resultNode]
	resultsPage         *fyne.Container
	searchOptions       *widget.Accordion
	strict              *kxwidget.Switch
	supportedCategories set.Set[app.EveEntityCategory]
	u                   *BaseUI
	w                   fyne.Window

	mu          sync.RWMutex
	recentItems []*app.EveEntity
}

func NewGameSearch(u *BaseUI) *GameSearch {
	a := &GameSearch{
		defaultCategories:   makeOptions(),
		entry:               widget.NewEntry(),
		indicator:           widget.NewProgressBarInfinite(),
		recentItems:         make([]*app.EveEntity, 0),
		resultCount:         widget.NewLabel(""),
		supportedCategories: infowindow.SupportedEveEntities(),
		u:                   u,
		w:                   u.MainWindow(),
	}
	a.ExtendBaseWidget(a)

	a.categories = iwidget.NewFilterChipGroup(a.defaultCategories, nil)
	a.categories.Selected = slices.Clone(a.defaultCategories)
	a.categories.OnChanged = func(s []string) {
		a.updateSearchOptionsTitle()
	}

	a.strict = kxwidget.NewSwitch(nil)
	a.strict.On = false
	a.strict.OnChanged = func(on bool) {
		a.updateSearchOptionsTitle()
	}
	a.resultCount = widget.NewLabel("")
	a.resultCount.Hide()
	a.results = a.makeResults()
	a.entry.ActionItem = iwidget.NewIconButton(theme.CancelIcon(), func() {
		a.Reset()
	})
	a.entry.PlaceHolder = "Search New Eden"
	a.entry.OnSubmitted = func(s string) {
		go a.doSearch(s)
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
				widget.NewButton("Reset", a.ResetOptions),
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
	a.showRecent()
	go func() {
		ids := a.u.Settings().RecentSearches()
		if len(ids) == 0 {
			return
		}
		ee, err := a.u.EveUniverseService().ListEntitiesForIDs(context.Background(), ids)
		if err != nil {
			slog.Error("failed to load recent items from settings", "error", err)
			return
		}
		a.setRecentItems(ee)
	}()
	return a
}

func (a *GameSearch) ResetOptions() {
	a.categories.SetSelected(a.defaultCategories)
	a.strict.SetOn(false)
	a.updateSearchOptionsTitle()
}

func (a *GameSearch) updateSearchOptionsTitle() {
	isDefault := func() bool {
		if a.strict.On {
			return false
		}
		if !set.NewFromSlice(a.categories.Selected).Equal(set.NewFromSlice(a.defaultCategories)) {
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

func (a *GameSearch) DoSearch(s string) {
	a.entry.SetText(s)
	go a.doSearch(s)
}

func (a *GameSearch) ToogleOptions(enabled bool) {
	if enabled {
		a.searchOptions.Open(0)
	} else {
		a.searchOptions.Close(0)
	}
	a.searchOptions.Refresh()
}

func (a *GameSearch) setRecentItems(ee []*app.EveEntity) {
	a.mu.Lock()
	a.recentItems = ee
	a.mu.Unlock()
	a.recent.Refresh()
}

func (a *GameSearch) storeRecentItems() {
	ids := xslices.Map(a.recentItems, func(x *app.EveEntity) int32 {
		return x.ID
	})
	a.u.Settings().SetRecentSearches(ids)
}

func (a *GameSearch) CreateRenderer() fyne.WidgetRenderer {
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

func (a *GameSearch) Focus() {
	a.w.Canvas().Focus(a.entry)
}

func (a *GameSearch) Reset() {
	a.entry.SetText("")
	a.clearResults()
}

func (a *GameSearch) SetWindow(w fyne.Window) {
	a.w = w
}

func (a *GameSearch) makeResults() *iwidget.Tree[resultNode] {
	t := iwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			if isBranch {
				return widget.NewLabel("Template")
			}
			return NewSearchResult(
				a.u.EveImageService(),
				a.u.EveUniverseService(),
				a.supportedCategories,
			)
		},
		func(n resultNode, isBranch bool, co fyne.CanvasObject) {
			if isBranch {
				co.(*widget.Label).SetText(n.String())
				return
			}
			co.(*SearchResult).Set(n.ee)
		},
	)
	t.OnSelected = func(n resultNode) {
		defer t.UnselectAll()
		if n.isCategory() {
			t.ToggleBranch(n)
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

func (a *GameSearch) showSupportedResult(o *app.EveEntity) {
	if !a.supportedCategories.Contains(o.Category) {
		return
	}
	a.u.ShowEveEntityInfoWindow(o)
}

func (a *GameSearch) makeRecentSelected() *widget.List {
	l := widget.NewList(
		func() int {
			a.mu.RLock()
			defer a.mu.RUnlock()
			return len(a.recentItems)
		},
		func() fyne.CanvasObject {
			return NewSearchResult(
				a.u.EveImageService(),
				a.u.EveUniverseService(),
				infowindow.SupportedEveEntities(),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			a.mu.RLock()
			defer a.mu.RUnlock()
			if id >= len(a.recentItems) {
				return
			}
			it := a.recentItems[id]
			co.(*SearchResult).Set(it)
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

func (a *GameSearch) clearResults() {
	a.results.Clear()
	a.resultCount.Hide()
	a.showRecent()
}

func (a *GameSearch) showResults() {
	a.recentPage.Hide()
	a.resultsPage.Show()
}

func (a *GameSearch) showRecent() {
	a.resultsPage.Hide()
	a.recentPage.Show()
}

func (a *GameSearch) doSearch(search string) {
	if a.u.IsOffline() {
		a.u.ShowInformationDialog(
			"Offline",
			"Can't search when offline",
			a.w,
		)
		return
	}
	a.clearResults()
	if search == "" {
		return
	}
	a.showResults()
	a.indicator.Show()
	a.indicator.Start()
	defer func() {
		a.indicator.Stop()
		a.indicator.Hide()
	}()
	categories := xslices.Map(a.categories.Selected, func(o string) app.SearchCategory {
		return option2searchCategory(o)
	})
	results, total, err := a.u.CharacterService().SearchESI(
		context.Background(),
		a.u.CurrentCharacterID(),
		search,
		categories,
		a.strict.On,
	)
	if err != nil {
		a.u.ShowErrorDialog("Search failed", err, a.u.MainWindow())
		return
	}
	if total == maxSearchResults {
		a.resultCount.Importance = widget.WarningImportance
		a.resultCount.Wrapping = fyne.TextWrapWord
		a.resultCount.SetText(fmt.Sprintf(
			"Search for \"%s\" exceeded the server limit of 500 results "+
				"and may not contain the items you are looking for.",
			search,
		))
	} else {
		a.resultCount.Importance = widget.MediumImportance
		a.resultCount.SetText(fmt.Sprintf("%d Results", total))
	}
	a.resultCount.Show()
	if total == 0 {
		return
	}
	t := iwidget.NewTreeData[resultNode]()
	var categoriesFound int
	for _, c := range categories {
		_, ok := results[c]
		if !ok {
			continue
		}
		categoriesFound++
		n := resultNode{category: c, count: len(results[c])}
		parentUID, err := t.Add(iwidget.RootUID, n)
		if err != nil {
			panic(err)
		}
		for _, o := range results[c] {
			n := resultNode{ee: o}
			t.Add(parentUID, n)
		}
	}
	a.results.Set(t)
	if categoriesFound == 1 {
		a.results.OpenAllBranches()
	}
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

func (sn resultNode) UID() widget.TreeNodeID {
	if sn.isCategory() {
		return "C_" + string(sn.category)
	}
	return fmt.Sprintf("EE_%d", sn.ee.ID)
}

func (sn resultNode) String() string {
	if sn.isCategory() {
		return fmt.Sprintf("%s (%d)", sn.category.String(), sn.count)
	}
	return sn.ee.Name
}
