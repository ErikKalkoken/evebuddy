package ui

import (
	"cmp"
	"context"
	"errors"
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
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
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
	isWatched       bool
	location        *app.EveLocation
	regionName      string
	searchTarget    string
	ship            *app.EveType
	skillpoints     optional.Optional[int]
	solarSystemName string
	tags            set.Set[string]
	trainingActive  optional.Optional[bool]
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

	columnSorter      *iwidget.ColumnSorter[characterOverviewRow]
	info              *widget.Label
	main              fyne.CanvasObject
	onUpdate          func(characters int)
	rows              []characterOverviewRow
	rowsFiltered      []characterOverviewRow
	search            *widget.Entry
	selectAlliance    *kxwidget.FilterChipSelect
	selectCorporation *kxwidget.FilterChipSelect
	selectRegion      *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortButton        *iwidget.SortButton
	top               *widget.Label
	u                 *baseUI
}

const (
	overviewColAlliance = iota + 1
	overviewColCharacter
	overviewColCorporation
	overviewColMail
	overviewColRegion
	overviewColSolarSystem
	overviewColSkillpoints
	overviewColWallet
)

func newCharacterOverview(u *baseUI) *characterOverview {
	columns := iwidget.NewDataColumns([]iwidget.DataColumn[characterOverviewRow]{{
		ID:    overviewColAlliance,
		Label: "Alliance",
		Sort: func(a, b characterOverviewRow) int {
			return xstrings.CompareIgnoreCase(a.allianceName(), b.allianceName())
		},
	}, {
		ID:    overviewColCharacter,
		Label: "Character",
		Sort: func(a, b characterOverviewRow) int {
			return xstrings.CompareIgnoreCase(a.characterName, b.characterName)
		},
	}, {
		ID:    overviewColCorporation,
		Label: "Corporation",
		Sort: func(a, b characterOverviewRow) int {
			return xstrings.CompareIgnoreCase(a.corporationName(), b.corporationName())
		},
	}, {
		ID:    overviewColMail,
		Label: "Unread",
		Sort: func(a, b characterOverviewRow) int {
			return cmp.Compare(a.unreadCount.ValueOrZero(), b.unreadCount.ValueOrZero())
		},
	}, {
		ID:    overviewColRegion,
		Label: "Region",
		Sort: func(a, b characterOverviewRow) int {
			return strings.Compare(a.regionName, b.regionName)
		},
	}, {
		ID:    overviewColSkillpoints,
		Label: "Skillpoints",
		Sort: func(a, b characterOverviewRow) int {
			return cmp.Compare(a.skillpoints.ValueOrZero(), b.skillpoints.ValueOrZero())
		},
	}, {
		ID:    overviewColSolarSystem,
		Label: "System",
		Sort: func(a, b characterOverviewRow) int {
			return strings.Compare(a.solarSystemName, b.solarSystemName)
		},
	}, {
		ID:    overviewColWallet,
		Label: "Wallet",
		Sort: func(a, b characterOverviewRow) int {
			return cmp.Compare(a.walletBalance.ValueOrZero(), b.walletBalance.ValueOrZero())
		},
	}})

	info := widget.NewLabel("Loading...")
	info.Importance = widget.LowImportance

	a := &characterOverview{
		columnSorter: iwidget.NewColumnSorter(columns, overviewColCharacter, iwidget.SortAsc),
		info:         info,
		rows:         make([]characterOverviewRow, 0),
		rowsFiltered: make([]characterOverviewRow, 0),
		search:       widget.NewEntry(),
		top:          makeTopLabel(),
		u:            u,
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
	if !a.u.isMobile {
		a.main = a.makeGrid()
	} else {
		a.main = a.makeList()
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

	// Signals
	a.u.generalSectionChanged.AddListener(func(ctx context.Context, arg generalSectionUpdated) {
		switch arg.section {
		case app.SectionEveCharacters:
			characters := set.Collect(xiter.MapSlice(a.rows, func(r characterOverviewRow) int32 {
				return r.characterID
			}))
			for characterID := range set.Intersection(characters, arg.changed).All() {
				a.updateItem(ctx, characterID)
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
	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		switch arg.section {
		case
			app.SectionCharacterLocation,
			app.SectionCharacterMailHeaders,
			app.SectionCharacterSkills,
			app.SectionCharacterWalletBalance:
			a.updateItem(ctx, arg.characterID)
		}
	})
	a.u.characterSectionUpdated.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		switch arg.section {
		case
			app.SectionCharacterSkillqueue:
			a.updateItem(ctx, arg.characterID)
		}
	})
	a.u.characterChanged.AddListener(func(ctx context.Context, characterID int32) {
		a.updateItem(ctx, characterID)
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
		container.NewStack(a.info, a.main),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *characterOverview) makeGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newCharacterCard(a.u.eis, false, a.u.ShowInfoWindow)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			co.(*characterCard).set(r)
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
			return newCharacterCard(a.u.eis, true, a.u.ShowInfoWindow)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			co.(*characterCard).set(r)
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
	alliance := a.selectAlliance.Selected
	corporation := a.selectCorporation.Selected
	region := a.selectRegion.Selected
	solarSystem := a.selectSolarSystem.Selected
	tag := a.selectTag.Selected
	search := strings.ToLower(a.search.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		// filter
		if alliance != "" {
			rows = slices.DeleteFunc(rows, func(r characterOverviewRow) bool {
				return r.allianceName() != alliance
			})
		}
		if corporation != "" {
			rows = slices.DeleteFunc(rows, func(r characterOverviewRow) bool {
				return r.corporationName() != corporation
			})
		}
		if region != "" {
			rows = slices.DeleteFunc(rows, func(r characterOverviewRow) bool {
				return r.regionName != region
			})
		}
		if solarSystem != "" {
			rows = slices.DeleteFunc(rows, func(r characterOverviewRow) bool {
				return r.solarSystemName != solarSystem
			})
		}
		if tag != "" {
			rows = slices.DeleteFunc(rows, func(r characterOverviewRow) bool {
				return !r.tags.Contains(tag)
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r characterOverviewRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		allianceOptions := xslices.Map(rows, func(r characterOverviewRow) string {
			return r.allianceName()
		})
		corporationOptions := xslices.Map(rows, func(r characterOverviewRow) string {
			return r.corporationName()
		})
		regionOptions := xslices.Map(rows, func(r characterOverviewRow) string {
			return r.regionName
		})
		solarSystemOptions := xslices.Map(rows, func(r characterOverviewRow) string {
			return r.solarSystemName
		})
		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r characterOverviewRow) set.Set[string] {
			return r.tags
		})...).All())

		fyne.Do(func() {
			a.selectAlliance.SetOptions(allianceOptions)
			a.selectCorporation.SetOptions(corporationOptions)
			a.selectRegion.SetOptions(regionOptions)
			a.selectSolarSystem.SetOptions(solarSystemOptions)
			a.selectTag.SetOptions(tagOptions)
			a.rowsFiltered = rows
			a.main.Refresh()
		})
	}()
}

func (a *characterOverview) update() {
	var rows []characterOverviewRow
	t, i, err := func() (string, widget.Importance, error) {
		cc, err := a.fetchRows(context.Background())
		if err != nil {
			return "", 0, err
		}
		if a.onUpdate != nil {
			a.onUpdate(len(cc))
		}
		if len(cc) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		rows = cc
		s := fmt.Sprintf("%d characters", len(cc))
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

func (a *characterOverview) updateItem(ctx context.Context, characterID int32) {
	logErr := func(err error) {
		slog.Error("characterOverview: Failed to update item", "characterID", characterID, "error", err)
	}
	c, err := a.u.cs.GetCharacter(ctx, characterID)
	if err != nil {
		logErr(err)
		return
	}
	r, err := a.fetchRow(ctx, c)
	if err != nil {
		logErr(err)
		return
	}
	fyne.Do(func() {
		id := slices.IndexFunc(a.rows, func(x characterOverviewRow) bool {
			return x.characterID == c.ID
		})
		if id == -1 {
			return
		}
		a.rows[id] = r
		a.filterRows(-1)
	})
}

func (a *characterOverview) fetchRows(ctx context.Context) ([]characterOverviewRow, error) {
	characters, err := a.u.cs.ListCharacters(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]characterOverviewRow, 0)
	for _, c := range characters {
		r, err := a.fetchRow(ctx, c)
		if errors.Is(err, app.ErrInvalid) {
			continue
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *characterOverview) fetchRow(ctx context.Context, c *app.Character) (characterOverviewRow, error) {
	if c == nil || c.EveCharacter == nil {
		return characterOverviewRow{}, app.ErrInvalid
	}
	r := characterOverviewRow{
		alliance:      c.EveCharacter.Alliance,
		characterID:   c.ID,
		characterName: c.EveCharacter.Name,
		corporation:   c.EveCharacter.Corporation,
		faction:       c.EveCharacter.Faction,
		isWatched:     c.IsTrainingWatched,
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
	total, unread, err := a.u.cs.GetMailCounts(ctx, c.ID)
	if err != nil {
		return r, err
	}
	if total > 0 {
		r.unreadCount = optional.New(unread)
	}
	d, err := a.u.cs.TotalTrainingTime(ctx, c.ID)
	if err != nil {
		return r, err
	}
	if !d.IsEmpty() {
		r.trainingActive.Set(d.ValueOrZero() > 0)
	}
	tags, err := a.u.cs.ListTagsForCharacter(ctx, c.ID)
	if err != nil {
		return r, err
	}
	r.tags = tags
	return r, nil
}

type characterCardEIS interface {
	AllianceLogoAsync(id int32, size int, setter func(r fyne.Resource))
	CharacterPortraitAsync(id int32, size int, setter func(r fyne.Resource))
	CorporationLogoAsync(id int32, size int, setter func(r fyne.Resource))
}

// characterCard is a widget that shows a card for a character.
// It has a large version designed for desktop and a small version designed for mobile.
type characterCard struct {
	widget.BaseWidget

	allianceLogo             *iwidget.TappableImage
	background               *canvas.Rectangle
	border                   *canvas.Rectangle
	characterName            *widget.Label
	corporationLogo          *iwidget.TappableImage
	eis                      characterCardEIS
	isSmall                  bool
	mails                    *widget.Label
	portrait                 *canvas.Image
	resourceTrainingActive   fyne.Resource
	resourceTrainingExpired  fyne.Resource
	resourceTrainingInactive fyne.Resource
	resourceTrainingUnknown  fyne.Resource
	ship                     *widget.Label
	showInfoWindow           func(c app.EveEntityCategory, id int32)
	skillpoints              *widget.Label
	solarSystem              *iwidget.RichText
	trainingStatus           *ttwidget.Icon
	wallet                   *widget.Label
}

func newCharacterCard(eis characterCardEIS, isSmall bool, showInfoWindow func(c app.EveEntityCategory, id int32)) *characterCard {
	const numberTemplate = "9.999.999.999"
	makeLabel := func(s string) *widget.Label {
		l := widget.NewLabel(s)
		l.Alignment = fyne.TextAlignTrailing
		l.Truncation = fyne.TextTruncateEllipsis
		return l
	}
	var portrait *canvas.Image
	if isSmall {
		portrait = iwidget.NewImageFromResource(
			icons.Characterplaceholder256Jpeg,
			fyne.NewSquareSize(88),
		)
	} else {
		portrait = iwidget.NewImageFromResource(
			icons.Characterplaceholder512Jpeg,
			fyne.NewSquareSize(200),
		)
	}
	resTraining := theme.MediaRecordIcon()
	trainingUnknown := theme.NewDisabledResource(icons.QuestionmarksmallSvg)
	w := &characterCard{
		allianceLogo:             iwidget.NewTappableImage(icons.Corporationplaceholder64Png, nil),
		background:               canvas.NewRectangle(theme.Color(theme.ColorNameHover)),
		border:                   canvas.NewRectangle(color.Transparent),
		characterName:            widget.NewLabel("Veronica Blomquist"),
		corporationLogo:          iwidget.NewTappableImage(icons.Corporationplaceholder64Png, nil),
		eis:                      eis,
		isSmall:                  isSmall,
		mails:                    makeLabel(numberTemplate),
		portrait:                 portrait,
		resourceTrainingActive:   theme.NewSuccessThemedResource(resTraining),
		resourceTrainingExpired:  theme.NewErrorThemedResource(resTraining),
		resourceTrainingInactive: theme.NewDisabledResource(resTraining),
		resourceTrainingUnknown:  trainingUnknown,
		ship:                     makeLabel("Merlin"),
		showInfoWindow:           showInfoWindow,
		skillpoints:              makeLabel(numberTemplate),
		solarSystem:              iwidget.NewRichText(),
		trainingStatus:           ttwidget.NewIcon(trainingUnknown),
		wallet:                   makeLabel(numberTemplate + " ISK"),
	}
	w.ExtendBaseWidget(w)
	var logoSize float32
	if isSmall {
		logoSize = app.IconUnitSize
	} else {
		logoSize = 40
	}
	w.allianceLogo.SetFillMode(canvas.ImageFillContain)
	w.allianceLogo.SetMinSize(fyne.NewSquareSize(logoSize))
	w.allianceLogo.SetCornerRadius(theme.InputRadiusSize())

	w.corporationLogo.SetFillMode(canvas.ImageFillContain)
	w.corporationLogo.SetMinSize(fyne.NewSquareSize(logoSize))
	w.corporationLogo.SetCornerRadius(theme.InputRadiusSize())

	w.background.CornerRadius = theme.InputRadiusSize()

	w.characterName.SizeName = theme.SizeNameSubHeadingText
	w.characterName.Truncation = fyne.TextTruncateEllipsis

	w.border.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	w.border.StrokeWidth = 1
	w.border.CornerRadius = theme.Size(theme.SizeNameInputRadius)

	return w
}

func (w *characterCard) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	if w.isSmall {
		c := container.NewBorder(
			nil,
			nil,
			container.New(layout.NewCustomPaddedLayout(0, 0, 0, 2*p),
				container.NewStack(
					w.background,
					container.New(layout.NewCustomPaddedLayout(2*p, 2*p, 3*p, 3*p),
						container.NewVBox(
							w.portrait,
							container.New(layout.NewCustomPaddedLayout(0, -p, 0, 0),
								container.NewHBox(
									w.corporationLogo, layout.NewSpacer(), w.allianceLogo,
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
					container.New(layout.NewCustomPaddedLayout(0, 0, -2*p, p), w.trainingStatus),
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
	logoBorder := &layout.CustomPaddedLayout{
		TopPadding:    1 * p,
		BottomPadding: 1 * p,
		LeftPadding:   1 * p,
		RightPadding:  1 * p,
	}
	skillpoints := ttwidget.NewIcon(theme.NewThemedResource(icons.SchoolSvg))
	skillpoints.SetToolTip("Skillpoints & Training status")
	wallet := ttwidget.NewIcon(theme.NewThemedResource(icons.CashSvg))
	wallet.SetToolTip("Wallet balance")
	mails := ttwidget.NewIcon(theme.MailComposeIcon())
	mails.SetToolTip("Number of unread mails")
	ship := ttwidget.NewIcon(theme.NewThemedResource(icons.ShipWheelSvg))
	ship.SetToolTip("Current ship")
	location := ttwidget.NewIcon(theme.NewThemedResource(icons.MapMarkerSvg))
	location.SetToolTip("Current location")
	c := container.NewBorder(
		container.New(layout.NewCustomPaddedLayout(0, 0, -p, -p), w.characterName),
		container.New(
			layout.NewCustomPaddedVBoxLayout(-2*p),
			container.NewBorder(
				nil,
				nil,
				skillpoints,
				container.New(layout.NewCustomPaddedLayout(0, 0, -p, p), w.trainingStatus),
				w.skillpoints,
			),
			container.NewBorder(
				nil,
				nil,
				wallet,
				nil,
				w.wallet,
			),
			container.NewBorder(
				nil,
				nil,
				mails,
				nil,
				w.mails,
			),
			container.NewBorder(
				nil,
				nil,
				ship,
				nil,
				w.ship,
			),
			container.NewBorder(
				nil,
				nil,
				location,
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

func (w *characterCard) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.background.FillColor = th.Color(theme.ColorNameHover, v)
	w.border.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	w.BaseWidget.Refresh()
}

func (w *characterCard) set(c characterOverviewRow) {
	var portraitSize, logoSize int
	if w.isSmall {
		portraitSize = 256
		logoSize = 64
	} else {
		portraitSize = 512
		logoSize = 64
	}

	w.eis.CharacterPortraitAsync(c.characterID, portraitSize, func(r fyne.Resource) {
		w.portrait.Resource = r
		w.portrait.Refresh()
	})

	if !w.isSmall {
		w.corporationLogo.OnTapped = func() {
			w.showInfoWindow(app.EveEntityCorporation, c.corporation.ID)
		}
		w.corporationLogo.SetToolTip(c.corporationName())
	}

	w.eis.CorporationLogoAsync(c.corporation.ID, logoSize, func(r fyne.Resource) {
		w.corporationLogo.SetResource(r)
	})

	if c.alliance != nil {
		if !w.isSmall {
			w.allianceLogo.OnTapped = func() {
				w.showInfoWindow(app.EveEntityAlliance, c.alliance.ID)
			}
			w.allianceLogo.SetToolTip(c.allianceName())
		}
		w.allianceLogo.Show()
		w.eis.AllianceLogoAsync(c.alliance.ID, logoSize, func(r fyne.Resource) {
			w.allianceLogo.SetResource(r)
		})
	} else {
		w.allianceLogo.Hide()
	}

	w.characterName.SetText(c.characterName)

	w.mails.SetText(c.unreadCount.StringFunc("?", func(v int) string {
		if v == 0 {
			return "-"
		}
		return humanize.Comma(int64(v))
	}))

	w.skillpoints.SetText(c.skillpoints.StringFunc("?", func(v int) string {
		return humanize.Comma(int64(v))
	}))

	if c.trainingActive.IsEmpty() {
		w.trainingStatus.SetResource(w.resourceTrainingUnknown)
		w.trainingStatus.SetToolTip("Training status unknown")
	} else {
		if c.trainingActive.ValueOrZero() {
			w.trainingStatus.SetResource(w.resourceTrainingActive)
			w.trainingStatus.SetToolTip("Training is active")
		} else {
			if c.isWatched {
				w.trainingStatus.SetResource(w.resourceTrainingExpired)
				w.trainingStatus.SetToolTip("Training has expired")
			} else {
				w.trainingStatus.SetResource(w.resourceTrainingInactive)
				w.trainingStatus.SetToolTip("Training is not active")
			}
		}
	}

	w.wallet.SetText(c.walletBalance.StringFunc("?", func(v float64) string {
		return humanize.Comma(int64(v)) + " ISK"
	}))

	w.ship.SetText(c.shipName())

	var rt []widget.RichTextSegment
	if c.location != nil && c.location.SolarSystem != nil {
		rt = c.location.SolarSystem.DisplayRichText()
	} else {
		rt = iwidget.RichTextSegmentsFromText("?")
	}
	rt = iwidget.AlignRichTextSegments(fyne.TextAlignTrailing, rt)
	w.solarSystem.Set(rt)
}
