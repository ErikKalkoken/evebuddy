package infowindow

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// characterInfo shows public information about a character.
type characterInfo struct {
	widget.BaseWidget

	alliance        *kxwidget.TappableLabel
	bio             *widget.Label
	corporation     *kxwidget.TappableLabel
	corporationLogo *canvas.Image
	description     *widget.Label
	employeeHistory *entityList
	id              int32
	iw              *InfoWindow
	membership      *widget.Label
	name            *widget.Label
	portrait        *kxwidget.TappableImage
	security        *widget.Label
	tabs            *container.AppTabs
	title           *widget.Label
	attributes      *attributeList
}

func newCharacterInfo(iw *InfoWindow, id int32) *characterInfo {
	alliance := kxwidget.NewTappableLabel("", nil)
	alliance.Wrapping = fyne.TextWrapWord
	corporation := kxwidget.NewTappableLabel("", nil)
	corporation.Wrapping = fyne.TextWrapWord
	portrait := kxwidget.NewTappableImage(icons.Characterplaceholder64Jpeg, nil)
	portrait.SetFillMode(canvas.ImageFillContain)
	portrait.SetMinSize(fyne.NewSquareSize(renderIconUnitSize))
	title := widget.NewLabel("")
	title.Wrapping = fyne.TextWrapWord
	bio := widget.NewLabel("")
	bio.Wrapping = fyne.TextWrapWord
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	a := &characterInfo{
		alliance:        alliance,
		bio:             bio,
		corporation:     corporation,
		corporationLogo: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		description:     description,
		id:              id,
		iw:              iw,
		membership:      widget.NewLabel(""),
		name:            makeInfoName(),
		portrait:        portrait,
		security:        widget.NewLabel(""),
		title:           title,
	}
	a.ExtendBaseWidget(a)
	a.attributes = newAttributeList(a.iw)
	a.employeeHistory = newEntityListFromItems(a.iw.show)
	attributes := container.NewTabItem("Attributes", a.attributes)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Bio", container.NewVScroll(a.bio)),
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		attributes,
		container.NewTabItem("Employment History", a.employeeHistory),
	)
	a.tabs.Select(attributes)
	return a
}

func (a *characterInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("character info update failed", "character", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load character: %s", a.iw.u.ErrorDisplay(err))
				a.name.Importance = widget.DangerImportance
				a.name.Refresh()
			})
		}
	}()
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			a.title,
		),
		container.NewBorder(
			nil,
			nil,
			a.corporationLogo,
			nil,
			container.New(
				layout.NewCustomPaddedVBoxLayout(-2*p),
				a.corporation,
				a.membership,
			),
		),
		widget.NewSeparator(),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.alliance,
			a.security,
		),
	)
	name := a.iw.u.StatusCacheService().CharacterName(a.id)
	name = strings.ReplaceAll(name, " ", "_")
	forums := iwidget.NewTappableIcon(icons.EvelogoPng, func() {
		a.iw.openURL(fmt.Sprintf("https://forums.eveonline.com/u/%s/summary", name))
	})
	forums.SetToolTip("Show on forums.eveonline.com")
	top := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			a.portrait,
			container.New(
				layout.NewCustomPaddedHBoxLayout(3*p),
				layout.NewSpacer(),
				a.iw.makeZkillboardIcon(a.id, infoCharacter),
				a.iw.makeEveWhoIcon(a.id, infoCharacter),
				forums,
				layout.NewSpacer(),
			),
		),
		nil,
		main,
	)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *characterInfo) load() error {
	ctx := context.Background()
	go func() {
		r, err := a.iw.u.EveImageService().CharacterPortrait(a.id, 256)
		if err != nil {
			slog.Error("character info: Failed to load portrait", "characterID", a.id, "error", err)
			return
		}
		fyne.Do(func() {
			a.portrait.SetResource(r)
		})
	}()
	go func() {
		history, err := a.iw.u.EveUniverseService().GetCharacterCorporationHistory(ctx, a.id)
		if err != nil {
			slog.Error("character info: Failed to load corporation history", "characterID", a.id, "error", err)
			return
		}
		if len(history) == 0 {
			fyne.Do(func() {
				a.membership.Hide()
			})
			return
		}
		items := xslices.Map(history, historyItem2EntityItem)
		fyne.Do(func() {
			a.employeeHistory.set(items...)
			current := history[0]
			duration := humanize.RelTime(current.StartDate, time.Now(), "", "")
			a.membership.SetText(fmt.Sprintf("for %s", duration))
		})
	}()
	o, err := a.iw.u.EveUniverseService().GetCharacterESI(ctx, a.id)
	if err != nil {
		return err
	}
	go func() {
		r, err := a.iw.u.EveImageService().CorporationLogo(o.Corporation.ID, app.IconPixelSize)
		if err != nil {
			slog.Error("character info: Failed to load corp logo", "characterID", a.id, "error", err)
			return
		}
		fyne.Do(func() {
			a.corporationLogo.Resource = r
			a.corporationLogo.Refresh()
		})
	}()
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.security.SetText(fmt.Sprintf("Security Status: %.1f", o.SecurityStatus))
		a.corporation.SetText(fmt.Sprintf("Member of %s", o.Corporation.Name))
		a.corporation.OnTapped = func() {
			a.iw.ShowEveEntity(o.Corporation)
		}
		a.portrait.OnTapped = func() {
			go fyne.Do(func() {
				a.iw.showZoomWindow(o.Name, a.id, a.iw.u.EveImageService().CharacterPortrait, a.iw.w)
			})
		}
	})
	fyne.Do(func() {
		a.bio.SetText(o.DescriptionPlain())
		a.description.SetText(o.RaceDescription())
	})
	fyne.Do(func() {
		if !o.HasAlliance() {
			a.alliance.Hide()
			return
		}
		a.alliance.SetText(o.Alliance.Name)
		a.alliance.OnTapped = func() {
			a.iw.ShowEveEntity(o.Alliance)
		}
	})
	fyne.Do(func() {
		if o.Title == "" {
			a.title.Hide()
			return
		}
		a.title.SetText("Title: " + o.Title)
	})
	attributes := []attributeItem{
		newAttributeItem("Born", o.Birthday.Format(app.DateTimeFormat)),
		newAttributeItem("Race", o.Race),
		newAttributeItem("Security Status", fmt.Sprintf("%.1f", o.SecurityStatus)),
		newAttributeItem("Corporation", o.Corporation),
	}
	if o.Alliance != nil {
		attributes = append(attributes, newAttributeItem("Alliance", o.Alliance))
	}
	if o.Faction != nil {
		attributes = append(attributes, newAttributeItem("Faction", o.Faction))
	}
	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			a.iw.u.App().Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	fyne.Do(func() {
		a.attributes.set(attributes)
	})
	return nil
}
