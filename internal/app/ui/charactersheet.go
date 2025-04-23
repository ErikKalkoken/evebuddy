package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type CharacterSheet struct {
	widget.BaseWidget

	allianceLogo    *kxwidget.TappableImage
	born            *widget.Label
	corporationLogo *kxwidget.TappableImage
	factionLogo     *kxwidget.TappableImage
	home            *iwidget.TappableRichText
	name            *iwidget.Label
	portrait        *kxwidget.TappableImage
	race            *kxwidget.TappableLabel
	security        *widget.Label
	sp              *widget.Label
	u               *BaseUI
	wealth          *widget.Label
}

func NewSheet(u *BaseUI) *CharacterSheet {
	makeLogo := func() *kxwidget.TappableImage {
		ti := kxwidget.NewTappableImage(icons.BlankSvg, nil)
		ti.SetFillMode(canvas.ImageFillContain)
		ti.SetMinSize(fyne.NewSquareSize(app.IconUnitSize))
		return ti
	}

	name := iwidget.NewLabelWithSize("", theme.SizeNameSubHeadingText)
	name.Wrapping = fyne.TextWrapWord
	home := iwidget.NewTappableRichTextWithText("?", nil)
	home.Wrapping = fyne.TextWrapWord

	portrait := kxwidget.NewTappableImage(icons.BlankSvg, nil)
	portrait.SetFillMode(canvas.ImageFillContain)
	portrait.SetMinSize(fyne.NewSquareSize(128))
	w := &CharacterSheet{
		allianceLogo:    makeLogo(),
		born:            widget.NewLabel("?"),
		corporationLogo: makeLogo(),
		factionLogo:     makeLogo(),
		home:            home,
		name:            name,
		portrait:        portrait,
		race:            kxwidget.NewTappableLabel("?", nil),
		security:        widget.NewLabel("?"),
		sp:              widget.NewLabel("?"),
		u:               u,
		wealth:          widget.NewLabel("?"),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (a *CharacterSheet) update() {
	c := a.u.currentCharacter()
	if c == nil || c.EveCharacter == nil {
		fyne.Do(func() {
			a.name.Text = "No character..."
			a.name.Importance = widget.WarningImportance
			a.name.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.name.SetText(c.EveCharacter.Name)
		a.race.SetText(c.EveCharacter.Race.Name)
		a.born.SetText(c.EveCharacter.Birthday.Format(app.DateTimeFormat))
		a.security.SetText(fmt.Sprintf("%.1f", c.EveCharacter.SecurityStatus))
		a.portrait.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntityCharacter, c.ID)
		}
		a.corporationLogo.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntityCorporation, c.EveCharacter.Corporation.ID)
		}
		a.race.OnTapped = func() {
			a.u.ShowRaceInfoWindow(c.EveCharacter.Race.ID)
		}
	})
	iwidget.RefreshTappableImageAsync(a.portrait, func() (fyne.Resource, error) {
		return a.u.EveImageService().CharacterPortrait(c.ID, 512)
	})
	iwidget.RefreshTappableImageAsync(a.corporationLogo, func() (fyne.Resource, error) {
		return a.u.EveImageService().CorporationLogo(c.EveCharacter.Corporation.ID, app.IconPixelSize)
	})
	fyne.Do(func() {
		if c.Home == nil {
			a.home.ParseMarkdown("")
			return
		}
		a.home.Segments = c.Home.DisplayRichText()
		a.home.OnTapped = func() {
			a.u.ShowLocationInfoWindow(c.Home.ID)
		}
		a.home.Refresh()
	})
	fyne.Do(func() {
		a.sp.SetText(ihumanize.OptionalComma(c.TotalSP, "?"))
		if c.AssetValue.IsEmpty() || c.WalletBalance.IsEmpty() {
			a.wealth.SetText("?")
			return
		}
		v := c.AssetValue.ValueOrZero() + c.WalletBalance.ValueOrZero()
		a.wealth.SetText(humanize.Comma(int64(v)) + " ISK")
	})
	fyne.Do(func() {
		if c.EveCharacter.Alliance == nil {
			a.allianceLogo.Hide()
			return
		}
		a.allianceLogo.Show()
		a.allianceLogo.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntityAlliance, c.EveCharacter.Alliance.ID)
		}
	})
	if c.EveCharacter.Alliance != nil {
		iwidget.RefreshTappableImageAsync(a.allianceLogo, func() (fyne.Resource, error) {
			return a.u.EveImageService().AllianceLogo(c.EveCharacter.Alliance.ID, app.IconPixelSize)
		})
	}
	fyne.Do(func() {
		if c.EveCharacter.Faction == nil {
			a.factionLogo.Hide()
			return
		}
		a.factionLogo.Show()
		a.factionLogo.OnTapped = func() {
			a.u.ShowInfoWindow(app.EveEntityFaction, c.EveCharacter.Faction.ID)
		}
	})
	if c.EveCharacter.Faction != nil {
		iwidget.RefreshTappableImageAsync(a.factionLogo, func() (fyne.Resource, error) {
			return a.u.EveImageService().FactionLogo(c.EveCharacter.Faction.ID, app.IconPixelSize)
		})
	}
}

func (a *CharacterSheet) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	const width = 140
	main := container.New(
		layout.NewCustomPaddedVBoxLayout(-p),
		NewDataRow(width, "Born", a.born),
		NewDataRow(width, "Race", a.race),
		NewDataRow(width, "Wealth", a.wealth),
		NewDataRow(width, "Security Status", a.security),
		NewDataRow(width, "Home Station", a.home),
		NewDataRow(width, "Total Skill Points", a.sp),
	)

	c := container.NewBorder(
		container.NewVBox(container.NewHBox(container.NewPadded(a.portrait))),
		nil,
		nil,
		nil,
		container.NewVBox(
			a.name,
			main,
			container.NewHBox(
				container.NewPadded(a.corporationLogo),
				container.NewPadded(a.allianceLogo),
				container.NewPadded(a.factionLogo)),
		),
	)
	return widget.NewSimpleRenderer(c)
}

type DataRow struct {
	widget.BaseWidget

	label   *widget.Label
	content fyne.CanvasObject
	width   float32
}

func NewDataRow(width float32, label string, content fyne.CanvasObject) *DataRow {
	w := &DataRow{
		label:   widget.NewLabel(label + ":"),
		content: content,
		width:   width,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *DataRow) CreateRenderer() fyne.WidgetRenderer {
	c := container.New(kxlayout.NewColumns(w.width), w.label, w.content)
	return widget.NewSimpleRenderer(c)
}
