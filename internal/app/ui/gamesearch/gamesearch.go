// Package gamesearch provides widgets for building the game search UI.
package gamesearch

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

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

// TODO: Make component permanent, e.g. to ensure cache is persistent between searches

const (
	maxSearchResults = 500 // max results returned from the server
)

type baseUI interface {
	Character() *characterservice.CharacterService
	EVEImage() ui.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	InfoViewer() ui.InfoViewer
	IsDeveloperMode() bool
	IsOffline() bool
	MainWindow() fyne.Window
	Settings() *settings.Settings
	Signals() *app.Signals
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

type GameSearch struct {
	widget.BaseWidget

	categories          *kxwidget.FilterChipGroup
	defaultCategories   []string
	entry               *widget.Entry
	iconCache           xsync.Map[int64, fyne.Resource]
	indicator           *widget.ProgressBarInfinite
	recent              *widget.List
	recentItems         []*app.EveEntity
	recentPage          *fyne.Container
	resultCount         *widget.Label
	results             *xwidget.Tree[resultNode]
	resultsPage         *fyne.Container
	searchOptions       *widget.Accordion
	strict              *kxwidget.Switch
	supportedCategories set.Set[app.EveEntityCategory]
	u                   baseUI
	w                   fyne.Window
}

func NewGameSearch(u baseUI) *GameSearch {
	a := &GameSearch{
		defaultCategories:   makeOptions(),
		entry:               widget.NewEntry(),
		indicator:           widget.NewProgressBarInfinite(),
		resultCount:         widget.NewLabel(""),
		supportedCategories: infoviewer.SupportedCategories(),
		u:                   u,
		w:                   u.MainWindow(),
	}
	a.ExtendBaseWidget(a)

	a.categories = kxwidget.NewFilterChipGroup(a.defaultCategories, func(_ []string) {
		a.updateSearchOptionsTitle()
	})
	a.categories.Selected = a.defaultCategories

	a.strict = kxwidget.NewSwitch(func(_ bool) {
		a.updateSearchOptionsTitle()
	})
	a.strict.On = false

	a.resultCount = widget.NewLabel("")
	a.resultCount.Hide()
	a.results = a.makeResults()
	a.entry.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.Reset()
	})
	a.entry.PlaceHolder = "Search New Eden"
	a.entry.OnSubmitted = func(s string) {
		go a.DoSearch(context.Background(), s)
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
		a.recentItems = xslices.Reset(a.recentItems)
		a.recent.Refresh()
		a.storeRecentItems()
	}
	a.recentPage = container.NewBorder(
		container.NewHBox(widget.NewLabel("Recent searches"), layout.NewSpacer(), clearRecent),
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
	if len(ids) == 0 {
		return
	}
	ee, err := a.u.EVEUniverse().ListEntitiesForIDs(ctx, ids)
	if errors.Is(err, app.ErrNotFound) {
		fyne.Do(func() {
			a.recentItems = xslices.Reset(a.recentItems)
			a.recent.Refresh()
		})
		return
	}
	if err != nil {
		slog.Error("failed to load recent items from settings", "error", err)
		return
	}
	fyne.Do(func() {
		a.recentItems = ee
		a.recent.Refresh()
		a.showRecent()
	})
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

func (a *GameSearch) ToogleOptions(enabled bool) {
	if enabled {
		a.searchOptions.Open(0)
	} else {
		a.searchOptions.Close(0)
	}
	a.searchOptions.Refresh()
}

func (a *GameSearch) storeRecentItems() {
	ids := xslices.Map(a.recentItems, func(x *app.EveEntity) int64 {
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

func (a *GameSearch) loadIconFunc() func(o *app.EveEntity, setIcon func(r fyne.Resource)) {
	return func(o *app.EveEntity, setIcon func(r fyne.Resource)) {
		xwidget.LoadResourceAsyncWithCache(
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

func (a *GameSearch) makeResults() *xwidget.Tree[resultNode] {
	t := xwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			if isBranch {
				return widget.NewLabel("Template")
			}
			return newSearchResult(a.loadIconFunc(), a.supportedCategories)
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
			return newSearchResult(a.loadIconFunc(), infoviewer.SupportedCategories())
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
		a.resultCount.Hide()
		a.showRecent()
	})
}

func (a *GameSearch) showRecent() {
	a.resultsPage.Hide()
	a.recentPage.Show()
}

func (a *GameSearch) SetEntry(s string) {
	a.entry.SetText(s)
}

func (a *GameSearch) DoSearch(ctx context.Context, search string) {
	if a.u.IsOffline() {
		fyne.Do(func() {
			ui.ShowInformation(
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
	results, total, err := a.u.Character().SearchESI(
		ctx,
		search,
		categories,
		a.strict.On,
	)
	if err != nil {
		fyne.Do(func() {
			ui.ShowErrorAndLog("Search failed", err, a.u.IsDeveloperMode(), a.u.MainWindow())
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
	var td xwidget.TreeData[resultNode]
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
