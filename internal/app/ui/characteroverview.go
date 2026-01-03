package ui

import (
	"cmp"
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

type characterOverviewRow struct {
	alliance        *app.EveEntity
	characterID     int32
	characterName   string
	corporation     *app.EveEntity
	faction         *app.EveEntity
	location        *app.EveLocation
	regionName      string
	searchTarget    string
	ship            *app.EveType
	skillpoints     optional.Optional[int]
	solarSystemName string
	tags            set.Set[string]
	unreadCount     optional.Optional[int]
	walletBalance   optional.Optional[float64]
}

func (r characterOverviewRow) allianceName() string {
	if r.alliance == nil {
		return ""
	}
	return r.alliance.Name
}

func (r characterOverviewRow) corporationName() string {
	if r.corporation == nil {
		return "?"
	}
	return r.corporation.Name
}

func (r characterOverviewRow) shipName() string {
	if r.ship == nil {
		return "?"
	}
	return r.ship.Name
}

type characterOverview struct {
	widget.BaseWidget

	body              fyne.CanvasObject
	columnSorter      *iwidget.ColumnSorter
	rows              []characterOverviewRow
	rowsFiltered      []characterOverviewRow
	search            *widget.Entry
	selectAlliance    *kxwidget.FilterChipSelect
	selectCorporation *kxwidget.FilterChipSelect
	selectRegion      *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortButton        *iwidget.SortButton
	info              *widget.Label
	top               *widget.Label
	u                 *baseUI
}

const (
	overviewColAlliance = iota
	overviewColCharacter
	overviewColCorporation
	overviewColMail
	overviewColRegion
	overviewColSolarSystem
	overviewColSkillpoints
	overviewColWallet
)

func newCharacterOverview(u *baseUI) *characterOverview {
	headers := iwidget.NewDataTableDef([]iwidget.ColumnDef{
		{
			Col:   overviewColAlliance,
			Label: "Alliance",
		},
		{
			Col:   overviewColCharacter,
			Label: "Character",
		},
		{
			Col:   overviewColCorporation,
			Label: "Corporation",
		},
		{
			Col:   overviewColMail,
			Label: "Unread",
		},
		{
			Col:   overviewColRegion,
			Label: "Region",
		},
		{
			Col:   overviewColSkillpoints,
			Label: "Skillpoints",
		},
		{
			Col:   overviewColSolarSystem,
			Label: "System",
		},
		{
			Col:   overviewColWallet,
			Label: "Wallet",
		},
	})

	info := widget.NewLabel("Loading...")
	info.Importance = widget.LowImportance

	a := &characterOverview{
		columnSorter: headers.NewColumnSorter(overviewColCharacter, iwidget.SortAsc),
		rows:         make([]characterOverviewRow, 0),
		rowsFiltered: make([]characterOverviewRow, 0),
		search:       widget.NewEntry(),
		top:          makeTopLabel(),
		u:            u,
		info:         info,
	}
	a.ExtendBaseWidget(a)

	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRows(-1)
	})
	a.search.OnChanged = func(s string) {
		a.filterRows(-1)
	}
	a.search.PlaceHolder = "Search character names"
	if a.u.isDesktop {
		a.body = a.makeGrid()
	} else {
		a.body = a.makeList()
	}

	a.selectAlliance = kxwidget.NewFilterChipSelect("Alliance", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectCorporation = kxwidget.NewFilterChipSelect("Corporation", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectRegion = kxwidget.NewFilterChipSelect("Region", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectSolarSystem = kxwidget.NewFilterChipSelect("System", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRows(-1)
	}, a.u.window)

	a.u.generalSectionChanged.AddListener(func(_ context.Context, arg generalSectionUpdated) {
		characterIDs := set.Collect(xiter.MapSlice(a.rows, func(r characterOverviewRow) int32 {
			return r.characterID
		}))
		switch arg.section {
		case app.SectionEveCharacters:
			if arg.changed.ContainsAny(characterIDs.All()) {
				a.update()
			}
		}
	})
	a.u.characterAdded.AddListener(func(_ context.Context, _ *app.Character) {
		a.update()
	})
	a.u.characterRemoved.AddListener(func(_ context.Context, _ *app.EntityShort[int32]) {
		a.update()
	})
	a.u.tagsChanged.AddListener(func(ctx context.Context, s struct{}) {
		a.update()
	})
	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		switch arg.section {
		case
			app.SectionCharacterLocation,
			app.SectionCharacterMailHeaders,
			app.SectionCharacterSkills,
			app.SectionCharacterWalletBalance:
			a.update()
		}
	})
	return a
}

func (a *characterOverview) CreateRenderer() fyne.WidgetRenderer {
	filters := container.NewHBox(
		a.selectAlliance,
		a.selectCorporation,
		a.selectRegion,
		a.selectSolarSystem,
		a.selectTag,
		a.sortButton,
	)
	c := container.NewBorder(
		container.NewVBox(
			a.top,
			a.search,
			container.NewHScroll(filters),
		),
		nil,
		nil,
		nil,
		container.NewStack(a.info, a.body),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *characterOverview) makeGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newCharacterCardLarge(a.u.eis, a.u.ShowInfoWindow)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			co.(*characterCardLarge).set(r)
		},
	)
	g.OnSelected = func(id widget.GridWrapItemID) {
		defer g.UnselectAll()
		if id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		if a.u.onShowCharacter != nil {
			a.u.onShowCharacter()
		}
		if a.u.currentCharacterID() == r.characterID {
			return
		}
		go func() {
			err := a.u.loadCharacter(r.characterID)
			if err != nil {
				slog.Error("Failed to load character", "characterID", r.characterID, "error", err)
				a.u.ShowSnackbar(fmt.Sprintf("Failed to load character: %s", a.u.humanizeError(err)))
				return
			}
		}()
	}
	return g
}

func (a *characterOverview) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newCharacterCardSmall(a.u.eis)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			co.(*characterCardSmall).set(r)
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.GridWrapItemID) {
		defer l.UnselectAll()
		if id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		if a.u.onShowCharacter != nil {
			a.u.onShowCharacter()
		}
		if a.u.currentCharacterID() == r.characterID {
			return
		}
		go func() {
			err := a.u.loadCharacter(r.characterID)
			if err != nil {
				slog.Error("Failed to load character", "characterID", r.characterID, "error", err)
				a.u.ShowSnackbar(fmt.Sprintf("Failed to load character: %s", a.u.humanizeError(err)))
				return
			}
		}()
	}
	return l
}

func (a *characterOverview) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectAlliance.Selected; x != "" {
		rows = slices.DeleteFunc(rows, func(r characterOverviewRow) bool {
			return r.allianceName() != x
		})
	}
	if x := a.selectCorporation.Selected; x != "" {
		rows = slices.DeleteFunc(rows, func(r characterOverviewRow) bool {
			return r.corporationName() != x
		})
	}
	if x := a.selectRegion.Selected; x != "" {
		rows = slices.DeleteFunc(rows, func(r characterOverviewRow) bool {
			return r.regionName != x
		})
	}
	if x := a.selectSolarSystem.Selected; x != "" {
		rows = slices.DeleteFunc(rows, func(r characterOverviewRow) bool {
			return r.solarSystemName != x
		})
	}
	if x := a.selectTag.Selected; x != "" {
		rows = slices.DeleteFunc(rows, func(o characterOverviewRow) bool {
			return !o.tags.Contains(x)
		})
	}
	// search filter
	if search := strings.ToLower(a.search.Text); search != "" {
		rows = slices.DeleteFunc(rows, func(r characterOverviewRow) bool {
			return !strings.Contains(r.searchTarget, search)
		})
	}
	// sort
	a.columnSorter.Sort(sortCol, func(sortCol int, dir iwidget.SortDir) {
		slices.SortFunc(rows, func(a, b characterOverviewRow) int {
			var x int
			switch sortCol {
			case overviewColAlliance:
				x = xstrings.CompareIgnoreCase(a.allianceName(), b.allianceName())
			case overviewColCharacter:
				x = xstrings.CompareIgnoreCase(a.characterName, b.characterName)
			case overviewColCorporation:
				x = xstrings.CompareIgnoreCase(a.corporationName(), b.corporationName())
			case overviewColMail:
				x = cmp.Compare(a.unreadCount.ValueOrZero(), b.unreadCount.ValueOrZero())
			case overviewColRegion:
				x = strings.Compare(a.regionName, b.regionName)
			case overviewColSkillpoints:
				x = cmp.Compare(a.skillpoints.ValueOrZero(), b.skillpoints.ValueOrZero())
			case overviewColSolarSystem:
				x = strings.Compare(a.solarSystemName, b.solarSystemName)
			case overviewColWallet:
				x = cmp.Compare(a.walletBalance.ValueOrZero(), b.walletBalance.ValueOrZero())
			}
			if dir == iwidget.SortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	a.selectAlliance.SetOptions(xslices.Map(rows, func(r characterOverviewRow) string {
		return r.allianceName()
	}))
	a.selectCorporation.SetOptions(xslices.Map(rows, func(r characterOverviewRow) string {
		return r.corporationName()
	}))
	a.selectRegion.SetOptions(xslices.Map(rows, func(r characterOverviewRow) string {
		return r.regionName
	}))
	a.selectSolarSystem.SetOptions(xslices.Map(rows, func(r characterOverviewRow) string {
		return r.solarSystemName
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
		unreadText := ihumanize.Optional(totals.unread, "?")
		skillpointsText := ihumanize.Optional(totals.skillpoints, "?")
		s := fmt.Sprintf(
			"%d characters • %s skillpoints • %s ISK wallet • %s unread",
			len(cc),
			skillpointsText,
			walletText,
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
		a.info.Hide()
		a.filterRows(-1)
	})
}

type overviewTotals struct {
	skillpoints optional.Optional[int]
	unread      optional.Optional[int]
	wallet      optional.Optional[float64]
}

func (*characterOverview) fetchRows(s services) ([]characterOverviewRow, overviewTotals, error) {
	var totals overviewTotals
	ctx := context.Background()
	characters, err := s.cs.ListCharacters(ctx)
	if err != nil {
		return nil, totals, err
	}
	cc := xslices.Map(characters, func(c *app.Character) characterOverviewRow {
		r := characterOverviewRow{
			alliance:      c.EveCharacter.Alliance,
			characterID:   c.ID,
			characterName: c.EveCharacter.Name,
			corporation:   c.EveCharacter.Corporation,
			faction:       c.EveCharacter.Faction,
			location:      c.Location,
			searchTarget:  strings.ToLower(c.EveCharacter.Name),
			ship:          c.Ship,
			skillpoints:   c.TotalSP,
			walletBalance: c.WalletBalance,
		}
		if c.Location != nil && c.Location.SolarSystem != nil {
			r.regionName = c.Location.SolarSystem.Constellation.Region.Name
			r.solarSystemName = c.Location.SolarSystem.Name
		}
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
	for _, c := range cc {
		if !c.unreadCount.IsEmpty() {
			totals.unread.Set(totals.unread.ValueOrZero() + c.unreadCount.ValueOrZero())
		}
		if !c.walletBalance.IsEmpty() {
			totals.wallet.Set(totals.wallet.ValueOrZero() + c.walletBalance.ValueOrZero())
		}
		if !c.skillpoints.IsEmpty() {
			totals.skillpoints.Set(totals.skillpoints.ValueOrZero() + c.skillpoints.ValueOrZero())
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

type characterCardEIS interface {
	AllianceLogo(id int32, size int) (fyne.Resource, error)
	CharacterPortrait(id int32, size int) (fyne.Resource, error)
	CorporationLogo(id int32, size int) (fyne.Resource, error)
}

// characterCardLarge is a widget that shows a card for a character.
// This version is designed for desktops.
type characterCardLarge struct {
	widget.BaseWidget

	allianceLogo    *iwidget.TappableImage
	border          *canvas.Rectangle
	characterName   *widget.Label
	corporationLogo *iwidget.TappableImage
	eis             characterCardEIS
	location        *widget.Label
	mails           *widget.Label
	portrait        *canvas.Image
	ship            *widget.Label
	showInfoWindow  func(c app.EveEntityCategory, id int32)
	skillpoints     *widget.Label
	solarSystem     *iwidget.RichText
	wallet          *widget.Label
}

func newCharacterCardLarge(eis characterCardEIS, showInfoWindow func(c app.EveEntityCategory, id int32)) *characterCardLarge {
	const numberTemplate = "9.999.999.999"
	makeLabel := func(s string) *widget.Label {
		l := widget.NewLabel(s)
		l.Alignment = fyne.TextAlignTrailing
		l.Truncation = fyne.TextTruncateEllipsis
		return l
	}
	portrait := iwidget.NewImageFromResource(
		icons.Characterplaceholder64Jpeg,
		fyne.NewSquareSize(200),
	)
	w := &characterCardLarge{
		allianceLogo:    iwidget.NewTappableImage(icons.Corporationplaceholder64Png, nil),
		border:          canvas.NewRectangle(color.Transparent),
		characterName:   widget.NewLabel("Veronica Blomquist"),
		corporationLogo: iwidget.NewTappableImage(icons.Corporationplaceholder64Png, nil),
		eis:             eis,
		location:        widget.NewLabel("Veronica Blomquist"),
		mails:           makeLabel(numberTemplate),
		portrait:        portrait,
		ship:            makeLabel("Merlin"),
		showInfoWindow:  showInfoWindow,
		skillpoints:     makeLabel(numberTemplate),
		solarSystem:     iwidget.NewRichText(),
		wallet:          makeLabel(numberTemplate + " ISK"),
	}
	w.ExtendBaseWidget(w)

	w.allianceLogo.SetFillMode(canvas.ImageFillContain)
	w.allianceLogo.SetMinSize(fyne.NewSquareSize(40))
	w.allianceLogo.SetCornerRadius(theme.InputRadiusSize())

	w.corporationLogo.SetFillMode(canvas.ImageFillContain)
	w.corporationLogo.SetMinSize(fyne.NewSquareSize(40))
	w.corporationLogo.SetCornerRadius(theme.InputRadiusSize())

	w.characterName.SizeName = theme.SizeNameSubHeadingText
	w.characterName.Truncation = fyne.TextTruncateEllipsis

	w.border.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	w.border.StrokeWidth = 1
	w.border.CornerRadius = theme.Size(theme.SizeNameInputRadius)

	w.location.Alignment = fyne.TextAlignCenter
	w.location.Truncation = fyne.TextTruncateEllipsis

	return w
}

func (w *characterCardLarge) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	logoBorder := &layout.CustomPaddedLayout{
		TopPadding:    1 * p,
		BottomPadding: 1 * p,
		LeftPadding:   1 * p,
		RightPadding:  1 * p,
	}
	c := container.NewBorder(
		container.New(layout.NewCustomPaddedLayout(0, 0, -p, -p), w.characterName),
		container.New(
			layout.NewCustomPaddedVBoxLayout(-2*p),
			container.NewBorder(
				nil,
				nil,
				widget.NewIcon(theme.NewThemedResource(icons.SchoolSvg)),
				nil,
				w.skillpoints,
			),
			container.NewBorder(
				nil,
				nil,
				widget.NewIcon(theme.NewThemedResource(icons.CashSvg)),
				nil,
				w.wallet,
			),
			container.NewBorder(
				nil,
				nil,
				widget.NewIcon(theme.MailComposeIcon()),
				nil,
				w.mails,
			),
			container.NewBorder(
				nil,
				nil,
				widget.NewIcon(theme.NewThemedResource(icons.ShipWheelSvg)),
				nil,
				w.ship,
			),
			container.NewBorder(
				nil,
				nil,
				widget.NewIcon(theme.NewThemedResource(icons.MapMarkerSvg)),
				nil,
				w.solarSystem,
			),
		),
		nil,
		nil,
		container.NewStack(
			w.portrait,
			container.New(
				&bottomLeftLayout{},
				container.New(logoBorder, w.corporationLogo),
			),
			container.New(
				&bottomRightLayout{},
				container.New(logoBorder, w.allianceLogo),
			),
		),
	)
	r := container.NewStack(
		container.New(layout.NewCustomPaddedLayout(p, p, 2*p, 2*p), c),
		w.border,
	)
	return widget.NewSimpleRenderer(r)
}

func (w *characterCardLarge) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.border.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	w.BaseWidget.Refresh()
}

func (w *characterCardLarge) set(r characterOverviewRow) {
	iwidget.RefreshImageAsync(w.portrait, func() (fyne.Resource, error) {
		return w.eis.CharacterPortrait(r.characterID, 512)
	})
	w.corporationLogo.OnTapped = func() {
		w.showInfoWindow(app.EveEntityCorporation, r.corporation.ID)
	}
	w.corporationLogo.SetToolTip(r.corporationName())
	iwidget.RefreshTappableImageAsync(w.corporationLogo, func() (fyne.Resource, error) {
		return w.eis.CorporationLogo(r.corporation.ID, 64)
	})
	if r.alliance != nil {
		w.allianceLogo.OnTapped = func() {
			w.showInfoWindow(app.EveEntityAlliance, r.alliance.ID)
		}
		w.allianceLogo.SetToolTip(r.allianceName())
		w.allianceLogo.Show()
		iwidget.RefreshTappableImageAsync(w.allianceLogo, func() (fyne.Resource, error) {
			return w.eis.AllianceLogo(r.alliance.ID, 64)
		})
	} else {
		w.allianceLogo.Hide()
	}
	w.characterName.SetText(r.characterName)
	w.mails.SetText(r.unreadCount.StringFunc("?", func(v int) string {
		if v == 0 {
			return "-"
		}
		return humanize.Comma(int64(v))
	}))
	w.skillpoints.SetText(r.skillpoints.StringFunc("?", func(v int) string {
		return humanize.Comma(int64(v))
	}))
	w.wallet.SetText(r.walletBalance.StringFunc("?", func(v float64) string {
		return humanize.Comma(int64(v)) + " ISK"
	}))
	w.ship.SetText(r.shipName())
	var rt []widget.RichTextSegment
	var location string
	if r.location != nil && r.location.SolarSystem != nil {
		rt = r.location.SolarSystem.DisplayRichText()
		location = r.location.DisplayName2()
	} else {
		rt = iwidget.RichTextSegmentsFromText("?")
		location = "?"
	}
	rt = iwidget.AlignRichTextSegments(fyne.TextAlignTrailing, rt)
	w.solarSystem.Set(rt)
	w.location.SetText(location)
}

// characterCardSmall is a widget that shows a card for a character.
// This version is designed for mobiles.
type characterCardSmall struct {
	widget.BaseWidget

	allianceLogo    *canvas.Image
	border          *canvas.Rectangle
	background      *canvas.Rectangle
	characterName   *widget.Label
	corporationLogo *canvas.Image
	eis             characterCardEIS
	mails           *widget.Label
	portrait        *canvas.Image
	ship            *widget.Label
	skillpoints     *widget.Label
	solarSystem     *iwidget.RichText
	wallet          *widget.Label
}

func newCharacterCardSmall(eis characterCardEIS) *characterCardSmall {
	const numberTemplate = "9.999.999.999"
	makeInfoLabel := func(s string) *widget.Label {
		l := widget.NewLabel(s)
		l.Alignment = fyne.TextAlignLeading
		l.Truncation = fyne.TextTruncateEllipsis
		return l
	}
	w := &characterCardSmall{
		background:    canvas.NewRectangle(theme.Color(theme.ColorNameHover)),
		border:        canvas.NewRectangle(color.Transparent),
		characterName: widget.NewLabel("Veronica Blomquist"),
		eis:           eis,
		mails:         makeInfoLabel(numberTemplate),
		ship:          makeInfoLabel("Merlin"),
		skillpoints:   makeInfoLabel(numberTemplate),
		solarSystem:   iwidget.NewRichText(),
		wallet:        makeInfoLabel(numberTemplate + " ISK"),
		allianceLogo: iwidget.NewImageFromResource(
			icons.Corporationplaceholder64Png,
			fyne.NewSquareSize(app.IconUnitSize),
		),
		corporationLogo: iwidget.NewImageFromResource(
			icons.Corporationplaceholder64Png,
			fyne.NewSquareSize(app.IconUnitSize),
		),
		portrait: iwidget.NewImageFromResource(
			icons.Characterplaceholder64Jpeg,
			fyne.NewSquareSize(88),
		),
	}
	w.ExtendBaseWidget(w)

	w.background.CornerRadius = theme.InputRadiusSize()

	w.border.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	w.border.StrokeWidth = 1
	w.border.CornerRadius = theme.Size(theme.SizeNameInputRadius)

	w.characterName.SizeName = theme.SizeNameSubHeadingText
	w.characterName.Truncation = fyne.TextTruncateEllipsis
	return w
}

func (w *characterCardSmall) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()

	c := container.NewBorder(
		nil,
		nil,
		container.New(layout.NewCustomPaddedLayout(0, 0, 0, 2*p),
			container.NewStack(
				w.background,
				container.New(layout.NewCustomPaddedLayout(2*p, 2*p, 3*p, 3*p),
					container.NewVBox(
						w.portrait,
						container.New(layout.NewCustomPaddedLayout(p/2, 0, 0, 0),
							container.NewHBox(
								w.corporationLogo,
								layout.NewSpacer(),
								w.allianceLogo,
							),
						),
					),
				),
			),
		),
		nil,
		container.New(layout.NewCustomPaddedVBoxLayout(-3*p),
			container.New(layout.NewCustomPaddedLayout(0, p, -2*p, 0), w.characterName),
			container.NewBorder(
				nil,
				nil,
				widget.NewIcon(theme.NewThemedResource(icons.SchoolSvg)),
				nil,
				w.skillpoints,
			),
			container.NewBorder(
				nil,
				nil,
				widget.NewIcon(theme.NewThemedResource(icons.CashSvg)),
				nil,
				w.wallet,
			),
			// container.NewBorder(
			// 	nil,
			// 	nil,
			// 	widget.NewIcon(theme.MailComposeIcon()),
			// 	nil,
			// 	w.mails,
			// ),
			container.NewBorder(
				nil,
				nil,
				widget.NewIcon(theme.NewThemedResource(icons.ShipWheelSvg)),
				nil,
				w.ship,
			),
			container.NewBorder(
				nil,
				nil,
				widget.NewIcon(theme.NewThemedResource(icons.MapMarkerSvg)),
				nil,
				w.solarSystem,
			),
		),
	)
	r := container.New(layout.NewCustomPaddedLayout(p, p, p, p), container.NewStack(c, w.border))
	return widget.NewSimpleRenderer(r)
}

func (w *characterCardSmall) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.background.FillColor = th.Color(theme.ColorNameHover, v)
	w.border.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	w.BaseWidget.Refresh()
}

func (w *characterCardSmall) set(r characterOverviewRow) {
	iwidget.RefreshImageAsync(w.portrait, func() (fyne.Resource, error) {
		return w.eis.CharacterPortrait(r.characterID, 512)
	})
	iwidget.RefreshImageAsync(w.corporationLogo, func() (fyne.Resource, error) {
		return w.eis.CorporationLogo(r.corporation.ID, 64)
	})
	if r.alliance != nil {
		w.allianceLogo.Show()
		iwidget.RefreshImageAsync(w.allianceLogo, func() (fyne.Resource, error) {
			return w.eis.AllianceLogo(r.alliance.ID, 64)
		})
	} else {
		w.allianceLogo.Hide()
	}
	w.characterName.SetText(r.characterName)
	w.mails.SetText(r.unreadCount.StringFunc("?", func(v int) string {
		if v == 0 {
			return "-"
		}
		return humanize.Comma(int64(v))
	}))
	w.skillpoints.SetText(r.skillpoints.StringFunc("?", func(v int) string {
		return humanize.Comma(int64(v))
	}))
	w.wallet.SetText(r.walletBalance.StringFunc("?", func(v float64) string {
		return humanize.Comma(int64(v)) + " ISK"
	}))
	w.ship.SetText(r.shipName())
	var rt []widget.RichTextSegment
	if r.location != nil && r.location.SolarSystem != nil {
		rt = r.location.SolarSystem.DisplayRichText()
	} else {
		rt = iwidget.RichTextSegmentsFromText("?")
	}
	w.solarSystem.Set(rt)
}
