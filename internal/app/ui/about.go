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
	v, err := github.NormalizeVersion(u.app.Metadata().Version)
	if err != nil {
		slog.Error("normalize local version", "error", err)
		v = "?"
	}
	local := widget.NewLabel(v)
	latest := widget.NewLabel("?")
	spinner := widget.NewActivity()
	if !u.IsOffline() {
		latest.Hide()
		spinner.Start()
		go func() {
			var s string
			var i widget.Importance
			var isBold bool
			v, err := u.availableUpdate()
			if err != nil {
				slog.Error("fetch github version for about", "error", err)
				s = "ERROR"
				i = widget.DangerImportance
			} else if v.IsRemoteNewer {
				s = v.Latest
				isBold = true
			} else {
				s = v.Latest
			}
			fyne.Do(func() {
				latest.Text = s
				latest.TextStyle.Bold = isBold
				latest.Importance = i
				latest.Refresh()
				spinner.Hide()
				latest.Show()
			})
		}()
	} else {
		spinner.Hide()
		latest.SetText("Offline")
		latest.Importance = widget.LowImportance
	}
	title := widget.NewLabel(u.appName())
	title.SizeName = theme.SizeNameSubHeadingText
	title.TextStyle.Bold = true

	_, size := u.MainWindow().Canvas().InteractiveArea()
	x := fmt.Sprintf("%d x %d", int(size.Width), int(size.Height))
	techInfos := container.New(layout.NewCustomPaddedVBoxLayout(0),
		container.NewHBox(widget.NewLabel("Main window size:"), layout.NewSpacer(), widget.NewLabel(x)),
	)
	discordURL, _ := url.Parse(DiscordServerURL)
	c := container.New(
		layout.NewCustomPaddedVBoxLayout(0),
		title,
		container.New(layout.NewCustomPaddedVBoxLayout(0),
			container.NewHBox(widget.NewLabel("Latest version:"), layout.NewSpacer(), container.NewStack(spinner, latest)),
			container.NewHBox(widget.NewLabel("You have:"), layout.NewSpacer(), local),
		),
		techInfos,
		widget.NewLabel("For support please open an issue on the web site or join our Discord server."),
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
