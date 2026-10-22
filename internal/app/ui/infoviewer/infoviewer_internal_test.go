package infoviewer

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

var _ baseUI = (*testdouble.UIFake)(nil)

// TestInfoViewer_show2_UsesMainWindowAsFallbackWhenInfoWindowClosed is a regression test
// for the bug where show2 used iw.w directly as the dialog parent. After an info window is
// opened and then closed, its OnClosed handler sets iw.w = nil. Any subsequent show2 call
// (e.g. tapping an unsupported entity type) would then pass nil to a Fyne dialog, causing
// a nil-pointer panic. The fix adds a fallback to iw.u.MainWindow() when iw.w is nil.
func TestInfoViewer_show2_UsesMainWindowAsFallbackWhenInfoWindowClosed(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
	// iw.w is nil here, simulating the state after an info window was opened and closed.
	iw := &InfoViewer{
		u: u,
		w: nil,
	}
	// entityID=0 takes the early-return error path which passes parentW to a dialog.
	// Before the fix, parentW was iw.w (nil) here, causing a panic.
	// After the fix, parentW falls back to iw.u.MainWindow(), which is always valid.
	assert.NotPanics(t, func() {
		iw.show2(showParams{variant: Undefined, entityID: 0})
	})
}

func TestInfoViewer_dialogParent(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	a := test.NewTempApp(t)
	u := testdouble.NewUIFake(testdouble.UIParams{App: a, Storage: st})
	t.Run("should return main window when no info window is open", func(t *testing.T) {
		iw := &InfoViewer{u: u}
		assert.Same(t, u.MainWindow(), iw.dialogParent())
	})
	t.Run("should return info window when open", func(t *testing.T) {
		w := a.NewWindow("Information")
		iw := &InfoViewer{u: u, w: w}
		assert.Same(t, w, iw.dialogParent())
	})
}

func TestAttributeList(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
	iw := &InfoViewer{u: u}
	texts := func(w fyne.CanvasObject) []string {
		var s []string
		for _, o := range test.LaidOutObjects(w) {
			if x, ok := o.(*widget.Label); ok && x.Visible() {
				s = append(s, x.Text)
			}
		}
		return s
	}
	t.Run("should show items from constructor", func(t *testing.T) {
		w := newAttributeList(iw, newAttributeItem("Alpha", "1"))
		win := test.NewWindow(w)
		defer win.Close()
		win.Resize(fyne.NewSize(300, 200))
		assert.Subset(t, texts(w), []string{"Alpha", "1"})
	})
	t.Run("should show items after set", func(t *testing.T) {
		w := newAttributeList(iw)
		win := test.NewWindow(w)
		defer win.Close()
		win.Resize(fyne.NewSize(300, 200))
		w.set([]attributeItem{newAttributeItem("Bravo", "2")})
		assert.Subset(t, texts(w), []string{"Bravo", "2"})
	})
}

func TestInfoViewer_Show2(t *testing.T) {
	t.Run("should show alliance for alliance type", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		// when
		iw.Show2(app.EveTypeAlliance, 42, 0)
		// then
		assert.Equal(t, &showParams{variant: Alliance, entityID: 42}, iw.current)
		assert.Equal(t, "Alliance: Information", iw.nav.Current().(*xwidget.AppBar).Title())
	})
	t.Run("should show type when item ID is missing", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		// when
		iw.Show2(app.EveTypeAlliance, 0, 0)
		// then
		assert.Equal(t, &showParams{variant: Type, entityID: app.EveTypeAlliance}, iw.current)
	})
	t.Run("should show error when type ID is missing", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		// when
		iw.Show2(0, 42, 0)
		// then
		assert.Nil(t, iw.current)
		assert.NotNil(t, u.MainWindow().Canvas().Overlays().Top())
	})
	t.Run("should show location for station category", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategoryStation})
		factory.CreateEveGroup(storage.CreateEveGroupParams{ID: 15, CategoryID: app.EveCategoryStation})
		et := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: 15})
		iw := New(u)
		iw.Show2(et.ID, 60_000_001, 7)
		require.Len(t, jobs, 1)
		// when
		jobs[0](t.Context())
		// then
		assert.Equal(t, &showParams{variant: Location, entityID: 60_000_001}, iw.current)
	})
	t.Run("should show location with character for structure category", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategoryStructure})
		factory.CreateEveGroup(storage.CreateEveGroupParams{ID: 1657, CategoryID: app.EveCategoryStructure})
		et := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: 1657})
		iw := New(u)
		iw.Show2(et.ID, 1_000_000_000_001, 7)
		require.Len(t, jobs, 1)
		// when
		jobs[0](t.Context())
		// then
		assert.Equal(t, &showParams{variant: Location, entityID: 1_000_000_000_001, characterID: 7}, iw.current)
	})
	t.Run("should show character for character group", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		factory.CreateEveGroup(storage.CreateEveGroupParams{ID: app.EveGroupCharacter})
		et := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: app.EveGroupCharacter})
		iw := New(u)
		iw.Show2(et.ID, 90_000_001, 0)
		require.Len(t, jobs, 1)
		// when
		jobs[0](t.Context())
		// then
		assert.Equal(t, &showParams{variant: Character, entityID: 90_000_001}, iw.current)
	})
	t.Run("should show error for unsupported type", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategoryShip})
		factory.CreateEveGroup(storage.CreateEveGroupParams{ID: 25, CategoryID: app.EveCategoryShip})
		et := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: 25})
		iw := New(u)
		iw.Show2(et.ID, 42, 0)
		require.Len(t, jobs, 1)
		// when
		jobs[0](t.Context())
		// then
		assert.Nil(t, iw.current)
		assert.NotNil(t, u.MainWindow().Canvas().Overlays().Top())
	})
}

func TestInfoViewer_show(t *testing.T) {
	t.Run("should open window with page", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		// when
		iw.ShowRace(1)
		// then
		assert.NotSame(t, u.MainWindow(), iw.w)
		assert.True(t, iw.nav.IsRoot())
		assert.Equal(t, "Race: Information", iw.nav.Current().(*xwidget.AppBar).Title())
		assert.Len(t, jobs, 1)
	})
	t.Run("should show title without suffix on mobile", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsMobile: true,
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		// when
		iw.ShowRace(1)
		// then
		assert.Equal(t, "Race", iw.nav.Current().(*xwidget.AppBar).Title())
	})
	t.Run("should push new page when window is open", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		iw.ShowRace(1)
		// when
		iw.ShowBloodline(2)
		// then
		assert.False(t, iw.nav.IsRoot())
		assert.Equal(t, "Bloodline: Information", iw.nav.Current().(*xwidget.AppBar).Title())
	})
	t.Run("should not push page again when already shown", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		iw.ShowRace(1)
		// when
		iw.ShowRace(1)
		// then
		assert.True(t, iw.nav.IsRoot())
		assert.Len(t, jobs, 1)
	})
	t.Run("should show solar system for solar system location", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		// when
		iw.ShowLocation(30_000_142)
		// then
		assert.Equal(t, "Solar System: Information", iw.nav.Current().(*xwidget.AppBar).Title())
	})
	t.Run("should show information for unknown location", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		// when
		iw.ShowLocation(app.UnknownLocationID)
		// then
		assert.Nil(t, iw.nav)
		assert.NotNil(t, u.MainWindow().Canvas().Overlays().Top())
	})
	t.Run("should show information again for same unknown location", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		iw.ShowLocation(app.UnknownLocationID)
		overlays := u.MainWindow().Canvas().Overlays()
		require.NotNil(t, overlays.Top())
		overlays.Remove(overlays.Top())
		// when
		iw.ShowLocation(app.UnknownLocationID)
		// then
		assert.NotNil(t, u.MainWindow().Canvas().Overlays().Top())
	})
	t.Run("should not push page again for same solar system location", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		iw.ShowLocation(30_000_142)
		// when
		iw.ShowLocation(30_000_142)
		// then
		assert.True(t, iw.nav.IsRoot())
		assert.Len(t, jobs, 1)
	})
	t.Run("should show information when offline", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Go:      func(_ func(context.Context)) {},
			Storage: st,
		})
		iw := New(u)
		// when
		iw.ShowRace(1)
		// then
		assert.Nil(t, iw.nav)
		assert.NotNil(t, u.MainWindow().Canvas().Overlays().Top())
	})
	t.Run("should show error for unsupported entity", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		// when
		iw.Show(&app.EveEntity{ID: 42, Category: app.EveEntityMailList})
		// then
		assert.Nil(t, iw.nav)
		assert.NotNil(t, u.MainWindow().Canvas().Overlays().Top())
	})
	t.Run("should load page", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		r := factory.CreateEveRace(storage.UpdateOrCreateEveRaceParams{Name: "Caldari"})
		iw := New(u)
		iw.ShowRace(r.ID)
		require.Len(t, jobs, 1)
		// when
		jobs[0](t.Context())
		// then
		page := iw.nav.Current().(*xwidget.AppBar)
		assert.Contains(t, labelTexts(page), "Caldari")
	})
	t.Run("should show error when page load fails", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder("GET", `=~^https://esi\.evetech\.net/universe/races`,
			httpmock.NewStringResponder(404, `{"error":"not found"}`))
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		iw.ShowRace(1)
		require.Len(t, jobs, 1)
		// when
		jobs[0](t.Context())
		// then
		page := iw.nav.Current().(*xwidget.AppBar)
		assert.True(t, slices.ContainsFunc(labelTexts(page), func(s string) bool {
			return strings.HasPrefix(s, "ERROR:")
		}))
	})
	t.Run("should not load page after it was popped", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		r := factory.CreateEveRace(storage.UpdateOrCreateEveRaceParams{Name: "Caldari"})
		iw := New(u)
		iw.ShowBloodline(1)
		iw.ShowRace(r.ID)
		require.Len(t, jobs, 2)
		page := iw.nav.Current().(*xwidget.AppBar)
		iw.nav.Pop()
		// when
		jobs[1](t.Context())
		// then
		assert.NotContains(t, labelTexts(page), "Caldari")
	})
	t.Run("should show page again after going back", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		iw.ShowRace(1)
		iw.ShowBloodline(2)
		iw.nav.Pop()
		// when
		iw.ShowBloodline(2)
		// then
		assert.False(t, iw.nav.IsRoot())
		assert.Equal(t, "Bloodline: Information", iw.nav.Current().(*xwidget.AppBar).Title())
	})
	t.Run("should not push previous page again after going back", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		iw.ShowRace(1)
		iw.ShowBloodline(2)
		iw.nav.Pop()
		// when
		iw.ShowRace(1)
		// then
		assert.True(t, iw.nav.IsRoot())
	})
	t.Run("should reset when window is closed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(job func(context.Context)) { jobs = append(jobs, job) },
			IsOnline: true,
			Storage:  st,
		})
		r := factory.CreateEveRace(storage.UpdateOrCreateEveRaceParams{Name: "Caldari"})
		iw := New(u)
		iw.ShowRace(r.ID)
		require.Len(t, jobs, 1)
		page := iw.nav.Current().(*xwidget.AppBar)
		// when
		iw.w.Close()
		jobs[0](t.Context())
		// then
		assert.Nil(t, iw.nav)
		assert.Nil(t, iw.w)
		assert.Empty(t, iw.pageCancels)
		assert.NotContains(t, labelTexts(page), "Caldari")
	})
	t.Run("should open new window after window was closed", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		iw.ShowRace(1)
		iw.w.Close()
		// when
		iw.ShowRace(1)
		// then
		assert.NotNil(t, iw.w)
		assert.True(t, iw.nav.IsRoot())
	})
}

// labelTexts returns the texts of all labels in o.
func labelTexts(o fyne.CanvasObject) []string {
	var s []string
	for _, x := range test.LaidOutObjects(o) {
		if l, ok := x.(*widget.Label); ok {
			s = append(s, l.Text)
		}
	}
	return s
}

func TestAttributeList_toUIItem(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	t.Cleanup(func() { db.Close() })
	u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
	iw := New(u)
	myURL, _ := url.Parse("https://www.example.com")
	myTime := time.Date(2026, 10, 22, 12, 30, 0, 0, time.UTC)
	cases := []struct {
		name           string
		value          any
		wantText       string
		wantImportance widget.Importance
	}{
		{"entity", &app.EveEntity{Name: "Alpha", Category: app.EveEntityAlliance}, "Alpha", widget.MediumImportance},
		{"nil entity", (*app.EveEntity)(nil), "?", widget.MediumImportance},
		{"faction", &app.EveFaction{Name: "Caldari State"}, "Caldari State", widget.MediumImportance},
		{"nil faction", (*app.EveFaction)(nil), "?", widget.MediumImportance},
		{"race", &app.EveRace{Name: "Caldari"}, "Caldari", widget.MediumImportance},
		{"nil race", (*app.EveRace)(nil), "?", widget.MediumImportance},
		{"bloodline", &eveBloodlineShort{Name: "Deteis"}, "Deteis", widget.MediumImportance},
		{"nil bloodline", (*eveBloodlineShort)(nil), "?", widget.MediumImportance},
		{"location", &app.EveLocation{Name: "Jita IV"}, "Jita IV", widget.MediumImportance},
		{"nil location", (*app.EveLocation)(nil), "?", widget.MediumImportance},
		{"type", &app.EveType{Name: "Merlin"}, "Merlin", widget.MediumImportance},
		{"nil type", (*app.EveType)(nil), "?", widget.MediumImportance},
		{"url", myURL, "https://www.example.com", widget.HighImportance},
		{"nil url", (*url.URL)(nil), "?", widget.MediumImportance},
		{"float32", float32(0.123), "12.3 %", widget.MediumImportance},
		{"time", myTime, "2026.10.22 12:30", widget.MediumImportance},
		{"zero time", time.Time{}, "-", widget.MediumImportance},
		{"int", 1234567, "1,234,567", widget.MediumImportance},
		{"float64", 1.5, "1.5", widget.MediumImportance},
		{"true", true, "yes", widget.SuccessImportance},
		{"false", false, "no", widget.DangerImportance},
		{"other", "abc", "abc", widget.MediumImportance},
	}
	for _, tc := range cases {
		t.Run("should format "+tc.name, func(t *testing.T) {
			// given
			w := newAttributeList(iw)
			// when
			got := w.toUIItem(newAttributeItem("Label", tc.value))
			// then
			assert.Equal(t, "Label", got.Label)
			assert.Equal(t, tc.wantText, got.Value)
			assert.Equal(t, tc.wantImportance, got.Importance)
		})
	}
}

func TestAttributeList_toUIItemActions(t *testing.T) {
	t.Run("should show info for supported entity", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		w := newAttributeList(iw)
		got := w.toUIItem(newAttributeItem("Label", &app.EveEntity{ID: 42, Category: app.EveEntityAlliance}))
		require.NotNil(t, got.InfoAction)
		// when
		got.InfoAction()
		// then
		assert.Equal(t, &showParams{variant: Alliance, entityID: 42}, iw.current)
	})
	t.Run("should have no info for unsupported entity", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		w := newAttributeList(New(u))
		// when
		got := w.toUIItem(newAttributeItem("Label", &app.EveEntity{ID: 42, Category: app.EveEntityMailList}))
		// then
		assert.Nil(t, got.InfoAction)
	})
	t.Run("should have no info for nil entity", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		w := newAttributeList(New(u))
		// when
		got := w.toUIItem(newAttributeItem("Label", (*app.EveEntity)(nil)))
		// then
		assert.Nil(t, got.InfoAction)
	})
	t.Run("should show info for objects", func(t *testing.T) {
		cases := []struct {
			name  string
			value any
			want  showParams
		}{
			{"location", &app.EveLocation{ID: 60_000_001}, showParams{variant: Location, entityID: 60_000_001}},
			{"bloodline", &eveBloodlineShort{ID: 3}, showParams{variant: Bloodline, entityID: 3}},
			{"faction", &app.EveFaction{ID: 500001}, showParams{variant: Faction, entityID: 500001}},
			{"race", &app.EveRace{ID: 1}, showParams{variant: Race, entityID: 1}},
			{"type", &app.EveType{ID: 603}, showParams{variant: Type, entityID: 603}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// given
				db, st, _ := testutil.NewDBOnDisk(t)
				t.Cleanup(func() { db.Close() })
				u := testdouble.NewUIFake(testdouble.UIParams{
					App:      test.NewTempApp(t),
					Go:       func(_ func(context.Context)) {},
					IsOnline: true,
					Storage:  st,
				})
				iw := New(u)
				w := newAttributeList(iw)
				got := w.toUIItem(newAttributeItem("Label", tc.value))
				require.NotNil(t, got.InfoAction)
				// when
				got.InfoAction()
				// then
				assert.Equal(t, &tc.want, iw.current)
			})
		}
	})
	t.Run("should open url", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		w := newAttributeList(New(u))
		var opened *url.URL
		w.openURL = func(x *url.URL) error {
			opened = x
			return nil
		}
		myURL, _ := url.Parse("https://www.example.com")
		got := w.toUIItem(newAttributeItem("Label", myURL))
		require.NotNil(t, got.Action)
		// when
		got.Action()
		// then
		assert.Same(t, myURL, opened)
	})
	t.Run("should run custom action with value", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		w := newAttributeList(New(u))
		var called any
		it := newAttributeItem("Label", 42)
		it.Action = func(v any) {
			called = v
		}
		got := w.toUIItem(it)
		require.NotNil(t, got.Action)
		// when
		got.Action()
		// then
		assert.Equal(t, 42, called)
	})
	t.Run("should have no action by default", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		w := newAttributeList(New(u))
		// when
		got := w.toUIItem(newAttributeItem("Label", 42))
		// then
		assert.Nil(t, got.Action)
	})
}

func TestInfoViewer_LinkIcons(t *testing.T) {
	cases := []struct {
		name        string
		makeIcon    func(iw *InfoViewer) *xwidget.Button
		wantURL     string
		wantToolTip string
	}{
		{
			"zKillboard alliance",
			func(iw *InfoViewer) *xwidget.Button { return iw.makeZKillboardIcon(42, Alliance) },
			"https://zkillboard.com/alliance/42/",
			"Show alliance on zKillboard.com",
		},
		{
			"zKillboard solar system",
			func(iw *InfoViewer) *xwidget.Button { return iw.makeZKillboardIcon(42, SolarSystem) },
			"https://zkillboard.com/system/42/",
			"Show solarsystem on zKillboard.com",
		},
		{
			"dotlan corporation",
			func(iw *InfoViewer) *xwidget.Button { return iw.makeDotlanIcon(42, Corporation) },
			"https://evemaps.dotlan.net/corp/42",
			"Show corporation on evemaps.dotlan.net",
		},
		{
			"dotlan region",
			func(iw *InfoViewer) *xwidget.Button { return iw.makeDotlanIcon(42, Region) },
			"https://evemaps.dotlan.net/region/42",
			"Show region on evemaps.dotlan.net",
		},
		{
			"evewho character",
			func(iw *InfoViewer) *xwidget.Button { return iw.makeEveWhoIcon(42, Character) },
			"https://evewho.com/character/42",
			"Show character on evewho.com",
		},
	}
	for _, tc := range cases {
		t.Run("should open "+tc.name, func(t *testing.T) {
			// given
			db, st, _ := testutil.NewDBOnDisk(t)
			t.Cleanup(func() { db.Close() })
			u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
			iw := New(u)
			var opened string
			iw.openURLFunc = func(x *url.URL) error {
				opened = x.String()
				return nil
			}
			icon := tc.makeIcon(iw)
			require.NotNil(t, icon.OnTapped)
			// when
			icon.OnTapped()
			// then
			assert.Equal(t, tc.wantURL, opened)
			assert.Equal(t, tc.wantToolTip, icon.ToolTip())
		})
	}
	t.Run("should not be tappable for unsupported kinds", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		iw := New(u)
		// when
		icons := []*xwidget.Button{
			iw.makeZKillboardIcon(42, Race),
			iw.makeDotlanIcon(42, Character),
			iw.makeEveWhoIcon(42, Region),
		}
		// then
		for _, icon := range icons {
			assert.Nil(t, icon.OnTapped)
			assert.Empty(t, icon.ToolTip())
		}
	})
}

func TestHistoryItem2EntityItem(t *testing.T) {
	t.Run("should show past membership", func(t *testing.T) {
		// given
		hi := app.MembershipHistoryItem{
			Days:         365,
			EndDate:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			Organization: &app.EveEntity{ID: 42, Name: "Alpha", Category: app.EveEntityCorporation},
			StartDate:    time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		}
		// when
		got := historyItem2EntityItem(hi)
		// then
		assert.Equal(t, "Alpha   **2024.01.02** to **2025.01.01** (365 days)", got.text)
		assert.Equal(t, int64(42), got.id)
		assert.Equal(t, Corporation, got.infoVariant)
	})
	t.Run("should show current membership of closed organization", func(t *testing.T) {
		// given
		hi := app.MembershipHistoryItem{
			Days:         1234,
			IsDeleted:    optional.New(true),
			Organization: &app.EveEntity{ID: 42, Name: "Alpha", Category: app.EveEntityCorporation},
			StartDate:    time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		}
		// when
		got := historyItem2EntityItem(hi)
		// then
		assert.Equal(t, "Alpha (closed)   **2024.01.02** to **this day** (1,234 days)", got.text)
	})
}

func TestEveEntity2InfoVariant(t *testing.T) {
	cases := []struct {
		category app.EveEntityCategory
		want     Kind
	}{
		{app.EveEntityAlliance, Alliance},
		{app.EveEntityCharacter, Character},
		{app.EveEntityConstellation, Constellation},
		{app.EveEntityCorporation, Corporation},
		{app.EveEntityFaction, Faction},
		{app.EveEntityInventoryType, Type},
		{app.EveEntityRegion, Region},
		{app.EveEntitySolarSystem, SolarSystem},
		{app.EveEntityStation, Location},
		{app.EveEntityMailList, Undefined},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("should return %s for %s", tc.want, tc.category), func(t *testing.T) {
			got := eveEntity2InfoVariant(&app.EveEntity{Category: tc.category})
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEntityList(t *testing.T) {
	t.Run("should show info when info icon is tapped", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		type shown struct {
			kind Kind
			id   int64
		}
		var got []shown
		w := newEntityListFromItems(func(k Kind, id int64) {
			got = append(got, shown{k, id})
		}, newEntityItem(42, "Alliance", "Alpha", Alliance))
		win := test.NewWindow(w)
		defer win.Close()
		win.Resize(fyne.NewSize(300, 200))
		var icon *xwidget.Button
		for _, o := range test.LaidOutObjects(w) {
			if x, ok := o.(*xwidget.Button); ok && x.Visible() {
				icon = x
			}
		}
		require.NotNil(t, icon)
		// when
		test.Tap(icon)
		// then
		assert.Equal(t, []shown{{Alliance, 42}}, got)
	})
	t.Run("should hide info icon for undefined kind", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newEntityListFromItems(func(_ Kind, _ int64) {}, newEntityItem(0, "Service", "Cloning", Undefined))
		win := test.NewWindow(w)
		defer win.Close()
		// when
		win.Resize(fyne.NewSize(300, 200))
		// then
		var icon *xwidget.Button
		for _, o := range test.LaidOutObjects(w) {
			if x, ok := o.(*xwidget.Button); ok {
				icon = x
			}
		}
		require.NotNil(t, icon)
		assert.False(t, icon.Visible())
	})
	t.Run("should show items after set", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newEntityList(func(_ Kind, _ int64) {})
		win := test.NewWindow(w)
		defer win.Close()
		win.Resize(fyne.NewSize(300, 200))
		// when
		w.set(newEntityItem(42, "Alliance", "Alpha", Alliance))
		// then
		assert.Contains(t, labelTexts(w), "Alliance")
	})
}

// tabLabelTexts returns the texts of all labels in the tab with title.
func tabLabelTexts(t *testing.T, tabs *container.AppTabs, title string) []string {
	t.Helper()
	for _, ti := range tabs.Items {
		if ti.Text != title {
			continue
		}
		w := test.NewWindow(ti.Content)
		t.Cleanup(w.Close)
		w.Resize(fyne.NewSize(600, 800))
		return labelTexts(ti.Content)
	}
	t.Fatalf("tab not found: %s", title)
	return nil
}
