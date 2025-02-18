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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

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
	Content                fyne.CanvasObject
	CommunicationsActions  []SettingAction
	CommunicationsSettings fyne.CanvasObject
	DesktopActions         []SettingAction
	DesktopContent         fyne.CanvasObject
	EveOnlineActions       []SettingAction
	EveOnlineContent       fyne.CanvasObject
	GeneralActions         []SettingAction
	GeneralContent         fyne.CanvasObject

	u      *BaseUI
	window fyne.Window
}

func (u *BaseUI) NewSettingsArea() *SettingsArea {
	a := &SettingsArea{u: u, window: u.Window}
	a.GeneralContent, a.GeneralActions = a.makeGeneralSettingsPage()
	a.DesktopContent, a.DesktopActions = a.makeDesktopSettingsPage()
	a.EveOnlineContent, a.EveOnlineActions = a.makeEVEOnlinePage()
	a.CommunicationsSettings, a.CommunicationsActions = a.makeNotificationPage()

	makeSettingsPage := func(title string, content fyne.CanvasObject, actions []SettingAction) fyne.CanvasObject {
		t := widget.NewLabel(title)
		t.TextStyle.Bold = true
		items := make([]*fyne.MenuItem, 0)
		for _, a := range actions {
			items = append(items, fyne.NewMenuItem(a.Label, a.Action))
		}
		options := widgets.NewContextMenuButtonWithIcon(theme.MenuIcon(), "", fyne.NewMenu("", items...))
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
		container.NewTabItem("Desktop", makeSettingsPage("Desktop", a.DesktopContent, a.DesktopActions)),
		container.NewTabItem("EVE Online", makeSettingsPage("EVE Online", a.EveOnlineContent, a.EveOnlineActions)),
		container.NewTabItem("Notifications", makeSettingsPage("Notifications", a.CommunicationsSettings, a.CommunicationsActions)),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	a.Content = tabs
	return a
}

func (a *SettingsArea) SetWindow(w fyne.Window) {
	a.window = w
}

func (a *SettingsArea) makeGeneralSettingsPage() (fyne.CanvasObject, []SettingAction) {
	// log level
	logLevel := widget.NewSelect(LogLevelNames(), func(s string) {
		a.u.FyneApp.Preferences().SetString(SettingLogLevel, s)
		slog.SetLogLoggerLevel(LogLevelName2Level(s))
	})
	logLevelSelected := a.u.FyneApp.Preferences().StringWithFallback(
		SettingLogLevel,
		SettingLogLevelDefault,
	)
	logLevel.SetSelected(logLevelSelected)

	// cache
	clearBtn := widget.NewButton("Clear NOW", func() {
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
			a.window,
		)
		m.OnSuccess = func() {
			d := dialog.NewInformation("Image cache", "Image cache cleared", a.window)
			d.Show()
		}
		m.OnError = func(err error) {
			slog.Error("Failed to clear image cache", "error", err)
			d := NewErrorDialog("Failed to clear image cache", err, a.window)
			d.Show()
		}
		m.Start()
	})
	settings := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:     "Image cache",
				Widget:   container.NewHBox(clearBtn),
				HintText: "Clear the local image cache",
			},
			{
				Text:     "Log level",
				Widget:   logLevel,
				HintText: "Current log level",
			},
		}}
	if a.u.IsMobile() {
		settings.Orientation = widget.Vertical
	}
	reset := SettingAction{
		Label: "Reset to defaults",
		Action: func() {
			logLevel.SetSelected(SettingLogLevelDefault)
		},
	}
	return settings, []SettingAction{reset}
}

func (a *SettingsArea) makeEVEOnlinePage() (fyne.CanvasObject, []SettingAction) {
	// max mails
	maxMails := kxwidget.NewSlider(0, settingMaxMailsMax)
	v1 := a.u.FyneApp.Preferences().IntWithFallback(settingMaxMails, settingMaxMailsDefault)
	maxMails.SetValue(float64(v1))
	maxMails.OnChangeEnded = func(v float64) {
		a.u.FyneApp.Preferences().SetInt(settingMaxMails, int(v))
	}

	// max transactions
	maxTransactions := kxwidget.NewSlider(0, settingMaxWalletTransactionsMax)
	v2 := a.u.FyneApp.Preferences().IntWithFallback(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault)
	maxTransactions.SetValue(float64(v2))
	maxTransactions.OnChangeEnded = func(v float64) {
		a.u.FyneApp.Preferences().SetInt(settingMaxWalletTransactions, int(v))
	}

	settings := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:     "Maximum mails",
				Widget:   maxMails,
				HintText: "Max number of mails downloaded. 0 = unlimited.",
			},
			{
				Text:     "Maximum wallet transaction",
				Widget:   maxTransactions,
				HintText: "Max wallet transactions downloaded. 0 = unlimited.",
			},
		},
	}
	if a.u.IsMobile() {
		settings.Orientation = widget.Vertical
	}
	reset := SettingAction{
		Label: "Reset to defaults",
		Action: func() {
			maxMails.SetValue(settingMaxMailsDefault)
			maxTransactions.SetValue(settingMaxWalletTransactionsDefault)
		},
	}
	return settings, []SettingAction{reset}
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

	items := make([]widgets.SettingItem, 0)

	// add global items
	items = append(items, widgets.NewSettingItemHeading("Global"))
	setCommunications := func(on bool) {
		a.u.FyneApp.Preferences().SetBool(settingNotifyCommunicationsEnabled, on)
		if on {
			a.u.FyneApp.Preferences().SetString(settingNotifyCommunicationsEarliest, time.Now().Format(time.RFC3339))
		}
	}
	items = append(items, widgets.NewSettingItemSwitch(
		"Notify communications",
		"Whether to notify new communications",
		func() bool {
			return a.u.FyneApp.Preferences().BoolWithFallback(
				settingNotifyCommunicationsEnabled,
				settingNotifyCommunicationsEnabledDefault,
			)
		},
		setCommunications,
	))
	setMail := func(on bool) {
		a.u.FyneApp.Preferences().SetBool(settingNotifyMailsEnabled, on)
		if on {
			a.u.FyneApp.Preferences().SetString(settingNotifyMailsEarliest, time.Now().Format(time.RFC3339))
		}
	}
	items = append(items, widgets.NewSettingItemSwitch(
		"Notify mails",
		"Whether to notify new mails",
		func() bool {
			return a.u.FyneApp.Preferences().BoolWithFallback(
				settingNotifyMailsEnabled,
				settingNotifyMailsEnabledDefault,
			)
		},
		setMail,
	))
	setPI := func(on bool) {
		a.u.FyneApp.Preferences().SetBool(settingNotifyPIEnabled, on)
		if on {
			a.u.FyneApp.Preferences().SetString(settingNotifyPIEarliest, time.Now().Format(time.RFC3339))
		}
	}
	items = append(items, widgets.NewSettingItemSwitch(
		"Planetary Industry",
		"Whether to notify about expired extractions",
		func() bool {
			return a.u.FyneApp.Preferences().BoolWithFallback(
				settingNotifyPIEnabled,
				settingNotifyPIEnabledDefault,
			)
		},
		setPI,
	))
	setTraining := func(on bool) {
		ctx := context.Background()
		if on {
			err := a.u.CharacterService.EnableAllTrainingWatchers(ctx)
			if err != nil {
				d := NewErrorDialog("failed to enable training notification", err, a.window)
				d.Show()
			} else {
				a.u.FyneApp.Preferences().SetBool(settingNotifyTrainingEnabled, true)
			}
		} else {
			err := a.u.CharacterService.DisableAllTrainingWatchers(ctx)
			if err != nil {
				d := NewErrorDialog("failed to disable training notification", err, a.window)
				d.Show()
			} else {
				a.u.FyneApp.Preferences().SetBool(settingNotifyTrainingEnabled, false)
			}
		}
	}
	items = append(items, widgets.NewSettingItemSwitch(
		"Notify Training",
		"Whether to notify abouthen skillqueue is empty",
		func() bool {
			return a.u.FyneApp.Preferences().BoolWithFallback(
				settingNotifyTrainingEnabled,
				settingNotifyTrainingEnabledDefault,
			)
		},
		setTraining,
	))
	setContracts := func(on bool) {
		a.u.FyneApp.Preferences().SetBool(settingNotifyContractsEnabled, on)
		if on {
			a.u.FyneApp.Preferences().SetString(settingNotifyContractsEarliest, time.Now().Format(time.RFC3339))
		}
	}
	items = append(items, widgets.NewSettingItemSwitch(
		"Notify Contracts",
		"Whether to notify when contract status changes",
		func() bool {
			return a.u.FyneApp.Preferences().BoolWithFallback(
				settingNotifyContractsEnabled,
				settingNotifyCommunicationsEnabledDefault,
			)
		},
		setContracts,
	))
	items = append(items, widgets.NewSettingItemSlider(
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
		func() fyne.Window {
			return a.window
		},
	))

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

	// create list for generated settings
	list := widgets.NewSettingList(items)
	c := container.NewBorder(
		widgets.NewLabelWithSize(
			"Choose which communication types can trigger a notification.",
			theme.SizeNameCaptionText,
		),
		nil,
		nil,
		nil,
		list,
	)
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
	return c, []SettingAction{reset, all, none, send}
}

func (a *SettingsArea) makeDesktopSettingsPage() (fyne.CanvasObject, []SettingAction) {
	// system tray
	sysTrayCheck := kxwidget.NewSwitch(func(b bool) {
		a.u.FyneApp.Preferences().SetBool(SettingSysTrayEnabled, b)
	})
	sysTrayCheck.On = a.u.FyneApp.Preferences().BoolWithFallback(
		SettingSysTrayEnabled,
		SettingSysTrayEnabledDefault,
	)

	// window
	resetWindow := widget.NewButton("Reset main window size", func() {
		a.window.Resize(fyne.NewSize(SettingWindowWidthDefault, SettingWindowHeightDefault))
	})

	settings := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:     "Close button",
				Widget:   sysTrayCheck,
				HintText: "App will minimize to system tray when closed (requires restart)",
			},
			{
				Text:     "Window",
				Widget:   resetWindow,
				HintText: "Resets window size to defaults",
			},
		}}
	reset := SettingAction{
		Label: "Reset",
		Action: func() {
			sysTrayCheck.SetState(SettingSysTrayEnabledDefault)
		},
	}
	return settings, []SettingAction{reset}
}
