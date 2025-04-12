package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// type colonyRow struct {
// 	character     string
// 	due           string
// 	dueColor      fyne.ThemeColorName
// 	extracting    string
// 	isExpired     bool
// 	planet        string
// 	planetType    app.EntityShort[int32]
// 	producing     string
// 	region        app.EntityShort[int32]
// 	security      string
// 	securityColor fyne.ThemeColorName
// 	solarSystemID int32
// 	characterID   int32
// }

type Colonies struct {
	widget.BaseWidget

	OnUpdate func(total, expired int)

	body    fyne.CanvasObject
	planets []*app.CharacterPlanet
	top     *widget.Label
	u       *BaseUI
}

func NewColonies(u *BaseUI) *Colonies {
	a := &Colonies{
		planets: make([]*app.CharacterPlanet, 0),
		top:     appwidget.MakeTopLabel(),
		u:       u,
	}
	a.ExtendBaseWidget(a)
	headers := []iwidget.HeaderDef{
		{Text: "Planet", Width: 150},
		{Text: "Type", Width: 100},
		{Text: "Extracting", Width: 200},
		{Text: "Due", Width: 150},
		{Text: "Producing", Width: 200},
		{Text: "Region", Width: 150},
		{Text: "Character", Width: columnWidthCharacter},
	}
	makeCell := func(col int, r *app.CharacterPlanet) []widget.RichTextSegment {
		switch col {
		case 0:
			return r.NameRichText()
		case 1:
			return iwidget.NewRichTextSegmentFromText(r.EvePlanet.TypeDisplay())
		case 2:
			return iwidget.NewRichTextSegmentFromText(r.Extracting())
		case 3:
			return r.DueRichText()
		case 4:
			return iwidget.NewRichTextSegmentFromText(r.Producing())
		case 5:
			return iwidget.NewRichTextSegmentFromText(r.EvePlanet.SolarSystem.Constellation.Region.Name)
		case 6:
			return iwidget.NewRichTextSegmentFromText(a.u.StatusCacheService().CharacterName(r.CharacterID))
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	if a.u.IsDesktop() {
		a.body = iwidget.MakeDataTableForDesktop(headers, &a.planets, makeCell, func(_ int, r *app.CharacterPlanet) {
			a.showColony(r)
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile(headers, &a.planets, makeCell, a.showColony)
	}
	return a
}

func (a *Colonies) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *Colonies) Update() {
	var s string
	var i widget.Importance
	var total, expired int
	if err := a.updateEntries(); err != nil {
		slog.Error("Failed to refresh wallet transaction UI", "err", err)
		s = "ERROR"
		i = widget.DangerImportance
	} else {
		total = len(a.planets)
		for _, c := range a.planets {
			if c.IsExpired() {
				expired++
			}
		}
		s = fmt.Sprintf("%d colonies", total)
		if expired > 0 {
			s += fmt.Sprintf(" • %d expired", expired)
		}
	}
	a.top.Text = s
	a.top.Importance = i
	a.top.Refresh()
	a.body.Refresh()
	if a.OnUpdate != nil {
		a.OnUpdate(total, expired)
	}
}

func (a *Colonies) updateEntries() error {
	oo, err := a.u.CharacterService().ListAllPlanets(context.TODO())
	if err != nil {
		return err
	}
	a.planets = oo
	return nil
}

func (a *Colonies) showColony(cp *app.CharacterPlanet) {
	characterName := a.u.StatusCacheService().CharacterName(cp.CharacterID)

	fi := []*widget.FormItem{
		widget.NewFormItem("Planet", iwidget.NewTappableRichText(
			func() {
				a.u.ShowEveEntityInfoWindow(cp.EvePlanet.SolarSystem.ToEveEntity())
			},
			cp.NameRichText()...,
		)),
		widget.NewFormItem("Type", kxwidget.NewTappableLabel(cp.EvePlanet.TypeDisplay(), func() {
			a.u.ShowEveEntityInfoWindow(cp.EvePlanet.Type.ToEveEntity())
		})),
		widget.NewFormItem("Region", kxwidget.NewTappableLabel(
			cp.EvePlanet.SolarSystem.Constellation.Region.Name,
			func() {
				a.u.ShowEveEntityInfoWindow(cp.EvePlanet.SolarSystem.Constellation.Region.ToEveEntity())
			})),
		widget.NewFormItem("Installations", widget.NewLabel(fmt.Sprint(len(cp.Pins)))),
		widget.NewFormItem("Character", kxwidget.NewTappableLabel(characterName, func() {
			a.u.ShowInfoWindow(app.EveEntityCharacter, cp.CharacterID)
		})),
	}
	f1 := widget.NewForm(fi...)
	f1.Orientation = widget.Adaptive

	extracting := container.NewVBox()
	for pp := range cp.ActiveExtractors() {
		if pp.ExpiryTime.IsEmpty() {
			continue
		}
		expiryTime := pp.ExpiryTime.ValueOrZero()
		icon, _ := pp.ExtractorProductType.Icon()
		row := container.NewHBox(
			iwidget.NewImageFromResource(icon, fyne.NewSquareSize(app.IconUnitSize)),
			kxwidget.NewTappableLabel(pp.ExtractorProductType.Name, func() {
				a.u.ShowEveEntityInfoWindow(pp.ExtractorProductType.ToEveEntity())
			}),
			widget.NewLabel(expiryTime.Format(app.DateTimeFormat)),
		)
		if expiryTime.Before(time.Now()) {
			l := widget.NewLabel("EXPIRED")
			l.Importance = widget.DangerImportance
			row.Add(l)
		}
		extracting.Add(row)
	}
	if len(extracting.Objects) == 0 {
		extracting.Add(widget.NewLabel("-"))
	}
	producing := container.NewVBox()
	for _, s := range cp.ProducedSchematics() {
		icon, _ := s.Icon()
		producing.Add(container.NewHBox(
			iwidget.NewImageFromResource(icon, fyne.NewSquareSize(app.IconUnitSize)),
			widget.NewLabel(s.Name),
		))
	}
	if len(producing.Objects) == 0 {
		producing.Add(widget.NewLabel("-"))
	}
	f2 := widget.NewForm(
		widget.NewFormItem("Extracting", extracting),
		widget.NewFormItem("Producing", producing),
	)
	top := container.NewHBox(f1, layout.NewSpacer())
	if a.u.IsDesktop() {
		res, _ := cp.EvePlanet.Type.Icon()
		image := iwidget.NewImageFromResource(res, fyne.NewSquareSize(100))
		top.Add(container.NewVBox(image))
	}
	c := container.NewVBox(top, f2)

	subTitle := fmt.Sprintf("%s - %s", cp.EvePlanet.Name, characterName)
	w := a.u.makeDetailWindow("Colony", subTitle, c)
	w.Show()
}
