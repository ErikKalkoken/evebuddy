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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

type settingAction struct {
	Label  string
	Action func()
}
type userSettings struct {
	widget.BaseWidget

	sb *iwidget.Snackbar
	u  *baseUI
	w  fyne.Window
}

func showSettingsWindow(u *baseUI) {
	w, ok, onClosed := u.getOrCreateWindowWithOnClosed("user-settings", "Settings")
	if !ok {
		w.Show()
		return
	}
	a := newSettings(u, w)
	w.SetContent(fynetooltip.AddWindowToolTipLayer(a, w.Canvas()))
	w.Resize(fyne.Size{Width: 700, Height: 500})
	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		a.sb.Stop()
	})
	w.SetCloseIntercept(func() {
		w.Close()
		fynetooltip.DestroyWindowToolTipLayer(w.Canvas())
	})
	w.Show()
}

func newSettings(u *baseUI, w fyne.Window) *userSettings {
	a := &userSettings{
		sb: iwidget.NewSnackbar(w),
		u:  u,
		w:  w,
	}
	a.ExtendBaseWidget(a)
	a.sb.Start()
	return a
}

func (a *userSettings) CreateRenderer() fyne.WidgetRenderer {
	makeSettingsPage := func(title string, content fyne.CanvasObject, actions fyne.CanvasObject) fyne.CanvasObject {
		ab := iwidget.NewAppBar(title, content, actions)
		ab.HideBackground = !a.u.isMobile
		return ab
	}
	generalContent, generalActions := a.makeGeneralPage()
	notificationContent, notificationActions := a.makeNotificationPage()
	tabs := container.NewAppTabs(
		container.NewTabItem("General", makeSettingsPage(
			"General",
			generalContent,
			generalActions,
		)),
		container.NewTabItem("Notifications", makeSettingsPage(
			"Notifications",
			notificationContent,
			notificationActions,
		)),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	return widget.NewSimpleRenderer(tabs)
}

func (a *userSettings) makeGeneralPage() (fyne.CanvasObject, *kxwidget.IconButton) {
	logLevel := NewSettingItemOptions(SettingItemOptions{
		label:        "Log level",
		hint:         "Set current log level",
		options:      a.u.settings.LogLevelNames(),
		defaultValue: a.u.settings.LogLevelDefault(),
		getter:       a.u.settings.LogLevel,
		setter: func(v string) {
			s := a.u.settings
			s.SetLogLevel(v)
			slog.SetLogLoggerLevel(s.LogLevelSlog())
		},
		isMobile: a.u.isMobile,
		window:   a.w,
	})

	developerMode := NewSettingItemSwitch(SettingItemSwitch{
		label:     "Developer Mode",
		hint:      "App shows additional technical information like Character IDs",
		getter:    a.u.settings.DeveloperMode,
		onChanged: a.u.settings.SetDeveloperMode,
	})

	items := []SettingItem{
		NewSettingItemHeading("Application"),
		logLevel,
		developerMode,
	}

	sysTray := NewSettingItemSwitch(SettingItemSwitch{
		defaultValue: a.u.settings.SysTrayEnabledDefault(),
		label:        "Run in background",
		hint:         "App will continue to run in background after window is closed (requires restart)",
		getter:       a.u.settings.SysTrayEnabled,
		onChanged:    a.u.settings.SetSysTrayEnabled,
	})
	if !a.u.isMobile {
		items = append(items, sysTray)
	}

	preferMarketTab := NewSettingItemSwitch(SettingItemSwitch{
		label:     "Prefer market tab",
		hint:      "Show market tab first for tradeable items",
		getter:    a.u.settings.PreferMarketTab,
		onChanged: a.u.settings.SetPreferMarketTab,
	})
	hideLimitedCorporations := NewSettingItemSwitch(SettingItemSwitch{
		defaultValue: a.u.settings.HideLimitedCorporationsDefault(),
		label:        "Hide limited corporations",
		hint:         "Hide corporations with no privileged access, e.g. corporation wallet",
		getter:       a.u.settings.HideLimitedCorporations,
		onChanged: func(enabled bool) {
			a.u.settings.SetHideLimitedCorporations(enabled)
			go a.u.corporationsChanged.Emit(context.Background(), struct{}{})
		},
	})

	items = slices.Concat(items, []SettingItem{
		NewSettingItemHeading("UI"),
		preferMarketTab,
		hideLimitedCorporations,
	})

	colorTheme := NewSettingItemOptions(SettingItemOptions{
		label:        "Appearance",
		hint:         "Choose the color scheme. 'Auto' uses the current OS theme.",
		options:      []string{string(settings.Auto), string(settings.Light), string(settings.Dark)},
		defaultValue: string(a.u.settings.ColorThemeDefault()),
		getter: func() string {
			return string(a.u.settings.ColorTheme())
		},
		setter: func(v string) {
			s := a.u.settings
			s.SetColorTheme(settings.ColorTheme(v))
			a.u.setColorTheme(settings.ColorTheme(v))
		},
		isMobile: a.u.isMobile,
		window:   a.w,
	})

	fyneScale := NewSettingItemSlider(SettingItemSliderParams{
		label:        "UI Scale",
		hint:         "Scaling factor of the user interface in percent. Requires restart.",
		minValue:     50,
		maxValue:     200,
		defaultValue: a.u.settings.FyneScaleDefault() * 100,
		step:         5,
		getter: func() float64 {
			return a.u.settings.FyneScale() * 100
		},
		setter: func(v float64) {
			a.u.settings.SetFyneScale(v / 100.0)
		},
		formatter: func(v any) string {
			return fmt.Sprintf("%v %%", v)
		},
		isMobile: a.u.isMobile,
		window:   a.w,
	})

	disableDPIDetection := NewSettingItemSwitch(SettingItemSwitch{
		label:     "Disable DPI detection",
		hint:      "Disables the automatic DPI detection. Requires restart.",
		getter:    a.u.settings.DisableDPIDetection,
		onChanged: a.u.settings.SetDisableDPIDetection,
	})

	if !a.u.isMobile {
		items = slices.Concat(items, []SettingItem{
			colorTheme,
			fyneScale,
			disableDPIDetection,
		})
	}

	vMin, vMax, vDef := a.u.settings.MaxMailsPresets()
	maxMail := NewSettingItemSlider(SettingItemSliderParams{
		label:        "Maximum mails",
		hint:         "Max number of mails downloaded. 0 = unlimited.",
		minValue:     float64(vMin),
		maxValue:     float64(vMax),
		defaultValue: float64(vDef),
		step:         1,
		getter: func() float64 {
			return float64(a.u.settings.MaxMails())
		},
		setter: func(v float64) {
			a.u.settings.SetMaxMails(int(v))
		},
		isMobile: a.u.isMobile,
		window:   a.w,
	})

	vMin, vMax, vDef = a.u.settings.MaxWalletTransactionsPresets()
	maxWallet := NewSettingItemSlider(SettingItemSliderParams{
		label:        "Maximum wallet transaction",
		hint:         "Max wallet transactions downloaded. 0 = unlimited.",
		minValue:     float64(vMin),
		maxValue:     float64(vMax),
		defaultValue: float64(vDef),
		step:         1,
		getter: func() float64 {
			return float64(a.u.settings.MaxWalletTransactions())
		},
		setter: func(v float64) {
			a.u.settings.SetMaxWalletTransactions(int(v))
		},
		isMobile: a.u.isMobile,
		window:   a.w,
	})

	vMin, vMax, vDef = a.u.settings.MarketOrderRetentionDaysPresets()
	marketOrdersRetention := NewSettingItemSlider(SettingItemSliderParams{
		label:        "Market order retention",
		hint:         "Number of days to keep historic market orders.",
		minValue:     float64(vMin),
		maxValue:     float64(vMax),
		defaultValue: float64(vDef),
		step:         1,
		getter: func() float64 {
			return float64(a.u.settings.MarketOrderRetentionDays())
		},
		setter: func(v float64) {
			a.u.settings.SetMarketOrdersRetentionDay(int(v))
		},
		isMobile: a.u.isMobile,
		window:   a.w,
	})

	items = slices.Concat(items, []SettingItem{
		NewSettingItemHeading("Section updates"),
		maxMail,
		maxWallet,
		marketOrdersRetention,
	})

	list := NewSettingList(items)

	clear := settingAction{
		Label: "Clear cache",
		Action: func() {
			w := a.w
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
				}, w,
			)
		},
	}
	reset := settingAction{
		Label: "Reset to defaults",
		Action: func() {
			developerMode.Reset()
			logLevel.Reset()
			sysTray.Reset()
			list.Refresh()
			colorTheme.Reset()
			disableDPIDetection.Reset()
			fyneScale.Reset()
			preferMarketTab.Reset()
			hideLimitedCorporations.Reset()
			maxMail.Reset()
			maxWallet.Reset()
		},
	}
	exportAppLog := settingAction{
		Label: "Export application log",
		Action: func() {
			a.showExportFileDialog(a.u.dataPaths["log"])
		},
	}
	exportCrashLog := settingAction{
		Label: "Export crash log",
		Action: func() {
			a.showExportFileDialog(a.u.dataPaths["crashfile"])
		},
	}
	deleteAppLog := settingAction{
		Label: "Delete application log",
		Action: func() {
			a.showDeleteFileDialog("application log", a.u.dataPaths["log"]+"*")
		},
	}
	deleteCrashLog := settingAction{
		Label: "Delete crash log",
		Action: func() {
			a.showDeleteFileDialog("crash log", a.u.dataPaths["crashfile"])
		},
	}
	actions := []settingAction{reset, clear, exportAppLog, exportCrashLog, deleteAppLog, deleteCrashLog}
	if !a.u.isMobile {
		actions = append(actions, settingAction{
			Label: "Resets main window size to defaults",
			Action: func() {
				a.u.settings.ResetWindowSize()
				a.u.MainWindow().Resize(a.u.settings.WindowSize())
			},
		})
	}
	if a.u.IsDeveloperMode() {
		actions = append(actions, settingAction{
			Label: "Show snackbar (debug)",
			Action: func() {
				a.sb.Show("This is a test")
			},
		})
		actions = append(actions, settingAction{
			Label: "Reset shown character (debug)",
			Action: func() {
				a.u.resetCharacter()
			},
		})
		actions = append(actions, settingAction{
			Label: "Reset shown corporation (debug)",
			Action: func() {
				a.u.resetCorporation()
			},
		})
	}
	return list, makeIconButtonFromActions(actions)
}

func (a *userSettings) showDeleteFileDialog(name, path string) {
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
				a.sb.Show("ERROR: Failed to delete " + name)
			} else {
				a.sb.Show(xstrings.Title(name) + " deleted")
			}
		}, a.w)
}

func (a *userSettings) showExportFileDialog(path string) {
	filename := filepath.Base(path)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		a.sb.Show("No file to export: " + filename)
		return
	} else if err != nil {
		a.u.showErrorDialog("Failed to open "+filename, err, a.w)
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
				a.sb.Show("File " + filename + " exported")
				return nil
			}()
			if err2 != nil {
				a.u.showErrorDialog("Failed to export "+filename, err, a.w)
			}
		}, a.w,
	)
	d.SetFileName(filename)
	a.u.ModifyShortcutsForDialog(d, a.w)
	d.Show()
}

func (a *userSettings) makeNotificationPage() (fyne.CanvasObject, *kxwidget.IconButton) {
	groupsAndTypes := make(map[app.EveNotificationGroup][]app.EveNotificationType)
	for n := range app.NotificationTypesSupported().All() {
		g := n.Group()
		groupsAndTypes[g] = append(groupsAndTypes[g], n)
	}
	groups := make([]app.EveNotificationGroup, 0)
	for c := range groupsAndTypes {
		groups = append(groups, c)
	}
	for _, g := range groups {
		slices.Sort(groupsAndTypes[g])
	}
	slices.Sort(groups)
	typesEnabled := a.u.settings.NotificationTypesEnabled()

	// add global items
	notifyCommunications := NewSettingItemSwitch(SettingItemSwitch{
		defaultValue: a.u.settings.NotifyCommunicationsEnabledDefault(),
		label:        "Notify communications",
		hint:         "Whether to notify new communications",
		getter:       a.u.settings.NotifyCommunicationsEnabled,
		onChanged:    a.u.settings.SetNotifyCommunicationsEnabled,
	})
	notifyMails := NewSettingItemSwitch(SettingItemSwitch{
		defaultValue: a.u.settings.NotifyMailsEnabledDefault(),
		label:        "Notify mails",
		hint:         "Whether to notify new mails",
		getter:       a.u.settings.NotifyMailsEnabled,
		onChanged: func(on bool) {
			a.u.settings.SetNotifyMailsEnabled(on)
			if on {
				a.u.settings.SetNotifyMailsEarliest(time.Now())
			}
		},
	})
	notifyPI := NewSettingItemSwitch(SettingItemSwitch{
		defaultValue: a.u.settings.NotifyPIEnabled(),
		label:        "Planetary Industry",
		hint:         "Whether to notify about expired extractions",
		getter:       a.u.settings.NotifyPIEnabled,
		onChanged: func(on bool) {
			a.u.settings.SetNotifyPIEnabled(on)
			if on {
				a.u.settings.SetNotifyPIEarliest(time.Now())
			}
		},
	})

	notifyTraining := NewSettingItemSwitch(SettingItemSwitch{
		defaultValue: a.u.settings.NotifyTrainingEnabled(),
		label:        "Notify Training",
		hint:         "Whether to notify when skillqueue is empty",
		getter:       a.u.settings.NotifyTrainingEnabled,
		onChanged: func(on bool) {
			ctx := context.Background()
			if on {
				err := a.u.cs.EnableAllTrainingWatchers(ctx)
				if err != nil {
					a.u.showErrorDialog("Failed to enable training notification", err, a.w)
				} else {
					a.u.settings.SetNotifyTrainingEnabled(on)
				}
			} else {
				err := a.u.cs.DisableAllTrainingWatchers(ctx)
				if err != nil {
					a.u.showErrorDialog("Failed to disable training notification", err, a.w)
				} else {
					a.u.settings.SetNotifyCommunicationsEnabled(false)
				}
			}
		},
	})

	notifyContracts := NewSettingItemSwitch(SettingItemSwitch{
		defaultValue: a.u.settings.NotifyContractsEnabledDefault(),
		label:        "Notify Contracts",
		hint:         "Whether to notify when contract status changes",
		getter:       a.u.settings.NotifyContractsEnabled,
		onChanged: func(on bool) {
			a.u.settings.SetNotifyContractsEnabled(on)
			if on {
				a.u.settings.SetNotifyContractsEarliest(time.Now())
			}
		},
	})

	vMin, vMax, vDef := a.u.settings.NotifyTimeoutHoursPresets()
	notifTimeout := NewSettingItemSlider(SettingItemSliderParams{
		label:        "Notify Timeout",
		hint:         "Events older then this value in hours will not be notified",
		minValue:     float64(vMin),
		maxValue:     float64(vMax),
		defaultValue: float64(vDef),
		step:         1.0,
		getter: func() float64 {
			return float64(a.u.settings.NotifyTimeoutHours())
		},
		setter: func(v float64) {
			a.u.settings.SetNotifyTimeoutHours(int(v))
		},
		isMobile: a.u.isMobile,
		window:   a.w,
	})

	items := []SettingItem{
		NewSettingItemHeading("Global"),
		notifyCommunications,
		notifyMails,
		notifyPI,
		notifyTraining,
		notifyContracts,
		notifTimeout,
	}
	items = append(items, NewSettingItemSeparator())
	items = append(items, NewSettingItemHeading("Communication Groups"))

	// add communication groups
	const groupHint = "Choose which communications to notify about"
	type groupPage struct {
		content fyne.CanvasObject
		actions []settingAction
	}
	groupPages := make(map[app.EveNotificationGroup]groupPage) // for pre-constructing group pages
	for _, g := range groups {
		groupPages[g] = func() groupPage {
			items2 := make([]SettingItem, 0)
			for _, nt := range groupsAndTypes[g] {
				ntStr := nt.String()
				ntDisplay := nt.Display()
				it := NewSettingItemSwitch(SettingItemSwitch{
					label: ntDisplay,
					hint:  "",
					getter: func() bool {
						return typesEnabled.Contains(ntStr)
					},
					onChanged: func(on bool) {
						if on {
							typesEnabled.Add(ntStr)
						} else {
							typesEnabled.Delete(ntStr)
						}
						a.u.settings.SetNotificationTypesEnabled(typesEnabled)
					},
				})
				items2 = append(items2, it)
			}
			list2 := NewSettingList(items2)
			enableAll := settingAction{
				Label: "Enable all",
				Action: func() {
					for _, it := range items2 {
						it.Setter(true)
					}
					list2.Refresh()
				},
			}
			disableAll := settingAction{
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
				actions: []settingAction{enableAll, disableAll},
			}
		}()

		it := NewSettingItemCustom(SettingItemCustom{
			label: g.String(),
			hint:  groupHint,
			getter: func() any {
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
			onSelected: func(it SettingItem, refresh func()) {
				p := groupPages[g]
				title := g.String()
				hint := widget.NewLabel(groupHint)
				hint.SizeName = theme.SizeNameCaptionText
				var d dialog.Dialog
				buttons := container.NewHBox(
					widget.NewButton("OK", func() {
						d.Hide()
					}),
					layout.NewSpacer(),
				)
				for _, a := range p.actions {
					buttons.Add(widget.NewButton(a.Label, a.Action))
				}
				c := container.NewBorder(nil, container.NewVBox(hint, buttons), nil, nil, p.content)
				w := a.w
				d = dialog.NewCustomWithoutButtons(title, c, w)
				a.u.ModifyShortcutsForDialog(d, w)
				d.Show()
				_, s := w.Canvas().InteractiveArea()
				d.Resize(fyne.NewSize(s.Width*0.8, s.Height*0.8))
				d.SetOnClosed(refresh)
			},
		})
		items = append(items, it)
	}

	list := NewSettingList(items)
	reset := settingAction{
		Label: "Reset to defaults",
		Action: func() {
			notifyCommunications.Reset()
			notifyContracts.Reset()
			notifyPI.Reset()
			notifyTraining.Reset()
			notifyMails.Reset()
			notifTimeout.Reset()
			typesEnabled.Clear()
			a.u.settings.ResetNotificationTypesEnabled()
			list.Refresh()
		},
	}
	updateTypes := func() {
		a.u.settings.SetNotificationTypesEnabled(typesEnabled)
		list.Refresh()
	}
	none := settingAction{
		Label: "Disable all communication groups",
		Action: func() {
			typesEnabled.Clear()
			updateTypes()
		},
	}
	all := settingAction{
		Label: "Enable all communication groups",
		Action: func() {
			for nt := range app.NotificationTypesSupported().All() {
				typesEnabled.Add(nt.String())
			}
			updateTypes()
		},
	}
	send := settingAction{
		Label: "Send test notification",
		Action: func() {
			n := fyne.NewNotification("Test", "This is a test notification from EVE Buddy.")
			a.u.App().SendNotification(n)
		},
	}
	return list, makeIconButtonFromActions([]settingAction{reset, all, none, send})
}

// func (a *userSettings) reportError(text string, err error) {
// 	slog.Error(text, "error", err)
// 	a.sb.Show(fmt.Sprintf("ERROR: %s: %s", text, err))
// }

func makeIconButtonFromActions(actions []settingAction) *kxwidget.IconButton {
	items := make([]*fyne.MenuItem, 0)
	for _, a := range actions {
		items = append(items, fyne.NewMenuItem(a.Label, a.Action))
	}
	return kxwidget.NewIconButtonWithMenu(
		theme.MoreHorizontalIcon(),
		fyne.NewMenu("", items...),
	)
}

// relative size of dialog window to current window
const (
	dialogWidthScale = 0.8 // except on mobile it is always 100%
	dialogHeightMin  = 100
)

type settingVariant uint

const (
	settingUndefined settingVariant = iota
	settingCustom
	settingHeading
	settingSeparator
	settingSwitch
)

// SettingItem represents an item in a setting list.
type SettingItem struct {
	Default   any
	Hint      string             // optional hint text
	Label     string             // label
	Getter    func() any         // returns the current value for this setting
	Setter    func(v any)        // sets the value for this setting
	Formatter func(v any) string // func to format the value

	onSelected func(it SettingItem, refresh func()) // action called when selected
	variant    settingVariant                       // the setting variant of this item
}

func (si SettingItem) Reset() {
	if si.Setter == nil || si.Default == nil {
		return
	}
	if si.Getter() == si.Default {
		return
	}
	si.Setter(si.Default)
}

// NewSettingItemHeading creates a heading in a setting list.
func NewSettingItemHeading(label string) SettingItem {
	return SettingItem{Label: label, variant: settingHeading}
}

// NewSettingItemSeparator creates a separator in a setting list.
func NewSettingItemSeparator() SettingItem {
	return SettingItem{variant: settingSeparator}
}

type SettingItemSwitch struct {
	defaultValue bool
	getter       func() bool
	hint         string
	label        string
	onChanged    func(bool)
}

// NewSettingItemSwitch creates a switch setting in a setting list.
func NewSettingItemSwitch(arg SettingItemSwitch) SettingItem {
	return SettingItem{
		Default: arg.defaultValue,
		Label:   arg.label,
		Hint:    arg.hint,
		Getter: func() any {
			return arg.getter()
		},
		Setter: func(v any) {
			arg.onChanged(v.(bool))
		},
		onSelected: func(it SettingItem, refresh func()) {
			it.Setter(!it.Getter().(bool))
			refresh()
		},
		variant: settingSwitch,
	}
}

type SettingItemCustom struct {
	label      string
	hint       string
	getter     func() any
	onSelected func(it SettingItem, refresh func())
}

// NewSettingItemCustom creates a custom setting in a setting list.
func NewSettingItemCustom(arg SettingItemCustom) SettingItem {
	return SettingItem{
		Label:      arg.label,
		Hint:       arg.hint,
		Getter:     arg.getter,
		onSelected: arg.onSelected,
		variant:    settingCustom,
	}
}

type SettingItemSliderParams struct {
	defaultValue float64
	formatter    func(v any) string
	getter       func() float64
	hint         string
	isMobile     bool
	label        string
	maxValue     float64
	minValue     float64
	setter       func(v float64)
	step         float64
	window       fyne.Window
}

func NewSettingItemSlider(arg SettingItemSliderParams) SettingItem {
	return SettingItem{
		Default: arg.defaultValue,
		Label:   arg.label,
		Hint:    arg.hint,
		Getter: func() any {
			return arg.getter()
		},
		Formatter: arg.formatter,
		Setter: func(v any) {
			switch x := v.(type) {
			case float64:
				arg.setter(x)
			case int:
				arg.setter(float64(x))
			default:
				panic("setting item: unsupported type: " + arg.label)
			}
		},
		onSelected: func(it SettingItem, refresh func()) {
			sl := kxwidget.NewSlider(arg.minValue, arg.maxValue)
			sl.SetValue(arg.getter())
			sl.SetStep(arg.step)
			sl.OnChangeEnded = arg.setter
			d := makeSettingDialog(makeSettingDialogParams{
				setting:  sl,
				label:    it.Label,
				hint:     it.Hint,
				isMobile: arg.isMobile,
				reset: func() {
					sl.SetValue(arg.defaultValue)
				},
				refresh: refresh,
				window:  arg.window,
			})
			d.Show()
		},
		variant: settingCustom,
	}
}

type SettingItemOptions struct {
	defaultValue string
	getter       func() string
	hint         string
	isMobile     bool
	label        string
	options      []string
	setter       func(v string)
	window       fyne.Window
}

func NewSettingItemOptions(arg SettingItemOptions) SettingItem {
	return SettingItem{
		Default: arg.defaultValue,
		Label:   arg.label,
		Hint:    arg.hint,
		Getter: func() any {
			return arg.getter()
		},
		Setter: func(v any) {
			arg.setter(v.(string))
		},
		onSelected: func(it SettingItem, refresh func()) {
			sel := widget.NewRadioGroup(arg.options, arg.setter)
			sel.Required = true
			sel.Selected = it.Getter().(string)
			d := makeSettingDialog(makeSettingDialogParams{
				setting: sel,
				label:   it.Label,
				hint:    it.Hint,
				reset: func() {
					sel.SetSelected(arg.defaultValue)
				},
				isMobile: arg.isMobile,
				refresh:  refresh,
				window:   arg.window,
			})
			d.Show()
		},
		variant:   settingCustom,
		Formatter: nil,
	}
}

type makeSettingDialogParams struct {
	hint     string
	isMobile bool
	label    string
	refresh  func()
	reset    func()
	setting  fyne.CanvasObject
	window   fyne.Window
}

func makeSettingDialog(arg makeSettingDialogParams) dialog.Dialog {
	var d dialog.Dialog
	buttons := container.NewHBox(
		widget.NewButton("OK", func() {
			d.Hide()
		}),
		layout.NewSpacer(),
		widget.NewButton("Reset", func() {
			arg.reset()
		}),
	)
	l := widget.NewLabel(arg.hint)
	l.SizeName = theme.SizeNameCaptionText
	c := container.NewBorder(
		nil,
		container.NewVBox(l, buttons),
		nil,
		nil,
		arg.setting,
	)
	// TODO: add modify shortcuts
	d = dialog.NewCustomWithoutButtons(arg.label, c, arg.window)
	_, s := arg.window.Canvas().InteractiveArea()
	var width float32
	if arg.isMobile {
		width = s.Width
	} else {
		width = s.Width * dialogWidthScale
	}
	d.Resize(fyne.NewSize(width, dialogHeightMin))
	d.SetOnClosed(arg.refresh)
	return d
}

// SettingList is a custom list widget for settings.
type SettingList struct {
	widget.List

	SelectDelay time.Duration
}

// NewSettingList returns a new SettingList widget.
func NewSettingList(items []SettingItem) *SettingList {
	w := &SettingList{SelectDelay: 200 * time.Millisecond}
	w.Length = func() int {
		return len(items)
	}
	w.CreateItem = func() fyne.CanvasObject {
		// p := theme.Padding()
		label := widget.NewLabel("Template")
		label.Truncation = fyne.TextTruncateClip
		hint := widget.NewLabel("")
		hint.Truncation = fyne.TextTruncateClip
		hint.SizeName = theme.SizeNameCaptionText
		c := container.NewPadded(container.NewBorder(
			nil,
			container.New(layout.NewCustomPaddedLayout(0, 0, 0, 0), widget.NewSeparator()),
			nil,
			container.NewVBox(layout.NewSpacer(), container.NewStack(kxwidget.NewSwitch(nil), widget.NewLabel("")), layout.NewSpacer()),
			container.New(layout.NewCustomPaddedVBoxLayout(0), layout.NewSpacer(), label, hint, layout.NewSpacer()),
		))
		return c
	}
	w.UpdateItem = func(id widget.ListItemID, co fyne.CanvasObject) {
		if id >= len(items) {
			return
		}
		it := items[id]
		border := co.(*fyne.Container).Objects[0].(*fyne.Container).Objects
		right := border[2].(*fyne.Container).Objects[1].(*fyne.Container).Objects
		sw := right[0].(*kxwidget.Switch)
		value := right[1].(*widget.Label)
		main := border[0].(*fyne.Container).Objects
		hint := main[2].(*widget.Label)
		if it.Hint != "" {
			hint.SetText(it.Hint)
			hint.Show()
		} else {
			hint.Hide()
		}
		label := main[1].(*widget.Label)
		label.Text = it.Label
		label.TextStyle.Bold = false
		switch it.variant {
		case settingHeading:
			label.TextStyle.Bold = true
			value.Hide()
			sw.Hide()
		case settingSwitch:
			value.Hide()
			sw.OnChanged = func(v bool) {
				it.Setter(v)
			}
			sw.On = it.Getter().(bool)
			sw.Show()
			sw.Refresh()
		case settingCustom:
			formatter := it.Formatter
			if formatter == nil {
				formatter = func(v any) string {
					return fmt.Sprint(v)
				}
			}
			value.SetText(formatter(it.Getter()))
			value.Show()
			sw.Hide()
		}
		sep := border[1].(*fyne.Container)
		if it.variant == settingSeparator {
			sep.Show()
			value.Hide()
			sw.Hide()
			label.Hide()
		} else {
			sep.Hide()
			label.Show()
			label.Refresh()
		}
		w.SetItemHeight(id, co.(*fyne.Container).MinSize().Height)
	}
	w.OnSelected = func(id widget.ListItemID) {
		if id >= len(items) {
			w.UnselectAll()
			return
		}
		it := items[id]
		if it.onSelected == nil {
			w.UnselectAll()
			return
		}
		it.onSelected(it, func() {
			w.RefreshItem(id)
		})
		go func() {
			time.Sleep(w.SelectDelay)
			fyne.Do(func() {
				w.UnselectAll()
			})
		}()
	}
	w.HideSeparators = true
	w.ExtendBaseWidget(w)
	return w
}
