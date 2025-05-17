package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type characterRow struct {
	alliance      *app.EveEntity
	assetValue    optional.Optional[float64]
	birthday      time.Time
	corporation   *app.EveEntity
	home          *app.EveLocation
	id            int32
	lastLoginAt   optional.Optional[time.Time]
	name          string
	security      float64
	unreadCount   optional.Optional[int]
	walletBalance optional.Optional[float64]
}

func (r characterRow) AllianceName() string {
	if r.alliance == nil {
		return ""
	}
	return r.alliance.Name
}

func (r characterRow) CorporationName() string {
	if r.corporation == nil {
		return ""
	}
	return r.corporation.Name
}

type OverviewCharacters struct {
	widget.BaseWidget

	body              fyne.CanvasObject
	columnSorter      *columnSorter
	rows              []characterRow
	rowsFiltered      []characterRow
	selectAlliance    *selectFilter
	selectCorporation *selectFilter
	sortButton        *sortButton
	top               *widget.Label
	u                 *BaseUI
}

func newOverviewCharacters(u *BaseUI) *OverviewCharacters {
	headers := []headerDef{
		{Text: "Character", Width: columnWidthCharacter},
		{Text: "Corporation", Width: 250},
		{Text: "Alliance", Width: 250},
		{Text: "Sec.", Width: 50},
		{Text: "Unread", Width: 100},
		{Text: "Wallet", Width: 100},
		{Text: "Assets", Width: 100},
		{Text: "Last Login", Width: 100},
		{Text: "Home", Width: columnWidthLocation},
		{Text: "Age", Width: 100},
	}
	a := &OverviewCharacters{
		columnSorter: newColumnSorter(headers),
		rows:         make([]characterRow, 0),
		top:          makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, c characterRow) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(c.name)
		case 1:
			return iwidget.NewRichTextSegmentFromText(c.corporation.Name)
		case 2:
			var s string
			if c.alliance != nil {
				s = c.alliance.Name
			}
			return iwidget.NewRichTextSegmentFromText(s)
		case 3:
			var color fyne.ThemeColorName
			text := fmt.Sprintf("%.1f", c.security)
			if c.security > 0 {
				color = theme.ColorNameSuccess
			} else if c.security < 0 {
				color = theme.ColorNameError
			} else {
				color = theme.ColorNameForeground
			}
			return iwidget.NewRichTextSegmentFromText(text, widget.RichTextStyle{
				ColorName: color,
			})
		case 4:
			return iwidget.NewRichTextSegmentFromText(ihumanize.Optional(c.unreadCount, "?"))
		case 5:
			return iwidget.NewRichTextSegmentFromText(ihumanize.OptionalFloat(c.walletBalance, 1, "?"))
		case 6:
			return iwidget.NewRichTextSegmentFromText(ihumanize.OptionalFloat(c.assetValue, 1, "?"))
		case 7:
			return iwidget.NewRichTextSegmentFromText(ihumanize.Optional(c.lastLoginAt, "?"))
		case 8:
			if c.home != nil {
				return c.home.DisplayRichText()
			}
		case 9:
			return iwidget.NewRichTextSegmentFromText(humanize.RelTime(c.birthday, time.Now(), "", ""))
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	if a.u.isDesktop {
		a.body = makeDataTable(headers, &a.rowsFiltered, makeCell, a.columnSorter, a.filterRows, func(_ int, oc characterRow) {
			u.ShowInfoWindow(app.EveEntityCharacter, oc.id)
		})
	} else {
		a.body = makeDataList(headers, &a.rowsFiltered, makeCell, func(oc characterRow) {
			u.ShowInfoWindow(app.EveEntityCharacter, oc.id)
		})
	}

	a.selectAlliance = newSelectFilter("Any alliance", func() {
		a.filterRows(-1)
	})
	a.selectCorporation = newSelectFilter("Any corporation", func() {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window, 5, 6, 7, 8, 9)
	return a
}

func (a *OverviewCharacters) CreateRenderer() fyne.WidgetRenderer {
	filters := container.NewHBox(a.selectAlliance, a.selectCorporation)
	if !a.u.isDesktop {
		filters.Add(a.sortButton)
	}
	c := container.NewBorder(
		container.NewVBox(a.top, container.NewHScroll(filters)),
		nil,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *OverviewCharacters) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	a.selectAlliance.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(o characterRow) bool {
			return o.AllianceName() == selected
		})
	})
	a.selectCorporation.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(o characterRow) bool {
			return o.CorporationName() == selected
		})
	})
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b characterRow) int {
			var x int
			switch sortCol {
			case 0:
				x = strings.Compare(a.name, b.name)
			case 1:
				x = strings.Compare(a.CorporationName(), b.CorporationName())
			case 2:
				x = strings.Compare(a.AllianceName(), b.AllianceName())
			case 3:
				x = cmp.Compare(a.security, b.security)
			case 4:
				x = cmp.Compare(a.unreadCount.ValueOrZero(), b.unreadCount.ValueOrZero())
			case 5:
				x = cmp.Compare(a.walletBalance.ValueOrZero(), b.walletBalance.ValueOrZero())
			case 6:
				x = cmp.Compare(a.assetValue.ValueOrZero(), b.assetValue.ValueOrZero())
			case 7:
				x = a.lastLoginAt.ValueOrZero().Compare(b.lastLoginAt.ValueOrZero())
			case 8:
				x = strings.Compare(a.home.DisplayName(), b.home.DisplayName())
			case 9:
				x = a.birthday.Compare(b.birthday)
			}
			if dir == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	a.selectAlliance.setOptions(xiter.MapSlice(rows, func(r characterRow) string {
		return r.AllianceName()
	}))
	a.selectCorporation.setOptions(xiter.MapSlice(rows, func(r characterRow) string {
		return r.CorporationName()
	}))
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *OverviewCharacters) update() {
	var rows []characterRow
	t, i, err := func() (string, widget.Importance, error) {
		cc, totals, err := a.fetchRows(a.u.services())
		if err != nil {
			return "", 0, err
		}
		if len(cc) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		rows = cc
		walletText := ihumanize.OptionalFloat(totals.wallet, 1, "?")
		assetsText := ihumanize.OptionalFloat(totals.assets, 1, "?")
		unreadText := ihumanize.Optional(totals.unread, "?")
		s := fmt.Sprintf(
			"%d characters • %s ISK wallet • %s ISK assets • %s unread",
			len(cc),
			walletText,
			assetsText,
			unreadText,
		)
		return s, widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh overview UI", "err", err)
		t = "ERROR: " + a.u.humanizeError(err)
		i = widget.DangerImportance
	}
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRows(-1)
	})
}

type overviewTotals struct {
	unread optional.Optional[int]
	wallet optional.Optional[float64]
	assets optional.Optional[float64]
}

func (*OverviewCharacters) fetchRows(s services) ([]characterRow, overviewTotals, error) {
	var totals overviewTotals
	ctx := context.Background()
	characters, err := s.cs.ListCharacters(ctx)
	if err != nil {
		return nil, totals, err
	}
	cc := xslices.Map(characters, func(m *app.Character) characterRow {
		return characterRow{
			alliance:      m.EveCharacter.Alliance,
			birthday:      m.EveCharacter.Birthday,
			corporation:   m.EveCharacter.Corporation,
			lastLoginAt:   m.LastLoginAt,
			id:            m.ID,
			name:          m.EveCharacter.Name,
			security:      m.EveCharacter.SecurityStatus,
			walletBalance: m.WalletBalance,
			home:          m.Home,
		}
	})
	for i, c := range cc {
		total, unread, err := s.cs.GetMailCounts(ctx, c.id)
		if err != nil {
			return nil, totals, err
		}
		if total > 0 {
			cc[i].unreadCount = optional.From(unread)
		}
	}
	for i, c := range cc {
		v, err := s.cs.AssetTotalValue(ctx, c.id)
		if err != nil {
			return nil, totals, err
		}
		cc[i].assetValue = v
	}
	for _, c := range cc {
		if !c.unreadCount.IsEmpty() {
			totals.unread.Set(totals.unread.ValueOrZero() + c.unreadCount.ValueOrZero())
		}
		if !c.walletBalance.IsEmpty() {
			totals.wallet.Set(totals.wallet.ValueOrZero() + c.walletBalance.ValueOrZero())
		}
		if !c.assetValue.IsEmpty() {
			totals.assets.Set(totals.assets.ValueOrZero() + c.assetValue.ValueOrZero())
		}
	}
	return cc, totals, nil
}
