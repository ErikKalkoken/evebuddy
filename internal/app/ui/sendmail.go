package ui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
)

type SendMailMode uint

const (
	SendMailNew SendMailMode = iota + 1
	SendMailReply
	SendMailReplyAll
	SendMailForward
)

func (u *BaseUI) ShowSendMailWindow(character *app.Character, mode SendMailMode, mail *app.CharacterMail) {
	title := u.MakeWindowTitle(fmt.Sprintf("New message [%s]", character.EveCharacter.Name))
	w := u.FyneApp.NewWindow(title)
	page, icon, action := u.MakeSendMailPage(character, mode, mail, w)
	b := widget.NewButtonWithIcon("Send", icon, func() {
		if action() {
			w.Hide()
		}
	})
	b.Importance = widget.HighImportance
	c := container.NewBorder(nil, b, nil, nil, page)
	w.SetContent(c)
	w.Resize(fyne.NewSize(600, 500))
	w.Show()
}

func (u *BaseUI) MakeSendMailPage(
	character *app.Character,
	mode SendMailMode,
	mail *app.CharacterMail,
	w fyne.Window,
) (fyne.CanvasObject, fyne.Resource, func() bool) {
	to := NewRecipients(func(onSelected func(*app.EveEntity)) {
		u.showAddDialog(character.ID, onSelected, w)
	})

	from := widget.NewEntry()
	from.PlaceHolder = character.EveCharacter.Name
	from.Disable()

	subject := widget.NewEntry()
	subject.Validator = newNonEmptyStringValidator()

	body := widget.NewEntry()
	body.MultiLine = true
	body.SetMinRowsVisible(14)
	body.Validator = newNonEmptyStringValidator()
	body.PlaceHolder = "Compose message"

	if mail != nil {
		const sep = "\n\n--------------------------------\n"
		switch mode {
		case SendMailReply:
			to.Set([]*app.EveEntity{mail.From})
			subject.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			body.SetText(sep + mail.String())
		case SendMailReplyAll:
			to.Set(mail.Recipients)
			subject.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			body.SetText(sep + mail.String())
		case SendMailForward:
			subject.SetText(fmt.Sprintf("Fw: %s", mail.Subject))
			body.SetText(sep + mail.String())
		default:
			panic(fmt.Errorf("undefined mode for create message: %v", mode))
		}
	}

	// sendAction tries to send the current mail and reports whether it was successful
	sendAction := func() bool {
		showErrorDialog := func(message string) {
			d := dialog.NewInformation("Failed to send mail", message, w)
			d.Show()
		}
		if to.IsEmpty() {
			showErrorDialog("A mail needs to have at least one recipient.")
			return false
		}
		if err := subject.Validate(); err != nil {
			showErrorDialog("The subject can not be empty.")
			return false
		}
		if err := body.Validate(); err != nil {
			showErrorDialog("The message can not be empty.")
			return false
		}
		ctx := context.Background()
		_, err := u.CharacterService.SendCharacterMail(
			ctx,
			character.ID,
			subject.Text,
			to.Recipients,
			body.Text,
		)
		if err != nil {
			showErrorDialog(err.Error())
			return false
		}
		return true
	}

	colums := kxlayout.NewColumns(60)
	page := container.NewBorder(
		container.NewVBox(
			container.New(colums, widget.NewLabel("From"), from),
			container.New(colums, widget.NewLabel("To"), to),
			container.New(colums, widget.NewLabel("Subject"), subject),
		),
		nil,
		nil,
		nil,
		body,
	)
	return page, theme.MailSendIcon(), sendAction
}

func (u *BaseUI) showAddDialog(characterID int32, onSelected func(ee *app.EveEntity), w fyne.Window) {
	var dlg dialog.Dialog
	items := make([]*app.EveEntity, 0)
	list := widget.NewList(
		func() int {
			return len(items)
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("Template")
			name.Truncation = fyne.TextTruncateClip
			category := widgets.NewLabelWithSize("Template", theme.SizeNameCaptionText)
			return container.NewBorder(
				nil,
				nil,
				nil,
				category,
				name,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(items) {
				return
			}
			it := items[id]
			row := co.(*fyne.Container).Objects
			row[0].(*widget.Label).SetText(it.Name)
			row[1].(*widgets.Label).SetText(it.CategoryDisplay())
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(items) {
			list.UnselectAll()
			return
		}
		onSelected(items[id])
		dlg.Hide()
	}
	showErrorDialog := func(search string, err error) {
		slog.Error("Failed to resolve names", "search", search, "error", err)
		d := dialog.NewInformation("Something went wrong", err.Error(), w)
		d.Show()
	}
	entry := widget.NewEntry()
	entry.PlaceHolder = "Type to start searching..."
	entry.ActionItem = widget.NewIcon(theme.SearchIcon())
	entry.OnChanged = func(search string) {
		if len(search) < 3 {
			items = items[:0]
			list.Refresh()
			return
		}
		ctx := context.Background()
		var err error
		items, err = u.EveUniverseService.ListEveEntitiesByPartialName(ctx, search)
		if err != nil {
			showErrorDialog(search, err)
			return
		}
		list.Refresh()
		go func() {
			missingIDs, err := u.CharacterService.AddEveEntitiesFromCharacterSearchESI(
				ctx,
				characterID,
				search,
			)
			if err != nil {
				showErrorDialog(search, err)
				return
			}
			if len(missingIDs) == 0 {
				return // no need to update when not changed
			}
			items, err = u.EveUniverseService.ListEveEntitiesByPartialName(ctx, search)
			if err != nil {
				showErrorDialog(search, err)
				return
			}
			list.Refresh()
		}()
	}
	rect := canvas.NewRectangle(color.Transparent)
	rect.StrokeColor = theme.Color(theme.ColorNameMenuBackground)
	rect.StrokeWidth = 1
	c := container.NewBorder(
		container.NewBorder(nil, nil, nil, widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
			dlg.Hide()
		}), entry),
		nil,
		nil,
		nil,
		container.NewStack(rect, list),
	)
	dlg = dialog.NewCustomWithoutButtons("Add recipient", c, w)
	_, s := w.Canvas().InteractiveArea()
	var f float32
	if fyne.CurrentDevice().IsMobile() {
		f = 1.0
	} else {
		f = 0.8
	}
	dlg.Resize(fyne.NewSize(s.Width*f, s.Height*f))
	dlg.Show()
}

func newNonEmptyStringValidator() fyne.StringValidator {
	myErr := errors.New("can not be empty")
	return func(text string) error {
		if len(text) == 0 {
			return myErr
		}
		return nil
	}
}
