package gui

import (
	"example/esiapp/internal/model"
	"fmt"
	"html"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/microcosm-cc/bluemonday"
)

type mail struct {
	content fyne.CanvasObject
	icons   *fyne.Container
	bodyC   *container.Scroll
	subject *widget.Label
	header  *widget.Label
	body    *widget.Label
	policy  *bluemonday.Policy
	mailID  uint64
}

func (e *eveApp) newMail() *mail {
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
	policy := bluemonday.StrictPolicy()
	m := mail{
		content: content,
		bodyC:   bodyWithScroll,
		subject: subject,
		header:  header,
		body:    body,
		policy:  policy,
		icons:   icons,
	}
	return &m
}

func (m *mail) update(mailID uint64) {
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
	t := strings.ReplaceAll(mail.Body, "<br>", "\n")
	body := html.UnescapeString(m.policy.Sanitize(t))
	m.updateContent(mail.Subject, header, body)
	// for _, i := range []int{0, 1, 2, 4} {
	// 	m.icons.Objects[i].(*widget.Button).Enable()
	// }
}

func (m *mail) clear() {
	m.updateContent("", "", "")
}

func (m *mail) updateContent(s string, h string, b string) {
	m.subject.SetText(s)
	m.header.SetText(h)
	m.body.SetText(b)
	m.bodyC.ScrollToTop()
}
