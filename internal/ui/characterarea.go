package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"

	"github.com/dustin/go-humanize"

	"example/evebuddy/internal/api/images"
	ihumanize "example/evebuddy/internal/helper/humanize"
	"example/evebuddy/internal/model"
)

type item struct {
	label string
	value string
}

// characterArea is the UI area that shows the character sheet
type characterArea struct {
	content fyne.CanvasObject
	items   *fyne.Container
	ui      *ui
}

func (u *ui) NewCharacterArea() *characterArea {
	items := container.NewVBox()
	c := characterArea{ui: u, content: container.NewScroll(items), items: items}
	return &c
}

func (c *characterArea) Redraw() {
	c.items.RemoveAll()
	character := c.ui.CurrentChar()
	if character == nil {
		return
	}

	icons := container.NewHBox()
	c.items.Add(icons)

	fg := theme.ForegroundColor()
	x := canvas.NewText(character.Name, fg)
	x.TextSize = theme.TextHeadingSize()
	c.items.Add(container.NewPadded(x))

	var r = []item{
		{"Corporation", character.Corporation.Name},
		{"Alliance", stringOrDefault(character.Alliance.Name, "-")},
		{"Faction", stringOrDefault(character.Faction.Name, "-")},
		{"Race", "PLACEHOLDER"},
		{"Gender", character.Gender},
		{"Born", character.Birthday.Format(myDateTime)},
		{"Security Status", fmt.Sprintf("%.1f", character.SecurityStatus)},
	}
	form1 := makeForm(r, fg)
	r = []item{
		{"Skill Points", numberOrDefault(character.SkillPoints, "?")},
		{"Wallet Balance", numberOrDefault(character.WalletBalance, "?")},
		{"Home Station", "PLACEHOLDER"},
		{"Location", stringOrDefault(character.SolarSystem.Name, "?")},
		{"Ship", "PLACEHOLDER"},
		{"Last Login", humanize.Time(character.LastLoginAt)},
	}
	form2 := makeForm(r, fg)
	c.items.Add(container.NewGridWithColumns(2, form1, form2))

	err := updateIcons(icons, character)
	if err != nil {
		slog.Error(err.Error())
	}
}

func makeForm(rows []item, fg color.Color) *fyne.Container {
	form1 := container.New(layout.NewFormLayout())
	for _, row := range rows {
		label := canvas.NewText(row.label+":", fg)
		label.TextStyle = fyne.TextStyle{Bold: true}
		value := canvas.NewText(row.value, fg)
		form1.Add(container.NewPadded(label))
		form1.Add(container.NewPadded(value))
	}
	return form1
}

func updateIcons(icons *fyne.Container, c *model.Character) error {
	u, err := images.CharacterPortraitURL(c.ID, 128)
	if err != nil {
		return err
	}
	character := canvas.NewImageFromURI(u)
	character.FillMode = canvas.ImageFillOriginal
	icons.Add(character)

	u, err = c.Corporation.IconURL(128)
	if err != nil {
		return err
	}
	corp := canvas.NewImageFromURI(u)
	corp.FillMode = canvas.ImageFillOriginal
	icons.Add(corp)

	if c.HasAlliance() {
		u, err = c.Alliance.IconURL(128)
		if err != nil {
			return err
		}
		image := canvas.NewImageFromURI(u)
		image.FillMode = canvas.ImageFillOriginal
		icons.Add(image)
	}

	if c.HasFaction() {
		u, err = c.Faction.IconURL(128)
		if err != nil {
			return err
		}
		image := canvas.NewImageFromURI(u)
		image.FillMode = canvas.ImageFillOriginal
		icons.Add(image)
	}

	icons.Add(layout.NewSpacer())
	return nil
}

func stringOrDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func numberOrDefault[T int | float64](v T, d string) string {
	if v == 0 {
		return d
	}
	return ihumanize.Number(float64(v), 1)
}

func (c *characterArea) StartUpdateTicker() {
	ticker := time.NewTicker(120 * time.Second)
	go func() {
		for {
			characterID := c.ui.CurrentCharID()
			if characterID != 0 {
				err := c.ui.service.UpdateCharacter(characterID)
				if err != nil {
					slog.Error(err.Error())
				} else {
					c.Redraw()
				}
			}
			<-ticker.C
		}
	}()
}
