package ui

import (
	"errors"
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

	var d *dialog.CustomDialog
	form := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:     "Maximum mails",
				Widget:   maxMailsEntry,
				HintText: "Maximum number of mails downloaded from the server",
			},
		},
		OnSubmit: func() {
			maxMails, err := strconv.Atoi(maxMailsEntry.Text)
			if err != nil {
				return
			}
			u.service.DictionarySetInt(model.SettingMaxMails, maxMails)
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
