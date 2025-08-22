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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type characterOverviewRow struct {
	alliance        *app.EveEntity
	assetValue      optional.Optional[float64]
	birthday        time.Time
	characterID     int32
	characterName   string
	corporation     *app.EveEntity
	faction         *app.EveEntity
	home            *app.EveLocation
	lastLoginAt     optional.Optional[time.Time]
	security        float64
	securityDisplay []widget.RichTextSegment
	tags            set.Set[string]
	unreadCount     optional.Optional[int]
	walletBalance   optional.Optional[float64]
}

func (r characterOverviewRow) AllianceName() string {
	if r.alliance == nil {
		return ""
	}
	return r.alliance.Name
}

func (r characterOverviewRow) CorporationName() string {
	if r.corporation == nil {
		return ""
	}
	return r.corporation.Name
}

type characterOverview struct {
	widget.BaseWidget

	body              fyne.CanvasObject
	columnSorter      *columnSorter
	rows              []characterOverviewRow
	rowsFiltered      []characterOverviewRow
	selectAlliance    *kxwidget.FilterChipSelect
	selectCorporation *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortButton        *sortButton
	top               *widget.Label
	u                 *baseUI
}

func newCharacterOverview(u *baseUI) *characterOverview {
	headers := []headerDef{
		{label: "Character", width: columnWidthEntity},
		{label: "Corporation", width: 250},
		{label: "Alliance", width: 250},
		{label: "Sec.", width: 50},
		{label: "Unread", width: 100},
		{label: "Wallet", width: 100},
		{label: "Assets", width: 100},
		{label: "Last Login", width: 100},
		{label: "Home", width: columnWidthLocation},
		{label: "Age", width: 100},
	}
	a := &characterOverview{
		columnSorter: newColumnSorter(headers),
		rows:         make([]characterOverviewRow, 0),
		top:          makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, r characterOverviewRow) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.RichTextSegmentsFromText(r.characterName)
		case 1:
			return iwidget.RichTextSegmentsFromText(r.corporation.Name)
		case 2:
			var s string
			if r.alliance != nil {
				s = r.alliance.Name
			}
			return iwidget.RichTextSegmentsFromText(s)
		case 3:
			return r.securityDisplay
		case 4:
			return iwidget.RichTextSegmentsFromText(ihumanize.Optional(r.unreadCount, "?"))
		case 5:
			return iwidget.RichTextSegmentsFromText(ihumanize.OptionalWithDecimals(r.walletBalance, 1, "?"))
		case 6:
			return iwidget.RichTextSegmentsFromText(ihumanize.OptionalWithDecimals(r.assetValue, 1, "?"))
		case 7:
			return iwidget.RichTextSegmentsFromText(ihumanize.Optional(r.lastLoginAt, "?"))
		case 8:
			if r.home != nil {
				return r.home.DisplayRichText()
			}
		case 9:
			return iwidget.RichTextSegmentsFromText(humanize.RelTime(r.birthday, time.Now(), "", ""))
		}
		return iwidget.RichTextSegmentsFromText("?")
	}
	if a.u.isDesktop {
		a.body = makeDataTable(headers, &a.rowsFiltered, makeCell, a.columnSorter, a.filterRows, func(_ int, r characterOverviewRow) {
			showCharacterOverviewDetailWindow(a.u, r)
		})
	} else {
		a.body = makeDataList(headers, &a.rowsFiltered, makeCell, func(r characterOverviewRow) {
			showCharacterOverviewDetailWindow(a.u, r)
		})
	}

	a.selectAlliance = kxwidget.NewFilterChipSelect("Alliance", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectCorporation = kxwidget.NewFilterChipSelect("Corporation", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window, 5, 6, 7, 8, 9)

	a.u.generalSectionChanged.AddListener(
		func(_ context.Context, arg generalSectionUpdated) {
			characterIDs := set.Collect(xiter.MapSlice(a.rows, func(r characterOverviewRow) int32 {
				return r.characterID
			}))
			switch arg.section {
			case app.SectionEveCharacters:
				if arg.changed.ContainsAny(characterIDs.All()) {
					a.update()
				}
			case app.SectionEveMarketPrices:
				a.update()
			}
		},
	)
	a.u.characterSectionChanged.AddListener(
		func(_ context.Context, arg characterSectionUpdated) {
			switch arg.section {
			case
				app.SectionCharacterMails,
				app.SectionCharacterWalletBalance,
				app.SectionCharacterAssets,
				app.SectionCharacterOnline:
				a.update()
			}
		},
	)
	return a
}

func (a *characterOverview) CreateRenderer() fyne.WidgetRenderer {
	filters := container.NewHBox(a.selectAlliance, a.selectCorporation, a.selectTag)
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

func (a *characterOverview) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectAlliance.Selected; x != "" {
		rows = xslices.Filter(rows, func(r characterOverviewRow) bool {
			return r.AllianceName() == x
		})
	}
	if x := a.selectCorporation.Selected; x != "" {
		rows = xslices.Filter(rows, func(r characterOverviewRow) bool {
			return r.CorporationName() == x
		})
	}
	if x := a.selectTag.Selected; x != "" {
		rows = xslices.Filter(rows, func(o characterOverviewRow) bool {
			return o.tags.Contains(x)
		})
	}
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b characterOverviewRow) int {
			var x int
			switch sortCol {
			case 0:
				x = strings.Compare(a.characterName, b.characterName)
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
	a.selectAlliance.SetOptions(xslices.Map(rows, func(r characterOverviewRow) string {
		return r.AllianceName()
	}))
	a.selectCorporation.SetOptions(xslices.Map(rows, func(r characterOverviewRow) string {
		return r.CorporationName()
	}))

	a.selectTag.SetOptions(slices.Sorted(set.Union(xslices.Map(rows, func(r characterOverviewRow) set.Set[string] {
		return r.tags
	})...).All()))
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *characterOverview) update() {
	var rows []characterOverviewRow
	t, i, err := func() (string, widget.Importance, error) {
		cc, totals, err := a.fetchRows(a.u.services())
		if err != nil {
			return "", 0, err
		}
		if len(cc) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		rows = cc
		walletText := ihumanize.OptionalWithDecimals(totals.wallet, 1, "?")
		assetsText := ihumanize.OptionalWithDecimals(totals.assets, 1, "?")
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

func (*characterOverview) fetchRows(s services) ([]characterOverviewRow, overviewTotals, error) {
	var totals overviewTotals
	ctx := context.Background()
	characters, err := s.cs.ListCharacters(ctx)
	if err != nil {
		return nil, totals, err
	}
	cc := xslices.Map(characters, func(m *app.Character) characterOverviewRow {
		r := characterOverviewRow{
			alliance:      m.EveCharacter.Alliance,
			birthday:      m.EveCharacter.Birthday,
			characterID:   m.ID,
			characterName: m.EveCharacter.Name,
			corporation:   m.EveCharacter.Corporation,
			faction:       m.EveCharacter.Faction,
			home:          m.Home,
			lastLoginAt:   m.LastLoginAt,
			security:      m.EveCharacter.SecurityStatus,
			walletBalance: m.WalletBalance,
		}
		var color fyne.ThemeColorName
		text := fmt.Sprintf("%.1f", r.security)
		if r.security > 0 {
			color = theme.ColorNameSuccess
		} else if r.security < 0 {
			color = theme.ColorNameError
		} else {
			color = theme.ColorNameForeground
		}
		r.securityDisplay = iwidget.RichTextSegmentsFromText(text, widget.RichTextStyle{
			ColorName: color,
		})
		return r
	})
	for i, c := range cc {
		total, unread, err := s.cs.GetMailCounts(ctx, c.characterID)
		if err != nil {
			return nil, totals, err
		}
		if total > 0 {
			cc[i].unreadCount = optional.New(unread)
		}
	}
	for i, c := range cc {
		v, err := s.cs.AssetTotalValue(ctx, c.characterID)
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
	for i, c := range cc {
		tags, err := s.cs.ListTagsForCharacter(ctx, c.characterID)
		if err != nil {
			return nil, totals, err
		}
		cc[i].tags = tags
	}
	return cc, totals, nil
}

// showCharacterOverviewDetailWindow shows details for a character overview in a new window.
func showCharacterOverviewDetailWindow(u *baseUI, r characterOverviewRow) {
	w, ok := u.getOrCreateWindow(fmt.Sprintf("characteroverview-%d", r.characterID), "Character: Overview", r.characterName)
	if !ok {
		w.Show()
		return
	}
	var home fyne.CanvasObject
	if r.home != nil {
		home = makeLocationLabel(r.home.ToShort(), u.ShowLocationInfoWindow)
	} else {
		home = widget.NewLabel("?")
	}

	fi := []*widget.FormItem{
		widget.NewFormItem("Owner", makeCharacterActionLabel(
			r.characterID,
			r.characterName,
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Corporation", makeEveEntityActionLabel(r.corporation, u.ShowEveEntityInfoWindow)),
		widget.NewFormItem("Alliance", makeEveEntityActionLabel(r.alliance, u.ShowEveEntityInfoWindow)),
		widget.NewFormItem("Faction", makeEveEntityActionLabel(r.faction, u.ShowEveEntityInfoWindow)),
		widget.NewFormItem("Security Status", widget.NewRichText(r.securityDisplay...)),
		widget.NewFormItem("Unread Mails", widget.NewLabel(ihumanize.Optional(r.unreadCount, "?"))),
		widget.NewFormItem("Wallet", widget.NewLabel(r.walletBalance.StringFunc("?", func(v float64) string {
			return formatISKAmount(v)
		}))),
		widget.NewFormItem("Assets", widget.NewLabel(r.assetValue.StringFunc("?", func(v float64) string {
			return formatISKAmount(v)
		}))),
		widget.NewFormItem("Last Login", widget.NewLabel(ihumanize.Optional(r.lastLoginAt, "?"))),
		widget.NewFormItem("Home", home),
		widget.NewFormItem("Age", widget.NewLabel(humanize.RelTime(r.birthday, time.Now(), "", ""))),
	}

	f := widget.NewForm(fi...)
	f.Orientation = widget.Adaptive
	subTitle := fmt.Sprintf("Overview of %s", r.characterName)
	setDetailWindow(detailWindowParams{
		content: f,
		imageLoader: func() (fyne.Resource, error) {
			return u.eis.CharacterPortrait(r.characterID, 256)
		},
		imageAction: func() {
			u.ShowInfoWindow(app.EveEntityCharacter, r.characterID)
		},
		minSize: fyne.NewSize(500, 450),
		title:   subTitle,
		window:  w,
	})
	w.Show()
}
