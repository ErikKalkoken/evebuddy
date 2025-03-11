package ui

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type corporationAttribute struct {
	label string
	value any
}

// CorporationInfoArea represents an area that shows public information about a character.
type CorporationInfoArea struct {
	Content fyne.CanvasObject

	alliance        *widget.Label
	allianceLogo    *canvas.Image
	attributes      []corporationAttribute
	attributeList   *widget.List
	corporation     *widget.Label
	corporationLogo *canvas.Image
	hq              *kxwidget.TappableLabel
	historyList     *widget.List
	historyItems    []app.MembershipHistoryItem
	tabs            *container.AppTabs
	u               *BaseUI
}

func NewCorporationInfoArea(u *BaseUI, corporationID int32) *CorporationInfoArea {
	alliance := widget.NewLabel("")
	alliance.Truncation = fyne.TextTruncateEllipsis
	corporation := widget.NewLabel("Loading...")
	corporation.Truncation = fyne.TextTruncateEllipsis
	hq := kxwidget.NewTappableLabel("", nil)
	hq.Truncation = fyne.TextTruncateEllipsis
	corporationLogo := iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(DefaultIconUnitSize))
	s := float32(DefaultIconPixelSize) * 1.3 / u.Window.Canvas().Scale()
	corporationLogo.SetMinSize(fyne.NewSquareSize(s))
	a := &CorporationInfoArea{
		alliance:        alliance,
		allianceLogo:    iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(DefaultIconUnitSize)),
		attributes:      make([]corporationAttribute, 0),
		corporation:     corporation,
		corporationLogo: corporationLogo,
		historyItems:    make([]app.MembershipHistoryItem, 0),
		hq:              hq,
		tabs:            container.NewAppTabs(),
		u:               u,
	}

	main := container.New(layout.NewCustomPaddedVBoxLayout(0),
		a.corporation,
		a.hq,
		container.NewBorder(
			nil,
			nil,
			a.allianceLogo,
			nil,
			a.alliance,
		),
	)
	a.attributeList = a.makeAttributes()
	a.historyList = a.makeHistory()
	top := container.NewBorder(nil, nil, container.NewVBox(a.corporationLogo), nil, main)
	a.Content = container.NewBorder(top, nil, nil, nil, a.tabs)

	go func() {
		err := a.load(corporationID)
		if err != nil {
			slog.Error("corporation info update failed", "id", corporationID, "error", err)
			a.corporation.Text = fmt.Sprintf("ERROR: Failed to load corporation: %s", ihumanize.Error(err))
			a.corporation.Importance = widget.DangerImportance
			a.corporation.Refresh()
		}
	}()
	return a
}

func (a *CorporationInfoArea) makeAttributes() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.attributes)
		},
		func() fyne.CanvasObject {
			value := widget.NewLabel("Value")
			value.Truncation = fyne.TextTruncateEllipsis
			value.Alignment = fyne.TextAlignTrailing
			icon := widget.NewIcon(theme.InfoIcon())
			label := widget.NewLabel("Label")
			return container.NewBorder(nil, nil, label, icon, value)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.attributes) {
				return
			}
			it := a.attributes[id]
			border := co.(*fyne.Container).Objects
			label := border[1].(*widget.Label)
			label.SetText(it.label)
			value := border[0].(*widget.Label)
			icon := border[2]
			icon.Hide()
			var s string
			var i widget.Importance
			switch x := it.value.(type) {
			case *app.EveEntity:
				s = x.Name
				if x.Category == app.EveEntityCharacter || x.Category == app.EveEntityCorporation {
					icon.Show()
				}
			case *url.URL:
				s = x.String()
				i = widget.HighImportance
			case float32:
				s = fmt.Sprintf("%.1f %%", x*100)
			case int:
				s = humanize.Comma(int64(x))
			case bool:
				if x {
					s = "yes"
					i = widget.SuccessImportance
				} else {
					s = "no"
					i = widget.DangerImportance
				}
			default:
				s = fmt.Sprint(x)
			}
			value.Text = s
			value.Importance = i
			value.Refresh()
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.attributes) {
			return
		}
		it := a.attributes[id]
		switch x := it.value.(type) {
		case *app.EveEntity:
			switch x.Category {
			case app.EveEntityCharacter:
				a.u.ShowCharacterInfoWindow(x.ID)
			case app.EveEntityCorporation:
				a.u.ShowCharacterInfoWindow(x.ID)
			}
		case *url.URL:
			err := a.u.FyneApp.OpenURL(x)
			if err != nil {
				a.u.Snackbar.Show(fmt.Sprintf("ERROR: Failed to open URL: %s", ihumanize.Error(err)))
			}
		}
	}
	return l
}

func (a *CorporationInfoArea) makeHistory() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.historyItems)
		},
		func() fyne.CanvasObject {
			l := widget.NewRichText()
			l.Truncation = fyne.TextTruncateEllipsis
			return l
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.historyItems) {
				return
			}
			it := a.historyItems[id]
			const dateFormat = "2006.01.02 15:04"
			var endDateStr string
			if !it.EndDate.IsZero() {
				endDateStr = it.EndDate.Format(dateFormat)
			} else {
				endDateStr = "this day"
			}
			var closed string
			if it.IsDeleted {
				closed = " (closed)"
			}
			text := fmt.Sprintf(
				"%s%s   **%s** to **%s** (%s days)",
				it.Organization.Name,
				closed,
				it.StartDate.Format(dateFormat),
				endDateStr,
				humanize.Comma(int64(it.Days)),
			)
			co.(*widget.RichText).ParseMarkdown(text)
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *CorporationInfoArea) load(corporationID int32) error {
	ctx := context.Background()
	go func() {
		r, err := a.u.EveImageService.CorporationLogo(corporationID, DefaultIconPixelSize)
		if err != nil {
			slog.Error("corporation info: Failed to load logo", "corporationID", corporationID, "error", err)
			return
		}
		a.corporationLogo.Resource = r
		a.corporationLogo.Refresh()
	}()
	c, err := a.u.EveUniverseService.GetEveCorporationESI(ctx, corporationID)
	if err != nil {
		return err
	}
	a.corporation.SetText(c.Name)
	if c.Alliance != nil {
		a.alliance.SetText("Member of " + c.Alliance.Name)
		go func() {
			r, err := a.u.EveImageService.AllianceLogo(c.Alliance.ID, DefaultIconPixelSize)
			if err != nil {
				slog.Error("corporation info: Failed to load alliance logo", "allianceID", c.Alliance.ID, "error", err)
				return
			}
			a.allianceLogo.Resource = r
			a.allianceLogo.Refresh()
		}()
	} else {
		a.alliance.Hide()
		a.allianceLogo.Hide()
	}
	desc := c.DescriptionPlain()
	if desc != "" {
		description := widget.NewLabel(desc)
		description.Wrapping = fyne.TextWrapWord
		a.tabs.Append(container.NewTabItem("Description", container.NewVScroll(description)))
	}
	if c.HomeStation != nil {
		a.hq.SetText("Headquarters: " + c.HomeStation.Name)
		a.hq.OnTapped = func() {
			a.u.ShowLocationInfoWindow(int64(c.HomeStation.ID))
		}
	} else {
		a.hq.Hide()
	}
	a.attributes = make([]corporationAttribute, 0)
	if c.Ceo != nil {
		a.attributes = append(a.attributes, corporationAttribute{"CEO", c.Ceo})
	}
	if c.Creator != nil {
		a.attributes = append(a.attributes, corporationAttribute{"Founder", c.Creator})
	}
	if c.Alliance != nil {
		a.attributes = append(a.attributes, corporationAttribute{"Alliance", c.Alliance})
	}
	a.attributes = append(a.attributes, corporationAttribute{"Ticker Name", c.Ticker})
	if c.Shares != 0 {
		a.attributes = append(a.attributes, corporationAttribute{"Shares", c.Shares})
	}
	a.attributes = append(a.attributes, corporationAttribute{"Member Count", c.MemberCount})
	if c.TaxRate != 0 {
		a.attributes = append(a.attributes, corporationAttribute{"ISK Tax Rate", c.TaxRate})
	}
	a.attributes = append(a.attributes, corporationAttribute{"War Eligability", c.WarEligible})
	if c.URL != "" {
		u, err := url.ParseRequestURI(c.URL)
		if err == nil {
			a.attributes = append(a.attributes, corporationAttribute{"URL", u})
		}
	}
	a.tabs.Append(container.NewTabItem("Attributes", a.attributeList))
	a.tabs.Append(container.NewTabItem("Alliance History", a.historyList))
	a.tabs.Refresh()

	go func() {
		history, err := a.u.EveUniverseService.GetCorporationAllianceHistory(ctx, corporationID)
		if err != nil {
			slog.Error("corporation info: Failed to load alliance history", "corporationID", corporationID, "error", err)
			return
		}
		a.historyItems = slices.Collect(xiter.FilterSlice(history, func(v app.MembershipHistoryItem) bool {
			return v.Organization != nil
		}))
		a.historyList.Refresh()
	}()
	return nil
}
