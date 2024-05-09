package model

import (
	"fmt"
	"html"
	"log"
	"regexp"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
)

// Special mail label IDs
const (
	MailLabelAll      = 1<<31 - 1
	MailLabelNone     = 0
	MailLabelInbox    = 1
	MailLabelSent     = 2
	MailLabelCorp     = 4
	MailLabelAlliance = 8
)

// A mail label for an Eve mail belonging to a character.
type MailLabel struct {
	ID            int64
	MyCharacterID int32
	Color         string
	LabelID       int32
	Name          string
	UnreadCount   int
}

var bodyPolicy = bluemonday.StrictPolicy()

// An Eve mail belonging to a character.
type Mail struct {
	Body          string
	MyCharacterID int32
	From          *EveEntity
	Labels        []*MailLabel
	IsRead        bool
	ID            int64
	MailID        int32
	Recipients    []*EveEntity
	Subject       string
	Timestamp     time.Time
}

// BodyPlain returns a mail's body as plain text.
func (m *Mail) BodyPlain() string {
	t := strings.ReplaceAll(m.Body, "<br>", "\n")
	b := html.UnescapeString(bodyPolicy.Sanitize(t))
	return b
}

// BodyForward returns a mail's body for a mail forward or reply.
func (m *Mail) ToString(format string) string {
	s := "\n---\n"
	s += m.MakeHeaderText(format)
	s += "\n\n"
	s += m.BodyPlain()
	return s
}

// MakeHeaderText returns the mail's header as formatted text.
func (m *Mail) MakeHeaderText(format string) string {
	var names []string
	for _, n := range m.Recipients {
		names = append(names, n.Name)
	}
	header := fmt.Sprintf(
		"From: %s\nSent: %s\nTo: %s",
		m.From.Name,
		m.Timestamp.Format(format),
		strings.Join(names, ", "),
	)
	return header
}

// RecipientNames returns the names of the recipients.
func (m *Mail) RecipientNames() []string {
	ss := make([]string, len(m.Recipients))
	for i, r := range m.Recipients {
		ss[i] = r.Name
	}
	return ss
}

func (m *Mail) BodyToMarkdown() string {
	t := strings.ReplaceAll(m.Body, "<loc>", "")
	t = strings.ReplaceAll(t, "</loc>", "")
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(t))
	if err != nil {
		log.Fatal(err)
	}
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if ok && href[:9] == "showinfo:" {
			var url string
			p := strings.Split(href[9:], "//")
			switch p[0] {
			case "1376":
				url = fmt.Sprintf("https://zkillboard.com/character/%s/", p[1])
			case "2":
				url = fmt.Sprintf("https://zkillboard.com/corporation/%s/", p[1])
			case "16159":
				url = fmt.Sprintf("https://zkillboard.com/alliance/%s/", p[1])
			case "5":
				url = fmt.Sprintf("https://zkillboard.com/system/%s/", p[1])
			default:
				url = "#"
			}
			s.SetAttr("href", url)
		}
		if ok && href[:9] == "killReport:" {
			s.SetAttr("href", "#")
		}
	})
	converter := md.NewConverter("", true, nil)
	textMD := converter.Convert(doc.Selection)
	return patchLinks(textMD)
}

// patchLinks will apply a workaround to address fyne issue #4340
func patchLinks(s string) string {
	r := regexp.MustCompile(`(\[\w*\]\(.*\))(\n\n)`)
	return string(r.ReplaceAll([]byte(s), []byte("$1 $2")))
}
