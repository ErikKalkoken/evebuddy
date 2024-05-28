package ui

import (
	"net/url"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (u *ui) showAboutDialog() {
	c := container.NewVBox()
	info := u.app.Metadata()
	appData := widget.NewRichTextFromMarkdown(
		"## " + appName(u.app) + "\n**Version:** " + info.Version)
	c.Add(appData)
	uri, _ := url.Parse("https://github.com/ErikKalkoken/evebuddy")
	c.Add(widget.NewHyperlink("Website", uri))
	c.Add(widget.NewLabel("\"EVE\", \"EVE Online\", \"CCP\", \nand all related logos and images \nare trademarks or registered trademarks of CCP hf."))
	c.Add(widget.NewLabel("(c) 2024 Erik Kalkoken"))
	d := dialog.NewCustom("About", "OK", c, u.window)
	d.Show()
}
