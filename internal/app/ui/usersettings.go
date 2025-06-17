package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type settingAction struct {
	Label  string
	Action func()
}
type userSettings struct {
	widget.BaseWidget

	needUpdate bool
	sb         *iwidget.Snackbar
	u          *baseUI
	w          fyne.Window
}

func showSettingsWindow(u *baseUI) {
	title := u.MakeWindowTitle("Settings")
	for _, w := range u.app.Driver().AllWindows() {
		if w.Title() == title {
			w.Show()
			return
		}
	}
	w := u.app.NewWindow(title)
	a := newSettings(u, w)
	w.SetContent(fynetooltip.AddWindowToolTipLayer(a, w.Canvas()))
	w.Resize(fyne.Size{Width: 700, Height: 500})
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
	w.SetOnClosed(func() {
		if a.needUpdate {
			u.updateCrossPages()
		}
		a.sb.Stop()
	})
	return a
}

func (a *userSettings) CreateRenderer() fyne.WidgetRenderer {
	makeSettingsPage := func(title string, content fyne.CanvasObject, actions fyne.CanvasObject) fyne.CanvasObject {
		return iwidget.NewAppBarWithTrailing(title, content, actions)
	}
	generalContent, generalActions := a.makeGeneralSettingsPage()
	notificationContent, notificationActions := a.makeNotificationPage()
	tagsContent, tagsActions := a.makeCharacterTagsPage()
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
		container.NewTabItem("Character Tags", makeSettingsPage(
			"Character Tags",
			tagsContent,
			tagsActions,
		)),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	return widget.NewSimpleRenderer(tabs)
}

func (a *userSettings) makeGeneralSettingsPage() (fyne.CanvasObject, *kxwidget.IconButton) {
	logLevel := NewSettingItemOptions(
		"Log level",
		"Set current log level",
		a.u.settings.LogLevelNames(),
		a.u.settings.LogLevelDefault(),
		func() string {
			return a.u.settings.LogLevel()
		},
		func(v string) {
			s := a.u.settings
			s.SetLogLevel(v)
			slog.SetLogLoggerLevel(s.LogLevelSlog())
		},
		a.w,
	)
	vMin, vMax, vDef := a.u.settings.MaxMailsPresets()
	maxMail := NewSettingItemSlider(
		"Maximum mails",
		"Max number of mails downloaded. 0 = unlimited.",
		float64(vMin),
		float64(vMax),
		float64(vDef),
		func() float64 {
			return float64(a.u.settings.MaxMails())
		},
		func(v float64) {
			a.u.settings.SetMaxMails(int(v))
		},
		a.w,
	)
	vMin, vMax, vDef = a.u.settings.MaxWalletTransactionsPresets()
	maxWallet := NewSettingItemSlider(
		"Maximum wallet transaction",
		"Max wallet transactions downloaded. 0 = unlimited.",
		float64(vMin),
		float64(vMax),
		float64(vDef),
		func() float64 {
			return float64(a.u.settings.MaxWalletTransactions())
		},
		func(v float64) {
			a.u.settings.SetMaxWalletTransactions(int(v))
		},
		a.w,
	)
	preferMarketTab := NewSettingItemSwitch(
		"Prefer market tab",
		"Show market tab for tradeable items",
		func() bool {
			return a.u.settings.PreferMarketTab()
		},
		func(v bool) {
			a.u.settings.SetPreferMarketTab(v)
		},
	)
	developerMode := NewSettingItemSwitch(
		"Developer Mode",
		"App shows additional technical information like Character IDs",
		func() bool {
			return a.u.settings.DeveloperMode()
		},
		func(v bool) {
			a.u.settings.SetDeveloperMode(v)
		},
	)

	items := []SettingItem{
		NewSettingItemHeading("Application"),
		logLevel,
		preferMarketTab,
		developerMode,
		NewSettingItemSeparator(),
		NewSettingItemHeading("EVE Online"),
		maxMail,
		maxWallet,
	}

	sysTray := NewSettingItemSwitch(
		"Close button",
		"App will minimize to system tray when closed (requires restart)",
		func() bool {
			return a.u.settings.SysTrayEnabled()
		},
		func(v bool) {
			a.u.settings.SetSysTrayEnabled(v)
		},
	)
	if a.u.isDesktop {
		items = slices.Insert(items, 2, sysTray)
	}

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
				}, w)
		}}
	reset := settingAction{
		Label: "Reset to defaults",
		Action: func() {
			a.u.settings.ResetPreferMarketTab()
			a.u.settings.ResetDeveloperMode()
			a.u.settings.ResetLogLevel()
			a.u.settings.ResetMaxMails()
			a.u.settings.ResetMaxWalletTransactions()
			a.u.settings.ResetSysTrayEnabled()
			list.Refresh()
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
	if a.u.isDesktop {
		actions = append(actions, settingAction{
			Label: "Resets main window size to defaults",
			Action: func() {
				a.u.settings.ResetWindowSize()
				a.u.MainWindow().Resize(a.u.settings.WindowSize())
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
				titler := cases.Title(language.English)
				a.sb.Show(titler.String(name) + " deleted")
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
				a.sb.Show("File " + filename + " exported")
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

func (a *userSettings) makeNotificationPage() (fyne.CanvasObject, *kxwidget.IconButton) {
	groupsAndTypes := make(map[app.NotificationGroup][]evenotification.Type)
	for n := range evenotification.SupportedTypes().All() {
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
	typesEnabled := a.u.settings.NotificationTypesEnabled()

	// add global items
	notifyCommunications := NewSettingItemSwitch(
		"Notify communications",
		"Whether to notify new communications",
		func() bool {
			return a.u.settings.NotifyCommunicationsEnabled()
		},
		func(on bool) {
			a.u.settings.SetNotifyCommunicationsEnabled(on)
			if on {
				a.u.settings.SetNotifyCommunicationsEarliest(time.Now())
			}
		},
	)
	notifyMails := NewSettingItemSwitch(
		"Notify mails",
		"Whether to notify new mails",
		func() bool {
			return a.u.settings.NotifyMailsEnabled()
		},
		func(on bool) {
			a.u.settings.SetNotifyMailsEnabled(on)
			if on {
				a.u.settings.SetNotifyMailsEarliest(time.Now())
			}
		},
	)
	notifyPI := NewSettingItemSwitch(
		"Planetary Industry",
		"Whether to notify about expired extractions",
		func() bool {
			return a.u.settings.NotifyPIEnabled()
		},
		func(on bool) {
			a.u.settings.SetNotifyPIEnabled(on)
			if on {
				a.u.settings.SetNotifyPIEarliest(time.Now())
			}
		},
	)
	notifyTraining := NewSettingItemSwitch(
		"Notify Training",
		"Whether to notify when skillqueue is empty",
		func() bool {
			return a.u.settings.NotifyTrainingEnabled()
		},
		func(on bool) {
			ctx := context.Background()
			if on {
				err := a.u.cs.EnableAllTrainingWatchers(ctx)
				if err != nil {
					a.u.ShowErrorDialog("failed to enable training notification", err, a.w)
				} else {
					a.u.settings.SetNotifyTrainingEnabled(on)
				}
			} else {
				err := a.u.cs.DisableAllTrainingWatchers(ctx)
				if err != nil {
					a.u.ShowErrorDialog("failed to disable training notification", err, a.w)
				} else {
					a.u.settings.SetNotifyCommunicationsEnabled(false)
				}
			}
		},
	)
	notifyContracts := NewSettingItemSwitch(
		"Notify Contracts",
		"Whether to notify when contract status changes",
		func() bool {
			return a.u.settings.NotifyContractsEnabled()
		},
		func(on bool) {
			a.u.settings.SetNotifyContractsEnabled(on)
			if on {
				a.u.settings.SetNotifyContractsEarliest(time.Now())
			}
		},
	)
	vMin, vMax, vDef := a.u.settings.NotifyTimeoutHoursPresets()
	notifTimeout := NewSettingItemSlider(
		"Notify Timeout",
		"Events older then this value in hours will not be notified",
		float64(vMin),
		float64(vMax),
		float64(vDef),
		func() float64 {
			return float64(a.u.settings.NotifyTimeoutHours())
		},
		func(v float64) {
			a.u.settings.SetNotifyTimeoutHours(int(v))
		},
		a.w,
	)
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
	groupPages := make(map[app.NotificationGroup]groupPage) // for pre-constructing group pages
	for _, g := range groups {
		groupPages[g] = func() groupPage {
			items2 := make([]SettingItem, 0)
			for _, nt := range groupsAndTypes[g] {
				ntStr := nt.String()
				ntDisplay := nt.Display()
				it := NewSettingItemSwitch(
					ntDisplay,
					"",
					func() bool {
						return typesEnabled.Contains(ntStr)
					},
					func(on bool) {
						if on {
							typesEnabled.Add(ntStr)
						} else {
							typesEnabled.Delete(ntStr)
						}
						a.u.settings.SetNotificationTypesEnabled(typesEnabled)
					},
				)
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

		it := NewSettingItemCustom(g.String(), groupHint,
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
			func(it SettingItem, refresh func()) {
				p := groupPages[g]
				title := g.String()
				hint := widget.NewLabel(groupHint)
				hint.SizeName = theme.SizeNameCaptionText
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
				w := a.w
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

	list := NewSettingList(items)
	reset := settingAction{
		Label: "Reset to defaults",
		Action: func() {
			a.u.settings.ResetNotifyCommunicationsEnabled()
			a.u.settings.ResetNotifyContractsEnabled()
			a.u.settings.ResetNotifyMailsEnabled()
			a.u.settings.ResetNotifyPIEnabled()
			a.u.settings.ResetNotifyTimeoutHours()
			a.u.settings.ResetNotifyTrainingEnabled()
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
			for nt := range evenotification.SupportedTypes().All() {
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

func (a *userSettings) makeCharacterTagsPage() (body fyne.CanvasObject, actions fyne.CanvasObject) {
	var selectedTag *app.CharacterTag
	var characterList *widget.List
	characters := make([]*app.EntityShort[int32], 0)
	var updateCharacters func(tag *app.CharacterTag)

	addCharacter := iwidget.NewTappableIcon(theme.ContentAddIcon(), func() {
		if selectedTag == nil {
			return
		}
		_, others, err := a.u.cs.ListCharactersForTag(context.Background(), selectedTag.ID)
		if err != nil {
			a.u.ShowErrorDialog("Failed to list characters for tag", err, a.w)
			characters = make([]*app.EntityShort[int32], 0)
			return
		}
		if len(others) == 0 {
			return
		}
		selected := make(map[int32]bool)
		list := widget.NewList(
			func() int {
				return len(others)
			},
			func() fyne.CanvasObject {
				check := widget.NewIcon(theme.CheckButtonIcon())
				portrait := iwidget.NewImageFromResource(
					icons.Characterplaceholder64Jpeg,
					fyne.NewSquareSize(app.IconUnitSize),
				)
				return container.NewBorder(
					nil,
					nil,
					container.NewHBox(check, portrait),
					nil,
					widget.NewLabel("Template"),
				)
			},
			func(id widget.ListItemID, co fyne.CanvasObject) {
				if id >= len(others) {
					return
				}
				box := co.(*fyne.Container).Objects
				character := others[id]
				box[0].(*widget.Label).SetText(character.Name)
				icons := box[1].(*fyne.Container).Objects

				portrait := icons[1].(*canvas.Image)
				go a.u.updateAvatar(character.ID, func(r fyne.Resource) {
					fyne.Do(func() {
						portrait.Resource = r
						portrait.Refresh()
					})
				})

				check := icons[0].(*widget.Icon)
				if selected[character.ID] {
					check.SetResource(theme.CheckButtonCheckedIcon())
				} else {
					check.SetResource(theme.CheckButtonIcon())
				}
			},
		)
		list.HideSeparators = true
		list.OnSelected = func(id widget.ListItemID) {
			list.UnselectAll()
			if id >= len(others) {
				return
			}
			character := others[id]
			selected[character.ID] = !selected[character.ID]
			list.RefreshItem(id)
		}
		d := dialog.NewCustomConfirm(
			"Add characters to tag: "+selectedTag.Name,
			"Add",
			"Cancel",
			list,
			func(confirmed bool) {
				if !confirmed {
					return
				}
				for characterID, v := range selected {
					if !v {
						return
					}
					err := a.u.cs.AddTagToCharacter(context.Background(), characterID, selectedTag.ID)
					if err != nil {
						a.u.ShowErrorDialog("Failed to add tag to character", err, a.w)
						return
					}
				}
				updateCharacters(selectedTag)
				a.needUpdate = true
			},
			a.w,
		)
		a.u.ModifyShortcutsForDialog(d, a.w)
		d.Show()
		_, s := a.w.Canvas().InteractiveArea()
		d.Resize(fyne.NewSize(s.Width*0.8, s.Height*0.8))
	})
	addCharacter.SetToolTip("Add character")
	addCharacter.Disable()

	var manageCharacters *iwidget.AppBar

	updateCharacters = func(tag *app.CharacterTag) {
		if tag == nil {
			return
		}
		selectedTag = tag
		manageCharacters.SetTitle("Tag: " + tag.Name)
		manageCharacters.Show()
		tagged, others, err := a.u.cs.ListCharactersForTag(context.Background(), tag.ID)
		if err != nil {
			a.u.ShowErrorDialog("Failed to list characters for tag", err, a.w)
			characters = make([]*app.EntityShort[int32], 0)
			return
		}
		characters = tagged
		characterList.Refresh()
		if len(others) > 0 {
			addCharacter.Enable()
		} else {
			addCharacter.Disable()
		}
	}

	p := theme.Padding()
	characterList = widget.NewList(
		func() int {
			return len(characters)
		},
		func() fyne.CanvasObject {
			remove := iwidget.NewTappableIcon(theme.CancelIcon(), nil)
			remove.SetToolTip("Remove character from tag")
			portrait := iwidget.NewImageFromResource(
				icons.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			name := widget.NewLabel("Template")
			return container.New(
				layout.NewCustomPaddedLayout(p, p, p, p),
				container.NewBorder(
					nil,
					nil,
					portrait,
					remove,
					name,
				),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(characters) {
				return
			}
			character := characters[id]
			box := co.(*fyne.Container).Objects[0].(*fyne.Container).Objects
			box[0].(*widget.Label).SetText(character.Name)

			portrait := box[1].(*canvas.Image)
			go a.u.updateAvatar(character.ID, func(r fyne.Resource) {
				fyne.Do(func() {
					portrait.Resource = r
					portrait.Refresh()
				})
			})

			remove := box[2].(*iwidget.TappableIcon)
			remove.OnTapped = func() {
				if selectedTag == nil {
					return
				}
				err := a.u.cs.RemoveTagFromCharacter(context.Background(), character.ID, selectedTag.ID)
				if err != nil {
					a.u.ShowErrorDialog("Failed to list characters", err, a.w)
					return
				}
				updateCharacters(selectedTag)
				a.needUpdate = true
			}
		},
	)
	characterList.HideSeparators = true
	characterList.OnSelected = func(id widget.ListItemID) {
		characterList.UnselectAll()
	}

	tags := make([]*app.CharacterTag, 0)
	var tagList *widget.List

	updateTags := func() {
		rows, err := a.u.cs.ListTags(context.Background())
		if err != nil {
			a.u.ShowErrorDialog("Failed to list tags", err, a.w)
			tags = make([]*app.CharacterTag, 0)
			return
		}
		tags = rows
		tagList.Refresh()
	}

	modifyTag := func(title, confirm string, execute func(name string) error) {
		names := set.Of(xslices.Map(tags, func(x *app.CharacterTag) string {
			return strings.ToLower(x.Name)
		})...)
		name := widget.NewEntry()
		name.Validator = func(s string) error {
			if len(s) == 0 {
				return errors.New("can not be empty")
			}
			if names.Contains(strings.ToLower(s)) {
				return errors.New("tag with same name already exists")
			}
			return nil
		}
		d := dialog.NewForm(title, confirm, "Cancel", []*widget.FormItem{
			widget.NewFormItem("Name", name),
		}, func(confirmed bool) {
			if !confirmed {
				return
			}
			if err := execute(name.Text); err != nil {
				a.u.ShowErrorDialog("Failed to modify tag", err, a.w)
				return
			}

			updateTags()
			a.needUpdate = true
		}, a.w,
		)
		a.u.ModifyShortcutsForDialog(d, a.w)
		d.Show()
		d.Resize(fyne.NewSize(300, 200))
		a.w.Canvas().Focus(name)
	}

	tagList = widget.NewList(
		func() int {
			return len(tags)
		},
		func() fyne.CanvasObject {
			delete := ttwidget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
			delete.Importance = widget.DangerImportance
			delete.SetToolTip("Delete tag")
			rename := ttwidget.NewButtonWithIcon("", theme.DocumentCreateIcon(), nil)
			rename.SetToolTip("Rename tag")
			name := widget.NewLabel("Template")
			return container.NewBorder(
				nil,
				nil,
				nil,
				container.NewHBox(rename, delete),
				name,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(tags) {
				return
			}
			tag := tags[id]
			box := co.(*fyne.Container).Objects
			box[0].(*widget.Label).SetText(tag.Name)
			icons := box[1].(*fyne.Container).Objects
			icons[0].(*ttwidget.Button).OnTapped = func() {
				modifyTag("Rename tag: "+tag.Name, "Rename", func(name string) error {
					return a.u.cs.RenameTag(context.Background(), tag.ID, name)
				})
			}
			icons[1].(*ttwidget.Button).OnTapped = func() {
				s := "Are you sure you want to delete tag " + tag.Name + "?"
				a.u.ShowConfirmDialog("Delete Tag", s, "Delete", func(confirmed bool) {
					if !confirmed {
						return
					}
					err := a.u.cs.DeleteTag(context.Background(), tag.ID)
					if err != nil {
						a.u.ShowErrorDialog("Failed to delete tag", err, a.w)
						return
					}
					updateTags()
					tagList.UnselectAll()
					selectedTag = nil
					characters = make([]*app.EntityShort[int32], 0)
					addCharacter.Disable()
					characterList.Refresh()
					addCharacter.Disable()
					manageCharacters.Hide()
				}, a.w)
			}
		},
	)
	tagList.HideSeparators = true
	tagList.OnSelected = func(id widget.ListItemID) {
		if id >= len(tags) {
			tagList.UnselectAll()
			return
		}
		tag := tags[id]
		updateCharacters(tag)
	}

	updateTags()

	manageCharacters = iwidget.NewAppBarWithTrailing(
		"",
		container.NewPadded(characterList),
		container.New(layout.NewCustomPaddedLayout(0, 0, 0, p), addCharacter),
	)
	manageCharacters.Hide()

	addTag := iwidget.NewTappableIcon(
		theme.ContentAddIcon(), func() {
			modifyTag("Create Character Tag", "Create", func(name string) error {
				_, err := a.u.cs.CreateTag(context.Background(), name)
				return err
			})
		},
	)
	addTag.SetToolTip("Add new tag")
	action := container.New(layout.NewCustomPaddedLayout(0, 0, 0, p), addTag)

	return container.NewVSplit(container.NewPadded(tagList), manageCharacters), action
}

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
	Hint   string      // optional hint text
	Label  string      // label
	Getter func() any  // returns the current value for this setting
	Setter func(v any) // sets the value for this setting

	onSelected func(it SettingItem, refresh func()) // action called when selected
	variant    settingVariant                       // the setting variant of this item
}

// NewSettingItemHeading creates a heading in a setting list.
func NewSettingItemHeading(label string) SettingItem {
	return SettingItem{Label: label, variant: settingHeading}
}

// NewSettingItemSeparator creates a separator in a setting list.
func NewSettingItemSeparator() SettingItem {
	return SettingItem{variant: settingSeparator}
}

// NewSettingItemSwitch creates a switch setting in a setting list.
func NewSettingItemSwitch(
	label, hint string,
	getter func() bool,
	onChanged func(bool),
) SettingItem {
	return SettingItem{
		Label: label,
		Hint:  hint,
		Getter: func() any {
			return getter()
		},
		Setter: func(v any) {
			onChanged(v.(bool))
		},
		onSelected: func(it SettingItem, refresh func()) {
			it.Setter(!it.Getter().(bool))
			refresh()
		},
		variant: settingSwitch,
	}
}

// NewSettingItemCustom creates a custom setting in a setting list.
func NewSettingItemCustom(
	label, hint string,
	getter func() any,
	onSelected func(it SettingItem, refresh func()),
) SettingItem {
	return SettingItem{
		Label:      label,
		Hint:       hint,
		Getter:     getter,
		onSelected: onSelected,
		variant:    settingCustom,
	}
}

func NewSettingItemSlider(
	label, hint string,
	minV, maxV, defaultV float64,
	getter func() float64,
	setter func(v float64),
	window fyne.Window,
) SettingItem {
	return SettingItem{
		Label: label,
		Hint:  hint,
		Getter: func() any {
			return getter()
		},
		Setter: func(v any) {
			switch x := v.(type) {
			case float64:
				setter(x)
			case int:
				setter(float64(x))
			default:
				panic("setting item: unsupported type: " + label)
			}
		},
		onSelected: func(it SettingItem, refresh func()) {
			sl := kxwidget.NewSlider(minV, maxV)
			sl.SetValue(float64(getter()))
			sl.OnChangeEnded = setter
			d := makeSettingDialog(
				sl,
				it.Label,
				it.Hint,
				func() {
					sl.SetValue(defaultV)
				},
				refresh,
				window,
			)
			d.Show()
		},
		variant: settingCustom,
	}
}

func NewSettingItemOptions(
	label, hint string,
	options []string,
	defaultV string,
	getter func() string,
	setter func(v string),
	window fyne.Window,
) SettingItem {
	return SettingItem{
		Label: label,
		Hint:  hint,
		Getter: func() any {
			return getter()
		},
		Setter: func(v any) {
			setter(v.(string))
		},
		onSelected: func(it SettingItem, refresh func()) {
			sel := widget.NewRadioGroup(options, setter)
			sel.SetSelected(it.Getter().(string))
			d := makeSettingDialog(
				sel,
				it.Label,
				it.Hint,
				func() {
					sel.SetSelected(defaultV)
				},
				refresh,
				window,
			)
			d.Show()
		},
		variant: settingCustom,
	}
}

func makeSettingDialog(
	setting fyne.CanvasObject,
	label, hint string,
	reset, refresh func(),
	w fyne.Window,
) dialog.Dialog {
	var d dialog.Dialog
	buttons := container.NewHBox(
		widget.NewButton("Close", func() {
			d.Hide()
		}),
		layout.NewSpacer(),
		widget.NewButton("Reset", func() {
			reset()
		}),
	)
	l := widget.NewLabel(hint)
	l.SizeName = theme.SizeNameCaptionText
	c := container.NewBorder(
		nil,
		container.NewVBox(l, buttons),
		nil,
		nil,
		setting,
	)
	// TODO: add modify shortcuts
	d = dialog.NewCustomWithoutButtons(label, c, w)
	_, s := w.Canvas().InteractiveArea()
	var width float32
	if fyne.CurrentDevice().IsMobile() {
		width = s.Width
	} else {
		width = s.Width * dialogWidthScale
	}
	d.Resize(fyne.NewSize(width, dialogHeightMin))
	d.SetOnClosed(refresh)
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
			value.SetText(fmt.Sprint(it.Getter()))
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
