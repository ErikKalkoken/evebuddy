// Package settingswindow provides a window to view and configure user settings.
package settingswindow

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
	"github.com/ErikKalkoken/evebuddy/internal/xmaps"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type ui interface {
	ClearAllCaches()
	DataPaths() xmaps.OrderedMap[string, string]
	ErrorDisplay(err error) string
	GetOrCreateWindowWithOnClosed(id string, titles ...string) (window fyne.Window, created bool, onClosed func())
	IsDeveloperMode() bool
	IsMobile() bool
	MainWindow() fyne.Window
	ResetCharacter(ctx context.Context)
	ResetCorporation(ctx context.Context)
	SetColorTheme(s settings.ColorTheme)
	SetDeveloperMode(b bool)
	Settings() *settings.Settings
	Signals() *app.Signals
}

func Show(s ui) {
	w, ok, onClosed := s.GetOrCreateWindowWithOnClosed("settingsWindow", "Settings")
	if !ok {
		w.Show()
		return
	}
	a := newSettingsWindow(s, w)
	w.SetContent(fynetooltip.AddWindowToolTipLayer(a, w.Canvas()))
	w.Resize(fyne.Size{Width: 700, Height: 500})
	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		a.sb.Stop()
	})
	w.SetCloseIntercept(func() {
		w.Close()
		fynetooltip.DestroyWindowToolTipLayer(w.Canvas())
	})
	w.Show()
}

type settingAction struct {
	Label  string
	Action func()
}

type settingsWindow struct {
	widget.BaseWidget

	sb *xwidget.Snackbar
	u  ui
	w  fyne.Window
}

func newSettingsWindow(u ui, w fyne.Window) *settingsWindow {
	a := &settingsWindow{
		sb: xwidget.NewSnackbar(w),
		u:  u,
		w:  w,
	}
	a.ExtendBaseWidget(a)
	a.sb.Start()
	return a
}

func (a *settingsWindow) CreateRenderer() fyne.WidgetRenderer {
	makeSettingsPage := func(title string, content fyne.CanvasObject, actions fyne.CanvasObject) fyne.CanvasObject {
		ab := xwidget.NewAppBar(title, content, actions)
		ab.HideBackground = !a.u.IsMobile()
		return ab
	}
	generalContent, generalActions := a.makeGeneralPage()
	notificationContent, notificationActions := a.makeNotificationPage()
	tabs := container.NewAppTabs(
		container.NewTabItem("General", makeSettingsPage(
			"General",
			generalContent,
			generalActions,
		)),
		container.NewTabItem("Notifications", makeSettingsPage(
			"Notifications",
			notificationContent,
			notificationActions,
		)),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	return widget.NewSimpleRenderer(tabs)
}

func (a *settingsWindow) makeGeneralPage() (fyne.CanvasObject, *kxwidget.IconButton) {
	logLevel := NewSettingItemOptions(SettingItemOptionsParams{
		label:        "Log level",
		hint:         "Set current log level",
		options:      a.u.Settings().LogLevelNames(),
		defaultValue: a.u.Settings().LogLevelDefault(),
		getter:       a.u.Settings().LogLevel,
		setter: func(v string) {
			s := a.u.Settings()
			s.SetLogLevel(v)
			slog.SetLogLoggerLevel(s.LogLevelSlog())
		},
		isMobile: a.u.IsMobile(),
		window:   a.w,
	})

	developerMode := NewSettingItemSwitch(SettingItemSwitchParams{
		label:  "Developer Mode",
		hint:   "App shows additional technical information like Character IDs",
		getter: a.u.Settings().DeveloperMode,
		onChanged: func(b bool) {
			a.u.Settings().SetDeveloperMode(b)
			a.u.SetDeveloperMode(b)
		},
	})

	items := []SettingItem{
		NewSettingItemHeading("Application"),
		logLevel,
		developerMode,
	}

	sysTray := NewSettingItemSwitch(SettingItemSwitchParams{
		defaultValue: a.u.Settings().SysTrayEnabledDefault(),
		label:        "Run in background",
		hint:         "App will continue to run in background after window is closed (requires restart)",
		getter:       a.u.Settings().SysTrayEnabled,
		onChanged:    a.u.Settings().SetSysTrayEnabled,
	})
	if !a.u.IsMobile() {
		items = append(items, sysTray)
	}

	preferMarketTab := NewSettingItemSwitch(SettingItemSwitchParams{
		label:     "Prefer market tab",
		hint:      "Show market tab first for tradeable items",
		getter:    a.u.Settings().PreferMarketTab,
		onChanged: a.u.Settings().SetPreferMarketTab,
	})
	hideLimitedCorporations := NewSettingItemSwitch(SettingItemSwitchParams{
		defaultValue: a.u.Settings().HideLimitedCorporationsDefault(),
		label:        "Hide limited corporations",
		hint:         "Hide corporations with no privileged access, e.g. corporation wallet",
		getter:       a.u.Settings().HideLimitedCorporations,
		onChanged: func(enabled bool) {
			a.u.Settings().SetHideLimitedCorporations(enabled)
			go a.u.Signals().CorporationsChanged.Emit(context.Background(), struct{}{})
		},
	})

	items = slices.Concat(items, []SettingItem{
		NewSettingItemHeading("UI"),
		preferMarketTab,
		hideLimitedCorporations,
	})

	colorTheme := NewSettingItemOptions(SettingItemOptionsParams{
		label:        "Appearance",
		hint:         "Choose the color scheme. 'Auto' uses the current OS theme.",
		options:      []string{string(settings.Auto), string(settings.Light), string(settings.Dark)},
		defaultValue: string(a.u.Settings().ColorThemeDefault()),
		getter: func() string {
			return string(a.u.Settings().ColorTheme())
		},
		setter: func(v string) {
			s := a.u.Settings()
			s.SetColorTheme(settings.ColorTheme(v))
			a.u.SetColorTheme(settings.ColorTheme(v))
		},
		isMobile: a.u.IsMobile(),
		window:   a.w,
	})

	fyneScale := NewSettingItemSlider(SettingItemSliderParams{
		label:        "UI Scale",
		hint:         "Scaling factor of the user interface in percent. Requires restart.",
		minValue:     50,
		maxValue:     200,
		defaultValue: a.u.Settings().FyneScaleDefault() * 100,
		step:         5,
		getter: func() float64 {
			return a.u.Settings().FyneScale() * 100
		},
		setter: func(v float64) {
			a.u.Settings().SetFyneScale(v / 100.0)
		},
		formatter: func(v any) string {
			return fmt.Sprintf("%v %%", v)
		},
		isMobile: a.u.IsMobile(),
		window:   a.w,
	})

	disableDPIDetection := NewSettingItemSwitch(SettingItemSwitchParams{
		label:     "Disable DPI detection",
		hint:      "Disables the automatic DPI detection. Requires restart.",
		getter:    a.u.Settings().DisableDPIDetection,
		onChanged: a.u.Settings().SetDisableDPIDetection,
	})

	if !a.u.IsMobile() {
		items = slices.Concat(items, []SettingItem{
			colorTheme,
			fyneScale,
			disableDPIDetection,
		})
	}

	vMin, vMax, vDef := a.u.Settings().MaxMailsPresets()
	maxMail := NewSettingItemSlider(SettingItemSliderParams{
		label:        "Maximum mails",
		hint:         "Max number of mails downloaded. 0 = unlimited.",
		minValue:     float64(vMin),
		maxValue:     float64(vMax),
		defaultValue: float64(vDef),
		step:         1,
		getter: func() float64 {
			return float64(a.u.Settings().MaxMails())
		},
		setter: func(v float64) {
			a.u.Settings().SetMaxMails(int(v))
		},
		isMobile: a.u.IsMobile(),
		window:   a.w,
	})

	vMin, vMax, vDef = a.u.Settings().MaxWalletTransactionsPresets()
	maxWallet := NewSettingItemSlider(SettingItemSliderParams{
		label:        "Maximum wallet transaction",
		hint:         "Max wallet transactions downloaded. 0 = unlimited.",
		minValue:     float64(vMin),
		maxValue:     float64(vMax),
		defaultValue: float64(vDef),
		step:         1,
		getter: func() float64 {
			return float64(a.u.Settings().MaxWalletTransactions())
		},
		setter: func(v float64) {
			a.u.Settings().SetMaxWalletTransactions(int(v))
		},
		isMobile: a.u.IsMobile(),
		window:   a.w,
	})

	vMin, vMax, vDef = a.u.Settings().MarketOrderRetentionDaysPresets()
	marketOrdersRetention := NewSettingItemSlider(SettingItemSliderParams{
		label:        "Market order retention",
		hint:         "Number of days to keep historic market orders.",
		minValue:     float64(vMin),
		maxValue:     float64(vMax),
		defaultValue: float64(vDef),
		step:         1,
		getter: func() float64 {
			return float64(a.u.Settings().MarketOrderRetentionDays())
		},
		setter: func(v float64) {
			a.u.Settings().SetMarketOrdersRetentionDay(int(v))
		},
		isMobile: a.u.IsMobile(),
		window:   a.w,
	})

	items = slices.Concat(items, []SettingItem{
		NewSettingItemHeading("Section updates"),
		maxMail,
		maxWallet,
		marketOrdersRetention,
	})

	list := newSettingList(items)

	clear := settingAction{
		Label: "Clear cache",
		Action: func() {
			w := a.w
			xdialog.ShowConfirm(
				"Clear Cache",
				"Are you sure you want to clear the cache?",
				"Clear",
				func(confirmed bool) {
					if !confirmed {
						return
					}
					m := kxmodal.NewProgressInfinite(
						"Clearing cache...",
						"",
						func() error {
							a.u.ClearAllCaches()
							return nil
						},
						w,
					)
					m.OnSuccess = func() {
						slog.Info("Cleared cache")
						a.sb.Show("Cache cleared")
					}
					m.OnError = func(err error) {
						slog.Error("Failed to clear cache", "error", err)
						a.sb.Show(fmt.Sprintf("Failed to clear cache: %s", a.u.ErrorDisplay(err)))
					}
					m.Start()
				}, w,
			)
		},
	}
	reset := settingAction{
		Label: "Reset to defaults",
		Action: func() {
			developerMode.Reset()
			logLevel.Reset()
			sysTray.Reset()
			list.Refresh()
			colorTheme.Reset()
			disableDPIDetection.Reset()
			fyneScale.Reset()
			preferMarketTab.Reset()
			hideLimitedCorporations.Reset()
			maxMail.Reset()
			maxWallet.Reset()
		},
	}
	exportAppLog := settingAction{
		Label: "Export application log",
		Action: func() {
			a.showExportFileDialog(a.u.DataPaths()["log"])
		},
	}
	exportCrashLog := settingAction{
		Label: "Export crash log",
		Action: func() {
			a.showExportFileDialog(a.u.DataPaths()["crashfile"])
		},
	}
	deleteAppLog := settingAction{
		Label: "Delete application log",
		Action: func() {
			a.showDeleteFileDialog("application log", a.u.DataPaths()["log"]+"*")
		},
	}
	deleteCrashLog := settingAction{
		Label: "Delete crash log",
		Action: func() {
			a.showDeleteFileDialog("crash log", a.u.DataPaths()["crashfile"])
		},
	}
	actions := []settingAction{reset, clear, exportAppLog, exportCrashLog, deleteAppLog, deleteCrashLog}
	if !a.u.IsMobile() {
		actions = append(actions, settingAction{
			Label: "Resets main window size to defaults",
			Action: func() {
				a.u.Settings().ResetWindowSize()
				a.u.MainWindow().Resize(a.u.Settings().WindowSize())
			},
		})
	}
	if a.u.IsDeveloperMode() {
		actions = append(actions, settingAction{
			Label: "Show snackbar (debug)",
			Action: func() {
				a.sb.Show("This is a test")
			},
		})
		actions = append(actions, settingAction{
			Label: "Reset shown character (debug)",
			Action: func() {
				go a.u.ResetCharacter(context.Background())
			},
		})
		actions = append(actions, settingAction{
			Label: "Reset shown corporation (debug)",
			Action: func() {
				go a.u.ResetCorporation(context.Background())
			},
		})
	}
	return list, makeIconButtonFromActions(actions)
}

func (a *settingsWindow) showDeleteFileDialog(name, path string) {
	xdialog.ShowConfirm(
		"Delete File",
		fmt.Sprintf("Are you sure you want to permanently delete this file?\n\n%s", name),
		"Delete",
		func(confirmed bool) {
			if !confirmed {
				return
			}
			err := func() error {
				files, err := filepath.Glob(path)
				if err != nil {
					return err
				}
				for _, f := range files {
					if err := os.Truncate(f, 0); err != nil {
						return err
					}
				}
				return nil
			}()
			if err != nil {
				slog.Error("delete "+name, "path", path, "error", err)
				a.sb.Show("ERROR: Failed to delete " + name)
			} else {
				a.sb.Show(xstrings.Title(name) + " deleted")
			}
		}, a.w)
}

func (a *settingsWindow) showExportFileDialog(path string) {
	filename := filepath.Base(path)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		a.sb.Show("No file to export: " + filename)
		return
	} else if err != nil {
		xdialog.ShowErrorAndLog("Failed to open "+filename, err, a.u.IsDeveloperMode(), a.w)
		return
	}
	d := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			err2 := func() error {
				if err != nil {
					return err
				}
				if writer == nil {
					return nil
				}
				defer writer.Close()
				if _, err := writer.Write(data); err != nil {
					return err
				}
				a.sb.Show("File " + filename + " exported")
				return nil
			}()
			if err2 != nil {
				xdialog.ShowErrorAndLog("Failed to export "+filename, err, a.u.IsDeveloperMode(), a.w)
			}
		}, a.w,
	)
	d.SetFileName(filename)
	xdesktop.DisableShortcutsForDialog(d, a.w)
	d.Show()
}

func (a *settingsWindow) makeNotificationPage() (fyne.CanvasObject, *kxwidget.IconButton) {
	groupsAndTypes := make(map[app.EveNotificationGroup][]app.EveNotificationType)
	for n := range app.NotificationTypesSupported().All() {
		g := n.Group()
		groupsAndTypes[g] = append(groupsAndTypes[g], n)
	}
	var groups []app.EveNotificationGroup
	for c := range groupsAndTypes {
		groups = append(groups, c)
	}
	for _, g := range groups {
		slices.Sort(groupsAndTypes[g])
	}
	slices.Sort(groups)
	typesEnabled := a.u.Settings().NotificationTypesEnabled()

	// add global items
	notifyCommunications := NewSettingItemSwitch(SettingItemSwitchParams{
		defaultValue: a.u.Settings().NotifyCommunicationsEnabledDefault(),
		label:        "Notify communications",
		hint:         "Whether to notify new communications",
		getter:       a.u.Settings().NotifyCommunicationsEnabled,
		onChanged:    a.u.Settings().SetNotifyCommunicationsEnabled,
	})
	notifyMails := NewSettingItemSwitch(SettingItemSwitchParams{
		defaultValue: a.u.Settings().NotifyMailsEnabledDefault(),
		label:        "Notify mails",
		hint:         "Whether to notify new mails",
		getter:       a.u.Settings().NotifyMailsEnabled,
		onChanged: func(on bool) {
			a.u.Settings().SetNotifyMailsEnabled(on)
			if on {
				a.u.Settings().SetNotifyMailsEarliest(time.Now())
			}
		},
	})
	notifyPI := NewSettingItemSwitch(SettingItemSwitchParams{
		defaultValue: a.u.Settings().NotifyPIEnabled(),
		label:        "Planetary Industry",
		hint:         "Whether to notify about expired extractions",
		getter:       a.u.Settings().NotifyPIEnabled,
		onChanged: func(on bool) {
			a.u.Settings().SetNotifyPIEnabled(on)
			if on {
				a.u.Settings().SetNotifyPIEarliest(time.Now())
			}
		},
	})

	notifyTraining := NewSettingItemSwitch(SettingItemSwitchParams{
		defaultValue: a.u.Settings().NotifyTrainingEnabled(),
		label:        "Notify Training",
		hint:         "Whether to notify when skillqueue is empty for watched characters",
		getter:       a.u.Settings().NotifyTrainingEnabled,
		onChanged: func(on bool) {
			a.u.Settings().SetNotifyTrainingEnabled(on)
		},
	})

	notifyContracts := NewSettingItemSwitch(SettingItemSwitchParams{
		defaultValue: a.u.Settings().NotifyContractsEnabledDefault(),
		label:        "Notify Contracts",
		hint:         "Whether to notify when contract status changes",
		getter:       a.u.Settings().NotifyContractsEnabled,
		onChanged: func(on bool) {
			a.u.Settings().SetNotifyContractsEnabled(on)
			if on {
				a.u.Settings().SetNotifyContractsEarliest(time.Now())
			}
		},
	})

	vMin, vMax, vDef := a.u.Settings().NotifyTimeoutHoursPresets()
	notifTimeout := NewSettingItemSlider(SettingItemSliderParams{
		label:        "Notify Timeout",
		hint:         "Events older then this value in hours will not be notified",
		minValue:     float64(vMin),
		maxValue:     float64(vMax),
		defaultValue: float64(vDef),
		step:         1.0,
		getter: func() float64 {
			return float64(a.u.Settings().NotifyTimeoutHours())
		},
		setter: func(v float64) {
			a.u.Settings().SetNotifyTimeoutHours(int(v))
		},
		isMobile: a.u.IsMobile(),
		window:   a.w,
	})

	items := []SettingItem{
		NewSettingItemHeading("Global"),
		notifyCommunications,
		notifyMails,
		notifyPI,
		notifyTraining,
		notifyContracts,
		notifTimeout,
	}
	items = append(items, NewSettingItemHeading("Communication Groups"))

	// add communication groups
	const groupHint = "Choose which communications to notify about"
	type groupPage struct {
		content fyne.CanvasObject
		actions []settingAction
	}
	groupPages := make(map[app.EveNotificationGroup]groupPage) // for pre-constructing group pages
	for _, g := range groups {
		groupPages[g] = func() groupPage {
			var items2 []SettingItem
			for _, nt := range groupsAndTypes[g] {
				ntStr := nt.String()
				ntDisplay := nt.Display()
				it := NewSettingItemSwitch(SettingItemSwitchParams{
					label: ntDisplay,
					hint:  "",
					getter: func() bool {
						return typesEnabled.Contains(ntStr)
					},
					onChanged: func(on bool) {
						if on {
							typesEnabled.Add(ntStr)
						} else {
							typesEnabled.Delete(ntStr)
						}
						a.u.Settings().SetNotificationTypesEnabled(typesEnabled)
					},
				})
				items2 = append(items2, it)
			}
			list2 := newSettingList(items2)
			enableAll := settingAction{
				Label: "Enable all",
				Action: func() {
					for _, it := range items2 {
						it.Setter(true)
					}
					list2.Refresh()
				},
			}
			disableAll := settingAction{
				Label: "Disable all",
				Action: func() {
					for _, it := range items2 {
						it.Setter(false)
					}
					list2.Refresh()
				},
			}
			return groupPage{
				content: list2,
				actions: []settingAction{enableAll, disableAll},
			}
		}()

		it := NewSettingItemCustom(SettingItemCustomParams{
			label: g.String(),
			hint:  groupHint,
			getter: func() any {
				var enabled int
				for _, nt := range groupsAndTypes[g] {
					if typesEnabled.Contains(nt.String()) {
						enabled++
					}
				}
				if total := len(groupsAndTypes[g]); total == enabled {
					return "All"
				} else if enabled > 0 {
					return "Some"
				}
				return "Off"
			},
			onSelected: func(it SettingItem, refresh func()) {
				p := groupPages[g]
				title := g.String()
				hint := widget.NewLabel(groupHint)
				hint.SizeName = theme.SizeNameCaptionText
				var d dialog.Dialog
				buttons := container.NewHBox(
					widget.NewButton("OK", func() {
						d.Hide()
					}),
					layout.NewSpacer(),
				)
				for _, a := range p.actions {
					buttons.Add(widget.NewButton(a.Label, a.Action))
				}
				c := container.NewBorder(nil, container.NewVBox(hint, buttons), nil, nil, p.content)
				w := a.w
				d = dialog.NewCustomWithoutButtons(title, c, w)
				xdesktop.DisableShortcutsForDialog(d, w)
				d.Show()
				_, s := w.Canvas().InteractiveArea()
				d.Resize(fyne.NewSize(s.Width*0.8, s.Height*0.8))
				d.SetOnClosed(refresh)
			},
		})
		items = append(items, it)
	}

	list := newSettingList(items)
	reset := settingAction{
		Label: "Reset to defaults",
		Action: func() {
			notifyCommunications.Reset()
			notifyContracts.Reset()
			notifyPI.Reset()
			notifyTraining.Reset()
			notifyMails.Reset()
			notifTimeout.Reset()
			typesEnabled.Clear()
			a.u.Settings().ResetNotificationTypesEnabled()
			list.Refresh()
		},
	}
	updateTypes := func() {
		a.u.Settings().SetNotificationTypesEnabled(typesEnabled)
		list.Refresh()
	}
	none := settingAction{
		Label: "Disable all communication groups",
		Action: func() {
			typesEnabled.Clear()
			updateTypes()
		},
	}
	all := settingAction{
		Label: "Enable all communication groups",
		Action: func() {
			for nt := range app.NotificationTypesSupported().All() {
				typesEnabled.Add(nt.String())
			}
			updateTypes()
		},
	}
	send := settingAction{
		Label: "Send test notification",
		Action: func() {
			go func() {
				n := fyne.NewNotification("Test", "This is a test notification from EVE Buddy.")
				fyne.CurrentApp().SendNotification(n)
			}()
		},
	}
	return list, makeIconButtonFromActions([]settingAction{reset, all, none, send})
}

// func (a *userSettings) reportError(text string, err error) {
// 	slog.Error(text, "error", err)
// 	a.sb.Show(fmt.Sprintf("ERROR: %s: %s", text, err))
// }

func makeIconButtonFromActions(actions []settingAction) *kxwidget.IconButton {
	var items []*fyne.MenuItem
	for _, a := range actions {
		items = append(items, fyne.NewMenuItem(a.Label, a.Action))
	}
	return kxwidget.NewIconButtonWithMenu(
		theme.MoreHorizontalIcon(),
		fyne.NewMenu("", items...),
	)
}

// relative size of dialog window to current window
const (
	dialogWidthScale = 0.8 // except on mobile it is always 100%
	dialogHeightMin  = 100
)

type settingVariant uint

const (
	settingUndefined settingVariant = iota
	settingCustom
	settingHeading
	settingSwitch
)

// SettingItem represents an item in a setting list.
type SettingItem struct {
	Default   any
	Hint      string             // optional hint text
	Label     string             // label
	Getter    func() any         // returns the current value for this setting
	Setter    func(v any)        // sets the value for this setting
	Formatter func(v any) string // func to format the value

	onSelected func(it SettingItem, refresh func()) // action called when selected
	variant    settingVariant                       // the setting variant of this item
}

func (si SettingItem) Reset() {
	if si.Setter == nil || si.Default == nil {
		return
	}
	if si.Getter() == si.Default {
		return
	}
	si.Setter(si.Default)
}

// NewSettingItemHeading creates a heading in a setting list.
func NewSettingItemHeading(label string) SettingItem {
	return SettingItem{Label: label, variant: settingHeading}
}

type SettingItemSwitchParams struct {
	defaultValue bool
	getter       func() bool
	hint         string
	label        string
	onChanged    func(bool)
}

// NewSettingItemSwitch creates a switch setting in a setting list.
func NewSettingItemSwitch(arg SettingItemSwitchParams) SettingItem {
	return SettingItem{
		Default: arg.defaultValue,
		Label:   arg.label,
		Hint:    arg.hint,
		Getter: func() any {
			return arg.getter()
		},
		Setter: func(v any) {
			arg.onChanged(v.(bool))
		},
		onSelected: func(it SettingItem, refresh func()) {
			it.Setter(!it.Getter().(bool))
			refresh()
		},
		variant: settingSwitch,
	}
}

type SettingItemCustomParams struct {
	label      string
	hint       string
	getter     func() any
	onSelected func(it SettingItem, refresh func())
}

// NewSettingItemCustom creates a custom setting in a setting list.
func NewSettingItemCustom(arg SettingItemCustomParams) SettingItem {
	return SettingItem{
		Label:      arg.label,
		Hint:       arg.hint,
		Getter:     arg.getter,
		onSelected: arg.onSelected,
		variant:    settingCustom,
	}
}

type SettingItemSliderParams struct {
	defaultValue float64
	formatter    func(v any) string
	getter       func() float64
	hint         string
	isMobile     bool
	label        string
	maxValue     float64
	minValue     float64
	setter       func(v float64)
	step         float64
	window       fyne.Window
}

func NewSettingItemSlider(arg SettingItemSliderParams) SettingItem {
	return SettingItem{
		Default: arg.defaultValue,
		Label:   arg.label,
		Hint:    arg.hint,
		Getter: func() any {
			return arg.getter()
		},
		Formatter: arg.formatter,
		Setter: func(v any) {
			switch x := v.(type) {
			case float64:
				arg.setter(x)
			case int:
				arg.setter(float64(x))
			default:
				panic("setting item: unsupported type: " + arg.label)
			}
		},
		onSelected: func(it SettingItem, refresh func()) {
			sl := kxwidget.NewSlider(arg.minValue, arg.maxValue)
			sl.SetValue(arg.getter())
			sl.SetStep(arg.step)
			sl.OnChangeEnded = arg.setter
			d := makeSettingDialog(makeSettingDialogParams{
				setting:  sl,
				label:    it.Label,
				hint:     it.Hint,
				isMobile: arg.isMobile,
				reset: func() {
					sl.SetValue(arg.defaultValue)
				},
				refresh: refresh,
				window:  arg.window,
			})
			d.Show()
		},
		variant: settingCustom,
	}
}

type SettingItemOptionsParams struct {
	defaultValue string
	getter       func() string
	hint         string
	isMobile     bool
	label        string
	options      []string
	setter       func(v string)
	window       fyne.Window
}

func NewSettingItemOptions(arg SettingItemOptionsParams) SettingItem {
	return SettingItem{
		Default: arg.defaultValue,
		Label:   arg.label,
		Hint:    arg.hint,
		Getter: func() any {
			return arg.getter()
		},
		Setter: func(v any) {
			arg.setter(v.(string))
		},
		onSelected: func(it SettingItem, refresh func()) {
			sel := widget.NewRadioGroup(arg.options, arg.setter)
			sel.Required = true
			sel.Selected = it.Getter().(string)
			d := makeSettingDialog(makeSettingDialogParams{
				setting: sel,
				label:   it.Label,
				hint:    it.Hint,
				reset: func() {
					sel.SetSelected(arg.defaultValue)
				},
				isMobile: arg.isMobile,
				refresh:  refresh,
				window:   arg.window,
			})
			d.Show()
		},
		variant:   settingCustom,
		Formatter: nil,
	}
}

type makeSettingDialogParams struct {
	hint     string
	isMobile bool
	label    string
	refresh  func()
	reset    func()
	setting  fyne.CanvasObject
	window   fyne.Window
}

func makeSettingDialog(arg makeSettingDialogParams) dialog.Dialog {
	var d dialog.Dialog
	buttons := container.NewHBox(
		widget.NewButton("OK", func() {
			d.Hide()
		}),
		layout.NewSpacer(),
		widget.NewButton("Reset", func() {
			arg.reset()
		}),
	)
	l := widget.NewLabel(arg.hint)
	l.SizeName = theme.SizeNameCaptionText
	c := container.NewBorder(
		nil,
		container.NewVBox(l, buttons),
		nil,
		nil,
		arg.setting,
	)
	// TODO: add modify shortcuts
	d = dialog.NewCustomWithoutButtons(arg.label, c, arg.window)
	_, s := arg.window.Canvas().InteractiveArea()
	var width float32
	if arg.isMobile {
		width = s.Width
	} else {
		width = s.Width * dialogWidthScale
	}
	d.Resize(fyne.NewSize(width, dialogHeightMin))
	d.SetOnClosed(arg.refresh)
	return d
}

// settingList is a custom list widget for settings.
type settingList struct {
	widget.List
}

// newSettingList returns a new SettingList widget.
func newSettingList(items []SettingItem) *settingList {
	w := &settingList{}
	w.ExtendBaseWidget(w)
	w.Length = func() int {
		return len(items)
	}
	w.CreateItem = func() fyne.CanvasObject {
		return newSettingListItem()
	}
	w.UpdateItem = func(id widget.ListItemID, co fyne.CanvasObject) {
		if id >= len(items) {
			return
		}
		li := co.(*settingListItem)
		li.set(items[id])
		w.SetItemHeight(id, li.MinSize().Height)
	}
	w.OnSelected = func(id widget.ListItemID) {
		defer w.UnselectAll()
		if id >= len(items) {
			return
		}
		it := items[id]
		if it.onSelected == nil {
			return
		}
		it.onSelected(it, func() {
			w.RefreshItem(id)
		})
	}
	w.HideSeparators = true
	return w
}

type settingListItem struct {
	widget.BaseWidget

	background *canvas.Rectangle
	hint       *widget.Label
	label      *widget.Label
	header     *widget.Label
	switch_    *kxwidget.Switch
	value      *widget.Label
	thief      *xwidget.HooverThief
}

func newSettingListItem() *settingListItem {
	label := widget.NewLabel("Template")
	label.Truncation = fyne.TextTruncateClip
	header := widget.NewLabel("Template")
	header.Truncation = fyne.TextTruncateClip
	header.TextStyle.Bold = true
	hint := widget.NewLabel("")
	hint.Truncation = fyne.TextTruncateClip
	hint.SizeName = theme.SizeNameCaptionText
	background := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	background.CornerRadius = 10
	w := &settingListItem{
		background: background,
		header:     header,
		hint:       hint,
		label:      label,
		switch_:    kxwidget.NewSwitch(nil),
		thief:      xwidget.NewHooverThief(),
		value:      widget.NewLabel(""),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *settingListItem) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSquareSize(theme.Padding()))
	c := container.NewBorder(
		nil,
		container.NewVBox(
			spacer,
			container.New(layout.NewCustomPaddedLayout(0, 0, 0, -2*p), w.header),
		),
		nil,
		container.NewVBox(
			layout.NewSpacer(),
			container.NewStack(w.switch_, w.value),
			layout.NewSpacer(),
		),
		container.New(layout.NewCustomPaddedVBoxLayout(0),
			layout.NewSpacer(),
			w.label,
			w.hint,
			layout.NewSpacer(),
		),
	)
	c2 := container.NewStack(
		container.New(layout.NewCustomPaddedLayout(1, 1, p, p), container.NewStack(w.background, c)),
		w.thief,
	)
	return widget.NewSimpleRenderer(c2)
}

func (w *settingListItem) set(r SettingItem) {
	if r.Hint != "" {
		w.hint.SetText(r.Hint)
		w.hint.Show()
	} else {
		w.hint.Hide()
	}
	switch r.variant {
	case settingHeading:
		w.header.SetText(r.Label)
		w.header.Show()
		w.label.Hide()
		w.value.Hide()
		w.switch_.Hide()
		w.background.Hide()
		w.thief.Show()
	case settingSwitch:
		w.label.SetText(r.Label)
		w.header.Hide()
		w.label.Show()
		w.value.Hide()
		w.switch_.OnChanged = func(v bool) {
			r.Setter(v)
		}
		w.switch_.On = r.Getter().(bool)
		w.switch_.Show()
		w.switch_.Refresh()
		w.background.Show()
		w.thief.Hide()
	case settingCustom:
		w.label.SetText(r.Label)
		w.header.Hide()
		w.label.Show()
		formatter := r.Formatter
		if formatter == nil {
			formatter = func(v any) string {
				return fmt.Sprint(v)
			}
		}
		w.value.SetText(formatter(r.Getter()))
		w.value.Show()
		w.switch_.Hide()
		w.background.Show()
		w.thief.Hide()
	}
}
