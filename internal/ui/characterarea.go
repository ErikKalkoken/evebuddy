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

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
)

const bigIconSize = 128

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

func (a *characterArea) Redraw() {
	a.items.RemoveAll()
	character := a.ui.CurrentChar()
	if character == nil {
		return
	}

	icons := container.NewHBox()
	a.items.Add(icons)

	defaultColor := theme.ForegroundColor()
	dangerColor := theme.ErrorColor()
	successColor := theme.SuccessColor()
	x := canvas.NewText(character.Character.Name, defaultColor)
	x.TextSize = theme.TextHeadingSize()
	a.items.Add(container.NewPadded(x))

	var secColor color.Color
	switch s := character.Character.SecurityStatus; {
	case s < 0:
		secColor = dangerColor
	case s > 0:
		secColor = successColor
	default:
		secColor = defaultColor
	}

	var r = []item{
		{"Corporation", character.Character.Corporation.Name, defaultColor},
		{"Alliance", stringOrFallback(character.Character.AllianceName(), "-"), defaultColor},
		{"Faction", stringOrFallback(character.Character.FactionName(), "-"), defaultColor},
		{"Race", character.Character.Race.Name, defaultColor},
		{"Gender", character.Character.Gender, defaultColor},
		{"Born", character.Character.Birthday.Format(myDateTime), defaultColor},
		{"Title", stringOrFallback(character.Character.Title, "-"), defaultColor},
	}
	form1 := makeForm(r)
	location := fmt.Sprintf(
		"%s %.1f (%s)",
		character.Location.Name,
		character.Location.SecurityStatus,
		character.Location.Constellation.Region.Name,
	)
	r = []item{
		{"Wallet Balance", numberOrDefault(character.WalletBalance, "?"), defaultColor},
		{"Skill Points", numberOrDefault(character.SkillPoints, "?"), defaultColor},
		{"Security Status", fmt.Sprintf("%.1f", character.Character.SecurityStatus), secColor},
		{"Location", location, defaultColor},
		{"Ship", character.Ship.Name, defaultColor},
		{"Last Login", humanize.Time(character.LastLoginAt), defaultColor},
	}
	form2 := makeForm(r)
	a.items.Add(container.NewGridWithColumns(2, form1, form2))

	err := updateIcons(a.ui, icons, character)
	if err != nil {
		slog.Error(err.Error())
	}
}

func makeForm(rows []item) *fyne.Container {
	const maxChars = 25
	fg := theme.ForegroundColor()
	form1 := container.New(layout.NewFormLayout())
	for _, row := range rows {
		label := canvas.NewText(row.label+":", fg)
		label.TextStyle = fyne.TextStyle{Bold: true}
		v := row.value
		if len(row.value) > maxChars {
			v = fmt.Sprintf("%s...", row.value[:maxChars])
		}
		value := canvas.NewText(v, row.color)
		form1.Add(container.NewPadded(label))
		form1.Add(container.NewPadded(value))
	}
	return form1
}

func updateIcons(u *ui, icons *fyne.Container, c *model.MyCharacter) error {
	r := u.imageManager.CharacterPortrait(c.ID, bigIconSize)
	character := canvas.NewImageFromResource(r)
	character.FillMode = canvas.ImageFillOriginal
	icons.Add(character)

	r = u.imageManager.CorporationLogo(c.ID, bigIconSize)
	corp := canvas.NewImageFromResource(r)
	corp.FillMode = canvas.ImageFillOriginal
	icons.Add(corp)

	if c.Character.HasAlliance() {
		r = u.imageManager.AllianceLogo(c.ID, bigIconSize)
		image := canvas.NewImageFromResource(r)
		image.FillMode = canvas.ImageFillOriginal
		icons.Add(image)
	}

	if c.Character.HasFaction() {
		r = u.imageManager.FactionLogo(c.ID, bigIconSize)
		image := canvas.NewImageFromResource(r)
		image.FillMode = canvas.ImageFillOriginal
		icons.Add(image)
	}

	r = u.imageManager.InventoryTypeRender(c.Ship.ID, bigIconSize)
	ship := canvas.NewImageFromResource(r)
	ship.FillMode = canvas.ImageFillOriginal
	icons.Add(ship)

	icons.Add(layout.NewSpacer())
	return nil
}

func (a *characterArea) StartUpdateTicker() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			func() {
				characterID := a.ui.CurrentCharID()
				if characterID == 0 {
					return
				}
				if !a.ui.service.SectionUpdatedExpired(characterID, service.UpdateSectionMyCharacter) {
					return
				}
				if err := a.ui.service.UpdateMyCharacter(characterID); err != nil {
					slog.Error(err.Error())
					return
				}
				a.Redraw()
			}()
			<-ticker.C
		}
	}()
}
