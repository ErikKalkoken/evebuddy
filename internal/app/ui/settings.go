package ui

import (
	"context"
	"log/slog"
	"maps"
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
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

// TODO: Improve switch API to allow switch not to be set on error

// Exported settings
const (
	SettingLogLevel              = "logLevel"
	SettingLogLevelDefault       = "warning"
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
	Content              fyne.CanvasObject
	NotificationActions  []SettingAction
	NotificationSettings fyne.CanvasObject
	GeneralActions       []SettingAction
	GeneralContent       fyne.CanvasObject

	u      *BaseUI
	window fyne.Window
}

func (u *BaseUI) NewSettingsArea() *SettingsArea {
	a := &SettingsArea{u: u, window: u.Window}
	a.GeneralContent, a.GeneralActions = a.makeGeneralSettingsPage()
	a.NotificationSettings, a.NotificationActions = a.makeNotificationPage()

	makeSettingsPage := func(title string, content fyne.CanvasObject, actions []SettingAction) fyne.CanvasObject {
		t := widget.NewLabel(title)
		t.TextStyle.Bold = true
		items := make([]*fyne.MenuItem, 0)
		for _, a := range actions {
			items = append(items, fyne.NewMenuItem(a.Label, a.Action))
		}
		options := widgets.NewContextMenuButtonWithIcon(theme.MoreHorizontalIcon(), "", fyne.NewMenu("", items...))
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
}

func (a *SettingsArea) currentWindow() fyne.Window {
	return a.window
}

func (a *SettingsArea) makeGeneralSettingsPage() (fyne.CanvasObject, []SettingAction) {
	setLogLevel := func(s string) {
		a.u.FyneApp.Preferences().SetString(SettingLogLevel, s)
		slog.SetLogLoggerLevel(LogLevelName2Level(s))
	}
	setMail := func(v float64) {
		a.u.FyneApp.Preferences().SetInt(settingMaxMails, int(v))
	}
	setTransactions := func(v float64) {
		a.u.FyneApp.Preferences().SetInt(settingMaxWalletTransactions, int(v))
	}
	items := []widgets.SettingItem{
		widgets.NewSettingItemHeading("Application"),
		widgets.NewSettingItemSelect(
			"Log level",
			"Set current log level",
			LogLevelNames(),
			func() string {
				return a.u.FyneApp.Preferences().StringWithFallback(
					SettingLogLevel,
					SettingLogLevelDefault,
				)
			},
			setLogLevel,
			a.currentWindow,
		),
		widgets.NewSettingItemSeperator(),
		widgets.NewSettingItemHeading("EVE Online"),
		widgets.NewSettingItemSlider(
			"Maximum mails",
			"Max number of mails downloaded. 0 = unlimited.",
			0,
			settingMaxMailsMax,
			func() float64 {
				return float64(a.u.FyneApp.Preferences().IntWithFallback(
					settingMaxMails,
					settingMaxMailsDefault))
			},
			setMail,
			a.currentWindow,
		),
		widgets.NewSettingItemSlider(
			"Maximum wallet transaction",
			"Max wallet transactions downloaded. 0 = unlimited.",
			0,
			settingMaxWalletTransactionsMax,
			func() float64 {
				return float64(a.u.FyneApp.Preferences().IntWithFallback(
					settingMaxWalletTransactions,
					settingMaxWalletTransactionsDefault))
			},
			setTransactions,
			a.currentWindow,
		),
	}

	setSysTray := func(b bool) {
		a.u.FyneApp.Preferences().SetBool(SettingSysTrayEnabled, b)
	}
	if a.u.IsDesktop() {
		items = slices.Insert(items, 2,
			widgets.NewSettingItemSwitch(
				"Close button",
				"App will minimize to system tray when closed (requires restart)",
				func() bool {
					return a.u.FyneApp.Preferences().BoolWithFallback(
						SettingSysTrayEnabled,
						SettingSysTrayEnabledDefault,
					)
				},
				setSysTray,
			),
		)
	}

	list := widgets.NewSettingList(items)

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
			setLogLevel(SettingLogLevelDefault)
			setMail(settingMaxMailsDefault)
			setTransactions(settingMaxWalletTransactionsDefault)
			setSysTray(SettingSysTrayEnabledDefault)
		},
	}
	actions := []SettingAction{reset, clear}
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
	setCommunications := func(on bool) {
		a.u.FyneApp.Preferences().SetBool(settingNotifyCommunicationsEnabled, on)
		if on {
			a.u.FyneApp.Preferences().SetString(settingNotifyCommunicationsEarliest, time.Now().Format(time.RFC3339))
		}
	}
	setContracts := func(on bool) {
		a.u.FyneApp.Preferences().SetBool(settingNotifyContractsEnabled, on)
		if on {
			a.u.FyneApp.Preferences().SetString(settingNotifyContractsEarliest, time.Now().Format(time.RFC3339))
		}
	}
	setMail := func(on bool) {
		a.u.FyneApp.Preferences().SetBool(settingNotifyMailsEnabled, on)
		if on {
			a.u.FyneApp.Preferences().SetString(settingNotifyMailsEarliest, time.Now().Format(time.RFC3339))
		}
	}
	setPI := func(on bool) {
		a.u.FyneApp.Preferences().SetBool(settingNotifyPIEnabled, on)
		if on {
			a.u.FyneApp.Preferences().SetString(settingNotifyPIEarliest, time.Now().Format(time.RFC3339))
		}
	}
	setTraining := func(on bool) {
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
	}
	items := []widgets.SettingItem{
		widgets.NewSettingItemHeading("Global"),
		widgets.NewSettingItemSwitch(
			"Notify communications",
			"Whether to notify new communications",
			func() bool {
				return a.u.FyneApp.Preferences().BoolWithFallback(
					settingNotifyCommunicationsEnabled,
					settingNotifyCommunicationsEnabledDefault,
				)
			},
			setCommunications,
		),
		widgets.NewSettingItemSwitch(
			"Notify mails",
			"Whether to notify new mails",
			func() bool {
				return a.u.FyneApp.Preferences().BoolWithFallback(
					settingNotifyMailsEnabled,
					settingNotifyMailsEnabledDefault,
				)
			},
			setMail,
		),
		widgets.NewSettingItemSwitch(
			"Planetary Industry",
			"Whether to notify about expired extractions",
			func() bool {
				return a.u.FyneApp.Preferences().BoolWithFallback(
					settingNotifyPIEnabled,
					settingNotifyPIEnabledDefault,
				)
			},
			setPI,
		),
		widgets.NewSettingItemSwitch(
			"Notify Training",
			"Whether to notify abouthen skillqueue is empty",
			func() bool {
				return a.u.FyneApp.Preferences().BoolWithFallback(
					settingNotifyTrainingEnabled,
					settingNotifyTrainingEnabledDefault,
				)
			},
			setTraining,
		),
		widgets.NewSettingItemSwitch(
			"Notify Contracts",
			"Whether to notify when contract status changes",
			func() bool {
				return a.u.FyneApp.Preferences().BoolWithFallback(
					settingNotifyContractsEnabled,
					settingNotifyCommunicationsEnabledDefault,
				)
			},
			setContracts,
		),
		widgets.NewSettingItemSlider(
			"Notify Timeout",
			"Events older then this value in hours will not be notified",
			1,
			settingNotifyTimeoutHoursMax,
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
		),
	}
	// add communication groups
	for _, g := range groups {
		items = append(items, widgets.NewSettingItemSeperator())
		items = append(items, widgets.NewSettingItemHeading("Communications: "+g.String()))
		for _, nt := range groupsAndTypes[g] {
			ntStr := nt.String()
			ntDisplay := nt.Display()
			it := widgets.NewSettingItemSwitch(
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
					a.u.FyneApp.Preferences().SetStringList(settingNotificationsTypesEnabled, typesEnabled.ToSlice())
				},
			)
			items = append(items, it)
		}
	}

	list := widgets.NewSettingList(items)
	reset := SettingAction{
		Label: "Reset to defaults",
		Action: func() {
			typesEnabled.Clear()
			setCommunications(settingNotifyCommunicationsEnabledDefault)
			setMail(settingNotifyMailsEnabledDefault)
			setPI(settingNotifyPIEnabledDefault)
			setTraining(settingNotifyTrainingEnabledDefault)
			setContracts(settingNotifyTrainingEnabledDefault)
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
