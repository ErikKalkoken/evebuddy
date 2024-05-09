package ui

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

// mailDetailArea is the UI area showing the current mail.
type mailDetailArea struct {
	content fyne.CanvasObject
	icons   *fyne.Container
	subject *widget.Label
	header  *widget.Label
	body    *widget.RichText
	mailID  int32
	ui      *ui

	mail *model.Mail
}

func (u *ui) NewMailArea() *mailDetailArea {
	a := mailDetailArea{
		ui: u,
	}
	icons := container.NewHBox(
		widget.NewButtonWithIcon("", theme.MailReplyIcon(), func() {
			u.ShowSendMessageWindow(CreateMessageReply, a.mail)
		}),
		widget.NewButtonWithIcon("", theme.MailReplyAllIcon(), func() {
			u.ShowSendMessageWindow(CreateMessageReplyAll, a.mail)
		}),
		widget.NewButtonWithIcon("", theme.MailForwardIcon(), func() {
			u.ShowSendMessageWindow(CreateMessageForward, a.mail)
		}),
		layout.NewSpacer(),
	)
	button := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		t := fmt.Sprintf("Are you sure you want to delete this mail?\n\n%s", a.mail.Subject)
		d := dialog.NewConfirm("Delete mail", t, func(confirmed bool) {
			if confirmed {
				err := u.service.DeleteMail(a.mail.MyCharacterID, a.mail.MailID)
				if err != nil {
					errorDialog := dialog.NewError(err, u.window)
					errorDialog.Show()
				} else {
					u.headerArea.Refresh()
				}
			}
		}, u.window)
		d.Show()
	})
	button.Importance = widget.DangerImportance
	icons.Add(button)
	a.icons = icons
	a.toogleIcons(false)

	subject := widget.NewLabel("")
	subject.TextStyle = fyne.TextStyle{Bold: true}
	subject.Truncation = fyne.TextTruncateEllipsis
	a.subject = subject

	header := widget.NewLabel("")
	header.Truncation = fyne.TextTruncateEllipsis
	a.header = header

	wrapper := container.NewVBox(icons, subject, header)

	body := widget.NewRichText()
	body.Wrapping = fyne.TextWrapBreak
	a.body = body

	a.content = container.NewBorder(wrapper, nil, nil, nil, container.NewVScroll(body))
	return &a
}

func (a *mailDetailArea) Clear() {
	a.updateContent("", "", "")
	a.toogleIcons(false)
}

func (a *mailDetailArea) SetMail(mailID int32, listItemID widget.ListItemID) {
	characterID := a.ui.CurrentCharID()
	mail, err := a.ui.service.GetMail(characterID, mailID)
	if err != nil {
		slog.Error("Failed to fetch mail", "mailID", mailID, "error", err)
		return
	}
	a.mailID = mailID
	if !mail.IsRead {
		go func() {
			err := func() error {
				err = a.ui.service.UpdateMailRead(characterID, mail.MailID)
				if err != nil {
					return err
				}
				a.ui.headerArea.Refresh()
				return nil
			}()
			if err != nil {
				slog.Error("Failed to mark mail as read", "characterID", characterID, "mailID", mail.MailID, "error", err)
			}
		}()
	}

	header := mail.MakeHeaderText(myDateTime)
	a.updateContent(mail.Subject, header, mail.BodyToMarkdown())
	a.toogleIcons(true)

}

func (a *mailDetailArea) toogleIcons(enabled bool) {
	for _, i := range []int{0, 1, 2, 4} {
		b := a.icons.Objects[i].(*widget.Button)
		if enabled {
			b.Enable()
		} else {
			b.Disable()
		}
	}
}

func (a *mailDetailArea) updateContent(s string, h string, b string) {
	a.subject.SetText(s)
	a.header.SetText(h)
	a.body.ParseMarkdown(b)
}
