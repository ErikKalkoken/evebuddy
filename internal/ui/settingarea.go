package ui

import (
	"errors"
	"log/slog"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

// settingsArea is the UI area for settings.
type settingsArea struct {
	content *fyne.Container
	ui      *ui
}

func (u *ui) ShowSettingsDialog() {
	d, err := makeSettingsDialog(u)
	if err != nil {
		dialog.NewError(err, u.window)
	} else {
		d.Show()
	}
}

func makeSettingsDialog(u *ui) (*dialog.CustomDialog, error) {
	maxMails, ok, err := u.service.DictionaryInt(model.SettingMaxMails)
	if err != nil {
		return nil, err
	}
	if !ok {
		maxMails = model.SettingMaxMailsDefault
	}
	maxMailsEntry := widget.NewEntry()
	maxMailsEntry.SetText(strconv.Itoa(int(maxMails)))
	maxMailsEntry.Validator = NewPositiveNumberValidator()

	clearBtn := widget.NewButton("Clear NOW", func() {
		d := dialog.NewConfirm(
			"Clear image cache",
			"Are you sure you want to clear the image cache?",
			func(confirmed bool) {
				if !confirmed {
					return
				}
				count := u.imageManager.ClearCache()
				slog.Info("Cleared images cache", "count", count)
			},
			u.window,
		)
		d.Show()
	})

	themeRadio := widget.NewRadioGroup(
		[]string{model.ThemeAuto, model.ThemeDark, model.ThemeLight}, func(s string) {},
	)
	name, ok, err := u.service.DictionaryString(model.SettingTheme)
	if err == nil && ok {
		themeRadio.SetSelected(name)
	}

	var d *dialog.CustomDialog
	form := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:     "Maximum mails",
				Widget:   maxMailsEntry,
				HintText: "Maximum number of mails downloaded from the server",
			},
			{
				Text:     "Theme",
				Widget:   themeRadio,
				HintText: "Chose the preferred theme",
			}, {
				Text:     "Image cache",
				Widget:   clearBtn,
				HintText: "Clear the local image cache",
			},
		},
		OnSubmit: func() {
			maxMails, err := strconv.Atoi(maxMailsEntry.Text)
			if err != nil {
				return
			}
			u.service.DictionarySetInt(model.SettingMaxMails, maxMails)
			u.SetTheme(themeRadio.Selected)
			u.service.DictionarySetString(model.SettingTheme, themeRadio.Selected)
			d.Hide()
		},
		OnCancel: func() {
			d.Hide()
		},
	}
	d = dialog.NewCustomWithoutButtons("Settings", form, u.window)
	return d, nil
}

func (u *ui) NewSettingsArea() *settingsArea {
	content := container.NewVBox()
	m := &settingsArea{
		ui:      u,
		content: content,
	}
	return m
}

func NewPositiveNumberValidator() fyne.StringValidator {
	myErr := errors.New("must be positive number")
	return func(text string) error {
		val, err := strconv.Atoi(text)
		if err != nil {
			return myErr
		}
		if val <= 0 {
			return myErr
		}
		return nil
	}
}
