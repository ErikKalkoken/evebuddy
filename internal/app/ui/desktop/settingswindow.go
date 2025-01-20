package desktop

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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type settingsWindow struct {
	content fyne.CanvasObject
	u       *DesktopUI
	window  fyne.Window
}

func (u *DesktopUI) showSettingsWindow() {
	if u.settingsWindow != nil {
		u.settingsWindow.Show()
		return
	}
	w := u.FyneApp.NewWindow(u.makeWindowTitle("Settings"))
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

func (u *DesktopUI) newSettingsWindow() *settingsWindow {
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
		w.u.FyneApp.Preferences().SetBool(settingSysTrayEnabled, b)
	})
	sysTrayEnabled := w.u.FyneApp.Preferences().BoolWithFallback(
		settingSysTrayEnabled,
		settingSysTrayEnabledDefault,
	)
	sysTrayCheck.SetState(sysTrayEnabled)

	// log level
	logLevel := widget.NewSelect(ui.LogLevelNames(), func(s string) {
		w.u.FyneApp.Preferences().SetString(ui.SettingLogLevel, s)
		slog.SetLogLoggerLevel(ui.LogLevelName2Level(s))
	})
	logLevelSelected := w.u.FyneApp.Preferences().StringWithFallback(
		ui.SettingLogLevel,
		ui.SettingLogLevelDefault,
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
			d := ui.NewErrorDialog("Failed to clear image cache", err, w.u.Window)
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
		logLevel.SetSelected(ui.SettingLogLevelDefault)
	}
	return makePage("General settings", settings, reset)
}

func (w *settingsWindow) makeEVEOnlinePage() fyne.CanvasObject {
	// max mails
	maxMails := kxwidget.NewSlider(0, ui.SettingMaxMailsMax)
	v1 := w.u.FyneApp.Preferences().IntWithFallback(ui.SettingMaxMails, ui.SettingMaxMailsDefault)
	maxMails.SetValue(float64(v1))
	maxMails.OnChangeEnded = func(v float64) {
		w.u.FyneApp.Preferences().SetInt(ui.SettingMaxMails, int(v))
	}

	// max transactions
	maxTransactions := kxwidget.NewSlider(0, ui.SettingMaxWalletTransactionsMax)
	v2 := w.u.FyneApp.Preferences().IntWithFallback(ui.SettingMaxWalletTransactions, ui.SettingMaxWalletTransactionsDefault)
	maxTransactions.SetValue(float64(v2))
	maxTransactions.OnChangeEnded = func(v float64) {
		w.u.FyneApp.Preferences().SetInt(ui.SettingMaxWalletTransactions, int(v))
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
		maxMails.SetValue(ui.SettingMaxMailsDefault)
		maxTransactions.SetValue(ui.SettingMaxWalletTransactionsDefault)
	}
	return makePage("Eve Online settings", settings, x)
}

func (w *settingsWindow) makeNotificationPage() fyne.CanvasObject {
	f1 := widget.NewForm()

	// mail toogle
	mailEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		w.u.FyneApp.Preferences().SetBool(ui.SettingNotifyMailsEnabled, on)
		if on {
			w.u.FyneApp.Preferences().SetString(ui.SettingNotifyMailsEarliest, time.Now().Format(time.RFC3339))
		}
	})
	mailEnabledCheck.SetState(w.u.FyneApp.Preferences().BoolWithFallback(
		ui.SettingNotifyMailsEnabled,
		ui.SettingNotifyMailsEnabledDefault,
	))
	f1.AppendItem(&widget.FormItem{
		Text:     "Mail",
		Widget:   mailEnabledCheck,
		HintText: "Wether to notify new mails",
	})

	// communications toogle
	communicationsEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		w.u.FyneApp.Preferences().SetBool(ui.SettingNotifyCommunicationsEnabled, on)
		if on {
			w.u.FyneApp.Preferences().SetString(ui.SettingNotifyCommunicationsEarliest, time.Now().Format(time.RFC3339))
		}
	})
	communicationsEnabledCheck.SetState(w.u.FyneApp.Preferences().BoolWithFallback(
		ui.SettingNotifyCommunicationsEnabled,
		ui.SettingNotifyCommunicationsEnabledDefault,
	))
	f1.AppendItem(&widget.FormItem{
		Text:     "Communications",
		Widget:   communicationsEnabledCheck,
		HintText: "Wether to notify new communications",
	})

	// PI toogle
	piEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		w.u.FyneApp.Preferences().SetBool(ui.SettingNotifyPIEnabled, on)
		if on {
			w.u.FyneApp.Preferences().SetString(ui.SettingNotifyPIEarliest, time.Now().Format(time.RFC3339))
		}
	})
	piEnabledCheck.SetState(w.u.FyneApp.Preferences().BoolWithFallback(
		ui.SettingNotifyPIEnabled,
		ui.SettingNotifyPIEnabledDefault,
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
				d := ui.NewErrorDialog("failed to enable training notification", err, w.window)
				d.Show()
			} else {
				w.u.FyneApp.Preferences().SetBool(ui.SettingNotifyTrainingEnabled, true)
			}
		} else {
			err := w.u.CharacterService.DisableAllTrainingWatchers(ctx)
			if err != nil {
				d := ui.NewErrorDialog("failed to disable training notification", err, w.window)
				d.Show()
			} else {
				w.u.FyneApp.Preferences().SetBool(ui.SettingNotifyTrainingEnabled, false)
			}
		}
	})
	trainingEnabledCheck.SetState(w.u.FyneApp.Preferences().BoolWithFallback(
		ui.SettingNotifyTrainingEnabled,
		ui.SettingNotifyTrainingEnabledDefault,
	))
	f1.AppendItem(&widget.FormItem{
		Text:     "Training",
		Widget:   trainingEnabledCheck,
		HintText: "Wether to notify when skillqueue is empty",
	})

	// Contracts toogle
	contractsEnabledCheck := kxwidget.NewSwitch(func(on bool) {
		w.u.FyneApp.Preferences().SetBool(ui.SettingNotifyContractsEnabled, on)
		if on {
			w.u.FyneApp.Preferences().SetString(ui.SettingNotifyContractsEarliest, time.Now().Format(time.RFC3339))
		}
	})
	contractsEnabledCheck.SetState(w.u.FyneApp.Preferences().BoolWithFallback(
		ui.SettingNotifyContractsEnabled,
		ui.SettingNotifyCommunicationsEnabledDefault,
	))
	f1.AppendItem(&widget.FormItem{
		Text:     "Contracts",
		Widget:   contractsEnabledCheck,
		HintText: "Wether to notify when contract status changes",
	})

	// notify timeout
	notifyTimeout := kxwidget.NewSlider(1, ui.SettingNotifyTimeoutHoursMax)
	v := w.u.FyneApp.Preferences().IntWithFallback(ui.SettingNotifyTimeoutHours, ui.SettingNotifyTimeoutHoursDefault)
	notifyTimeout.SetValue(float64(v))
	notifyTimeout.OnChangeEnded = func(v float64) {
		w.u.FyneApp.Preferences().SetInt(ui.SettingNotifyTimeoutHours, int(v))
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
	typesEnabled := set.NewFromSlice(w.u.FyneApp.Preferences().StringList(ui.SettingNotificationsTypesEnabled))
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
				w.u.FyneApp.Preferences().SetStringList(ui.SettingNotificationsTypesEnabled, typesEnabled.ToSlice())
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
		mailEnabledCheck.SetState(ui.SettingNotifyMailsEnabledDefault)
		communicationsEnabledCheck.SetState(ui.SettingNotifyCommunicationsEnabledDefault)
		piEnabledCheck.SetState(ui.SettingNotifyPIEnabledDefault)
		trainingEnabledCheck.SetState(ui.SettingNotifyTrainingEnabledDefault)
		contractsEnabledCheck.SetState(ui.SettingNotifyTrainingEnabledDefault)
		notifyTimeout.SetValue(ui.SettingNotifyTimeoutHoursDefault)
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
