package settings

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	asettings "github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xmaps"
)

func TestMain(m *testing.M) {
	// Leftover goroutines from one test can render text concurrently with the next,
	// which is not safe in Fyne's shared text shaper.
	runAsync = func(f func()) { f() }
	os.Exit(m.Run())
}

func TestShow(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("can render window", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		// when
		Show(u)
		// then
		w, created := u.GetOrCreateWindow("settingsWindow")
		assert.False(t, created)
		assert.NotNil(t, w.Canvas().Capture())
		w.Close()
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		Show(u)
		n := len(fyne.CurrentApp().Driver().AllWindows())
		// when
		Show(u)
		// then
		assert.Equal(t, n, len(fyne.CurrentApp().Driver().AllWindows()))
		w, _ := u.GetOrCreateWindow("settingsWindow")
		w.Close()
	})
}

func TestSettingItemReset(t *testing.T) {
	t.Run("does nothing when setter is undefined", func(t *testing.T) {
		// given
		it := SettingItem{Default: 1, Getter: func() any { return 2 }}
		// when/then
		assert.NotPanics(t, it.Reset)
	})
	t.Run("does nothing when default is undefined", func(t *testing.T) {
		// given
		var calls int
		it := SettingItem{
			Getter: func() any { return 2 },
			Setter: func(_ any) { calls++ },
		}
		// when
		it.Reset()
		// then
		assert.Equal(t, 0, calls)
	})
	t.Run("does nothing when value is already the default", func(t *testing.T) {
		// given
		var calls int
		it := SettingItem{
			Default: 1,
			Getter:  func() any { return 1 },
			Setter:  func(_ any) { calls++ },
		}
		// when
		it.Reset()
		// then
		assert.Equal(t, 0, calls)
	})
	t.Run("sets default when value differs", func(t *testing.T) {
		// given
		var got any
		it := SettingItem{
			Default: 1,
			Getter:  func() any { return 2 },
			Setter:  func(v any) { got = v },
		}
		// when
		it.Reset()
		// then
		assert.Equal(t, 1, got)
	})
}

func TestSettingItemSwitch(t *testing.T) {
	t.Run("selecting toggles the value and refreshes", func(t *testing.T) {
		// given
		var v bool
		it := NewSettingItemSwitch(SettingItemSwitchParams{
			label:     "Alpha",
			getter:    func() bool { return v },
			onChanged: func(b bool) { v = b },
		})
		var refreshed bool
		// when
		it.onSelected(it, func() { refreshed = true })
		// then
		assert.True(t, v)
		assert.True(t, refreshed)
	})
}

func TestSettingItemSlider(t *testing.T) {
	t.Run("setter accepts float64", func(t *testing.T) {
		// given
		var v float64
		it := NewSettingItemSlider(SettingItemSliderParams{
			label:  "Alpha",
			getter: func() float64 { return v },
			setter: func(x float64) { v = x },
		})
		// when
		it.Setter(float64(3))
		// then
		assert.Equal(t, float64(3), v)
	})
	t.Run("setter accepts int", func(t *testing.T) {
		// given
		var v float64
		it := NewSettingItemSlider(SettingItemSliderParams{
			label:  "Alpha",
			getter: func() float64 { return v },
			setter: func(x float64) { v = x },
		})
		// when
		it.Setter(3)
		// then
		assert.Equal(t, float64(3), v)
	})
	t.Run("setter panics for unsupported type", func(t *testing.T) {
		// given
		it := NewSettingItemSlider(SettingItemSliderParams{
			label:  "Alpha",
			getter: func() float64 { return 0 },
			setter: func(_ float64) {},
		})
		// when/then
		assert.Panics(t, func() {
			it.Setter("3")
		})
	})
	t.Run("can change value in dialog", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := test.NewTempWindow(t, nil)
		w.Resize(fyne.NewSize(700, 500))
		v := float64(10)
		it := NewSettingItemSlider(SettingItemSliderParams{
			label:        "Alpha",
			minValue:     0,
			maxValue:     100,
			defaultValue: 50,
			step:         1,
			getter:       func() float64 { return v },
			setter:       func(x float64) { v = x },
			window:       w,
		})
		it.onSelected(it, func() {})
		sl := findInDialog[*kxwidget.Slider](t, w)
		// when
		sl.OnChangeEnded(20)
		// then
		assert.Equal(t, float64(20), v)
	})
	t.Run("reset button in dialog stores the default", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := test.NewTempWindow(t, nil)
		w.Resize(fyne.NewSize(700, 500))
		v := float64(10)
		it := NewSettingItemSlider(SettingItemSliderParams{
			label:        "Alpha",
			minValue:     0,
			maxValue:     100,
			defaultValue: 50,
			step:         1,
			getter:       func() float64 { return v },
			setter:       func(x float64) { v = x },
			window:       w,
		})
		it.onSelected(it, func() {})
		// when
		tapDialogButton(t, w, "Reset")
		// then
		assert.Equal(t, float64(50), v)
	})
	t.Run("ok button closes dialog and refreshes", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := test.NewTempWindow(t, nil)
		w.Resize(fyne.NewSize(700, 500))
		it := NewSettingItemSlider(SettingItemSliderParams{
			label:    "Alpha",
			minValue: 0,
			maxValue: 100,
			step:     1,
			getter:   func() float64 { return 10 },
			setter:   func(_ float64) {},
			window:   w,
		})
		var refreshed bool
		it.onSelected(it, func() { refreshed = true })
		// when
		tapDialogButton(t, w, "OK")
		// then
		assert.True(t, refreshed)
		assert.Nil(t, w.Canvas().Overlays().Top())
	})
}

func TestSettingItemOptions(t *testing.T) {
	t.Run("can select option in dialog", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := test.NewTempWindow(t, nil)
		w.Resize(fyne.NewSize(700, 500))
		v := "a"
		it := NewSettingItemOptions(SettingItemOptionsParams{
			label:        "Alpha",
			options:      []string{"a", "b", "c"},
			defaultValue: "c",
			getter:       func() string { return v },
			setter:       func(x string) { v = x },
			window:       w,
		})
		it.onSelected(it, func() {})
		rg := findInDialog[*widget.RadioGroup](t, w)
		// when
		rg.SetSelected("b")
		// then
		assert.Equal(t, "b", v)
	})
	t.Run("reset button in dialog stores the default", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := test.NewTempWindow(t, nil)
		w.Resize(fyne.NewSize(700, 500))
		v := "a"
		it := NewSettingItemOptions(SettingItemOptionsParams{
			label:        "Alpha",
			options:      []string{"a", "b", "c"},
			defaultValue: "c",
			getter:       func() string { return v },
			setter:       func(x string) { v = x },
			window:       w,
		})
		it.onSelected(it, func() {})
		// when
		tapDialogButton(t, w, "Reset")
		// then
		assert.Equal(t, "c", v)
	})
}

func TestSettingListItem(t *testing.T) {
	test.NewTempApp(t)

	t.Run("shows heading", func(t *testing.T) {
		// given
		w := newSettingListItem()
		test.NewTempWindow(t, w)
		// when
		w.set(NewSettingItemHeading("Alpha"))
		// then
		assert.Equal(t, "Alpha", w.header.Text)
		assert.True(t, w.header.Visible())
		assert.False(t, w.label.Visible())
		assert.False(t, w.value.Visible())
		assert.False(t, w.sw.Visible())
		assert.False(t, w.hint.Visible())
	})
	t.Run("shows switch", func(t *testing.T) {
		// given
		w := newSettingListItem()
		test.NewTempWindow(t, w)
		var v bool
		// when
		w.set(NewSettingItemSwitch(SettingItemSwitchParams{
			label:     "Alpha",
			hint:      "Bravo",
			getter:    func() bool { return true },
			onChanged: func(b bool) { v = b },
		}))
		// then
		assert.Equal(t, "Alpha", w.label.Text)
		assert.Equal(t, "Bravo", w.hint.Text)
		assert.True(t, w.hint.Visible())
		assert.True(t, w.sw.Visible())
		assert.True(t, w.sw.On)
		assert.False(t, w.value.Visible())
		assert.False(t, w.header.Visible())
		w.sw.OnChanged(false)
		assert.False(t, v)
	})
	t.Run("shows custom value with default formatting", func(t *testing.T) {
		// given
		w := newSettingListItem()
		test.NewTempWindow(t, w)
		// when
		w.set(NewSettingItemCustom(SettingItemCustomParams{
			label:  "Alpha",
			getter: func() any { return 42 },
		}))
		// then
		assert.Equal(t, "Alpha", w.label.Text)
		assert.Equal(t, "42", w.value.Text)
		assert.True(t, w.value.Visible())
		assert.False(t, w.sw.Visible())
		assert.False(t, w.hint.Visible())
	})
	t.Run("shows custom value with formatter", func(t *testing.T) {
		// given
		w := newSettingListItem()
		test.NewTempWindow(t, w)
		// when
		w.set(NewSettingItemSlider(SettingItemSliderParams{
			label:     "Alpha",
			getter:    func() float64 { return 42 },
			setter:    func(_ float64) {},
			formatter: func(_ any) string { return "formatted" },
		}))
		// then
		assert.Equal(t, "formatted", w.value.Text)
	})
}

func TestSettingList(t *testing.T) {
	t.Run("selecting an item calls its action and clears the selection", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		var v bool
		list := newSettingList([]SettingItem{
			NewSettingItemHeading("Alpha"),
			NewSettingItemSwitch(SettingItemSwitchParams{
				label:     "Bravo",
				getter:    func() bool { return v },
				onChanged: func(b bool) { v = b },
			}),
		})
		test.NewTempWindow(t, list)
		var selected []widget.ListItemID
		list.OnUnselected = func(id widget.ListItemID) { selected = append(selected, id) }
		// when
		list.Select(1)
		// then
		assert.True(t, v)
		assert.Equal(t, []widget.ListItemID{1}, selected)
	})
	t.Run("selecting a heading does nothing", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		list := newSettingList([]SettingItem{
			NewSettingItemHeading("Alpha"),
		})
		test.NewTempWindow(t, list)
		// when/then
		assert.NotPanics(t, func() {
			list.Select(0)
		})
	})
}

func TestGeneralPage(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("shows desktop only items on desktop", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		a := makeSettings(t, u)
		// when
		list, actions := a.makeGeneralPage()
		// then
		labels := itemLabels(list.items)
		assert.Contains(t, labels, "Run in background")
		assert.Contains(t, labels, "UI Scale")
		assert.Contains(t, labels, "Disable DPI detection")
		assert.Contains(t, actionLabels(actions), "Resets main window size to defaults")
	})
	t.Run("hides desktop only items on mobile", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st, IsMobile: true})
		t.Cleanup(u.Settings().Close)
		a := makeSettings(t, u)
		// when
		list, actions := a.makeGeneralPage()
		// then
		labels := itemLabels(list.items)
		assert.NotContains(t, labels, "Run in background")
		assert.NotContains(t, labels, "UI Scale")
		assert.NotContains(t, labels, "Disable DPI detection")
		assert.NotContains(t, actionLabels(actions), "Resets main window size to defaults")
	})
	t.Run("hiding limited corporations emits signal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		var emitted bool
		u.Signals().CorporationsChanged.AddListener(func(_ context.Context, _ struct{}) {
			emitted = true
		})
		a := makeSettings(t, u)
		list, _ := a.makeGeneralPage()
		it := findItem(t, list.items, "Hide limited corporations")
		// when
		it.Setter(true)
		// then
		assert.True(t, u.Settings().HideLimitedCorporations())
		assert.True(t, emitted)
	})
	t.Run("changing appearance updates color theme", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		var got asettings.ColorTheme
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:               test.NewTempApp(t),
			Storage:           st,
			SetColorThemeFunc: func(s asettings.ColorTheme) { got = s },
		})
		t.Cleanup(u.Settings().Close)
		a := makeSettings(t, u)
		list, _ := a.makeGeneralPage()
		it := findItem(t, list.items, "Appearance")
		// when
		it.Setter(string(asettings.Dark))
		// then
		assert.Equal(t, asettings.Dark, u.Settings().ColorTheme())
		assert.Equal(t, asettings.Dark, got)
	})
	t.Run("changing developer mode updates UI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		a := makeSettings(t, u)
		list, _ := a.makeGeneralPage()
		it := findItem(t, list.items, "Developer mode")
		// when
		it.Setter(true)
		// then
		assert.True(t, u.Settings().DeveloperMode())
		assert.True(t, u.IsDeveloperMode())
	})
	t.Run("changing log level updates logger", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		orig := slog.SetLogLoggerLevel(slog.LevelInfo)
		t.Cleanup(func() { slog.SetLogLoggerLevel(orig) })
		a := makeSettings(t, u)
		list, _ := a.makeGeneralPage()
		it := findItem(t, list.items, "Log level")
		// when
		it.Setter("debug")
		// then
		assert.Equal(t, "debug", u.Settings().LogLevel())
		assert.True(t, slog.Default().Enabled(context.Background(), slog.LevelDebug))
	})
	t.Run("reset restores defaults", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		orig := slog.SetLogLoggerLevel(slog.LevelInfo)
		t.Cleanup(func() { slog.SetLogLoggerLevel(orig) })
		s := u.Settings()
		s.SetApprovedContactCost(100)
		s.SetColorTheme(asettings.Dark)
		s.SetDeveloperMode(true)
		s.SetDisableDPIDetection(true)
		s.SetFyneScale(1.5)
		s.SetHideLimitedCorporations(true)
		s.SetLogLevel("debug")
		s.SetMarketOrdersRetentionDay(10)
		s.SetMaxMails(10)
		s.SetMaxWalletTransactions(10)
		s.SetPreferMarketTab(true)
		s.SetSysTrayEnabled(false)
		a := makeSettings(t, u)
		_, actions := a.makeGeneralPage()
		// when
		findAction(t, actions, "Reset to defaults").Action()
		// then
		assert.Equal(t, 0, s.ApprovedContactCost())
		assert.Equal(t, s.ColorThemeDefault(), s.ColorTheme())
		assert.False(t, s.DeveloperMode())
		assert.False(t, s.DisableDPIDetection())
		assert.Equal(t, s.FyneScaleDefault(), s.FyneScale())
		assert.Equal(t, s.HideLimitedCorporationsDefault(), s.HideLimitedCorporations())
		assert.Equal(t, s.LogLevelDefault(), s.LogLevel())
		_, _, def := s.MarketOrderRetentionDaysPresets()
		assert.Equal(t, def, s.MarketOrderRetentionDays())
		_, _, def = s.MaxMailsPresets()
		assert.Equal(t, def, s.MaxMails())
		_, _, def = s.MaxWalletTransactionsPresets()
		assert.Equal(t, def, s.MaxWalletTransactions())
		assert.False(t, s.PreferMarketTab())
		assert.Equal(t, s.SysTrayEnabledDefault(), s.SysTrayEnabled())
	})
	t.Run("reset updates shown values", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		u.Settings().SetFyneScale(1.5)
		a := makeSettings(t, u)
		list, actions := a.makeGeneralPage()
		a.w.SetContent(list)
		a.w.Resize(fyne.NewSize(700, 2000))
		require.Equal(t, "150 %", shownValue(t, list, "UI Scale"))
		// when
		findAction(t, actions, "Reset to defaults").Action()
		// then
		assert.Equal(t, "100 %", shownValue(t, list, "UI Scale"))
	})
	t.Run("shows message when log file to export does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:       test.NewTempApp(t),
			Storage:   st,
			DataPaths: xmaps.OrderedMap[string, string]{"log": filepath.Join(t.TempDir(), "evebuddy.log")},
		})
		t.Cleanup(u.Settings().Close)
		a := makeSettings(t, u)
		var messages []string
		a.displaySnackbar = func(text string) { messages = append(messages, text) }
		_, actions := a.makeGeneralPage()
		// when
		findAction(t, actions, "Export application log").Action()
		// then
		require.Len(t, messages, 1)
		assert.Equal(t, "No file to export: evebuddy.log", messages[0])
	})
	t.Run("can reset main window size", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		u.Settings().SetWindowSize(fyne.NewSize(300, 200))
		a := makeSettings(t, u)
		_, actions := a.makeGeneralPage()
		// when
		findAction(t, actions, "Resets main window size to defaults").Action()
		// then
		assert.Equal(t, fyne.NewSize(1000, 600), u.Settings().WindowSize())
		assert.Equal(t, fyne.NewSize(1000, 600), u.MainWindow().Canvas().Size())
	})
}

func TestNotificationPage(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()

	// a group and its notification types
	group := slices.Sorted(app.NotificationTypesSupported().All())[0].Group()
	var groupTypes []string
	for nt := range app.NotificationTypesSupported().All() {
		if nt.Group() == group {
			groupTypes = append(groupTypes, nt.String())
		}
	}
	require.Greater(t, len(groupTypes), 1)

	t.Run("enabling mails sets earliest", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		a := makeSettings(t, u)
		list, _ := a.makeNotificationPage()
		it := findItem(t, list.items, "Notify mails")
		// when
		it.Setter(true)
		// then
		assert.True(t, u.Settings().NotifyMailsEnabled())
		assert.WithinDuration(t, time.Now(), u.Settings().NotifyMailsEarliest(), 5*time.Second)
	})
	t.Run("enabling PI sets earliest", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		a := makeSettings(t, u)
		list, _ := a.makeNotificationPage()
		it := findItem(t, list.items, "Planetary Industry")
		// when
		it.Setter(true)
		// then
		assert.True(t, u.Settings().NotifyPIEnabled())
		assert.WithinDuration(t, time.Now(), u.Settings().NotifyPIEarliest(), 5*time.Second)
	})
	t.Run("enabling contracts sets earliest", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		a := makeSettings(t, u)
		list, _ := a.makeNotificationPage()
		it := findItem(t, list.items, "Notify Contracts")
		// when
		it.Setter(true)
		// then
		assert.True(t, u.Settings().NotifyContractsEnabled())
		assert.WithinDuration(t, time.Now(), u.Settings().NotifyContractsEarliest(), 5*time.Second)
	})
	t.Run("group shows off when no type is enabled", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		a := makeSettings(t, u)
		// when
		list, _ := a.makeNotificationPage()
		// then
		assert.Equal(t, "Off", findItem(t, list.items, group.String()).Getter())
	})
	t.Run("group shows some when some types are enabled", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		u.Settings().SetNotificationTypesEnabled(set.Of(groupTypes[0]))
		a := makeSettings(t, u)
		// when
		list, _ := a.makeNotificationPage()
		// then
		assert.Equal(t, "Some", findItem(t, list.items, group.String()).Getter())
	})
	t.Run("group shows all when all types are enabled", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		u.Settings().SetNotificationTypesEnabled(set.Of(groupTypes...))
		a := makeSettings(t, u)
		// when
		list, _ := a.makeNotificationPage()
		// then
		assert.Equal(t, "All", findItem(t, list.items, group.String()).Getter())
	})
	t.Run("can enable all types of a group in dialog", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		a := makeSettings(t, u)
		list, _ := a.makeNotificationPage()
		a.w.SetContent(list)
		list.Select(findItemIndex(t, list.items, group.String()))
		// when
		tapDialogButton(t, a.w, "Enable all")
		// then
		assert.ElementsMatch(t, groupTypes, slices.Collect(u.Settings().NotificationTypesEnabled().All()))
	})
	t.Run("can disable all types of a group in dialog", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		u.Settings().SetNotificationTypesEnabled(set.Of(groupTypes...))
		a := makeSettings(t, u)
		list, _ := a.makeNotificationPage()
		a.w.SetContent(list)
		list.Select(findItemIndex(t, list.items, group.String()))
		// when
		tapDialogButton(t, a.w, "Disable all")
		// then
		assert.Equal(t, 0, u.Settings().NotificationTypesEnabled().Size())
	})
	t.Run("closing group dialog updates shown status", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		a := makeSettings(t, u)
		list, _ := a.makeNotificationPage()
		a.w.SetContent(list)
		a.w.Resize(fyne.NewSize(700, 2000))
		list.Select(findItemIndex(t, list.items, group.String()))
		tapDialogButton(t, a.w, "Enable all")
		// when
		tapDialogButton(t, a.w, "OK")
		// then
		assert.Equal(t, "All", shownValue(t, list, group.String()))
	})
	t.Run("can enable all groups", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		a := makeSettings(t, u)
		_, actions := a.makeNotificationPage()
		// when
		findAction(t, actions, "Enable all communication groups").Action()
		// then
		assert.Equal(t, app.NotificationTypesSupported().Size(), u.Settings().NotificationTypesEnabled().Size())
	})
	t.Run("can disable all groups", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		u.Settings().SetNotificationTypesEnabled(set.Of(groupTypes...))
		a := makeSettings(t, u)
		_, actions := a.makeNotificationPage()
		// when
		findAction(t, actions, "Disable all communication groups").Action()
		// then
		assert.Equal(t, 0, u.Settings().NotificationTypesEnabled().Size())
	})
	t.Run("reset restores defaults", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		s := u.Settings()
		s.SetNotificationTypesEnabled(set.Of(groupTypes...))
		s.SetNotifyCommunicationsEnabled(true)
		s.SetNotifyContractsEnabled(true)
		s.SetNotifyMailsEnabled(true)
		s.SetNotifyPIEnabled(true)
		s.SetNotifyTimeoutHours(1)
		s.SetNotifyTrainingEnabled(true)
		a := makeSettings(t, u)
		_, actions := a.makeNotificationPage()
		// when
		findAction(t, actions, "Reset to defaults").Action()
		// then
		assert.Equal(t, 0, s.NotificationTypesEnabled().Size())
		assert.Equal(t, s.NotifyCommunicationsEnabledDefault(), s.NotifyCommunicationsEnabled())
		assert.Equal(t, s.NotifyContractsEnabledDefault(), s.NotifyContractsEnabled())
		assert.Equal(t, s.NotifyMailsEnabledDefault(), s.NotifyMailsEnabled())
		assert.Equal(t, s.NotifyPIEnabledDefault(), s.NotifyPIEnabled())
		_, _, def := s.NotifyTimeoutHoursPresets()
		assert.Equal(t, def, s.NotifyTimeoutHours())
		assert.Equal(t, s.NotifyTrainingEnabledDefault(), s.NotifyTrainingEnabled())
	})
	t.Run("can send test notification", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		t.Cleanup(u.Settings().Close)
		a := makeSettings(t, u)
		_, actions := a.makeNotificationPage()
		// when/then
		test.AssertNotificationSent(t, fyne.NewNotification("Test", "This is a test notification from EVE Buddy."), func() {
			findAction(t, actions, "Send test notification").Action()
		})
	})
}

// makeSettings returns a new settings widget in its own window.
func makeSettings(t *testing.T, u baseUI) *settings {
	t.Helper()
	w := test.NewTempWindow(t, nil)
	w.Resize(fyne.NewSize(700, 500))
	a := newSettings(u, w)
	t.Cleanup(a.sb.Stop)
	return a
}

func itemLabels(items []SettingItem) []string {
	var labels []string
	for _, it := range items {
		labels = append(labels, it.Label)
	}
	return labels
}

func actionLabels(actions []settingAction) []string {
	var labels []string
	for _, a := range actions {
		labels = append(labels, a.Label)
	}
	return labels
}

func findItemIndex(t *testing.T, items []SettingItem, label string) int {
	t.Helper()
	i := slices.IndexFunc(items, func(it SettingItem) bool {
		return it.Label == label
	})
	require.NotEqual(t, -1, i, "item not found: %s", label)
	return i
}

func findItem(t *testing.T, items []SettingItem, label string) SettingItem {
	t.Helper()
	return items[findItemIndex(t, items, label)]
}

func findAction(t *testing.T, actions []settingAction, label string) settingAction {
	t.Helper()
	i := slices.IndexFunc(actions, func(a settingAction) bool {
		return a.Label == label
	})
	require.NotEqual(t, -1, i, "action not found: %s", label)
	return actions[i]
}

// findInDialog returns the first object of type T in the dialog shown in w.
func findInDialog[T fyne.CanvasObject](t *testing.T, w fyne.Window) T {
	t.Helper()
	top := w.Canvas().Overlays().Top()
	require.NotNil(t, top, "no dialog shown")
	for _, o := range test.LaidOutObjects(top) {
		if x, ok := o.(T); ok {
			return x
		}
	}
	var zero T
	require.Fail(t, "object not found in dialog")
	return zero
}

// tapDialogButton taps the button with label in the dialog shown in w.
func tapDialogButton(t *testing.T, w fyne.Window, label string) {
	t.Helper()
	top := w.Canvas().Overlays().Top()
	require.NotNil(t, top, "no dialog shown")
	for _, o := range test.LaidOutObjects(top) {
		if b, ok := o.(*widget.Button); ok && b.Text == label {
			test.Tap(b)
			return
		}
	}
	require.Fail(t, "button not found in dialog: "+label)
}

// shownValue returns the value shown in the list for the item with label.
func shownValue(t *testing.T, list *settingList, label string) string {
	t.Helper()
	for _, o := range test.LaidOutObjects(list) {
		if li, ok := o.(*settingListItem); ok && li.label.Text == label && li.label.Visible() {
			return li.value.Text
		}
	}
	require.Fail(t, "item not shown: "+label)
	return ""
}
