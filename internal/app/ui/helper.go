package ui

import (
	"fmt"
	"log/slog"
	"regexp"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

func EntityNameOrFallback[T int | int32 | int64](e *app.EntityShort[T], fallback string) string {
	if e == nil {
		return fallback
	}
	return e.Name
}

// NewImageResourceAsync shows a placeholder resource and refreshes it once the main resource is loaded asynchronously.
func NewImageResourceAsync(placeholder fyne.Resource, loader func() (fyne.Resource, error)) *canvas.Image {
	image := canvas.NewImageFromResource(placeholder)
	RefreshImageResourceAsync(image, loader)
	return image
}

// RefreshImageResourceAsync refreshes the resource of an image asynchronously.
// This prevents fyne to wait with rendering an image until a resource is fully loaded from a web server.
func RefreshImageResourceAsync(image *canvas.Image, loader func() (fyne.Resource, error)) {
	go func() {
		r, err := loader()
		if err != nil {
			slog.Warn("Failed to fetch image resource", "err", err)
			r = theme.BrokenImageIcon()
		}
		image.Resource = r
		image.Refresh()
	}()
}

func SkillDisplayName[N int | int32 | int64 | uint | uint32 | uint64](name string, level N) string {
	return fmt.Sprintf("%s %s", name, ihumanize.RomanLetter(level))
}

func BoolIconResource(ok bool) fyne.Resource {
	if ok {
		return theme.NewSuccessThemedResource(theme.ConfirmIcon())
	}
	return theme.NewErrorThemedResource(theme.CancelIcon())
}

// NewCustomHyperlink returns a new hyperlink with a custom action.
func NewCustomHyperlink(text string, onTapped func()) *widget.Hyperlink {
	x := widget.NewHyperlink(text, nil)
	x.OnTapped = onTapped
	return x
}

// markdownStripLinks strips all links from a text in markdown.
func markdownStripLinks(s string) string {
	r := regexp.MustCompile(`\[(.+?)\]\((.+?)\)`)
	return r.ReplaceAllString(s, "**$1**")
}

func NewSubHeading(text string) *widget.RichText {
	x := widget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyle{
			ColorName: theme.ColorNameForeground,
			Inline:    false,
			SizeName:  theme.SizeNameSubHeadingText,
		},
		Text: text,
	})
	return x
}
