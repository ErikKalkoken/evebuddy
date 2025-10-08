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
	DiscordServerURL = "https://discord.gg/tVSCQEVJnJ"
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

	discordURL, _ := url.Parse(DiscordServerURL)
	support := widget.NewLabel("For support please open an issue on our web site or join our Discord server.")
	support.Wrapping = fyne.TextWrapWord
	c := container.New(
		layout.NewCustomPaddedVBoxLayout(0),
		title,
		container.NewHBox(currentVersion, releaseNotes),
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
	if !u.IsDeveloperMode() {
		techInfos.Hide()
	}
	return c
}
