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
)

// mailDetailArea is the UI area showing the current mail.
type mailDetailArea struct {
	content fyne.CanvasObject
	icons   *fyne.Container
	bodyC   *container.Scroll
	subject *widget.Label
	header  *widget.Label
	body    *widget.Label
	mailID  int32
	ui      *ui
}

func (u *ui) NewMailArea() *mailDetailArea {
	icons := container.NewHBox()

	subject := widget.NewLabel("")
	subject.TextStyle = fyne.TextStyle{Bold: true}
	subject.Truncation = fyne.TextTruncateEllipsis

	header := widget.NewLabel("")
	header.Truncation = fyne.TextTruncateEllipsis

	wrapper := container.NewVBox(icons, subject, header)

	body := widget.NewLabel("")
	body.Wrapping = fyne.TextWrapBreak
	bodyWithScroll := container.NewVScroll(body)
	content := container.NewBorder(wrapper, nil, nil, nil, bodyWithScroll)
	m := mailDetailArea{
		content: content,
		bodyC:   bodyWithScroll,
		subject: subject,
		header:  header,
		body:    body,
		icons:   icons,
		ui:      u,
	}
	return &m
}

func (m *mailDetailArea) Clear() {
	m.updateContent("", "", "")
}

func (m *mailDetailArea) Redraw(mailID int32, listItemID widget.ListItemID) {
	characterID := m.ui.CurrentCharID()
	mail, err := m.ui.service.GetMail(characterID, mailID)
	if err != nil {
		slog.Error("Failed to render mail", "mailID", mailID, "error", err)
		return
	}
	m.mailID = mailID
	if !mail.IsRead {
		go func() {
			err := func() error {
				err = m.ui.service.UpdateMailRead(characterID, mail.MailID)
				if err != nil {
					return err
				}
				err = m.ui.headerArea.listData.SetValue(listItemID, int(m.mailID))
				if err != nil {
					return err
				}
				m.ui.headerArea.list.RefreshItem(listItemID)
				return nil
			}()
			if err != nil {
				slog.Error("Failed to mark mail as read", "characterID", characterID, "mailID", mail.MailID, "error", err)
			}
		}()
	}
	m.icons.RemoveAll()
	m.icons.Add(
		widget.NewButtonWithIcon("", theme.MailReplyIcon(), func() {
			m.ui.ShowCreateMessageWindow(CreateMessageReply, &mail)
		}),
	)
	m.icons.Add(
		widget.NewButtonWithIcon("", theme.MailReplyAllIcon(), func() {
			m.ui.ShowCreateMessageWindow(CreateMessageReplyAll, &mail)
		}),
	)
	m.icons.Add(
		widget.NewButtonWithIcon("", theme.MailForwardIcon(), func() {
			m.ui.ShowCreateMessageWindow(CreateMessageForward, &mail)
		}),
	)
	m.icons.Add(layout.NewSpacer())
	button := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		t := fmt.Sprintf("Are you sure you want to delete this mail?\n\n%s", mail.Subject)
		d := dialog.NewConfirm("Delete mail", t, func(confirmed bool) {
			if confirmed {
				err := m.ui.service.DeleteMail(mail.CharacterID, mail.MailID)
				if err != nil {
					errorDialog := dialog.NewError(err, m.ui.window)
					errorDialog.Show()
				} else {
					m.ui.headerArea.RedrawCurrent()
				}
			}
		}, m.ui.window)
		d.Show()
	})
	button.Importance = widget.DangerImportance
	m.icons.Add(button)
	header := mail.MakeHeaderText(myDateTime)
	b := mail.BodyPlain()
	m.updateContent(mail.Subject, header, b)
	for _, i := range []int{0, 1, 2, 4} {
		m.icons.Objects[i].(*widget.Button).Enable()
	}
}

func (m *mailDetailArea) updateContent(s string, h string, b string) {
	m.subject.SetText(s)
	m.header.SetText(h)
	m.body.SetText(b)
	m.bodyC.ScrollToTop()
}