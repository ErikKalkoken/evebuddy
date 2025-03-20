package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// TODO: Add settings API to allow this widget to be moved into own package
// TODO: Improve switch API to allow switch not to be set on error

// Exported settings
const (
	SettingLogLevel              = "logLevel"
	SettingLogLevelDefault       = "info"
	SettingSysTrayEnabled        = "settingSysTrayEnabled"
	SettingSysTrayEnabledDefault = false
	SettingTabsMainID            = "tabs-main-id"
	SettingWindowHeight          = "window-height"
	SettingWindowHeightDefault   = 600
	SettingWindowWidth           = "window-width"
	SettingWindowWidthDefault    = 1000
)

// Local settings
const (
	settingLastCharacterID                    = "settingLastCharacterID"
	settingMaxMails                           = "settingMaxMails"
	settingMaxMailsDefault                    = 1_000
	settingMaxMailsMax                        = 10_000
	settingMaxWalletTransactions              = "settingMaxWalletTransactions"
	settingMaxWalletTransactionsDefault       = 1_000
	settingMaxWalletTransactionsMax           = 10_000
	settingNotificationsTypesEnabled          = "settingNotificationsTypesEnabled"
	settingNotifyCommunicationsEarliest       = "settingNotifyCommunicationsEarliest"
	settingNotifyCommunicationsEnabled        = "settingNotifyCommunicationsEnabled"
	settingNotifyCommunicationsEnabledDefault = false
	settingNotifyContractsEarliest            = "settingNotifyContractsEarliest"
	settingNotifyContractsEnabled             = "settingNotifyContractsEnabled"
	settingNotifyContractsEnabledDefault      = false
	settingNotifyMailsEarliest                = "settingNotifyMailsEarliest"
	settingNotifyMailsEnabled                 = "settingNotifyMailsEnabled"
	settingNotifyMailsEnabledDefault          = false
	settingNotifyPIEarliest                   = "settingNotifyPIEarliest"
	settingNotifyPIEnabled                    = "settingNotifyPIEnabled"
	settingNotifyPIEnabledDefault             = false
	settingNotifyTimeoutHours                 = "settingNotifyTimeoutHours"
	settingNotifyTimeoutHoursDefault          = 30 * 24
	settingNotifyTimeoutHoursMax              = 90 * 24
	settingNotifyTrainingEarliest             = "settingNotifyTrainingEarliest"
	settingNotifyTrainingEnabled              = "settingNotifyTrainingEnabled"
	settingNotifyTrainingEnabledDefault       = false
	settingDeveloperMode                      = "developer-mode"
)

// SettingKeys returns all setting keys. Mostly to know what to delete.
func SettingKeys() []string {
	return []string{
		settingLastCharacterID,
		settingMaxMails,
		settingMaxWalletTransactions,
		settingNotificationsTypesEnabled,
		settingNotifyCommunicationsEarliest,
		settingNotifyCommunicationsEnabled,
		settingNotifyContractsEarliest,
		settingNotifyContractsEnabled,
		settingNotifyMailsEarliest,
		settingNotifyMailsEnabled,
		settingNotifyPIEarliest,
		settingNotifyPIEnabled,
		settingNotifyTimeoutHours,
		settingNotifyTrainingEarliest,
		settingNotifyTrainingEnabled,
		SettingSysTrayEnabled,
		SettingTabsMainID,
		SettingWindowHeight,
		SettingWindowWidth,
	}
}

var logLevelName2Level = map[string]slog.Level{
	"debug":   slog.LevelDebug,
	"error":   slog.LevelError,
	"info":    slog.LevelInfo,
	"warning": slog.LevelWarn,
}

func LogLevelName2Level(s string) slog.Level {
	l, ok := logLevelName2Level[s]
	if !ok {
		l = slog.LevelInfo
	}
	return l
}

func LogLevelNames() []string {
	x := slices.Collect(maps.Keys(logLevelName2Level))
	slices.Sort(x)
	return x
}

type SettingAction struct {
	Label  string
	Action func()
}

type Settings struct {
	widget.BaseWidget

	NotificationActions          []SettingAction
	NotificationSettings         fyne.CanvasObject // TODO: Refactor into widget
	GeneralActions               []SettingAction
	GeneralContent               fyne.CanvasObject // TODO: Refactor into widget
	CommunicationGroupContent    fyne.CanvasObject // TODO: Refactor into widget
	OnCommunicationGroupSelected func(title string, content fyne.CanvasObject, actions []SettingAction)

	showSnackbar func(string)
	snackbar     *iwidget.Snackbar
	u            app.UI
	window       fyne.Window
}

func NewSettings(u app.UI) *Settings {
	a := &Settings{
		showSnackbar: u.ShowSnackbar,
		u:            u,
		window:       u.MainWindow(),
	}
	a.ExtendBaseWidget(a)
	a.GeneralContent, a.GeneralActions = a.makeGeneralSettingsPage()
	a.NotificationSettings, a.NotificationActions = a.makeNotificationPage()
	return a
}

func (a *Settings) CreateRenderer() fyne.WidgetRenderer {
	makeSettingsPage := func(title string, content fyne.CanvasObject, actions []SettingAction) fyne.CanvasObject {
		t := widget.NewLabel(title)
		t.TextStyle.Bold = true
		items := make([]*fyne.MenuItem, 0)
		for _, a := range actions {
			items = append(items, fyne.NewMenuItem(a.Label, a.Action))
		}
		options := iwidget.NewContextMenuButtonWithIcon(theme.MoreHorizontalIcon(), "", fyne.NewMenu("", items...))
		return container.NewBorder(
			container.NewVBox(container.NewHBox(t, layout.NewSpacer(), options), widget.NewSeparator()),
			nil,
			nil,
			nil,
			container.NewScroll(content),
		)
	}
	tabs := container.NewAppTabs(
		container.NewTabItem("General", makeSettingsPage("General", a.GeneralContent, a.GeneralActions)),
		container.NewTabItem("Notifications", makeSettingsPage("Notifications", a.NotificationSettings, a.NotificationActions)),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	return widget.NewSimpleRenderer(tabs)
}

func (a *Settings) SetWindow(w fyne.Window) {
	a.window = w
	if a.snackbar != nil {
		a.snackbar.Stop()
	}
	a.snackbar = iwidget.NewSnackbar(w)
	a.snackbar.Start()
	a.showSnackbar = func(s string) {
		a.snackbar.Show(s)
	}
}

func (a *Settings) currentWindow() fyne.Window {
	return a.window
}

func (a *Settings) makeGeneralSettingsPage() (fyne.CanvasObject, []SettingAction) {
	logLevel := iwidget.NewSettingItemOptions(
		"Log level",
		"Set current log level",
		LogLevelNames(),
		SettingLogLevelDefault,
		func() string {
			return a.u.App().Preferences().StringWithFallback(
				SettingLogLevel,
				SettingLogLevelDefault,
			)
		},
		func(s string) {
			a.u.App().Preferences().SetString(SettingLogLevel, s)
			slog.SetLogLoggerLevel(LogLevelName2Level(s))
		},
		a.currentWindow,
	)
	maxMail := iwidget.NewSettingItemSlider(
		"Maximum mails",
		"Max number of mails downloaded. 0 = unlimited.",
		0,
		settingMaxMailsMax,
		settingMaxMailsDefault,
		func() float64 {
			return float64(a.u.App().Preferences().IntWithFallback(
				settingMaxMails,
				settingMaxMailsDefault))
		},
		func(v float64) {
			a.u.App().Preferences().SetInt(settingMaxMails, int(v))
		},
		a.currentWindow,
	)
	maxWallet := iwidget.NewSettingItemSlider(
		"Maximum wallet transaction",
		"Max wallet transactions downloaded. 0 = unlimited.",
		0,
		settingMaxWalletTransactionsMax,
		settingMaxWalletTransactionsDefault,
		func() float64 {
			return float64(a.u.App().Preferences().IntWithFallback(
				settingMaxWalletTransactions,
				settingMaxWalletTransactionsDefault))
		},
		func(v float64) {
			a.u.App().Preferences().SetInt(settingMaxWalletTransactions, int(v))
		},
		a.currentWindow,
	)
	developerMode := iwidget.NewSettingItemSwitch(
		"Developer Mode",
		"App shows addditional technical information like Character IDs",
		func() bool {
			return a.u.App().Preferences().Bool(settingDeveloperMode)
		},
		func(b bool) {
			a.u.App().Preferences().SetBool(settingDeveloperMode, b)
		},
	)

	items := []iwidget.SettingItem{
		iwidget.NewSettingItemHeading("Application"),
		logLevel,
		developerMode,
		iwidget.NewSettingItemSeperator(),
		iwidget.NewSettingItemHeading("EVE Online"),
		maxMail,
		maxWallet,
	}

	systray := iwidget.NewSettingItemSwitch(
		"Close button",
		"App will minimize to system tray when closed (requires restart)",
		func() bool {
			return a.u.App().Preferences().BoolWithFallback(
				SettingSysTrayEnabled,
				SettingSysTrayEnabledDefault,
			)
		},
		func(b bool) {
			a.u.App().Preferences().SetBool(SettingSysTrayEnabled, b)
		},
	)
	if a.u.IsDesktop() {
		items = slices.Insert(items, 2, systray)
	}

	list := iwidget.NewSettingList(items)

	clear := SettingAction{
		"Clear cache",
		func() {
			w := a.currentWindow()
			a.u.ShowConfirmDialog(
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
						a.u.ShowSnackbar("Cache cleared")
					}
					m.OnError = func(err error) {
						slog.Error("Failed to clear cache", "error", err)
						a.u.ShowSnackbar(fmt.Sprintf("Failed to clear cache: %s", humanize.Error(err)))
					}
					m.Start()
				}, w)
		}}
	reset := SettingAction{
		Label: "Reset to defaults",
		Action: func() {
			logLevel.Setter(SettingLogLevelDefault)
			maxMail.Setter(settingMaxMailsDefault)
			maxWallet.Setter(settingMaxWalletTransactionsDefault)
			systray.Setter(SettingSysTrayEnabledDefault)
			developerMode.Setter(false)
			list.Refresh()
		},
	}
	exportAppLog := SettingAction{
		Label: "Export application log",
		Action: func() {
			a.showExportFileDialog(a.u.DataPaths()["log"])
		},
	}
	exportCrashLog := SettingAction{
		Label: "Export crash log",
		Action: func() {
			a.showExportFileDialog(a.u.DataPaths()["crashfile"])
		},
	}
	deleteAppLog := SettingAction{
		Label: "Delete application log",
		Action: func() {
			a.showDeleteFileDialog("application log", a.u.DataPaths()["log"]+"*")
		},
	}
	deleteCrashLog := SettingAction{
		Label: "Delete creash log",
		Action: func() {
			a.showDeleteFileDialog("crash log", a.u.DataPaths()["crashfile"])
		},
	}
	actions := []SettingAction{reset, clear, exportAppLog, exportCrashLog, deleteAppLog, deleteCrashLog}
	if a.u.IsDesktop() {
		actions = append(actions, SettingAction{
			Label: "Resets main window size to defaults",
			Action: func() {
				a.u.MainWindow().Resize(fyne.NewSize(SettingWindowWidthDefault, SettingWindowHeightDefault))
			},
		})
	}
	return list, actions
}

func (a *Settings) showDeleteFileDialog(name, path string) {
	a.u.ShowConfirmDialog(
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
				a.snackbar.Show("ERROR: Failed to delete " + name)
			} else {
				titler := cases.Title(language.English)
				a.snackbar.Show(titler.String(name) + " deleted")
			}
		}, a.window)
}

func (a *Settings) showExportFileDialog(path string) {
	filename := filepath.Base(path)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		a.snackbar.Show("No file to export: " + filename)
		return
	} else if err != nil {
		a.u.ShowErrorDialog("Failed to open "+filename, err, a.window)
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
				a.snackbar.Show("File " + filename + " exported")
				return nil
			}()
			if err2 != nil {
				a.u.ShowErrorDialog("Failed to export "+filename, err, a.window)
			}
		}, a.window,
	)
	d.SetFileName(filename)
	a.u.ModifyShortcutsForDialog(d, a.window)
	d.Show()
}

func (a *Settings) makeNotificationPage() (fyne.CanvasObject, []SettingAction) {
	groupsAndTypes := make(map[app.NotificationGroup][]evenotification.Type)
	for _, n := range evenotification.SupportedGroups() {
		c := evenotification.Type2group[n]
		groupsAndTypes[c] = append(groupsAndTypes[c], n)
	}
	groups := make([]app.NotificationGroup, 0)
	for c := range groupsAndTypes {
		groups = append(groups, c)
	}
	for _, g := range groups {
		slices.Sort(groupsAndTypes[g])
	}
	slices.Sort(groups)
	typesEnabled := set.NewFromSlice(a.u.App().Preferences().StringList(settingNotificationsTypesEnabled))

	// add global items
	notifyCommunications := iwidget.NewSettingItemSwitch(
		"Notify communications",
		"Whether to notify new communications",
		func() bool {
			return a.u.App().Preferences().BoolWithFallback(
				settingNotifyCommunicationsEnabled,
				settingNotifyCommunicationsEnabledDefault,
			)
		},
		func(on bool) {
			a.u.App().Preferences().SetBool(settingNotifyCommunicationsEnabled, on)
			if on {
				a.u.App().Preferences().SetString(
					settingNotifyCommunicationsEarliest,
					time.Now().Format(time.RFC3339))
			}
		},
	)
	notifyMails := iwidget.NewSettingItemSwitch(
		"Notify mails",
		"Whether to notify new mails",
		func() bool {
			return a.u.App().Preferences().BoolWithFallback(
				settingNotifyMailsEnabled,
				settingNotifyMailsEnabledDefault,
			)
		},
		func(on bool) {
			a.u.App().Preferences().SetBool(settingNotifyMailsEnabled, on)
			if on {
				a.u.App().Preferences().SetString(
					settingNotifyMailsEarliest,
					time.Now().Format(time.RFC3339))
			}
		},
	)
	notifyPI := iwidget.NewSettingItemSwitch(
		"Planetary Industry",
		"Whether to notify about expired extractions",
		func() bool {
			return a.u.App().Preferences().BoolWithFallback(
				settingNotifyPIEnabled,
				settingNotifyPIEnabledDefault,
			)
		},
		func(on bool) {
			a.u.App().Preferences().SetBool(settingNotifyPIEnabled, on)
			if on {
				a.u.App().Preferences().SetString(
					settingNotifyPIEarliest,
					time.Now().Format(time.RFC3339))
			}
		},
	)
	notifyTraining := iwidget.NewSettingItemSwitch(
		"Notify Training",
		"Whether to notify abouthen skillqueue is empty",
		func() bool {
			return a.u.App().Preferences().BoolWithFallback(
				settingNotifyTrainingEnabled,
				settingNotifyTrainingEnabledDefault,
			)
		},
		func(on bool) {
			ctx := context.Background()
			if on {
				err := a.u.CharacterService().EnableAllTrainingWatchers(ctx)
				if err != nil {
					a.u.ShowErrorDialog("failed to enable training notification", err, a.currentWindow())
				} else {
					a.u.App().Preferences().SetBool(settingNotifyTrainingEnabled, true)
				}
			} else {
				err := a.u.CharacterService().DisableAllTrainingWatchers(ctx)
				if err != nil {
					a.u.ShowErrorDialog("failed to disable training notification", err, a.currentWindow())
				} else {
					a.u.App().Preferences().SetBool(settingNotifyTrainingEnabled, false)
				}
			}
		},
	)
	notifyContracts := iwidget.NewSettingItemSwitch(
		"Notify Contracts",
		"Whether to notify when contract status changes",
		func() bool {
			return a.u.App().Preferences().BoolWithFallback(
				settingNotifyContractsEnabled,
				settingNotifyCommunicationsEnabledDefault,
			)
		},
		func(on bool) {
			a.u.App().Preferences().SetBool(settingNotifyContractsEnabled, on)
			if on {
				a.u.App().Preferences().SetString(
					settingNotifyContractsEarliest,
					time.Now().Format(time.RFC3339))
			}
		},
	)
	notifTimeout := iwidget.NewSettingItemSlider(
		"Notify Timeout",
		"Events older then this value in hours will not be notified",
		1,
		settingNotifyTimeoutHoursMax,
		settingNotifyTimeoutHoursDefault,
		func() float64 {
			return float64(a.u.App().Preferences().IntWithFallback(
				settingNotifyTimeoutHours,
				settingNotifyTimeoutHoursDefault,
			))
		},
		func(v float64) {
			a.u.App().Preferences().SetInt(settingNotifyTimeoutHours, int(v))
		},
		a.currentWindow,
	)
	items := []iwidget.SettingItem{
		iwidget.NewSettingItemHeading("Global"),
		notifyCommunications,
		notifyMails,
		notifyPI,
		notifyTraining,
		notifyContracts,
		notifTimeout,
	}
	items = append(items, iwidget.NewSettingItemSeperator())
	items = append(items, iwidget.NewSettingItemHeading("Communication Groups"))

	// add communication groups
	const groupHint = "Choose which communications to notfy about"
	type groupPage struct {
		content fyne.CanvasObject
		actions []SettingAction
	}
	groupPages := make(map[app.NotificationGroup]groupPage) // for pre-constructing group pages
	for _, g := range groups {
		groupPages[g] = func() groupPage {
			items2 := make([]iwidget.SettingItem, 0)
			for _, nt := range groupsAndTypes[g] {
				ntStr := nt.String()
				ntDisplay := nt.Display()
				it := iwidget.NewSettingItemSwitch(
					ntDisplay,
					"",
					func() bool {
						return typesEnabled.Contains(ntStr)
					},
					func(on bool) {
						if on {
							typesEnabled.Add(ntStr)
						} else {
							typesEnabled.Remove(ntStr)
						}
						a.u.App().Preferences().SetStringList(
							settingNotificationsTypesEnabled,
							typesEnabled.ToSlice())
					},
				)
				items2 = append(items2, it)
			}
			list2 := iwidget.NewSettingList(items2)
			enableAll := SettingAction{
				Label: "Enable all",
				Action: func() {
					for _, it := range items2 {
						it.Setter(true)
					}
					list2.Refresh()
				},
			}
			disableAll := SettingAction{
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
				actions: []SettingAction{enableAll, disableAll},
			}
		}()

		it := iwidget.NewSettingItemCustom(g.String(), groupHint,
			func() any {
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
			func(it iwidget.SettingItem, refresh func()) {
				p := groupPages[g]
				title := g.String()
				hint := iwidget.NewLabelWithSize(groupHint, theme.SizeNameCaptionText)
				if a.OnCommunicationGroupSelected != nil {
					c := container.NewBorder(hint, nil, nil, nil, p.content)
					a.OnCommunicationGroupSelected(title, c, p.actions)
					return
				}
				var d dialog.Dialog
				buttons := container.NewHBox(
					widget.NewButton("Close", func() {
						d.Hide()
					}),
					layout.NewSpacer(),
				)
				for _, a := range p.actions {
					buttons.Add(widget.NewButton(a.Label, a.Action))
				}
				c := container.NewBorder(nil, container.NewVBox(hint, buttons), nil, nil, p.content)
				w := a.currentWindow()
				d = dialog.NewCustomWithoutButtons(title, c, w)
				a.u.ModifyShortcutsForDialog(d, w)
				d.Show()
				_, s := w.Canvas().InteractiveArea()
				d.Resize(fyne.NewSize(s.Width*0.8, s.Height*0.8))
				d.SetOnClosed(refresh)
			},
		)
		items = append(items, it)
	}

	list := iwidget.NewSettingList(items)
	reset := SettingAction{
		Label: "Reset to defaults",
		Action: func() {
			typesEnabled.Clear()
			notifyCommunications.Setter(settingNotifyCommunicationsEnabledDefault)
			notifyMails.Setter(settingNotifyMailsEnabledDefault)
			notifyPI.Setter(settingNotifyPIEnabledDefault)
			notifyTraining.Setter(settingNotifyTrainingEnabledDefault)
			notifyContracts.Setter(settingNotifyTrainingEnabledDefault)
			notifTimeout.Setter(settingNotifyTimeoutHoursDefault)
			list.Refresh()
		},
	}
	updateTypes := func() {
		a.u.App().Preferences().SetStringList(
			settingNotificationsTypesEnabled,
			typesEnabled.ToSlice(),
		)
		list.Refresh()
	}
	none := SettingAction{
		Label: "Disable all communication groups",
		Action: func() {
			typesEnabled.Clear()
			updateTypes()
		},
	}
	all := SettingAction{
		Label: "Enable all communication groups",
		Action: func() {
			for _, nt := range evenotification.SupportedGroups() {
				typesEnabled.Add(nt.String())
			}
			updateTypes()
		},
	}
	send := SettingAction{
		Label: "Send test notification",
		Action: func() {
			n := fyne.NewNotification("Test", "This is a test notification from EVE Buddy.")
			a.u.App().SendNotification(n)
		},
	}
	return list, []SettingAction{reset, all, none, send}
}
