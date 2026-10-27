package core

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/icrowley/fake"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/github"
)

const (
	discordServerURL = "https://discord.gg/tVSCQEVJnJ"
)

// aboutPage is a widget that shows information about the app.
type aboutPage struct {
	widget.BaseWidget

	currentVersionLabel *widget.Label
	dummyEntry          *widget.Entry
	releaseNotesLink    *widget.Hyperlink
	rootURL             *url.URL
	showSnackbarButton  *widget.Button
	techInfoBox         *fyne.Container
	u                   *baseUI
	updateAvailableLink *widget.Hyperlink
	updateAvailableRow  *fyne.Container
}

// newAboutPage returns a new about page and starts checking for an update.
func newAboutPage(u *baseUI) *aboutPage {
	rootURL := ui.WebsiteRootURL()
	updateAvailableLink := widget.NewHyperlink("Download", rootURL.JoinPath("releases"))
	w := &aboutPage{
		currentVersionLabel: widget.NewLabel(""),
		dummyEntry:          widget.NewEntry(),
		releaseNotesLink:    widget.NewHyperlink("What's new", nil),
		rootURL:             rootURL,
		u:                   u,
		updateAvailableLink: updateAvailableLink,
		updateAvailableRow: container.NewHBox(
			widget.NewLabelWithStyle("Update available", fyne.TextAlignLeading, fyne.TextStyle{
				Bold: true,
			}),
			updateAvailableLink,
		),
	}
	w.ExtendBaseWidget(w)

	v, err := github.NormalizeVersion(u.app.Metadata().Version)
	isVersionValid := err == nil
	if !isVersionValid {
		slog.Error("normalize local version", "error", err)
		w.currentVersionLabel.SetText("?")
		w.releaseNotesLink.Hide()
	} else {
		w.currentVersionLabel.SetText(v)
		w.releaseNotesLink.SetURL(rootURL.JoinPath("releases", "v"+v))
	}

	_, size := u.MainWindow().Canvas().InteractiveArea()
	x := fmt.Sprintf("%d x %d", int(size.Width), int(size.Height))
	w.techInfoBox = container.New(layout.NewCustomPaddedVBoxLayout(0),
		container.NewHBox(widget.NewLabel("Main window size:"), layout.NewSpacer(), widget.NewLabel(x)),
	)
	w.showSnackbarButton = widget.NewButton("Show Snackbar (debug)", func() {
		u.DisplaySnackbar(fake.Paragraph())
	})
	w.dummyEntry.PlaceHolder = "Enter sb to fire a snackbar"
	var fired bool
	w.dummyEntry.OnChanged = func(s string) {
		if s == "sb" && !fired {
			fired = true
			u.DisplaySnackbar(fake.Paragraph())
		}
		if s == "" {
			fired = false
		}
	}
	if !u.IsDeveloperMode() {
		w.techInfoBox.Hide()
		w.showSnackbarButton.Hide()
		w.dummyEntry.Hide()
	}

	w.updateAvailableRow.Hide()
	// the check would fail on an invalid version
	if isVersionValid && !u.isOfflineMode {
		u.Go(w.checkForUpdate)
	}
	return w
}

func (w *aboutPage) CreateRenderer() fyne.WidgetRenderer {
	title := widget.NewLabel(ui.Name())
	title.SizeName = theme.SizeNameSubHeadingText
	title.TextStyle.Bold = true
	support := widget.NewLabel("For support please open an issue on our web site or join our Discord server.")
	support.Wrapping = fyne.TextWrapWord
	discordURL, _ := url.Parse(discordServerURL)
	c := container.New(
		layout.NewCustomPaddedVBoxLayout(0),
		title,
		container.NewHBox(w.currentVersionLabel, w.releaseNotesLink),
		w.updateAvailableRow,
		w.techInfoBox,
		w.showSnackbarButton,
		w.dummyEntry,
		support,
		container.NewHBox(
			widget.NewHyperlink("Website", w.rootURL),
			widget.NewHyperlink("Downloads", w.rootURL.JoinPath("releases")),
			widget.NewHyperlink("Discord", discordURL),
		),
		widget.NewLabel("\"EVE\", \"EVE Online\", \"CCP\", \nand all related logos and images \nare trademarks or registered trademarks of CCP hf."),
		widget.NewLabel("(c) 2024-26 Erik Kalkoken"),
	)
	return widget.NewSimpleRenderer(c)
}

// checkForUpdate shows a download link when a newer version is available.
func (w *aboutPage) checkForUpdate(ctx context.Context) {
	v, err := w.u.availableUpdate(ctx)
	if err != nil {
		if !app.IsContextCanceled(ctx, err) {
			slog.Error("fetch latest github version for about page", "err", err)
		}
		return
	}
	if !v.IsRemoteNewer {
		return
	}
	fyne.Do(func() {
		if w.u.Signals().IsShuttingDown() {
			return
		}
		w.updateAvailableLink.SetURL(w.rootURL.JoinPath("releases", "tag", "v"+v.Latest))
		w.updateAvailableRow.Show()
		w.Refresh() // the row may not have been visible before
	})
}
