package infowindow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// characterInfo shows public information about a character.
type characterInfo struct {
	widget.BaseWidget
	baseInfo

	alliance        *widget.Hyperlink
	attributes      *attributeList
	bio             *widget.Label
	corporation     *widget.Hyperlink
	corporationLogo *canvas.Image
	description     *widget.Label
	employeeHistory *entityList
	id              int64
	membership      *widget.Label
	ownedIcon       *ttwidget.Icon
	portrait        *xwidget.TappableImage
	security        *widget.Label
	tabs            *container.AppTabs
	title           *widget.Label
}

func newCharacterInfo(iw *InfoWindow, id int64) *characterInfo {
	alliance := widget.NewHyperlink("", nil)
	alliance.Wrapping = fyne.TextWrapWord
	corporation := widget.NewHyperlink("", nil)
	corporation.Wrapping = fyne.TextWrapWord
	portrait := xwidget.NewTappableImage(icons.BlankSvg, nil)
	portrait.SetFillMode(canvas.ImageFillContain)
	portrait.SetMinSize(iw.renderIconSize())
	title := widget.NewLabel("")
	title.Wrapping = fyne.TextWrapWord
	bio := widget.NewLabel("")
	bio.Wrapping = fyne.TextWrapWord
	ownedIcon := ttwidget.NewIcon(theme.NewSuccessThemedResource(icons.CheckDecagramSvg))
	ownedIcon.SetToolTip("You own this character")
	ownedIcon.Hide()
	a := &characterInfo{
		alliance:        alliance,
		bio:             bio,
		corporation:     corporation,
		corporationLogo: xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		description:     newLabelWithWrapAndSelectable(""),
		id:              id,
		membership:      widget.NewLabel(""),
		ownedIcon:       ownedIcon,
		portrait:        portrait,
		security:        widget.NewLabel(""),
		title:           title,
	}
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.attributes = newAttributeList(a.iw)
	a.employeeHistory = newEntityListFromItems(a.iw.show)
	attributes := container.NewTabItem("Attributes", a.attributes)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Bio", container.NewVScroll(a.bio)),
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		attributes,
	)
	ee := app.EveEntity{ID: id, Category: app.EveEntityCharacter}
	if !ee.IsNPC().ValueOrZero() {
		a.tabs.Append(container.NewTabItem("Employment History", a.employeeHistory))
	}
	a.tabs.Select(attributes)
	return a
}

func (a *characterInfo) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	main := container.NewVBox(
		container.NewBorder(
			nil,
			nil,
			nil,
			container.NewPadded(a.ownedIcon),
			container.NewVBox(
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
						container.NewBorder(
							nil,
							nil,
							container.New(layout.NewCustomPaddedLayout(0, 0, 0, -3*p), widget.NewLabel("Member of")),
							nil,
							a.corporation,
						),
						a.membership,
					),
				),
			),
		),
		widget.NewSeparator(),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.alliance,
			a.security,
		),
	)
	forums := xwidget.NewTappableIcon(icons.EvelogoPng, func() {
		go func() {
			ec, err := a.iw.u.EVEUniverse().GetCharacterESI(context.Background(), a.id)
			if err != nil {
				a.iw.sb.Show("Failed to get character for forum: " + a.iw.u.ErrorDisplay(err))
				return
			}
			name := strings.ReplaceAll(ec.Name, " ", "_")
			fyne.Do(func() {
				a.iw.openURL(fmt.Sprintf("https://forums.eveonline.com/u/%s/summary", name))
			})
		}()
	})
	forums.SetToolTip("Show on forums.eveonline.com")
	top := container.NewBorder(
		nil,
		nil,
		container.New(
			layout.NewCustomPaddedVBoxLayout(2*p),
			a.portrait,
			container.New(
				layout.NewCustomPaddedHBoxLayout(3*p),
				layout.NewSpacer(),
				a.iw.makeZKillboardIcon(a.id, infoCharacter),
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

func (a *characterInfo) update(ctx context.Context) error {
	fyne.Do(func() {
		a.iw.u.EVEImage().CharacterPortraitAsync(a.id, 256, func(r fyne.Resource) {
			a.portrait.SetResource(r)
		})
	})
	o, _, err := a.iw.u.EVEUniverse().GetOrCreateCharacterESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.iw.u.EVEImage().CorporationLogoAsync(o.Corporation.ID, app.IconPixelSize, func(r fyne.Resource) {
			a.corporationLogo.Resource = r
			a.corporationLogo.Refresh()
		})
	})

	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.security.SetText(fmt.Sprintf("Security Status: %s", o.SecurityStatus.StringFunc("?", func(v float64) string {
			return fmt.Sprintf("%.1f", v)
		})))
		a.corporation.SetText(o.Corporation.Name)
		a.corporation.OnTapped = func() {
			a.iw.Show(o.Corporation)
		}
		a.portrait.OnTapped = func() {
			a.iw.showZoomWindow(o.Name, a.id, a.iw.u.EVEImage().CharacterPortraitAsync, a.iw.w)
		}
	})
	fyne.Do(func() {
		a.bio.SetText(o.DescriptionPlain())
		a.description.SetText(o.Race.Description)
		a.tabs.Refresh()
	})
	fyne.Do(func() {
		v, ok := o.Alliance.Value()
		if !ok {
			a.alliance.Hide()
			return
		}
		a.alliance.SetText(v.Name)
		a.alliance.OnTapped = func() {
			a.iw.Show(v)
		}
	})
	fyne.Do(func() {
		v, ok := o.Title.Value()
		if !ok {
			a.title.Hide()
			return
		}
		a.title.SetText("Title: " + v)
	})

	characterIDs, err := a.iw.u.Character().ListCharacterIDs(ctx)
	if err != nil {
		return err
	}
	isOwned := characterIDs.Contains(a.id)
	if isOwned {
		a.ownedIcon.Show()
	}

	attributes, err := a.makeAttributes(ctx, o, isOwned)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.attributes.set(attributes)
		a.tabs.Refresh()
	})

	history, err := a.iw.u.EVEUniverse().FetchCharacterCorporationHistory(ctx, a.id)
	if err != nil {
		return err
	}
	if len(history) == 0 {
		fyne.Do(func() {
			a.membership.Hide()
		})
		return nil
	}
	items := xslices.Map(history, historyItem2EntityItem)
	duration := humanize.RelTime(history[0].StartDate, time.Now(), "", "")
	fyne.Do(func() {
		a.employeeHistory.set(items...)
		a.membership.SetText(fmt.Sprintf("for %s", duration))
		a.tabs.Refresh()
	})
	return nil
}

func (a *characterInfo) makeAttributes(ctx context.Context, o *app.EveCharacter, isOwned bool) ([]attributeItem, error) {
	attributes := []attributeItem{
		newAttributeItem("Born", o.Birthday.Format(app.DateTimeFormat)),
		newAttributeItem("Race", o.Race),
		newAttributeItem("Security Status", o.SecurityStatus.StringFunc("?", func(v float64) string {
			return fmt.Sprintf("%.1f", v)
		})),
		newAttributeItem("Corporation", o.Corporation),
	}
	if v, ok := o.Alliance.Value(); ok {
		attributes = append(attributes, newAttributeItem("Alliance", v))
	}
	if v, ok := o.Alliance.Value(); ok {
		attributes = append(attributes, newAttributeItem("Faction", v))
	}
	var u any
	if v, ok := o.EveEntity().IsNPC().Value(); ok {
		u = v
	} else {
		u = "?"
	}
	attributes = append(attributes, newAttributeItem("NPC", u))
	if isOwned {
		c, err := a.iw.u.Character().GetCharacter(ctx, a.id)
		if err != nil {
			return nil, err
		}
		if v, ok := c.Home.Value(); ok {
			attributes = append(attributes, newAttributeItem("Home", v))
		}
		if v, ok := c.Location.Value(); ok {
			attributes = append(attributes, newAttributeItem("Location", v))
		}
		if v, ok := c.LastLoginAt.Value(); ok {
			attributes = append(attributes, newAttributeItem("Last Login", v))
		}
	}
	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			fyne.CurrentApp().Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	return attributes, nil
}
