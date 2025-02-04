package ui

import (
	"context"
	"log/slog"
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

type settingsWindow struct {
	content fyne.CanvasObject
	u       *UI
	window  fyne.Window
}

func (u *UI) showSettingsWindow() {
	if u.settingsWindow != nil {
		u.settingsWindow.Show()
		return
	}
	w := u.fyneApp.NewWindow(u.makeWindowTitle("Settings"))
	sw := u.newSettingsWindow()
	w.SetContent(sw.content)
	w.Resize(fyne.Size{Width: 700, Height: 500})
	w.SetOnClosed(func() {
		u.settingsWindow = nil
	})
	u.settingsWindow = w
	sw.window = w
	w.Show()
}

func (u *UI) newSettingsWindow() *settingsWindow {
	sw := &settingsWindow{u: u}
	tabs := container.NewAppTabs(
		container.NewTabItem("General", sw.makeGeneralPage()),
		container.NewTabItem("Eve Online", sw.makeEVEOnlinePage()),
		container.NewTabItem("Notifications", sw.makeNotificationPage()),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	sw.content = tabs
	return sw
}

func (w *settingsWindow) makeGeneralPage() fyne.CanvasObject {
	// system tray
	sysTrayCheck := kxwidget.NewSwitch(func(b bool) {
		w.u.fyneApp.Preferences().SetBool(settingSysTrayEnabled, b)
	})
	sysTrayEnabled := w.u.fyneApp.Preferences().BoolWithFallback(
		settingSysTrayEnabled,
		settingSysTrayEnabledDefault,
	)
	sysTrayCheck.SetState(sysTrayEnabled)

	// log level
	logLevel := widget.NewSelect(LogLevelNames(), func(s string) {
		w.u.fyneApp.Preferences().SetString(SettingLogLevel, s)
		slog.SetLogLoggerLevel(LogLevelName2Level(s))
	})
	logLevelSelected := w.u.fyneApp.Preferences().StringWithFallback(
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
				if err := w.u.EveImageService.ClearCache(); err != nil {
					return err
				}
				slog.Info("Cleared image cache")
				return nil
			},
			w.window,
		)
		m.OnSuccess = func() {
			d := dialog.NewInformation("Image cache", "Image cache cleared", w.window)
			d.Show()
		}
		m.OnError = func(err error) {
			slog.Error("Failed to clear image cache", "error", err)
			d := NewErrorDialog("Failed to clear image cache", err, w.u.window)
			d.Show()
		}
		m.Start()
	})
	settings := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:     "Close button",
				Widget:   sysTrayCheck,
				HintText: "App will minimize to system tray when closed (requires restart)",
			},
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
	reset := func() {
		sysTrayCheck.SetState(settingSysTrayEnabledDefault)
		logLevel.SetSelected(SettingLogLevelDefault)
	}
	return makePage("General settings", settings, reset)
}

func (w *settingsWindow) makeEVEOnlinePage() fyne.CanvasObject {
	// max mails
	maxMails := kxwidget.NewSlider(0, settingMaxMailsMax)
	v1 := w.u.fyneApp.Preferences().IntWithFallback(settingMaxMails, settingMaxMailsDefault)
	maxMails.SetValue(float64(v1))
	maxMails.OnChangeEnded = func(v float64) {
		w.u.fyneApp.Preferences().SetInt(settingMaxMails, int(v))
	}

	// max transactions
	maxTransactions := kxwidget.NewSlider(0, settingMaxWalletTransactionsMax)
	v2 := w.u.fyneApp.Preferences().IntWithFallback(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault)
	maxTransactions.SetValue(float64(v2))
	maxTransactions.OnChangeEnded = func(v float64) {
		w.u.fyneApp.Preferences().SetInt(settingMaxWalletTransactions, int(v))
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
	x := func() {
		maxMails.SetValue(settingMaxMailsDefault)
		maxTransactions.SetValue(settingMaxWalletTransactionsDefault)
	}
	return makePage("Eve Online settings", settings, x)
}

func (w *settingsWindow) makeNotificationPage() fyne.CanvasObject {
	f1 := widget.NewForm()

	// mail toogle
	mailEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		w.u.fyneApp.Preferences().SetBool(settingNotifyMailsEnabled, on)
		if on {
			w.u.fyneApp.Preferences().SetString(settingNotifyMailsEarliest, time.Now().Format(time.RFC3339))
		}
	})
	mailEnabledCheck.SetState(w.u.fyneApp.Preferences().BoolWithFallback(
		settingNotifyMailsEnabled,
		settingNotifyMailsEnabledDefault,
	))
	f1.AppendItem(&widget.FormItem{
		Text:     "Mail",
		Widget:   mailEnabledCheck,
		HintText: "Wether to notify new mails",
	})

	// communications toogle
	communicationsEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		w.u.fyneApp.Preferences().SetBool(settingNotifyCommunicationsEnabled, on)
		if on {
			w.u.fyneApp.Preferences().SetString(settingNotifyCommunicationsEarliest, time.Now().Format(time.RFC3339))
		}
	})
	communicationsEnabledCheck.SetState(w.u.fyneApp.Preferences().BoolWithFallback(
		settingNotifyCommunicationsEnabled,
		settingNotifyCommunicationsEnabledDefault,
	))
	f1.AppendItem(&widget.FormItem{
		Text:     "Communications",
		Widget:   communicationsEnabledCheck,
		HintText: "Wether to notify new communications",
	})

	// PI toogle
	piEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		w.u.fyneApp.Preferences().SetBool(settingNotifyPIEnabled, on)
		if on {
			w.u.fyneApp.Preferences().SetString(settingNotifyPIEarliest, time.Now().Format(time.RFC3339))
		}
	})
	piEnabledCheck.SetState(w.u.fyneApp.Preferences().BoolWithFallback(
		settingNotifyPIEnabled,
		settingNotifyPIEnabledDefault,
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
			err := w.u.CharacterService.EnableAllTrainingWatchers(ctx)
			if err != nil {
				d := NewErrorDialog("failed to enable training notification", err, w.window)
				d.Show()
			} else {
				w.u.fyneApp.Preferences().SetBool(settingNotifyTrainingEnabled, true)
			}
		} else {
			err := w.u.CharacterService.DisableAllTrainingWatchers(ctx)
			if err != nil {
				d := NewErrorDialog("failed to disable training notification", err, w.window)
				d.Show()
			} else {
				w.u.fyneApp.Preferences().SetBool(settingNotifyTrainingEnabled, false)
			}
		}
	})
	trainingEnabledCheck.SetState(w.u.fyneApp.Preferences().BoolWithFallback(
		settingNotifyTrainingEnabled,
		settingNotifyTrainingEnabledDefault,
	))
	f1.AppendItem(&widget.FormItem{
		Text:     "Training",
		Widget:   trainingEnabledCheck,
		HintText: "Wether to notify when skillqueue is empty",
	})

	// Contracts toogle
	contractsEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		w.u.fyneApp.Preferences().SetBool(settingNotifyContractsEnabled, on)
		if on {
			w.u.fyneApp.Preferences().SetString(settingNotifyContractsEarliest, time.Now().Format(time.RFC3339))
		}
	})
	contractsEnabledCheck.SetState(w.u.fyneApp.Preferences().BoolWithFallback(
		settingNotifyContractsEnabled,
		settingNotifyCommunicationsEnabledDefault,
	))
	f1.AppendItem(&widget.FormItem{
		Text:     "Contracts",
		Widget:   contractsEnabledCheck,
		HintText: "Wether to notify when contract status changes",
	})

	// notify timeout
	notifyTimeout := kxwidget.NewSlider(1, settingNotifyTimeoutHoursMax)
	v := w.u.fyneApp.Preferences().IntWithFallback(settingNotifyTimeoutHours, settingNotifyTimeoutHoursDefault)
	notifyTimeout.SetValue(float64(v))
	notifyTimeout.OnChangeEnded = func(v float64) {
		w.u.fyneApp.Preferences().SetInt(settingNotifyTimeoutHours, int(v))
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
	typesEnabled := set.NewFromSlice(w.u.fyneApp.Preferences().StringList(settingNotificationsTypesEnabled))
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
				w.u.fyneApp.Preferences().SetStringList(settingNotificationsTypesEnabled, typesEnabled.ToSlice())
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
		mailEnabledCheck.SetState(settingNotifyMailsEnabledDefault)
		communicationsEnabledCheck.SetState(settingNotifyCommunicationsEnabledDefault)
		piEnabledCheck.SetState(settingNotifyPIEnabledDefault)
		trainingEnabledCheck.SetState(settingNotifyTrainingEnabledDefault)
		contractsEnabledCheck.SetState(settingNotifyTrainingEnabledDefault)
		notifyTimeout.SetValue(settingNotifyTimeoutHoursDefault)
		for _, sw := range notifsAll {
			sw.SetState(false)
		}
	}
	return makePage("Notification settings", c, reset)
}

func makePage(title string, content fyne.CanvasObject, resetSettings func()) fyne.CanvasObject {
	l := widget.NewLabel(title)
	l.Importance = widget.HighImportance
	l.TextStyle.Bold = true
	return container.NewBorder(
		container.NewVBox(l, widget.NewSeparator()),
		container.NewHBox(widget.NewButton("Reset", resetSettings)),
		nil,
		nil,
		container.NewVScroll(content),
	)
}
