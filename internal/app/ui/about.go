package ui

import (
	"fmt"
	"log/slog"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/github"
)

const (
	discordServerURL = "https://discord.gg/tVSCQEVJnJ"
)

func makeAboutPage(u *baseUI) fyne.CanvasObject {
	title := widget.NewLabel(u.appName())
	title.SizeName = theme.SizeNameSubHeadingText
	title.TextStyle.Bold = true

	v, err := github.NormalizeVersion(u.app.Metadata().Version)
	if err != nil {
		slog.Error("normalize local version", "error", err)
		v = "?"
	}
	currentVersion := widget.NewLabel(v)
	releaseNotes := widget.NewHyperlink("What's new", u.websiteRootURL().JoinPath("releases", "v"+v))

	_, size := u.MainWindow().Canvas().InteractiveArea()
	x := fmt.Sprintf("%d x %d", int(size.Width), int(size.Height))
	techInfos := container.New(layout.NewCustomPaddedVBoxLayout(0),
		container.NewHBox(widget.NewLabel("Main window size:"), layout.NewSpacer(), widget.NewLabel(x)),
	)
	if !u.IsDeveloperMode() {
		techInfos.Hide()
	}
	discordURL, _ := url.Parse(discordServerURL)
	support := widget.NewLabel("For support please open an issue on our web site or join our Discord server.")
	support.Wrapping = fyne.TextWrapWord
	updateAvailable := widget.NewHyperlink("Update available", u.websiteRootURL().JoinPath("releases"))
	updateAvailable.Hide()
	go func() {
		v, err := u.availableUpdate()
		if err != nil {
			slog.Error("Failed to fetch available updates")
			return
		}
		if !v.IsRemoteNewer {
			return
		}
		updateAvailable.URL = u.websiteRootURL().JoinPath("releases", "tag", "v"+v.Latest)
		updateAvailable.Show()
	}()
	c := container.New(
		layout.NewCustomPaddedVBoxLayout(0),
		title,
		container.NewHBox(currentVersion, releaseNotes),
		updateAvailable,
		techInfos,
		support,
		container.NewHBox(
			widget.NewHyperlink("Website", u.websiteRootURL()),
			widget.NewHyperlink("Downloads", u.websiteRootURL().JoinPath("releases")),
			widget.NewHyperlink("Discord", discordURL),
		),
		widget.NewLabel("\"EVE\", \"EVE Online\", \"CCP\", \nand all related logos and images \nare trademarks or registered trademarks of CCP hf."),
		widget.NewLabel("(c) 2024-25 Erik Kalkoken"),
	)
	return c
}
