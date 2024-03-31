package ui

import (
	"example/esiapp/internal/model"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// mailArea is the UI area showing the current mail.
type mailArea struct {
	content fyne.CanvasObject
	icons   *fyne.Container
	bodyC   *container.Scroll
	subject *widget.Label
	header  *widget.Label
	body    *widget.Label
	mailID  uint64
}

func (e *ui) newMailArea() *mailArea {
	btnReply := widget.NewButtonWithIcon("", theme.MailReplyIcon(), func() {
	})
	btnReplyAll := widget.NewButtonWithIcon("", theme.MailReplyAllIcon(), func() {
	})
	btnForward := widget.NewButtonWithIcon("", theme.MailForwardIcon(), func() {
	})
	btnDelete := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
	})
	icons := container.NewHBox(btnReply, btnReplyAll, btnForward, layout.NewSpacer(), btnDelete)
	for _, i := range []int{0, 1, 2, 4} {
		icons.Objects[i].(*widget.Button).Disable()
	}

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
	m := mailArea{
		content: content,
		bodyC:   bodyWithScroll,
		subject: subject,
		header:  header,
		body:    body,
		icons:   icons,
	}
	return &m
}

func (m *mailArea) update(mailID uint64) {
	mail, err := model.FetchMail(mailID)
	if err != nil {
		slog.Error("Failed to render mail", "mailID", mailID, "error", err)
		return
	}
	m.mailID = mailID
	var names []string
	for _, n := range mail.Recipients {
		names = append(names, n.Name)
	}
	header := fmt.Sprintf(
		"From: %s\nSent: %s\nTo: %s",
		mail.From.Name,
		mail.Timestamp.Format(myDateTime),
		strings.Join(names, ", "),
	)
	b := mail.BodyPlain()
	m.updateContent(mail.Subject, header, b)
	// for _, i := range []int{0, 1, 2, 4} {
	// 	m.icons.Objects[i].(*widget.Button).Enable()
	// }
}

func (m *mailArea) clear() {
	m.updateContent("", "", "")
}

func (m *mailArea) updateContent(s string, h string, b string) {
	m.subject.SetText(s)
	m.header.SetText(h)
	m.body.SetText(b)
	m.bodyC.ScrollToTop()
}
