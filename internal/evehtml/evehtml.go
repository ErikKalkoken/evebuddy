// Package evehtml provides converters to deal with Eve Online HTML.
package evehtml

import (
	"fmt"
	"html"
	"log"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
)

var reHorizontalRuler = regexp.MustCompile(`(--+)`)
var reLinks = regexp.MustCompile(`(\[\w*\]\(.*\))(\n\n)`)

// ToPlain converts custom Eve Online HTML text to plain text and returns it.
func ToPlain(xml string) string {
	t := strings.ReplaceAll(xml, "<br>", "\n")
	bodyPolicy := bluemonday.StrictPolicy()
	b := html.UnescapeString(bodyPolicy.Sanitize(t))
	return b
}

// ToMarkdown converts custom Eve Online HTML text to markdown and returns it.
func ToMarkdown(xml string) string {
	t := strings.ReplaceAll(xml, "<loc>", "")
	t = strings.ReplaceAll(t, "</loc>", "")
	t = reHorizontalRuler.ReplaceAllString(t, `<br><hr><br>`)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(t))
	if err != nil {
		log.Fatal(err)
	}
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		u, err := url.Parse(href)
		if err != nil {
			slog.Warn("Failed to parse href", "href", href)
			s.SetAttr("href", "#") // Ignore link if it can not be parsed
			return
		}
		switch u.Scheme {
		case "showinfo":
			var url string
			p := strings.Split(u.Opaque, "//")
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
		case "killreport":
			fallthrough
		case "fitting":
			s.SetAttr("href", "#")
		}
	})
	converter := md.NewConverter("", true, nil)
	textMD := converter.Convert(doc.Selection)
	return patchLinks(textMD)
}

// patchLinks will apply a workaround to address fyne issue #4340
func patchLinks(s string) string {
	return string(reLinks.ReplaceAll([]byte(s), []byte("$1 $2")))
}

// Strip removes all XML/HTML from a given string and return the result.
func Strip(xml string) string {
	bodyPolicy := bluemonday.StrictPolicy()
	b := html.UnescapeString(bodyPolicy.Sanitize(xml))
	return b
}
