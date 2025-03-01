package ui

import (
	"context"
	"errors"
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

	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

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

type SettingsArea struct {
	Content                      fyne.CanvasObject
	NotificationActions          []SettingAction
	NotificationSettings         fyne.CanvasObject
	GeneralActions               []SettingAction
	GeneralContent               fyne.CanvasObject
	CommunicationGroupContent    fyne.CanvasObject
	OnCommunicationGroupSelected func(title string, content fyne.CanvasObject, actions []SettingAction)

	snackbar *iwidget.Snackbar
	u        *BaseUI
	window   fyne.Window
}

func (u *BaseUI) NewSettingsArea() *SettingsArea {
	a := &SettingsArea{
		snackbar: u.Snackbar,
		u:        u,
		window:   u.Window,
	}
	a.GeneralContent, a.GeneralActions = a.makeGeneralSettingsPage()
	a.NotificationSettings, a.NotificationActions = a.makeNotificationPage()

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
	a.Content = tabs
	return a
}

func (a *SettingsArea) SetWindow(w fyne.Window) {
	a.window = w
	a.snackbar = iwidget.NewSnackbar(w)
	a.snackbar.Start()
}

func (a *SettingsArea) currentWindow() fyne.Window {
	return a.window
}

func (a *SettingsArea) makeGeneralSettingsPage() (fyne.CanvasObject, []SettingAction) {
	logLevel := iwidget.NewSettingItemOptions(
		"Log level",
		"Set current log level",
		LogLevelNames(),
		SettingLogLevelDefault,
		func() string {
			return a.u.FyneApp.Preferences().StringWithFallback(
				SettingLogLevel,
				SettingLogLevelDefault,
			)
		},
		func(s string) {
			a.u.FyneApp.Preferences().SetString(SettingLogLevel, s)
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
			return float64(a.u.FyneApp.Preferences().IntWithFallback(
				settingMaxMails,
				settingMaxMailsDefault))
		},
		func(v float64) {
			a.u.FyneApp.Preferences().SetInt(settingMaxMails, int(v))
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
			return float64(a.u.FyneApp.Preferences().IntWithFallback(
				settingMaxWalletTransactions,
				settingMaxWalletTransactionsDefault))
		},
		func(v float64) {
			a.u.FyneApp.Preferences().SetInt(settingMaxWalletTransactions, int(v))
		},
		a.currentWindow,
	)
	items := []iwidget.SettingItem{
		iwidget.NewSettingItemHeading("Application"),
		logLevel,
		iwidget.NewSettingItemSeperator(),
		iwidget.NewSettingItemHeading("EVE Online"),
		maxMail,
		maxWallet,
	}

	sysTray := iwidget.NewSettingItemSwitch(
		"Close button",
		"App will minimize to system tray when closed (requires restart)",
		func() bool {
			return a.u.FyneApp.Preferences().BoolWithFallback(
				SettingSysTrayEnabled,
				SettingSysTrayEnabledDefault,
			)
		},
		func(b bool) {
			a.u.FyneApp.Preferences().SetBool(SettingSysTrayEnabled, b)
		},
	)
	if a.u.IsDesktop() {
		items = slices.Insert(items, 2, sysTray)
	}

	list := iwidget.NewSettingList(items)

	clear := SettingAction{
		"Clear the local image cache",
		func() {
			w := a.currentWindow()
			m := kxmodal.NewProgressInfinite(
				"Clearing cache...",
				"",
				func() error {
					if err := a.u.EveImageService.ClearCache(); err != nil {
						return err
					}
					slog.Info("Cleared image cache")
					return nil
				},
				w,
			)
			m.OnSuccess = func() {
				d := dialog.NewInformation("Image cache", "Image cache cleared", w)
				d.Show()
			}
			m.OnError = func(err error) {
				slog.Error("Failed to clear image cache", "error", err)
				d := NewErrorDialog("Failed to clear image cache", err, w)
				d.Show()
			}
			m.Start()
		}}
	reset := SettingAction{
		Label: "Reset to defaults",
		Action: func() {
			logLevel.Setter(SettingLogLevelDefault)
			maxMail.Setter(settingMaxMailsDefault)
			maxWallet.Setter(settingMaxWalletTransactionsDefault)
			sysTray.Setter(SettingSysTrayEnabledDefault)
			list.Refresh()
		},
	}
	exportAppLog := SettingAction{
		Label: "Export application log",
		Action: func() {
			a.showExportFileDialog(a.u.DataPaths["log"])
		},
	}
	exportCrashLog := SettingAction{
		Label: "Export crash log",
		Action: func() {
			a.showExportFileDialog(a.u.DataPaths["crashfile"])
		},
	}
	deleteAppLog := SettingAction{
		Label: "Delete application log",
		Action: func() {
			a.showDeleteFileDialog("application log", a.u.DataPaths["log"]+"*")
		},
	}
	deleteCrashLog := SettingAction{
		Label: "Delete creash log",
		Action: func() {
			a.showDeleteFileDialog("crash log", a.u.DataPaths["crashfile"])
		},
	}
	actions := []SettingAction{reset, clear, exportAppLog, deleteAppLog, exportCrashLog, deleteCrashLog}
	if a.u.IsDesktop() {
		actions = append(actions, SettingAction{
			Label: "Resets main window size to defaults",
			Action: func() {
				a.u.Window.Resize(fyne.NewSize(SettingWindowWidthDefault, SettingWindowHeightDefault))
			},
		})
	}
	return list, actions
}

func (a *SettingsArea) showDeleteFileDialog(name, path string) {
	d := dialog.NewConfirm("Delete "+name, "Are you sure?", func(confirmed bool) {
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
			a.snackbar.Show(Titler.String(name) + " deleted")
		}
	}, a.window)
	d.Show()
}

func (a *SettingsArea) showExportFileDialog(path string) {
	filename := filepath.Base(path)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		a.snackbar.Show("No file to export: " + filename)
		return
	} else if err != nil {
		ShowErrorDialog("Failed to open "+filename, err, a.window)
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
				ShowErrorDialog("Failed to export "+filename, err, a.window)
			}
		}, a.window,
	)
	d.SetFileName(filename)
	d.Show()
}

func (a *SettingsArea) makeNotificationPage() (fyne.CanvasObject, []SettingAction) {
	groupsAndTypes := make(map[evenotification.Group][]evenotification.Type)
	for _, n := range evenotification.SupportedGroups() {
		c := evenotification.Type2group[n]
		groupsAndTypes[c] = append(groupsAndTypes[c], n)
	}
	groups := make([]evenotification.Group, 0)
	for c := range groupsAndTypes {
		groups = append(groups, c)
	}
	for _, g := range groups {
		slices.Sort(groupsAndTypes[g])
	}
	slices.Sort(groups)
	typesEnabled := set.NewFromSlice(a.u.FyneApp.Preferences().StringList(settingNotificationsTypesEnabled))

	// add global items
	notifyCommunications := iwidget.NewSettingItemSwitch(
		"Notify communications",
		"Whether to notify new communications",
		func() bool {
			return a.u.FyneApp.Preferences().BoolWithFallback(
				settingNotifyCommunicationsEnabled,
				settingNotifyCommunicationsEnabledDefault,
			)
		},
		func(on bool) {
			a.u.FyneApp.Preferences().SetBool(settingNotifyCommunicationsEnabled, on)
			if on {
				a.u.FyneApp.Preferences().SetString(
					settingNotifyCommunicationsEarliest,
					time.Now().Format(time.RFC3339))
			}
		},
	)
	notifyMails := iwidget.NewSettingItemSwitch(
		"Notify mails",
		"Whether to notify new mails",
		func() bool {
			return a.u.FyneApp.Preferences().BoolWithFallback(
				settingNotifyMailsEnabled,
				settingNotifyMailsEnabledDefault,
			)
		},
		func(on bool) {
			a.u.FyneApp.Preferences().SetBool(settingNotifyMailsEnabled, on)
			if on {
				a.u.FyneApp.Preferences().SetString(
					settingNotifyMailsEarliest,
					time.Now().Format(time.RFC3339))
			}
		},
	)
	notifyPI := iwidget.NewSettingItemSwitch(
		"Planetary Industry",
		"Whether to notify about expired extractions",
		func() bool {
			return a.u.FyneApp.Preferences().BoolWithFallback(
				settingNotifyPIEnabled,
				settingNotifyPIEnabledDefault,
			)
		},
		func(on bool) {
			a.u.FyneApp.Preferences().SetBool(settingNotifyPIEnabled, on)
			if on {
				a.u.FyneApp.Preferences().SetString(
					settingNotifyPIEarliest,
					time.Now().Format(time.RFC3339))
			}
		},
	)
	notifyTraining := iwidget.NewSettingItemSwitch(
		"Notify Training",
		"Whether to notify abouthen skillqueue is empty",
		func() bool {
			return a.u.FyneApp.Preferences().BoolWithFallback(
				settingNotifyTrainingEnabled,
				settingNotifyTrainingEnabledDefault,
			)
		},
		func(on bool) {
			ctx := context.Background()
			if on {
				err := a.u.CharacterService.EnableAllTrainingWatchers(ctx)
				if err != nil {
					d := NewErrorDialog("failed to enable training notification", err, a.currentWindow())
					d.Show()
				} else {
					a.u.FyneApp.Preferences().SetBool(settingNotifyTrainingEnabled, true)
				}
			} else {
				err := a.u.CharacterService.DisableAllTrainingWatchers(ctx)
				if err != nil {
					d := NewErrorDialog("failed to disable training notification", err, a.currentWindow())
					d.Show()
				} else {
					a.u.FyneApp.Preferences().SetBool(settingNotifyTrainingEnabled, false)
				}
			}
		},
	)
	notifyContracts := iwidget.NewSettingItemSwitch(
		"Notify Contracts",
		"Whether to notify when contract status changes",
		func() bool {
			return a.u.FyneApp.Preferences().BoolWithFallback(
				settingNotifyContractsEnabled,
				settingNotifyCommunicationsEnabledDefault,
			)
		},
		func(on bool) {
			a.u.FyneApp.Preferences().SetBool(settingNotifyContractsEnabled, on)
			if on {
				a.u.FyneApp.Preferences().SetString(
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
			return float64(a.u.FyneApp.Preferences().IntWithFallback(
				settingNotifyTimeoutHours,
				settingNotifyTimeoutHoursDefault,
			))
		},
		func(v float64) {
			a.u.FyneApp.Preferences().SetInt(settingNotifyTimeoutHours, int(v))
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
	groupPages := make(map[evenotification.Group]groupPage) // for pre-constructing group pages
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
						a.u.FyneApp.Preferences().SetStringList(
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
		a.u.FyneApp.Preferences().SetStringList(
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
			a.u.FyneApp.SendNotification(n)
		},
	}
	return list, []SettingAction{reset, all, none, send}
}
