package gamesearch

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestMain(m *testing.M) {
	// Leftover goroutines from one test can render text concurrently with the next,
	// which is not safe in Fyne's shared text shaper.
	runAsync = func(f func()) { f() }
	searchDelay = 0
	loadResourceAsync = func(initial fyne.Resource, getter func() (fyne.Resource, bool), updater func(fyne.Resource), loader func() (fyne.Resource, error), setter func(fyne.Resource)) {
		if r, ok := getter(); ok {
			updater(r)
			return
		}
		updater(initial)
		r, err := loader()
		if err != nil {
			slog.Warn("Failed to fetch image resource", "err", err)
			r = theme.BrokenImageIcon()
		}
		setter(r)
		updater(r)
	}
	os.Exit(m.Run())
}

const searchURL = `=~^https://esi\.evetech\.net/characters/\d+/search`

// infoViewerFake records which entities were shown.
type infoViewerFake struct {
	shown []*app.EveEntity
}

func (iv *infoViewerFake) Show(o *app.EveEntity) {
	iv.shown = append(iv.shown, o)
}

func (*infoViewerFake) Show2(_, _, _ int64)   {}
func (*infoViewerFake) ShowBloodline(_ int64) {}
func (*infoViewerFake) ShowLocation(_ int64)  {}
func (*infoViewerFake) ShowRace(_ int64)      {}
func (*infoViewerFake) ShowType(_, _ int64)   {}

func eveEntityIDs(ee []*app.EveEntity) []int64 {
	return xslices.Map(ee, func(x *app.EveEntity) int64 {
		return x.ID
	})
}

func TestGameSearch_Init(t *testing.T) {
	t.Run("disables search until recent searches are loaded", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})

		a := NewGameSearch(u)

		assert.True(t, a.searchEntry.Disabled())
		assert.Equal(t, "Loading...", a.searchEntry.PlaceHolder)
		assert.False(t, a.clearRecent.Visible())
		assert.False(t, a.resultsPage.Visible())
	})
	t.Run("enables search after loading", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		e := factory.CreateEveEntity()
		u.Settings().SetRecentSearches([]int64{e.ID})
		a := NewGameSearch(u)

		a.init(t.Context())

		assert.Equal(t, []int64{e.ID}, eveEntityIDs(a.recentItems))
		assert.False(t, a.searchEntry.Disabled())
		assert.Equal(t, "Search New Eden", a.searchEntry.PlaceHolder)
		assert.True(t, a.clearRecent.Visible())
	})
	t.Run("enables search when nothing was saved", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := NewGameSearch(u)

		a.init(t.Context())

		assert.Empty(t, a.recentItems)
		assert.False(t, a.searchEntry.Disabled())
		assert.Equal(t, "Search New Eden", a.searchEntry.PlaceHolder)
		assert.True(t, a.clearRecent.Visible())
	})
	t.Run("enables search when loading fails", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		e := factory.CreateEveEntity()
		u.Settings().SetRecentSearches([]int64{e.ID})
		a := NewGameSearch(u)
		if _, err := db.Exec("DROP TABLE eve_entities"); err != nil { // let loading fail
			t.Fatal(err)
		}

		a.init(t.Context())

		assert.Empty(t, a.recentItems)
		assert.Equal(t, []int64{e.ID}, u.Settings().RecentSearches())
		assert.False(t, a.searchEntry.Disabled())
		assert.Equal(t, "Search New Eden", a.searchEntry.PlaceHolder)
		assert.True(t, a.clearRecent.Visible())
	})
	t.Run("does not save picks after loading fails", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), InfoViewer: &infoViewerFake{}, Storage: st})
		e := factory.CreateEveEntity()
		picked := factory.CreateEveEntityCharacter()
		u.Settings().SetRecentSearches([]int64{e.ID})
		a := NewGameSearch(u)
		if _, err := db.Exec("DROP TABLE eve_entities"); err != nil { // let loading fail
			t.Fatal(err)
		}
		a.init(t.Context())
		td := xwidget.NewTreeData[searchResultNode]()
		category := &searchResultNode{category: app.SearchCharacter, count: 1}
		require.NoError(t, td.Add(nil, category, true))
		item := &searchResultNode{ee: picked}
		require.NoError(t, td.Add(category, item, false))
		a.results.Set(td)

		a.results.SelectNode(item)

		assert.Equal(t, []int64{picked.ID}, eveEntityIDs(a.recentItems))
		assert.Equal(t, []int64{e.ID}, u.Settings().RecentSearches())
	})
	t.Run("prunes unknown IDs", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		existing := factory.CreateEveEntity()
		u.Settings().SetRecentSearches([]int64{existing.ID, 99999999})
		a := NewGameSearch(u)

		a.init(t.Context())

		assert.Equal(t, []int64{existing.ID}, eveEntityIDs(a.recentItems))
		assert.Equal(t, []int64{existing.ID}, u.Settings().RecentSearches())
	})
	t.Run("keeps search disabled when shutting down", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		e := factory.CreateEveEntity()
		u.Settings().SetRecentSearches([]int64{e.ID})
		a := NewGameSearch(u)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		a.init(ctx)

		assert.Empty(t, a.recentItems)
		assert.Equal(t, []int64{e.ID}, u.Settings().RecentSearches())
		assert.True(t, a.searchEntry.Disabled())
		assert.Equal(t, "Loading...", a.searchEntry.PlaceHolder)
		assert.False(t, a.clearRecent.Visible())
	})
}

func TestSearchCategoryOptions(t *testing.T) {
	t.Run("maps all search categories to options and back", func(t *testing.T) {
		for _, c := range app.SearchCategories() {
			// when
			got := option2searchCategory(searchCategory2option(c))
			// then
			assert.Equal(t, c, got)
		}
	})
	t.Run("panics for unknown search category", func(t *testing.T) {
		assert.Panics(t, func() {
			searchCategory2option(app.SearchCategory("unknown"))
		})
	})
	t.Run("panics for unknown option", func(t *testing.T) {
		assert.Panics(t, func() {
			option2searchCategory("Unknown")
		})
	})
	t.Run("makes sorted options for all search categories", func(t *testing.T) {
		// when
		got := makeOptions()
		// then
		assert.Equal(t, []string{
			"Agents",
			"Alliances",
			"Characters",
			"Constellations",
			"Corporations",
			"Factions",
			"Regions",
			"Stations",
			"Systems",
			"Types",
		}, got)
	})
}

func TestSearchResultNode(t *testing.T) {
	t.Run("category node", func(t *testing.T) {
		// given
		n := searchResultNode{category: app.SearchSolarSystem, count: 3}
		// then
		assert.True(t, n.isCategory())
		assert.Equal(t, "Solar System (3)", n.String())
		assert.Equal(t, "Solar System-0", n.UID())
	})
	t.Run("entity node", func(t *testing.T) {
		// given
		n := searchResultNode{ee: &app.EveEntity{ID: 42, Name: "Alpha", Category: app.EveEntityCharacter}}
		// then
		assert.False(t, n.isCategory())
		assert.Equal(t, "Alpha", n.String())
		assert.Equal(t, "-42", n.UID())
	})
}

func TestGameSearch_SearchOptions(t *testing.T) {
	t.Run("shows default title initially", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})

		a := NewGameSearch(u)

		assert.Equal(t, "Search options", a.searchOptions.Items[0].Title)
	})
	t.Run("shows changed title when strict is on", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := NewGameSearch(u)

		a.strict.SetOn(true)

		assert.Equal(t, "Search options (changed)", a.searchOptions.Items[0].Title)
	})
	t.Run("shows changed title when categories differ from default", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := NewGameSearch(u)

		a.categories.SetSelected([]string{"Characters"})

		assert.Equal(t, "Search options (changed)", a.searchOptions.Items[0].Title)
	})
	t.Run("can reset options to default", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := NewGameSearch(u)
		a.categories.SetSelected([]string{"Characters"})
		a.strict.SetOn(true)

		a.resetOptions()

		assert.ElementsMatch(t, makeOptions(), a.categories.Selected)
		assert.False(t, a.strict.On)
		assert.Equal(t, "Search options", a.searchOptions.Items[0].Title)
	})
	t.Run("toggles strict when tapping its label", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := NewGameSearch(u)
		w := test.NewWindow(a)
		t.Cleanup(w.Close)
		a.searchOptions.Open(0)
		var label *kxwidget.TappableLabel
		for _, o := range test.LaidOutObjects(a) {
			if x, ok := o.(*kxwidget.TappableLabel); ok && x.Text == "Strict search" {
				label = x
			}
		}
		require.NotNil(t, label)

		test.Tap(label)

		assert.True(t, a.strict.On)
		assert.Equal(t, "Search options (changed)", a.searchOptions.Items[0].Title)
	})
}

func TestGameSearch_DoSearch(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)
	t.Run("shows info dialog when offline", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := NewGameSearch(u)
		// when
		a.startSearch("alpha")
		// then
		assert.Equal(t, 0, httpmock.GetTotalCallCount())
		assert.False(t, a.resultsPage.Visible())
		assert.NotNil(t, a.w.Canvas().Overlays().Top())
	})
	t.Run("shows info dialog only once when typing quickly while offline", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		var wg sync.WaitGroup
		runAsync = func(f func()) { wg.Go(f) }
		searchDelay = 200 * time.Millisecond
		t.Cleanup(func() {
			runAsync = func(f func()) { f() }
			searchDelay = 0
		})
		a := NewGameSearch(u)
		// when
		a.startSearch("alpha")
		a.startSearch("alphab")
		wg.Wait()
		// then
		assert.Equal(t, 0, httpmock.GetTotalCallCount())
		assert.Len(t, a.w.Canvas().Overlays().List(), 1)
	})
	t.Run("searches only once when typing quickly", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		e := factory.CreateEveEntityCharacter()
		var searches []string
		httpmock.RegisterResponder("GET", searchURL, func(r *http.Request) (*http.Response, error) {
			searches = append(searches, r.URL.Query().Get("search"))
			return httpmock.NewJsonResponse(200, map[string][]int64{"character": {e.ID}})
		})
		var wg sync.WaitGroup
		runAsync = func(f func()) { wg.Go(f) }
		searchDelay = 200 * time.Millisecond
		t.Cleanup(func() {
			runAsync = func(f func()) { f() }
			searchDelay = 0
		})
		a := NewGameSearch(u)
		// when
		a.startSearch("alpha")
		a.startSearch("alphab")
		a.startSearch("alphabc")
		wg.Wait()
		// then
		assert.Equal(t, []string{"alphabc"}, searches)
		assert.Equal(t, "1 Results", a.resultCount.Text)
	})
	t.Run("shows recent searches for empty search", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		a := NewGameSearch(u)
		// when
		a.startSearch("")
		// then
		assert.Equal(t, 0, httpmock.GetTotalCallCount())
		assert.True(t, a.recentPage.Visible())
		assert.False(t, a.resultsPage.Visible())
	})
	t.Run("shows results from one category with branch opened", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		e1 := factory.CreateEveEntityCharacter(app.EveEntity{Name: "Alpha"})
		e2 := factory.CreateEveEntityCharacter(app.EveEntity{Name: "Bravo"})
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{
			"character": {e1.ID, e2.ID},
		}))
		a := NewGameSearch(u)
		// when
		a.startSearch("alpha")
		// then
		assert.True(t, a.resultsPage.Visible())
		assert.False(t, a.recentPage.Visible())
		assert.False(t, a.indicator.Visible())
		assert.True(t, a.resultCount.Visible())
		assert.Equal(t, "2 Results", a.resultCount.Text)
		assert.Equal(t, widget.MediumImportance, a.resultCount.Importance)
		td := a.results.Data()
		categories := td.Children(nil)
		require.Len(t, categories, 1)
		assert.Equal(t, "Character (2)", categories[0].String())
		assert.True(t, a.results.IsBranchOpenNode(categories[0]))
		items := xslices.Map(td.Children(categories[0]), func(n *searchResultNode) string {
			return n.String()
		})
		assert.Equal(t, []string{"Alpha", "Bravo"}, items)
	})
	t.Run("shows results from several categories with branches closed", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		e1 := factory.CreateEveEntityCharacter()
		e2 := factory.CreateEveEntityCorporation()
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{
			"character":   {e1.ID},
			"corporation": {e2.ID},
		}))
		a := NewGameSearch(u)
		// when
		a.startSearch("alpha")
		// then
		assert.Equal(t, "2 Results", a.resultCount.Text)
		categories := a.results.Data().Children(nil)
		got := xslices.Map(categories, func(n *searchResultNode) string {
			return n.String()
		})
		assert.Equal(t, []string{"Character (1)", "Corporation (1)"}, got)
		for _, n := range categories {
			assert.False(t, a.results.IsBranchOpenNode(n))
		}
	})
	t.Run("shows warning when results exceed the server limit", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		var ids []int64
		for range maxSearchResults {
			ids = append(ids, factory.CreateEveEntityCharacter().ID)
		}
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{
			"character": ids,
		}))
		a := NewGameSearch(u)
		// when
		a.startSearch("alpha")
		// then
		assert.Equal(t, widget.WarningImportance, a.resultCount.Importance)
		assert.Contains(t, a.resultCount.Text, "exceeded the server limit of 500 results")
	})
	t.Run("shows no results", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{}))
		a := NewGameSearch(u)
		// when
		a.startSearch("alpha")
		// then
		assert.True(t, a.resultsPage.Visible())
		assert.Equal(t, "0 Results", a.resultCount.Text)
		assert.True(t, a.results.Data().IsEmpty())
	})
	t.Run("searches with selected categories and strict option", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		var req *http.Request
		httpmock.RegisterResponder("GET", searchURL, func(r *http.Request) (*http.Response, error) {
			req = r
			return httpmock.NewJsonResponse(200, map[string][]int64{})
		})
		a := NewGameSearch(u)
		a.categories.SetSelected([]string{"Characters", "Types"})
		a.strict.SetOn(true)
		// when
		a.startSearch("alpha")
		// then
		require.NotNil(t, req)
		q := req.URL.Query()
		assert.ElementsMatch(t, []string{"character", "inventory_type"}, q["categories"])
		assert.Equal(t, "true", q.Get("strict"))
		assert.Equal(t, "alpha", q.Get("search"))
	})
	t.Run("reports error when search fails", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var snackbar string
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:                 test.NewTempApp(t),
			DisplaySnackbarFunc: func(s string) { snackbar = s },
			IsOnline:            true,
			Storage:             st,
		})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewStringResponder(500, "internal error"))
		a := NewGameSearch(u)
		// when
		a.startSearch("alpha")
		// then
		assert.Equal(t, "Search failed", snackbar)
		assert.False(t, a.indicator.Visible())
		assert.False(t, a.indicator.Running())
		assert.False(t, a.resultCount.Visible())
	})
	t.Run("shows info dialog on window set later", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		fakeApp := test.NewTempApp(t)
		u := testdouble.NewUIFake(testdouble.UIParams{App: fakeApp, Storage: st})
		a := NewGameSearch(u)
		mainWindow := a.w
		w := fakeApp.NewWindow("Search")
		t.Cleanup(w.Close)
		a.SetWindow(w, func(string) {})
		// when
		a.startSearch("alpha")
		// then
		assert.NotNil(t, w.Canvas().Overlays().Top())
		assert.Nil(t, mainWindow.Canvas().Overlays().Top())
	})
	t.Run("reports error to snackbar set later", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var mainSnackbar string
		fakeApp := test.NewTempApp(t)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:                 fakeApp,
			DisplaySnackbarFunc: func(s string) { mainSnackbar = s },
			IsOnline:            true,
			Storage:             st,
		})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewStringResponder(500, "internal error"))
		a := NewGameSearch(u)
		w := fakeApp.NewWindow("Search")
		t.Cleanup(w.Close)
		var snackbar string
		a.SetWindow(w, func(s string) { snackbar = s })
		// when
		a.startSearch("alpha")
		// then
		assert.Equal(t, "Search failed", snackbar)
		assert.Empty(t, mainSnackbar)
	})
	t.Run("cancels running search when a new one starts", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var snackbar string
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:                 test.NewTempApp(t),
			DisplaySnackbarFunc: func(s string) { snackbar = s },
			IsOnline:            true,
			Storage:             st,
		})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		e1 := factory.CreateEveEntityCharacter()
		e2 := factory.CreateEveEntityCharacter()
		started := make(chan struct{})
		httpmock.RegisterResponder("GET", searchURL, func(r *http.Request) (*http.Response, error) {
			if r.URL.Query().Get("search") == "alpha" {
				close(started)
				<-r.Context().Done() // responds only when canceled
				return nil, r.Context().Err()
			}
			return httpmock.NewJsonResponse(200, map[string][]int64{"character": {e1.ID, e2.ID}})
		})
		var wg sync.WaitGroup
		runAsync = func(f func()) { wg.Go(f) }
		t.Cleanup(func() { runAsync = func(f func()) { f() } })
		a := NewGameSearch(u)
		a.startSearch("alpha")
		<-started
		// when
		a.startSearch("alphab")
		wg.Wait()
		// then
		assert.Equal(t, "2 Results", a.resultCount.Text)
		categories := a.results.Data().Children(nil)
		require.Len(t, categories, 1)
		assert.Equal(t, "Character (2)", categories[0].String())
		assert.False(t, a.indicator.Visible())
		assert.False(t, a.indicator.Running())
		assert.Empty(t, snackbar)
	})
	t.Run("cancels running search on reset", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var snackbar string
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:                 test.NewTempApp(t),
			DisplaySnackbarFunc: func(s string) { snackbar = s },
			IsOnline:            true,
			Storage:             st,
		})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		started := make(chan struct{})
		httpmock.RegisterResponder("GET", searchURL, func(r *http.Request) (*http.Response, error) {
			close(started)
			<-r.Context().Done() // responds only when canceled
			return nil, r.Context().Err()
		})
		var wg sync.WaitGroup
		runAsync = func(f func()) { wg.Go(f) }
		t.Cleanup(func() { runAsync = func(f func()) { f() } })
		a := NewGameSearch(u)
		a.startSearch("alpha")
		<-started
		// when
		a.Reset()
		wg.Wait()
		// then
		assert.False(t, a.resultCount.Visible())
		assert.True(t, a.recentPage.Visible())
		assert.False(t, a.resultsPage.Visible())
		assert.False(t, a.indicator.Visible())
		assert.False(t, a.indicator.Running())
		assert.Empty(t, snackbar)
	})
}

func TestGameSearch_SelectResult(t *testing.T) {
	t.Run("shows entity and adds it to front of recent searches", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		iv := &infoViewerFake{}
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), InfoViewer: iv, Storage: st})
		e1 := factory.CreateEveEntityCharacter()
		e2 := factory.CreateEveEntityCharacter()
		u.Settings().SetRecentSearches([]int64{e1.ID, e2.ID})
		a := NewGameSearch(u)
		a.init(t.Context())
		td := xwidget.NewTreeData[searchResultNode]()
		category := &searchResultNode{category: app.SearchCharacter, count: 1}
		require.NoError(t, td.Add(nil, category, true))
		item := &searchResultNode{ee: e2}
		require.NoError(t, td.Add(category, item, false))
		a.results.Set(td)
		// when
		a.results.SelectNode(item)
		// then
		assert.Equal(t, []int64{e2.ID}, eveEntityIDs(iv.shown))
		assert.Equal(t, []int64{e2.ID, e1.ID}, eveEntityIDs(a.recentItems))
		assert.Equal(t, []int64{e2.ID, e1.ID}, u.Settings().RecentSearches())
	})
	t.Run("adds entity of unsupported category to recent searches without showing it", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		iv := &infoViewerFake{}
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), InfoViewer: iv, Storage: st})
		e := factory.CreateEveEntityWithCategory(app.EveEntityMailList)
		a := NewGameSearch(u)
		a.init(t.Context())
		td := xwidget.NewTreeData[searchResultNode]()
		category := &searchResultNode{category: app.SearchCharacter, count: 1}
		require.NoError(t, td.Add(nil, category, true))
		item := &searchResultNode{ee: e}
		require.NoError(t, td.Add(category, item, false))
		a.results.Set(td)
		// when
		a.results.SelectNode(item)
		// then
		assert.Empty(t, iv.shown)
		assert.Equal(t, []int64{e.ID}, eveEntityIDs(a.recentItems))
	})
	t.Run("toggles branch when selecting a category", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		iv := &infoViewerFake{}
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), InfoViewer: iv, Storage: st})
		e := factory.CreateEveEntityCharacter()
		a := NewGameSearch(u)
		a.init(t.Context())
		td := xwidget.NewTreeData[searchResultNode]()
		category := &searchResultNode{category: app.SearchCharacter, count: 1}
		require.NoError(t, td.Add(nil, category, true))
		require.NoError(t, td.Add(category, &searchResultNode{ee: e}, false))
		a.results.Set(td)
		// when
		a.results.SelectNode(category)
		// then
		assert.True(t, a.results.IsBranchOpenNode(category))
		assert.Empty(t, iv.shown)
		assert.Empty(t, a.recentItems)
	})
}

func TestGameSearch_Recent(t *testing.T) {
	t.Run("shows entity when selecting a recent search", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		iv := &infoViewerFake{}
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), InfoViewer: iv, Storage: st})
		e := factory.CreateEveEntityCharacter()
		u.Settings().SetRecentSearches([]int64{e.ID})
		a := NewGameSearch(u)
		a.init(t.Context())
		// when
		a.recent.Select(0)
		// then
		assert.Equal(t, []int64{e.ID}, eveEntityIDs(iv.shown))
	})
	t.Run("shows same entity again when selecting a recent search twice", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		iv := &infoViewerFake{}
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), InfoViewer: iv, Storage: st})
		e := factory.CreateEveEntityCharacter()
		u.Settings().SetRecentSearches([]int64{e.ID})
		a := NewGameSearch(u)
		a.init(t.Context())
		a.recent.Select(0)
		// when
		a.recent.Select(0)
		// then
		assert.Equal(t, []int64{e.ID, e.ID}, eveEntityIDs(iv.shown))
	})
	t.Run("can clear recent searches", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		e := factory.CreateEveEntityCharacter()
		u.Settings().SetRecentSearches([]int64{e.ID})
		a := NewGameSearch(u)
		a.init(t.Context())
		// when
		test.Tap(a.clearRecent)
		// then
		assert.Empty(t, a.recentItems)
		assert.Empty(t, u.Settings().RecentSearches())
	})
}

func TestGameSearch_SearchEntry(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)
	t.Run("searches when entering at least 3 characters", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		e := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{
			"character": {e.ID},
		}))
		a := NewGameSearch(u)
		a.init(t.Context())
		// when
		a.searchEntry.SetText("abc")
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		assert.True(t, a.resultsPage.Visible())
		assert.Equal(t, "1 Results", a.resultCount.Text)
	})
	t.Run("does not search when entering less than 3 characters", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		a := NewGameSearch(u)
		a.init(t.Context())
		// when
		a.searchEntry.SetText("ab")
		// then
		assert.Equal(t, 0, httpmock.GetTotalCallCount())
		assert.False(t, a.resultsPage.Visible())
	})
	t.Run("resets when clearing the entry", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		e := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{
			"character": {e.ID},
		}))
		a := NewGameSearch(u)
		a.init(t.Context())
		a.searchEntry.SetText("abc")
		// when
		a.searchEntry.SetText("")
		// then
		assert.True(t, a.recentPage.Visible())
		assert.False(t, a.resultsPage.Visible())
		assert.False(t, a.resultCount.Visible())
		assert.True(t, a.results.Data().IsEmpty())
	})
	t.Run("can reset", func(t *testing.T) {
		// given
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		e := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{
			"character": {e.ID},
		}))
		a := NewGameSearch(u)
		a.init(t.Context())
		a.searchEntry.SetText("abc")
		// when
		a.Reset()
		// then
		assert.Equal(t, "", a.searchEntry.Text)
		assert.True(t, a.recentPage.Visible())
		assert.False(t, a.resultsPage.Visible())
		assert.True(t, a.results.Data().IsEmpty())
	})
}

func TestSearchResult(t *testing.T) {
	loadIcon := func(_ *app.EveEntity, setIcon func(r fyne.Resource)) {
		setIcon(icons.Typeplaceholder64Png)
	}
	t.Run("shows entity of supported category", func(t *testing.T) {
		// given
		w := newSearchResult(loadIcon, set.Of(app.EveEntityCharacter))
		// when
		w.set(&app.EveEntity{ID: 1, Name: "Alpha", Category: app.EveEntityCharacter})
		// then
		assert.Equal(t, "Alpha", w.name.Text)
		assert.Equal(t, widget.MediumImportance, w.name.Importance)
		assert.Equal(t, icons.Typeplaceholder64Png, w.image.Resource)
	})
	t.Run("shows entity of unsupported category with low importance", func(t *testing.T) {
		// given
		w := newSearchResult(loadIcon, set.Of(app.EveEntityCharacter))
		// when
		w.set(&app.EveEntity{ID: 1, Name: "Alpha", Category: app.EveEntityMailList})
		// then
		assert.Equal(t, "Alpha", w.name.Text)
		assert.Equal(t, widget.LowImportance, w.name.Importance)
	})
}

func TestGameSearch_CanRender(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)
	t.Run("before init", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := NewGameSearch(u)
		w := test.NewWindow(a)
		t.Cleanup(w.Close)

		w.Resize(fyne.NewSize(600, 800))

		assert.True(t, slices.Contains(test.LaidOutObjects(a), fyne.CanvasObject(a.searchEntry)))
	})
	t.Run("with recent searches, results and options open", func(t *testing.T) {
		httpmock.Reset()
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		e1 := factory.CreateEveEntityCharacter()
		et := factory.CreateEveType()
		e2 := factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: et.ID, Name: et.Name})
		u.Settings().SetRecentSearches([]int64{e1.ID})
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{
			"inventory_type": {e2.ID},
		}))
		a := NewGameSearch(u)
		w := test.NewWindow(a)
		t.Cleanup(w.Close)
		w.Resize(fyne.NewSize(600, 800))
		a.init(t.Context())
		a.searchOptions.Open(0)

		a.startSearch("alpha")
		w.Resize(fyne.NewSize(400, 600))

		got := make(map[string]fyne.Resource)
		for _, o := range test.LaidOutObjects(a) {
			if r, ok := o.(*searchResult); ok {
				got[r.name.Text] = r.image.Resource
			}
		}
		assert.Equal(t, map[string]fyne.Resource{
			e1.Name: icons.Characterplaceholder64Jpeg, // recent searches
			e2.Name: icons.Typeplaceholder64Png,       // search results
		}, got)
	})
}
