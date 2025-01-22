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
	"fyne.io/fyne/v2/widget"
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
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

func (u *BaseUI) MakeGeneralSettingsPage(w fyne.Window) (fyne.CanvasObject, func()) {
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
	if !u.IsDesktop() {
		settings.Orientation = widget.Vertical
	}
	reset := func() {
		logLevel.SetSelected(SettingLogLevelDefault)
	}
	return settings, reset
}

func (u *BaseUI) MakeEVEOnlinePage() (fyne.CanvasObject, func()) {
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
				HintText: "Maximum number of mails downloaded. 0 = unlimited.",
			},
			{
				Text:     "Maximum wallet transaction",
				Widget:   maxTransactions,
				HintText: "Maximum number of wallet transaction downloaded. 0 = unlimited.",
			},
		},
	}
	if !u.IsDesktop() {
		settings.Orientation = widget.Vertical
	}
	x := func() {
		maxMails.SetValue(SettingMaxMailsDefault)
		maxTransactions.SetValue(SettingMaxWalletTransactionsDefault)
	}
	return settings, x
}

func (u *BaseUI) MakeNotificationPage(w fyne.Window) (fyne.CanvasObject, func()) {
	if w == nil {
		w = u.Window
	}
	f1 := widget.NewForm()

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
	f1.AppendItem(&widget.FormItem{
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
	f1.AppendItem(&widget.FormItem{
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
	f1.AppendItem(&widget.FormItem{
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
	f1.AppendItem(&widget.FormItem{
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
	f1.AppendItem(&widget.FormItem{
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
	f1.AppendItem(&widget.FormItem{
		Text:     "Notification timeout",
		Widget:   notifyTimeout,
		HintText: "Events older then this value in hours will not be notified",
	})

	// communications types
	f2 := widget.NewForm()
	categoriesAndTypes := make(map[evenotification.Category][]evenotification.Type)
	for _, n := range evenotification.SupportedTypes() {
		c := evenotification.Type2category[n]
		categoriesAndTypes[c] = append(categoriesAndTypes[c], n)
	}
	categories := make([]evenotification.Category, 0)
	for c := range categoriesAndTypes {
		categories = append(categories, c)
	}
	slices.Sort(categories)
	typesEnabled := set.NewFromSlice(u.FyneApp.Preferences().StringList(SettingNotificationsTypesEnabled))
	notifsAll := make([]*kxwidget.Switch, 0)
	for _, c := range categories {
		f2.Append("", widget.NewLabel(c.String()))
		nts := categoriesAndTypes[c]
		notifsCategory := make([]*kxwidget.Switch, 0)
		for _, nt := range nts {
			sw := kxwidget.NewSwitch(func(on bool) {
				if on {
					typesEnabled.Add(nt.String())
				} else {
					typesEnabled.Remove(nt.String())
				}
				u.FyneApp.Preferences().SetStringList(SettingNotificationsTypesEnabled, typesEnabled.ToSlice())
			})
			if typesEnabled.Contains(nt.String()) {
				sw.On = true
			}
			f2.AppendItem(widget.NewFormItem(nt.Display(), sw))
			notifsCategory = append(notifsCategory, sw)
			notifsAll = append(notifsAll, sw)
		}
		enableAll := widget.NewButton("Enable all", func() {
			for _, sw := range notifsCategory {
				sw.SetState(true)
			}
		})
		disableAll := widget.NewButton("Disable all", func() {
			for _, sw := range notifsCategory {
				sw.SetState(false)
			}
		})
		f2.Append("", container.NewHBox(enableAll, disableAll))
		f2.Append("", container.NewPadded())
	}
	if !u.IsDesktop() {
		f1.Orientation = widget.Vertical
		f2.Orientation = widget.Vertical
	}

	title1 := widget.NewLabel("Global")
	title1.TextStyle.Bold = true
	title2 := widget.NewLabel("Communication Types")
	title2.TextStyle.Bold = true
	c := container.NewVBox(
		title1,
		f1,
		container.NewPadded(),
		title2,
		f2,
	)
	reset := func() {
		mailEnabledCheck.SetState(SettingNotifyMailsEnabledDefault)
		communicationsEnabledCheck.SetState(SettingNotifyCommunicationsEnabledDefault)
		piEnabledCheck.SetState(SettingNotifyPIEnabledDefault)
		trainingEnabledCheck.SetState(SettingNotifyTrainingEnabledDefault)
		contractsEnabledCheck.SetState(SettingNotifyTrainingEnabledDefault)
		notifyTimeout.SetValue(SettingNotifyTimeoutHoursDefault)
		for _, sw := range notifsAll {
			sw.SetState(false)
		}
	}
	return c, reset
}
