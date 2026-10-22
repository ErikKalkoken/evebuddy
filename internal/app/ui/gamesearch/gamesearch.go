// Package gamesearch provides widgets for building the game search UI.
package gamesearch

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infoviewer"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	maxSearchResults = 500 // max results returned from the server
)

// Hooks that tests can replace.
var (
	runAsync          = func(f func()) { go f() }
	loadResourceAsync = xwidget.LoadResourceAsyncWithCache
	// waitSearchDelay waits briefly, so typing triggers only one search.
	// Reports false when the search was canceled while waiting.
	waitSearchDelay = func(ctx context.Context) bool {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(300 * time.Millisecond):
			return true
		}
	}
	// searchTimeout limits the ESI call, as a dead connection can otherwise hang for minutes.
	searchTimeout = 15 * time.Second
)

type baseUI interface {
	Character() *characterservice.CharacterService
	EVEImage() ui.EVEImageService
	ErrorDisplay(err error) string
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	InfoViewer() ui.InfoViewer
	IsOffline() bool
	MainWindow() fyne.Window
	Settings() *settings.Settings
	Signals() *app.Signals
}

type searchResultNode struct {
	category app.SearchCategory
	count    int
	ee       *app.EveEntity
}

func (r *searchResultNode) isCategory() bool {
	return r.ee == nil
}

func (r searchResultNode) String() string {
	if r.isCategory() {
		return fmt.Sprintf("%s (%d)", r.category.String(), r.count)
	}
	return r.ee.Name
}

func (r searchResultNode) UID() widget.TreeNodeID {
	return fmt.Sprintf("%s-%d", r.category, r.ee.IDOrZero())
}

type GameSearch struct {
	widget.BaseWidget

	cancelSearch        context.CancelFunc
	categories          *kxwidget.FilterChipGroup
	clearRecent         *widget.Hyperlink
	defaultCategories   []string
	searchEntry         *xwidget.SearchEntry
	iconCache           xsync.Map[int64, fyne.Resource]
	recent              *widget.List
	recentItems         []*app.EveEntity
	recentLoadFailed    bool
	recentPage          *fyne.Container
	results             *xwidget.Tree[searchResultNode]
	resultsPage         *fyne.Container
	searchOptions       *widget.Accordion
	spinner             *kxwidget.Spinner
	statusLabel         *widget.Label
	strict              *kxwidget.Switch
	supportedCategories set.Set[app.EveEntityCategory]
	u                   baseUI
	w                   fyne.Window
}

func NewGameSearch(u baseUI) *GameSearch {
	a := &GameSearch{
		defaultCategories:   makeOptions(),
		spinner:             kxwidget.NewSpinner(),
		statusLabel:         widget.NewLabel(""),
		supportedCategories: infoviewer.SupportedCategories(),
		u:                   u,
		w:                   u.MainWindow(),
	}
	a.ExtendBaseWidget(a)

	searchIfValid := func() {
		if s := a.searchEntry.Text; len(s) >= 3 {
			a.startSearch(s)
		}
	}

	a.categories = kxwidget.NewFilterChipGroup(a.defaultCategories, func(_ []string) {
		a.updateSearchOptionsTitle()
		searchIfValid()
	})
	a.categories.Selected = a.defaultCategories

	a.strict = kxwidget.NewSwitch(func(_ bool) {
		a.updateSearchOptionsTitle()
		searchIfValid()
	})
	a.strict.On = false

	a.statusLabel.Wrapping = fyne.TextWrapWord
	a.statusLabel.Hide()
	a.results = a.makeResults()

	a.searchEntry = xwidget.NewSearchEntry(
		"Loading...",
		func(s string) {
			if len(s) < 3 {
				a.clearSearch()
				return
			}
			a.startSearch(s)
		},
	)
	a.searchEntry.Disable() // until init, so recent searches can't change before they are loaded
	a.spinner.Hide()

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
		a.statusLabel,
		nil,
		nil,
		nil,
		container.NewStack(
			a.results,
			container.NewCenter(container.NewStack(
				xwidget.NewSpacer(fyne.NewSquareSize(2*theme.IconInlineSize())),
				a.spinner,
			)),
		),
	)
	a.resultsPage.Hide()
	a.clearRecent = widget.NewHyperlink("Clear", nil)
	a.clearRecent.OnTapped = func() {
		xslices.Clear(&a.recentItems)
		a.recent.Refresh()
		a.storeRecentItems()
	}
	a.clearRecent.Hide() // until init
	a.recentPage = container.NewBorder(
		container.NewHBox(widget.NewLabel("Recent searches"), layout.NewSpacer(), a.clearRecent),
		nil,
		nil,
		nil,
		a.recent,
	)

	// Signals
	a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
		a.init(ctx)
	})
	return a
}

func (a *GameSearch) init(ctx context.Context) {
	ids := a.u.Settings().RecentSearches()
	var ee []*app.EveEntity
	var err error
	if len(ids) > 0 {
		ee, err = a.u.EVEUniverse().ListExistingEntitiesForIDs(ctx, ids)
		if ctx.Err() != nil {
			return // shutting down
		}
		if err != nil {
			slog.Error("failed to load recent items from settings", "error", err)
		}
	}
	fyne.Do(func() {
		a.recentItems = ee
		a.recentLoadFailed = err != nil
		if err == nil && len(ee) < len(ids) {
			a.storeRecentItems() // prune unknown IDs
		}
		a.recent.Refresh()
		a.searchEntry.SetPlaceHolder("Search New Eden")
		a.searchEntry.Enable()
		a.clearRecent.Show()
	})
}

func (a *GameSearch) resetOptions() {
	a.categories.SetSelected(a.defaultCategories)
	a.strict.SetOn(false)
	a.updateSearchOptionsTitle()
}

func (a *GameSearch) updateSearchOptionsTitle() {
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

func (a *GameSearch) storeRecentItems() {
	if a.recentLoadFailed {
		return // keep the saved list, as it could not be loaded
	}
	ids := xslices.Map(a.recentItems, func(x *app.EveEntity) int64 {
		return x.ID
	})
	a.u.Settings().SetRecentSearches(ids)
}

func (a *GameSearch) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(
			a.searchEntry,
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
	a.w.Canvas().Focus(a.searchEntry)
}

func (a *GameSearch) Reset() {
	a.searchEntry.SetText("")
	a.clearSearch()
}

// clearSearch cancels the running search and shows the recent searches.
func (a *GameSearch) clearSearch() {
	a.stopSearch()
	a.spinner.Stop()
	a.spinner.Hide()
	a.clearResults()
}

func (a *GameSearch) SetWindow(w fyne.Window) {
	a.w = w
}

func (a *GameSearch) loadIconFunc() func(o *app.EveEntity, setIcon func(r fyne.Resource)) {
	return func(o *app.EveEntity, setIcon func(r fyne.Resource)) {
		loadResourceAsync(
			icons.BlankSvg,
			func() (fyne.Resource, bool) {
				return a.iconCache.Load(o.ID)
			},
			setIcon,
			func() (fyne.Resource, error) {
				switch o.Category {
				case app.EveEntityInventoryType:
					et, err := a.u.EVEUniverse().GetOrCreateTypeESI(context.Background(), o.ID)
					if err != nil {
						return nil, err
					}
					switch et.Group.Category.ID {
					case app.EveCategorySKINs:
						return a.u.EVEImage().InventoryTypeSKIN(et.ID, ui.IconPixelSize)
					case app.EveCategoryBlueprint:
						return a.u.EVEImage().InventoryTypeBPO(et.ID, ui.IconPixelSize)
					default:
						return a.u.EVEImage().InventoryTypeIcon(et.ID, ui.IconPixelSize)
					}
				default:
					return a.u.EVEImage().EveEntityLogo(o, ui.IconPixelSize)
				}
			},
			func(r fyne.Resource) {
				a.iconCache.Store(o.ID, r)
			},
		)
	}
}

func (a *GameSearch) makeResults() *xwidget.Tree[searchResultNode] {
	t := xwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			if isBranch {
				return widget.NewLabel("Template")
			}
			return newSearchResult(a.loadIconFunc(), a.supportedCategories)
		},
		func(n *searchResultNode, isBranch bool, co fyne.CanvasObject) {
			if isBranch {
				co.(*widget.Label).SetText(n.String())
				return
			}
			co.(*searchResult).set(n.ee)
		},
	)
	t.OnSelectedNode = func(n *searchResultNode) {
		defer t.UnselectAll()
		if n.isCategory() {
			t.ToggleBranchNode(n)
			return
		}
		a.showSupportedResult(n.ee)
		a.recentItems = slices.DeleteFunc(a.recentItems, func(a *app.EveEntity) bool {
			return a.ID == n.ee.ID
		})
		a.recentItems = slices.Insert(a.recentItems, 0, n.ee)
		a.storeRecentItems()
		a.recent.Refresh()
	}
	return t
}

func (a *GameSearch) showSupportedResult(o *app.EveEntity) {
	if !a.supportedCategories.Contains(o.Category) {
		return
	}
	a.u.InfoViewer().Show(o)
}

func (a *GameSearch) makeRecentSelected() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.recentItems)
		},
		func() fyne.CanvasObject {
			return newSearchResult(a.loadIconFunc(), a.supportedCategories)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.recentItems) {
				return
			}
			it := a.recentItems[id]
			co.(*searchResult).set(it)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.recentItems) {
			return
		}
		it := a.recentItems[id]
		a.showSupportedResult(it)
	}
	return l
}

func (a *GameSearch) clearResults() {
	fyne.Do(func() {
		a.results.Clear()
		a.statusLabel.Hide()
		a.showRecent()
	})
}

func (a *GameSearch) showRecent() {
	a.resultsPage.Hide()
	a.recentPage.Show()
}

// startSearch starts a new search and cancels the previous one.
// Options are read here, because widgets must only be accessed on the main thread.
func (a *GameSearch) startSearch(search string) {
	a.stopSearch()
	categories := xslices.Map(a.categories.Selected, func(o string) app.SearchCategory {
		return option2searchCategory(o)
	})
	if len(categories) == 0 {
		a.spinner.Stop()
		a.spinner.Hide()
		a.results.Clear()
		a.recentPage.Hide()
		a.resultsPage.Show()
		a.showStatus("Select at least one category", widget.MediumImportance)
		return
	}
	strict := a.strict.On
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelSearch = cancel
	runAsync(func() {
		a.doSearch(ctx, search, categories, strict)
	})
}

func (a *GameSearch) stopSearch() {
	if a.cancelSearch == nil {
		return
	}
	a.cancelSearch()
	a.cancelSearch = nil
}

func (a *GameSearch) showStatus(text string, importance widget.Importance) {
	a.statusLabel.Importance = importance
	a.statusLabel.Text = text
	a.statusLabel.Refresh()
	a.statusLabel.Show()
}

// doSearch runs a search. Results of a canceled search are discarded,
// so a slow search can't overwrite the results of a newer one.
func (a *GameSearch) doSearch(ctx context.Context, search string, categories []app.SearchCategory, strict bool) {
	fyne.Do(func() {
		if ctx.Err() != nil {
			return
		}
		a.results.Clear()
		a.statusLabel.Hide()
		a.recentPage.Hide()
		a.resultsPage.Show()
		a.spinner.Show()
		a.spinner.Start()
		a.resultsPage.Refresh() // Fyne won't repaint a never-visible object on Show
	})
	defer func() {
		fyne.Do(func() {
			if ctx.Err() != nil {
				return // spinner belongs to the newer search
			}
			a.spinner.Stop()
			a.spinner.Hide()
		})
	}()
	if !waitSearchDelay(ctx) {
		return
	}
	if a.u.IsOffline() {
		fyne.Do(func() {
			if ctx.Err() != nil {
				return
			}
			a.showRecent()
			ui.ShowInformation(
				"Offline",
				"Can't search when offline",
				a.w,
			)
		})
		return
	}
	esiCtx, cancel := context.WithTimeout(ctx, searchTimeout) // separate, so ctx only reports newer searches
	defer cancel()
	results, _, err := a.u.Character().SearchESI(
		esiCtx,
		search,
		categories,
		strict,
	)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		slog.Error("Search failed", "error", err)
		msg := a.u.ErrorDisplay(err)
		if esiCtx.Err() != nil {
			msg = "search timed out"
		}
		fyne.Do(func() {
			if ctx.Err() != nil {
				return
			}
			a.showStatus("ERROR: "+msg, widget.DangerImportance)
		})
		return
	}
	var total int // entries, not unique IDs, as an entity can be found in several categories
	for _, c := range categories {
		total += len(results[c])
	}
	fyne.Do(func() {
		if ctx.Err() != nil {
			return
		}
		if total >= maxSearchResults {
			a.showStatus(fmt.Sprintf(
				"Search for \"%s\" exceeded the server limit of 500 results "+
					"and may not contain the items you are looking for.",
				search,
			), widget.WarningImportance)
		} else {
			a.showStatus(fmt.Sprintf("%d Results", total), widget.MediumImportance)
		}
	})
	if total == 0 {
		return
	}
	td := xwidget.NewTreeData[searchResultNode]()
	var categoriesFound int
	for _, c := range categories {
		items, ok := results[c]
		if !ok {
			continue
		}
		categoriesFound++
		itemsCount := len(items)
		category := &searchResultNode{category: c, count: itemsCount}
		err := td.Add(nil, category, itemsCount > 0)
		if err != nil {
			slog.Error("game search: adding category node", "node", category, "err", err)
			continue
		}
		for _, o := range items {
			entity := &searchResultNode{category: c, ee: o}
			err := td.Add(category, entity, false)
			if err != nil {
				slog.Error("game search: adding item node", "node", entity, "err", err)
				continue
			}
		}
	}
	fyne.Do(func() {
		if ctx.Err() != nil {
			return
		}
		a.results.Set(td)
		if categoriesFound == 1 {
			a.results.OpenAllBranches()
		}
	})
}

type searchResult struct {
	widget.BaseWidget

	loadIcon            func(o *app.EveEntity, setIcon func(r fyne.Resource))
	image               *canvas.Image
	name                *widget.Label
	supportedCategories set.Set[app.EveEntityCategory]
}

func newSearchResult(loadIcon func(o *app.EveEntity, setIcon func(r fyne.Resource)), supportedCategories set.Set[app.EveEntityCategory]) *searchResult {
	image := xwidget.NewImageFromResource(
		icons.BlankSvg,
		fyne.NewSquareSize(ui.IconUnitSize),
	)
	w := &searchResult{
		loadIcon:            loadIcon,
		image:               image,
		name:                widget.NewLabel(""),
		supportedCategories: supportedCategories,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *searchResult) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, container.NewPadded(w.image), nil, w.name)
	return widget.NewSimpleRenderer(c)
}

func (w *searchResult) set(o *app.EveEntity) {
	var i widget.Importance
	if !w.supportedCategories.Contains(o.Category) {
		i = widget.LowImportance
	}
	w.name.Importance = i
	w.name.Text = o.Name
	w.name.Refresh()

	w.loadIcon(o, func(r fyne.Resource) {
		w.image.Resource = r
		w.image.Refresh()
	})
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
