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
	color color.Color
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

	defaultColor := theme.ForegroundColor()
	dangerColor := theme.ErrorColor()
	successColor := theme.SuccessColor()
	x := canvas.NewText(character.Name, defaultColor)
	x.TextSize = theme.TextHeadingSize()
	c.items.Add(container.NewPadded(x))

	var secColor color.Color
	switch s := character.SecurityStatus; {
	case s < 0:
		secColor = dangerColor
	case s > 0:
		secColor = successColor
	default:
		secColor = defaultColor
	}

	var r = []item{
		{"Corporation", character.Corporation.Name, defaultColor},
		{"Alliance", stringOrDefault(character.Alliance.Name, "-"), defaultColor},
		{"Faction", stringOrDefault(character.Faction.Name, "-"), defaultColor},
		{"Race", character.Race.Name, defaultColor},
		{"Gender", character.Gender, defaultColor},
		{"Born", character.Birthday.Format(myDateTime), defaultColor},
	}
	form1 := makeForm(r)
	r = []item{
		{"Wallet Balance", numberOrDefault(character.WalletBalance, "?"), defaultColor},
		{"Skill Points", numberOrDefault(character.SkillPoints, "?"), defaultColor},
		{"Security Status", fmt.Sprintf("%.1f", character.SecurityStatus), secColor},
		{"Location", stringOrDefault(character.Location.Name, "?"), defaultColor},
		{"Ship", character.Ship.Name, defaultColor},
		{"Last Login", humanize.Time(character.LastLoginAt), defaultColor},
	}
	form2 := makeForm(r)
	c.items.Add(container.NewGridWithColumns(2, form1, form2))

	err := updateIcons(icons, character)
	if err != nil {
		slog.Error(err.Error())
	}
}

func makeForm(rows []item) *fyne.Container {
	fg := theme.ForegroundColor()
	form1 := container.New(layout.NewFormLayout())
	for _, row := range rows {
		label := canvas.NewText(row.label+":", fg)
		label.TextStyle = fyne.TextStyle{Bold: true}
		v := row.value
		if len(row.value) > 20 {
			v = fmt.Sprintf("%s...", row.value[:25])
		}
		value := canvas.NewText(v, row.color)
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

	u, err = c.Ship.IconURL(128)
	if err != nil {
		return err
	}
	ship := canvas.NewImageFromURI(u)
	ship.FillMode = canvas.ImageFillOriginal
	icons.Add(ship)

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
				err := c.ui.service.UpdateCharacterDetails(characterID)
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
