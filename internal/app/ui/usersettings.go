package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// TODO: Improve switch API to allow switch not to be set on error

type UserSettings struct {
	widget.BaseWidget

	NotificationActions          []app.SettingAction
	NotificationSettings         fyne.CanvasObject // TODO: Refactor into widget
	GeneralActions               []app.SettingAction
	GeneralContent               fyne.CanvasObject // TODO: Refactor into widget
	CommunicationGroupContent    fyne.CanvasObject // TODO: Refactor into widget
	OnCommunicationGroupSelected func(title string, content fyne.CanvasObject, actions []app.SettingAction)

	showSnackbar func(string)
	sb           *iwidget.Snackbar
	u            *BaseUI
	w            fyne.Window
}

func NewSettings(u *BaseUI) *UserSettings {
	a := &UserSettings{
		showSnackbar: u.ShowSnackbar,
		u:            u,
		w:            u.MainWindow(),
	}
	a.ExtendBaseWidget(a)
	a.GeneralContent, a.GeneralActions = a.makeGeneralSettingsPage()
	a.NotificationSettings, a.NotificationActions = a.makeNotificationPage()
	return a
}

func (a *UserSettings) CreateRenderer() fyne.WidgetRenderer {
	makeSettingsPage := func(title string, content fyne.CanvasObject, actions []app.SettingAction) fyne.CanvasObject {
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

func (a *UserSettings) SetWindow(w fyne.Window) {
	a.w = w
	if a.sb != nil {
		a.sb.Stop()
	}
	a.sb = iwidget.NewSnackbar(w)
	a.sb.Start()
	a.showSnackbar = func(s string) {
		a.sb.Show(s)
	}
}

func (a *UserSettings) currentWindow() fyne.Window {
	return a.w
}

func (a *UserSettings) makeGeneralSettingsPage() (fyne.CanvasObject, []app.SettingAction) {
	logLevel := iwidget.NewSettingItemOptions(
		"Log level",
		"Set current log level",
		a.u.Settings().LogLevelNames(),
		a.u.Settings().LogLevelDefault(),
		func() string {
			return a.u.Settings().LogLevel()
		},
		func(v string) {
			s := a.u.Settings()
			s.SetLogLevel(v)
			slog.SetLogLoggerLevel(s.LogLevelSlog())
		},
		a.currentWindow,
	)
	vMin, vMax, vDef := a.u.Settings().MaxMailsPresets()
	maxMail := iwidget.NewSettingItemSlider(
		"Maximum mails",
		"Max number of mails downloaded. 0 = unlimited.",
		float64(vMin),
		float64(vMax),
		float64(vDef),
		func() float64 {
			return float64(a.u.Settings().MaxMails())
		},
		func(v float64) {
			a.u.Settings().SetMaxMails(int(v))
		},
		a.currentWindow,
	)
	vMin, vMax, vDef = a.u.Settings().MaxWalletTransactionsPresets()
	maxWallet := iwidget.NewSettingItemSlider(
		"Maximum wallet transaction",
		"Max wallet transactions downloaded. 0 = unlimited.",
		float64(vMin),
		float64(vMax),
		float64(vDef),
		func() float64 {
			return float64(a.u.Settings().MaxWalletTransactions())
		},
		func(v float64) {
			a.u.Settings().SetMaxWalletTransactions(int(v))
		},
		a.currentWindow,
	)
	preferMarketTab := iwidget.NewSettingItemSwitch(
		"Prefer market tab",
		"Show market tab for tradeable items",
		func() bool {
			return a.u.Settings().PreferMarketTab()
		},
		func(v bool) {
			a.u.Settings().SetPreferMarketTab(v)
		},
	)
	developerMode := iwidget.NewSettingItemSwitch(
		"Developer Mode",
		"App shows addditional technical information like Character IDs",
		func() bool {
			return a.u.Settings().DeveloperMode()
		},
		func(v bool) {
			a.u.Settings().SetDeveloperMode(v)
		},
	)

	items := []iwidget.SettingItem{
		iwidget.NewSettingItemHeading("Application"),
		logLevel,
		preferMarketTab,
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
			return a.u.Settings().SysTrayEnabled()
		},
		func(v bool) {
			a.u.Settings().SetSysTrayEnabled(v)
		},
	)
	if a.u.isDesktop() {
		items = slices.Insert(items, 2, systray)
	}

	list := iwidget.NewSettingList(items)

	clear := app.SettingAction{
		Label: "Clear cache",
		Action: func() {
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
						a.u.ShowSnackbar(fmt.Sprintf("Failed to clear cache: %s", a.u.humanizeError(err)))
					}
					m.Start()
				}, w)
		}}
	reset := app.SettingAction{
		Label: "Reset to defaults",
		Action: func() {
			a.u.Settings().ResetPreferMarketTab()
			a.u.Settings().ResetDeveloperMode()
			a.u.Settings().ResetLogLevel()
			a.u.Settings().ResetMaxMails()
			a.u.Settings().ResetMaxWalletTransactions()
			a.u.Settings().ResetSysTrayEnabled()
			list.Refresh()
		},
	}
	exportAppLog := app.SettingAction{
		Label: "Export application log",
		Action: func() {
			a.showExportFileDialog(a.u.dataPaths["log"])
		},
	}
	exportCrashLog := app.SettingAction{
		Label: "Export crash log",
		Action: func() {
			a.showExportFileDialog(a.u.dataPaths["crashfile"])
		},
	}
	deleteAppLog := app.SettingAction{
		Label: "Delete application log",
		Action: func() {
			a.showDeleteFileDialog("application log", a.u.dataPaths["log"]+"*")
		},
	}
	deleteCrashLog := app.SettingAction{
		Label: "Delete crash log",
		Action: func() {
			a.showDeleteFileDialog("crash log", a.u.dataPaths["crashfile"])
		},
	}
	actions := []app.SettingAction{reset, clear, exportAppLog, exportCrashLog, deleteAppLog, deleteCrashLog}
	if a.u.isDesktop() {
		actions = append(actions, app.SettingAction{
			Label: "Resets main window size to defaults",
			Action: func() {
				a.u.Settings().ResetWindowSize()
				a.u.MainWindow().Resize(a.u.Settings().WindowSize())
			},
		})
	}
	return list, actions
}

func (a *UserSettings) showDeleteFileDialog(name, path string) {
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
				a.showSnackbar("ERROR: Failed to delete " + name)
			} else {
				titler := cases.Title(language.English)
				a.showSnackbar(titler.String(name) + " deleted")
			}
		}, a.w)
}

func (a *UserSettings) showExportFileDialog(path string) {
	filename := filepath.Base(path)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		a.showSnackbar("No file to export: " + filename)
		return
	} else if err != nil {
		a.u.ShowErrorDialog("Failed to open "+filename, err, a.w)
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
				a.showSnackbar("File " + filename + " exported")
				return nil
			}()
			if err2 != nil {
				a.u.ShowErrorDialog("Failed to export "+filename, err, a.w)
			}
		}, a.w,
	)
	d.SetFileName(filename)
	a.u.ModifyShortcutsForDialog(d, a.w)
	d.Show()
}

func (a *UserSettings) makeNotificationPage() (fyne.CanvasObject, []app.SettingAction) {
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
	typesEnabled := a.u.Settings().NotificationTypesEnabled()

	// add global items
	notifyCommunications := iwidget.NewSettingItemSwitch(
		"Notify communications",
		"Whether to notify new communications",
		func() bool {
			return a.u.Settings().NotifyCommunicationsEnabled()
		},
		func(on bool) {
			a.u.Settings().SetNotifyCommunicationsEnabled(on)
			if on {
				a.u.Settings().SetNotifyCommunicationsEarliest(time.Now())
			}
		},
	)
	notifyMails := iwidget.NewSettingItemSwitch(
		"Notify mails",
		"Whether to notify new mails",
		func() bool {
			return a.u.Settings().NotifyMailsEnabled()
		},
		func(on bool) {
			a.u.Settings().SetNotifyMailsEnabled(on)
			if on {
				a.u.Settings().SetNotifyMailsEarliest(time.Now())
			}
		},
	)
	notifyPI := iwidget.NewSettingItemSwitch(
		"Planetary Industry",
		"Whether to notify about expired extractions",
		func() bool {
			return a.u.Settings().NotifyPIEnabled()
		},
		func(on bool) {
			a.u.Settings().SetNotifyPIEnabled(on)
			if on {
				a.u.Settings().SetNotifyPIEarliest(time.Now())
			}
		},
	)
	notifyTraining := iwidget.NewSettingItemSwitch(
		"Notify Training",
		"Whether to notify abouthen skillqueue is empty",
		func() bool {
			return a.u.Settings().NotifyTrainingEnabled()
		},
		func(on bool) {
			ctx := context.Background()
			if on {
				err := a.u.cs.EnableAllTrainingWatchers(ctx)
				if err != nil {
					a.u.ShowErrorDialog("failed to enable training notification", err, a.currentWindow())
				} else {
					a.u.Settings().SetNotifyTrainingEnabled(on)
				}
			} else {
				err := a.u.cs.DisableAllTrainingWatchers(ctx)
				if err != nil {
					a.u.ShowErrorDialog("failed to disable training notification", err, a.currentWindow())
				} else {
					a.u.Settings().SetNotifyCommunicationsEnabled(false)
				}
			}
		},
	)
	notifyContracts := iwidget.NewSettingItemSwitch(
		"Notify Contracts",
		"Whether to notify when contract status changes",
		func() bool {
			return a.u.Settings().NotifyContractsEnabled()
		},
		func(on bool) {
			a.u.Settings().SetNotifyContractsEnabled(on)
			if on {
				a.u.Settings().SetNotifyContractsEarliest(time.Now())
			}
		},
	)
	vMin, vMax, vDef := a.u.Settings().NotifyTimeoutHoursPresets()
	notifTimeout := iwidget.NewSettingItemSlider(
		"Notify Timeout",
		"Events older then this value in hours will not be notified",
		float64(vMin),
		float64(vMax),
		float64(vDef),
		func() float64 {
			return float64(a.u.Settings().NotifyTimeoutHours())
		},
		func(v float64) {
			a.u.Settings().SetNotifyTimeoutHours(int(v))
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
		actions []app.SettingAction
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
						a.u.Settings().SetNotificationTypesEnabled(typesEnabled)
					},
				)
				items2 = append(items2, it)
			}
			list2 := iwidget.NewSettingList(items2)
			enableAll := app.SettingAction{
				Label: "Enable all",
				Action: func() {
					for _, it := range items2 {
						it.Setter(true)
					}
					list2.Refresh()
				},
			}
			disableAll := app.SettingAction{
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
				actions: []app.SettingAction{enableAll, disableAll},
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
	reset := app.SettingAction{
		Label: "Reset to defaults",
		Action: func() {
			a.u.Settings().ResetNotifyCommunicationsEnabled()
			a.u.Settings().ResetNotifyContractsEnabled()
			a.u.Settings().ResetNotifyMailsEnabled()
			a.u.Settings().ResetNotifyPIEnabled()
			a.u.Settings().ResetNotifyTimeoutHours()
			a.u.Settings().ResetNotifyTrainingEnabled()
			typesEnabled.Clear()
			a.u.Settings().ResetNotificationTypesEnabled()
			list.Refresh()
		},
	}
	updateTypes := func() {
		a.u.Settings().SetNotificationTypesEnabled(typesEnabled)
		list.Refresh()
	}
	none := app.SettingAction{
		Label: "Disable all communication groups",
		Action: func() {
			typesEnabled.Clear()
			updateTypes()
		},
	}
	all := app.SettingAction{
		Label: "Enable all communication groups",
		Action: func() {
			for _, nt := range evenotification.SupportedGroups() {
				typesEnabled.Add(nt.String())
			}
			updateTypes()
		},
	}
	send := app.SettingAction{
		Label: "Send test notification",
		Action: func() {
			n := fyne.NewNotification("Test", "This is a test notification from EVE Buddy.")
			a.u.App().SendNotification(n)
		},
	}
	return list, []app.SettingAction{reset, all, none, send}
}
