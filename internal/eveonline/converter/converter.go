package converter

import (
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
)

// XMLtoMarkdown convert Eve Online XML text to markdown and return it.
func XMLtoMarkdown(xml string) string {
	t := strings.ReplaceAll(xml, "<loc>", "")
	t = strings.ReplaceAll(t, "</loc>", "")
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
	r := regexp.MustCompile(`(\[\w*\]\(.*\))(\n\n)`)
	return string(r.ReplaceAll([]byte(s), []byte("$1 $2")))
}
