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

const (
	SettingLastCharacterID                    = "settingLastCharacterID"
	SettingLogLevel                           = "logLevel"
	SettingLogLevelDefault                    = "warning"
	SettingMaxMails                           = "settingMaxMails"
	SettingMaxMailsDefault                    = 1_000
	SettingMaxMailsMax                        = 10_000
	SettingMaxWalletTransactions              = "settingMaxWalletTransactions"
	SettingMaxWalletTransactionsDefault       = 1_000
	SettingMaxWalletTransactionsMax           = 10_000
	SettingNotificationsTypesEnabled          = "settingNotificationsTypesEnabled"
	SettingNotifyCommunicationsEarliest       = "settingNotifyCommunicationsEarliest"
	SettingNotifyCommunicationsEnabled        = "settingNotifyCommunicationsEnabled"
	SettingNotifyCommunicationsEnabledDefault = false
	SettingNotifyContractsEarliest            = "settingNotifyContractsEarliest"
	SettingNotifyContractsEnabled             = "settingNotifyContractsEnabled"
	SettingNotifyContractsEnabledDefault      = false
	SettingNotifyMailsEarliest                = "settingNotifyMailsEarliest"
	SettingNotifyMailsEnabled                 = "settingNotifyMailsEnabled"
	SettingNotifyMailsEnabledDefault          = false
	SettingNotifyPIEarliest                   = "settingNotifyPIEarliest"
	SettingNotifyPIEnabled                    = "settingNotifyPIEnabled"
	SettingNotifyPIEnabledDefault             = false
	SettingNotifyTimeoutHours                 = "settingNotifyTimeoutHours"
	SettingNotifyTimeoutHoursDefault          = 30 * 24
	SettingNotifyTimeoutHoursMax              = 90 * 24
	SettingNotifyTrainingEarliest             = "settingNotifyTrainingEarliest"
	SettingNotifyTrainingEnabled              = "settingNotifyTrainingEnabled"
	SettingNotifyTrainingEnabledDefault       = false
)

// SettingKeys returns all setting keys. Mostly to know what to delete.
func SettingKeys() []string {
	return []string{
		SettingLastCharacterID,
		SettingMaxMails,
		SettingMaxWalletTransactions,
		SettingNotificationsTypesEnabled,
		SettingNotifyCommunicationsEnabled,
		SettingNotifyCommunicationsEarliest,
		SettingNotifyContractsEnabled,
		SettingNotifyContractsEarliest,
		SettingNotifyMailsEnabled,
		SettingNotifyMailsEarliest,
		SettingNotifyPIEnabled,
		SettingNotifyPIEarliest,
		SettingNotifyTimeoutHours,
		SettingNotifyTrainingEnabled,
		SettingNotifyTrainingEarliest,
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

func (u *BaseUI) MakeGeneralSettingsPage(w fyne.Window) (fyne.CanvasObject, []*fyne.MenuItem) {
	if w == nil {
		w = u.Window
	}

	// log level
	logLevel := widget.NewSelect(LogLevelNames(), func(s string) {
		u.FyneApp.Preferences().SetString(SettingLogLevel, s)
		slog.SetLogLoggerLevel(LogLevelName2Level(s))
	})
	logLevelSelected := u.FyneApp.Preferences().StringWithFallback(
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
				if err := u.EveImageService.ClearCache(); err != nil {
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
	if u.IsMobile() {
		settings.Orientation = widget.Vertical
	}
	reset := &fyne.MenuItem{
		Label: "Reset to defaults",
		Action: func() {
			logLevel.SetSelected(SettingLogLevelDefault)
		},
	}
	return settings, []*fyne.MenuItem{reset}
}

func (u *BaseUI) MakeEVEOnlinePage() (fyne.CanvasObject, []*fyne.MenuItem) {
	// max mails
	maxMails := kxwidget.NewSlider(0, SettingMaxMailsMax)
	v1 := u.FyneApp.Preferences().IntWithFallback(SettingMaxMails, SettingMaxMailsDefault)
	maxMails.SetValue(float64(v1))
	maxMails.OnChangeEnded = func(v float64) {
		u.FyneApp.Preferences().SetInt(SettingMaxMails, int(v))
	}

	// max transactions
	maxTransactions := kxwidget.NewSlider(0, SettingMaxWalletTransactionsMax)
	v2 := u.FyneApp.Preferences().IntWithFallback(SettingMaxWalletTransactions, SettingMaxWalletTransactionsDefault)
	maxTransactions.SetValue(float64(v2))
	maxTransactions.OnChangeEnded = func(v float64) {
		u.FyneApp.Preferences().SetInt(SettingMaxWalletTransactions, int(v))
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
	if u.IsMobile() {
		settings.Orientation = widget.Vertical
	}
	reset := &fyne.MenuItem{
		Label: "Reset to defaults",
		Action: func() {
			maxMails.SetValue(SettingMaxMailsDefault)
			maxTransactions.SetValue(SettingMaxWalletTransactionsDefault)
		},
	}
	return settings, []*fyne.MenuItem{reset}
}

func (u *BaseUI) MakeNotificationGeneralPage(w fyne.Window) (fyne.CanvasObject, []*fyne.MenuItem) {
	if w == nil {
		w = u.Window
	}
	form := widget.NewForm()
	if u.IsMobile() {
		form.Orientation = widget.Vertical
	}

	// mail toogle
	mailEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		u.FyneApp.Preferences().SetBool(SettingNotifyMailsEnabled, on)
		if on {
			u.FyneApp.Preferences().SetString(SettingNotifyMailsEarliest, time.Now().Format(time.RFC3339))
		}
	})
	mailEnabledCheck.SetState(u.FyneApp.Preferences().BoolWithFallback(
		SettingNotifyMailsEnabled,
		SettingNotifyMailsEnabledDefault,
	))
	form.AppendItem(&widget.FormItem{
		Text:     "Mail",
		Widget:   mailEnabledCheck,
		HintText: "Wether to notify new mails",
	})

	// communications toogle
	communicationsEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		u.FyneApp.Preferences().SetBool(SettingNotifyCommunicationsEnabled, on)
		if on {
			u.FyneApp.Preferences().SetString(SettingNotifyCommunicationsEarliest, time.Now().Format(time.RFC3339))
		}
	})
	communicationsEnabledCheck.SetState(u.FyneApp.Preferences().BoolWithFallback(
		SettingNotifyCommunicationsEnabled,
		SettingNotifyCommunicationsEnabledDefault,
	))
	form.AppendItem(&widget.FormItem{
		Text:     "Communications",
		Widget:   communicationsEnabledCheck,
		HintText: "Wether to notify new communications",
	})

	// PI toogle
	piEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		u.FyneApp.Preferences().SetBool(SettingNotifyPIEnabled, on)
		if on {
			u.FyneApp.Preferences().SetString(SettingNotifyPIEarliest, time.Now().Format(time.RFC3339))
		}
	})
	piEnabledCheck.SetState(u.FyneApp.Preferences().BoolWithFallback(
		SettingNotifyPIEnabled,
		SettingNotifyPIEnabledDefault,
	))
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
			err := u.CharacterService.EnableAllTrainingWatchers(ctx)
			if err != nil {
				d := NewErrorDialog("failed to enable training notification", err, w)
				d.Show()
			} else {
				u.FyneApp.Preferences().SetBool(SettingNotifyTrainingEnabled, true)
			}
		} else {
			err := u.CharacterService.DisableAllTrainingWatchers(ctx)
			if err != nil {
				d := NewErrorDialog("failed to disable training notification", err, w)
				d.Show()
			} else {
				u.FyneApp.Preferences().SetBool(SettingNotifyTrainingEnabled, false)
			}
		}
	})
	trainingEnabledCheck.SetState(u.FyneApp.Preferences().BoolWithFallback(
		SettingNotifyTrainingEnabled,
		SettingNotifyTrainingEnabledDefault,
	))
	form.AppendItem(&widget.FormItem{
		Text:     "Training",
		Widget:   trainingEnabledCheck,
		HintText: "Wether to notify when skillqueue is empty",
	})

	// Contracts toogle
	contractsEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		u.FyneApp.Preferences().SetBool(SettingNotifyContractsEnabled, on)
		if on {
			u.FyneApp.Preferences().SetString(SettingNotifyContractsEarliest, time.Now().Format(time.RFC3339))
		}
	})
	contractsEnabledCheck.SetState(u.FyneApp.Preferences().BoolWithFallback(
		SettingNotifyContractsEnabled,
		SettingNotifyCommunicationsEnabledDefault,
	))
	form.AppendItem(&widget.FormItem{
		Text:     "Contracts",
		Widget:   contractsEnabledCheck,
		HintText: "Wether to notify when contract status changes",
	})

	// notify timeout
	notifyTimeout := kxwidget.NewSlider(1, SettingNotifyTimeoutHoursMax)
	v := u.FyneApp.Preferences().IntWithFallback(SettingNotifyTimeoutHours, SettingNotifyTimeoutHoursDefault)
	notifyTimeout.SetValue(float64(v))
	notifyTimeout.OnChangeEnded = func(v float64) {
		u.FyneApp.Preferences().SetInt(SettingNotifyTimeoutHours, int(v))
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
			u.FyneApp.SendNotification(n)
		}),
		HintText: "Send a test notification to verify it works",
	})

	reset := &fyne.MenuItem{
		Label: "Reset to defaults",
		Action: func() {
			mailEnabledCheck.SetState(SettingNotifyMailsEnabledDefault)
			communicationsEnabledCheck.SetState(SettingNotifyCommunicationsEnabledDefault)
			piEnabledCheck.SetState(SettingNotifyPIEnabledDefault)
			trainingEnabledCheck.SetState(SettingNotifyTrainingEnabledDefault)
			contractsEnabledCheck.SetState(SettingNotifyTrainingEnabledDefault)
			notifyTimeout.SetValue(SettingNotifyTimeoutHoursDefault)
		},
	}
	return form, []*fyne.MenuItem{reset}
}

func (u *BaseUI) MakeNotificationTypesPage(w fyne.Window) (fyne.CanvasObject, []*fyne.MenuItem) {
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

	typesEnabled := set.NewFromSlice(u.FyneApp.Preferences().StringList(SettingNotificationsTypesEnabled))
	p := theme.Padding()
	list := widget.NewList(
		func() int {
			return len(categories)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabel("Title")
			title.Truncation = fyne.TextTruncateEllipsis
			sub := widgets.NewLabelWithSize("Enabled", theme.SizeNameCaptionText)
			sub.Wrapping = fyne.TextWrapBreak
			return container.NewBorder(
				nil, nil, nil, nil,
				container.NewBorder(
					container.New(layout.NewCustomPaddedLayout(0, -p, 0, 0), title),
					nil,
					nil,
					nil,
					container.New(layout.NewCustomPaddedLayout(-p, 0, 0, 0), sub),
				),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			c := categories[id]
			outer := co.(*fyne.Container).Objects
			inner := outer[0].(*fyne.Container).Objects
			title := inner[1].(*fyne.Container).Objects[0].(*widget.Label)
			title.SetText(c.String())
			sub := inner[0].(*fyne.Container).Objects[0].(*widgets.Label)
			var enabled int
			for _, n := range categoriesAndTypes[c] {
				if x := n.String(); typesEnabled.Contains(x) {
					enabled++
				}
			}
			var s string
			switch enabled {
			case 0:
				s = "Off"
			case len(categoriesAndTypes[c]):
				s = "All"
			default:
				s = "Some"
			}
			sub.SetText(s)
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
					u.FyneApp.Preferences().SetStringList(SettingNotificationsTypesEnabled, typesEnabled.ToSlice())
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
			u.FyneApp.Preferences().SetStringList(SettingNotificationsTypesEnabled, typesEnabled.ToSlice())
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
						u.FyneApp.Preferences().SetStringList(SettingNotificationsTypesEnabled, typesEnabled.ToSlice())
						list2.Refresh()
					}),
					fyne.NewMenuItem(
						"Disable all",
						func() {
							for _, nt := range categoriesAndTypes[f] {
								typesEnabled.Remove(nt.String())
							}
							u.FyneApp.Preferences().SetStringList(SettingNotificationsTypesEnabled, typesEnabled.ToSlice())
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

		d = dialog.NewCustom("Notification Types", "Close", c, w)
		d.Show()
		d.Resize(fyne.NewSize(400, 600))
		d.SetOnClosed(func() {
			list.Refresh()
		})
	}

	updateTypes := func() {
		u.FyneApp.Preferences().SetStringList(SettingNotificationsTypesEnabled, typesEnabled.ToSlice())
		list.Refresh()
	}
	none := &fyne.MenuItem{
		Label: "Disable all",
		Action: func() {
			typesEnabled.Clear()
			updateTypes()
		},
	}
	all := &fyne.MenuItem{
		Label: "Enable all",
		Action: func() {
			for _, nt := range evenotification.SupportedTypes() {
				typesEnabled.Add(nt.String())
			}
			updateTypes()
		},
	}
	return list, []*fyne.MenuItem{none, all}
}
