package ui

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infowindow"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

const (
	maxSearchResults = 500 // max results returned from the server
)

type SearchArea struct {
	Content fyne.CanvasObject

	categories          *iwidget.FilterChipGroup
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

func NewSearchArea(u *BaseUI) *SearchArea {
	a := &SearchArea{
		entry:               widget.NewEntry(),
		indicator:           widget.NewProgressBarInfinite(),
		resultCount:         widget.NewLabel(""),
		supportedCategories: infowindow.SupportedEveEntities(),
		u:                   u,
		w:                   u.Window,
	}

	defaultStrict := false
	defaultCategories := makeOptions()
	updateSearchOptionsTitle := func() {
		isDefault := func() bool {
			if a.strict.On != defaultStrict {
				return false
			}
			if !set.NewFromSlice(a.categories.Selected).Equal(set.NewFromSlice(defaultCategories)) {
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
	a.categories = iwidget.NewFilterChipGroup(defaultCategories, nil)
	a.categories.Selected = slices.Clone(defaultCategories)
	a.categories.OnChanged = func(s []string) {
		updateSearchOptionsTitle()
	}

	a.strict = kxwidget.NewSwitch(nil)
	a.strict.On = defaultStrict
	a.strict.OnChanged = func(on bool) {
		updateSearchOptionsTitle()
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
				widget.NewButton("Reset", func() {
					a.categories.SetSelected(defaultCategories)
					a.strict.SetOn(false)
					updateSearchOptionsTitle()
				}),
			),
		),
	)
	updateSearchOptionsTitle()

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
		a.mu.Lock()
		a.recentItems = make([]*app.EveEntity, 0)
		a.mu.Unlock()
		a.recent.Refresh()
	}
	a.recentPage = container.NewBorder(
		container.NewHBox(widget.NewLabel("Recent searches"), layout.NewSpacer(), clearRecent),
		nil,
		nil,
		nil,
		a.recent,
	)
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
	a.showRecent()
	a.Content = c
	return a
}

func (a *SearchArea) Focus() {
	a.w.Canvas().Focus(a.entry)
}

func (a *SearchArea) Reset() {
	a.entry.SetText("")
	a.clearResults()
}

func (a *SearchArea) SetWindow(w fyne.Window) {
	a.w = w
}

func (a *SearchArea) makeResults() *iwidget.Tree[resultNode] {
	t := iwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			if isBranch {
				return widget.NewLabel("Template")
			}
			return appwidget.NewSearchResult(
				a.u.EveImageService,
				a.u.EveUniverseService,
				a.supportedCategories,
			)
		},
		func(n resultNode, isBranch bool, co fyne.CanvasObject) {
			if isBranch {
				co.(*widget.Label).SetText(n.String())
				return
			}
			co.(*appwidget.SearchResult).Set(n.ee)
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
		a.mu.Unlock()
		a.recent.Refresh()
	}
	return t
}

func (a *SearchArea) showSupportedResult(o *app.EveEntity) {
	if !a.supportedCategories.Contains(o.Category) {
		return
	}
	iw := infowindow.New(
		a.u.CurrentCharacterID,
		a.u.CharacterService,
		a.u.EveUniverseService,
		a.u.EveImageService,
		a.w,
	)
	iw.ShowEveEntity(o)
}

func (a *SearchArea) makeRecentSelected() *widget.List {
	l := widget.NewList(
		func() int {
			a.mu.RLock()
			defer a.mu.RUnlock()
			return len(a.recentItems)
		},
		func() fyne.CanvasObject {
			return appwidget.NewSearchResult(
				a.u.EveImageService,
				a.u.EveUniverseService,
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
			co.(*appwidget.SearchResult).Set(it)
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

func (a *SearchArea) clearResults() {
	a.results.Clear()
	a.resultCount.Hide()
	a.showRecent()
}

func (a *SearchArea) showResults() {
	a.recentPage.Hide()
	a.resultsPage.Show()
}

func (a *SearchArea) showRecent() {
	a.resultsPage.Hide()
	a.recentPage.Show()
}

func (a *SearchArea) doSearch(search string) {
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
	categories := slices.Collect(xiter.MapSlice(a.categories.Selected, func(o string) character.SearchCategory {
		return option2searchCategory(o)
	}))
	results, total, err := a.u.CharacterService.SearchESI(
		context.Background(),
		a.u.CurrentCharacterID(),
		search,
		categories,
		a.strict.On,
	)
	if err != nil {
		d2 := iwidget.NewErrorDialog("Search failed", err, a.u.Window)
		d2.Show()
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

var searchCategory2optionMap = map[character.SearchCategory]string{
	character.SearchAgent:         "Agents",
	character.SearchAlliance:      "Alliances",
	character.SearchCharacter:     "Characters",
	character.SearchConstellation: "Constellations",
	character.SearchCorporation:   "Corporations",
	character.SearchFaction:       "Factions",
	character.SearchRegion:        "Regions",
	character.SearchSolarSystem:   "Systems",
	character.SearchStation:       "Stations",
	character.SearchType:          "Types",
}

func searchCategory2option(c character.SearchCategory) string {
	x, ok := searchCategory2optionMap[c]
	if !ok {
		panic(fmt.Sprintf("searchCategory2option: \"%s\" not found", c))
	}
	return x
}

func option2searchCategory(o string) character.SearchCategory {
	for k, v := range searchCategory2optionMap {
		if v == o {
			return k
		}
	}
	panic(fmt.Sprintf("option2searchCategory: \"%s\" not found", o))
}

func makeOptions() []string {
	options := slices.Collect(xiter.MapSlice(character.SearchCategories(), func(c character.SearchCategory) string {
		return searchCategory2option(c)
	}))
	slices.Sort(options)
	return options
}

type resultNode struct {
	category character.SearchCategory
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
