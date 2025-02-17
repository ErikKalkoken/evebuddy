package ui

import (
	"context"
	"fmt"
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
	NotificationsActions   []SettingAction
	NotificationsContent   fyne.CanvasObject

	u      *BaseUI
	window fyne.Window
}

func (u *BaseUI) NewSettingsArea() *SettingsArea {
	a := &SettingsArea{u: u, window: u.Window}
	a.GeneralContent, a.GeneralActions = a.makeGeneralSettingsPage()
	a.DesktopContent, a.DesktopActions = a.makeDesktopSettingsPage()
	a.EveOnlineContent, a.EveOnlineActions = a.makeEVEOnlinePage()
	a.NotificationsContent, a.NotificationsActions = a.makeNotificationsPage()
	a.CommunicationsSettings, a.CommunicationsActions = a.makeCommunicationsPage()

	makeSettingsPage := func(title string, content fyne.CanvasObject, actions []SettingAction) fyne.CanvasObject {
		t := widget.NewLabel(title)
		t.TextStyle.Bold = true
		top := container.NewHBox(t, layout.NewSpacer())
		for _, a := range actions {
			top.Add(widget.NewButton(a.Label, a.Action))
		}
		return container.NewBorder(
			container.NewVBox(top, widget.NewSeparator()),
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
		container.NewTabItem("Notifications", makeSettingsPage("Notifications", a.NotificationsContent, a.NotificationsActions)),
		container.NewTabItem("Communications", makeSettingsPage("Communications", a.CommunicationsSettings, a.CommunicationsActions)),
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

func (a *SettingsArea) makeNotificationsPage() (fyne.CanvasObject, []SettingAction) {
	form := widget.NewForm()
	if a.u.IsMobile() {
		form.Orientation = widget.Vertical
	}

	// mail toogle
	mailEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		a.u.FyneApp.Preferences().SetBool(settingNotifyMailsEnabled, on)
		if on {
			a.u.FyneApp.Preferences().SetString(settingNotifyMailsEarliest, time.Now().Format(time.RFC3339))
		}
	})
	mailEnabledCheck.On = a.u.FyneApp.Preferences().BoolWithFallback(
		settingNotifyMailsEnabled,
		settingNotifyMailsEnabledDefault,
	)
	form.AppendItem(&widget.FormItem{
		Text:     "Mail",
		Widget:   mailEnabledCheck,
		HintText: "Wether to notify new mails",
	})

	// communications toogle
	communicationsEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		a.u.FyneApp.Preferences().SetBool(settingNotifyCommunicationsEnabled, on)
		if on {
			a.u.FyneApp.Preferences().SetString(settingNotifyCommunicationsEarliest, time.Now().Format(time.RFC3339))
		}
	})
	communicationsEnabledCheck.On = a.u.FyneApp.Preferences().BoolWithFallback(
		settingNotifyCommunicationsEnabled,
		settingNotifyCommunicationsEnabledDefault,
	)
	form.AppendItem(&widget.FormItem{
		Text:     "Communications",
		Widget:   communicationsEnabledCheck,
		HintText: "Wether to notify new communications",
	})

	// PI toogle
	piEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		a.u.FyneApp.Preferences().SetBool(settingNotifyPIEnabled, on)
		if on {
			a.u.FyneApp.Preferences().SetString(settingNotifyPIEarliest, time.Now().Format(time.RFC3339))
		}
	})
	piEnabledCheck.On = a.u.FyneApp.Preferences().BoolWithFallback(
		settingNotifyPIEnabled,
		settingNotifyPIEnabledDefault,
	)
	form.AppendItem(&widget.FormItem{
		Text:     "Planetary Industry",
		Widget:   piEnabledCheck,
		HintText: "Wether to notify about expired extractions",
	})

	// Training toogle
	// TODO: Improve switch API to allow switch not to be set on error
	trainingEnabledCheck := kxwidget.NewSwitch(func(on bool) {
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
	})
	trainingEnabledCheck.On = a.u.FyneApp.Preferences().BoolWithFallback(
		settingNotifyTrainingEnabled,
		settingNotifyTrainingEnabledDefault,
	)
	form.AppendItem(&widget.FormItem{
		Text:     "Training",
		Widget:   trainingEnabledCheck,
		HintText: "Wether to notify when skillqueue is empty",
	})

	// Contracts toogle
	contractsEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		a.u.FyneApp.Preferences().SetBool(settingNotifyContractsEnabled, on)
		if on {
			a.u.FyneApp.Preferences().SetString(settingNotifyContractsEarliest, time.Now().Format(time.RFC3339))
		}
	})
	contractsEnabledCheck.On = a.u.FyneApp.Preferences().BoolWithFallback(
		settingNotifyContractsEnabled,
		settingNotifyCommunicationsEnabledDefault,
	)
	form.AppendItem(&widget.FormItem{
		Text:     "Contracts",
		Widget:   contractsEnabledCheck,
		HintText: "Wether to notify when contract status changes",
	})

	// notify timeout
	notifyTimeout := kxwidget.NewSlider(1, settingNotifyTimeoutHoursMax)
	v := a.u.FyneApp.Preferences().IntWithFallback(settingNotifyTimeoutHours, settingNotifyTimeoutHoursDefault)
	notifyTimeout.SetValue(float64(v))
	notifyTimeout.OnChangeEnded = func(v float64) {
		a.u.FyneApp.Preferences().SetInt(settingNotifyTimeoutHours, int(v))
	}
	form.AppendItem(&widget.FormItem{
		Text:     "Notification timeout",
		Widget:   notifyTimeout,
		HintText: "Events older then this value in hours will not be notified",
	})

	form.AppendItem(&widget.FormItem{
		Text: "Test notification",
		Widget: widget.NewButton("Send now", func() {
			n := fyne.NewNotification("Test", "This is a test notification from EVE Buddy.")
			a.u.FyneApp.SendNotification(n)
		}),
		HintText: "Send a test notification to verify it works",
	})

	reset := SettingAction{
		Label: "Reset to defaults",
		Action: func() {
			mailEnabledCheck.SetState(settingNotifyMailsEnabledDefault)
			communicationsEnabledCheck.SetState(settingNotifyCommunicationsEnabledDefault)
			piEnabledCheck.SetState(settingNotifyPIEnabledDefault)
			trainingEnabledCheck.SetState(settingNotifyTrainingEnabledDefault)
			contractsEnabledCheck.SetState(settingNotifyTrainingEnabledDefault)
			notifyTimeout.SetValue(settingNotifyTimeoutHoursDefault)
		},
	}
	return form, []SettingAction{reset}
}

func (a *SettingsArea) makeCommunicationsPage() (fyne.CanvasObject, []SettingAction) {
	categoriesAndTypes := make(map[evenotification.Folder][]evenotification.Type)
	for _, n := range evenotification.SupportedTypes() {
		c := evenotification.Type2folder[n]
		categoriesAndTypes[c] = append(categoriesAndTypes[c], n)
	}
	categories := make([]evenotification.Folder, 0)
	for c := range categoriesAndTypes {
		categories = append(categories, c)
	}
	slices.Sort(categories)

	typesEnabled := set.NewFromSlice(a.u.FyneApp.Preferences().StringList(settingNotificationsTypesEnabled))
	p := theme.Padding()
	list := widget.NewList(
		func() int {
			return len(categories)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabel("Title")
			title.Wrapping = fyne.TextWrapBreak
			title.TextStyle.Bold = true
			return container.NewBorder(
				nil,
				nil,
				nil,
				widget.NewLabel("State"),
				title,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			c := categories[id]
			border := co.(*fyne.Container).Objects
			title := border[0].(*widget.Label)
			title.SetText(c.String())
			state := border[1].(*widget.Label)
			var enabled int
			for _, n := range categoriesAndTypes[c] {
				if x := n.String(); typesEnabled.Contains(x) {
					enabled++
				}
			}
			var s string
			total := len(categoriesAndTypes[c])
			switch enabled {
			case 0:
				s = "Off"
			case total:
				s = "All enabled"
			default:
				s = fmt.Sprintf("%d / %d enabled", enabled, total)
			}
			state.SetText(s)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		f := categories[id]

		list2 := widget.NewList(
			func() int {
				return len(categoriesAndTypes[f])
			},
			func() fyne.CanvasObject {
				title := widget.NewLabel("Title")
				title.Truncation = fyne.TextTruncateEllipsis
				return container.NewBorder(
					nil,
					nil,
					nil,
					container.NewVBox(layout.NewSpacer(), kxwidget.NewSwitch(nil), layout.NewSpacer()),
					container.New(layout.NewCustomPaddedLayout(0, -p, 0, 0), title),
				)
			},
			func(id widget.ListItemID, co fyne.CanvasObject) {
				nt := categoriesAndTypes[f][id]
				outer := co.(*fyne.Container).Objects
				inner := outer[0].(*fyne.Container).Objects
				title := inner[0].(*widget.Label)
				title.SetText(nt.Display())
				s := outer[1].(*fyne.Container).Objects[1].(*kxwidget.Switch)
				s.OnChanged = func(on bool) {
					if on {
						typesEnabled.Add(nt.String())
					} else {
						typesEnabled.Remove(nt.String())
					}
					a.u.FyneApp.Preferences().SetStringList(settingNotificationsTypesEnabled, typesEnabled.ToSlice())
				}
				s.SetState(typesEnabled.Contains(nt.String()))
			},
		)
		list2.OnSelected = func(id widget.ListItemID) {
			defer list2.UnselectAll()
			nt := categoriesAndTypes[f][id]
			if typesEnabled.Contains(nt.String()) {
				typesEnabled.Remove(nt.String())
			} else {
				typesEnabled.Add(nt.String())
			}
			a.u.FyneApp.Preferences().SetStringList(settingNotificationsTypesEnabled, typesEnabled.ToSlice())
			list2.RefreshItem(id)
		}

		var d dialog.Dialog
		title := widget.NewLabel(f.String())
		title.TextStyle.Bold = true
		top := container.NewBorder(
			nil, nil, nil, widgets.NewIconButtonWithMenu(
				theme.MenuIcon(), fyne.NewMenu("", fyne.NewMenuItem(
					"Enable all",
					func() {
						for _, nt := range categoriesAndTypes[f] {
							typesEnabled.Add(nt.String())
						}
						a.u.FyneApp.Preferences().SetStringList(settingNotificationsTypesEnabled, typesEnabled.ToSlice())
						list2.Refresh()
					}),
					fyne.NewMenuItem(
						"Disable all",
						func() {
							for _, nt := range categoriesAndTypes[f] {
								typesEnabled.Remove(nt.String())
							}
							a.u.FyneApp.Preferences().SetStringList(settingNotificationsTypesEnabled, typesEnabled.ToSlice())
							list2.Refresh()
						}))),
			title,
		)
		c := container.NewBorder(
			container.NewVBox(top, widget.NewSeparator()),
			nil,
			nil,
			nil,
			list2,
		)

		d = dialog.NewCustom("Notification Types", "Close", c, a.window)
		d.Show()
		d.Resize(fyne.NewSize(400, 600))
		d.SetOnClosed(func() {
			list.Refresh()
		})
	}

	updateTypes := func() {
		a.u.FyneApp.Preferences().SetStringList(settingNotificationsTypesEnabled, typesEnabled.ToSlice())
		list.Refresh()
	}
	none := SettingAction{
		Label: "Disable all",
		Action: func() {
			typesEnabled.Clear()
			updateTypes()
		},
	}
	all := SettingAction{
		Label: "Enable all",
		Action: func() {
			for _, nt := range evenotification.SupportedTypes() {
				typesEnabled.Add(nt.String())
			}
			updateTypes()
		},
	}
	return list, []SettingAction{none, all}
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
