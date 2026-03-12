package characters

import (
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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xlayout"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type characterOverviewRow struct {
	alliance        optional.Optional[*app.EveEntity]
	characterID     int64
	characterName   string
	corporation     *app.EveEntity
	faction         optional.Optional[*app.EveEntity]
	isWatched       bool
	location        optional.Optional[*app.EveLocation]
	regionName      string
	searchTarget    string
	ship            optional.Optional[*app.EveType]
	skillpoints     optional.Optional[int64]
	solarSystemName string
	tags            set.Set[string]
	trainingActive  optional.Optional[bool]
	unreadCount     optional.Optional[int]
	walletBalance   optional.Optional[float64]
}

func (r characterOverviewRow) allianceName() string {
	return optional.Map(r.alliance, "", func(v *app.EveEntity) string {
		return v.Name
	})
}

func (r characterOverviewRow) corporationName() string {
	if r.corporation == nil {
		return "?"
	}
	return r.corporation.Name
}

func (r characterOverviewRow) shipName() string {
	return optional.Map(r.ship, "", func(v *app.EveType) string {
		return v.Name
	})
}

type Overview struct {
	widget.BaseWidget

	OnUpdate func(characters int)

	footer            *widget.Label
	columnSorter      *xwidget.ColumnSorter[characterOverviewRow]
	loadInfo          *widget.Label
	main              fyne.CanvasObject
	rows              []characterOverviewRow
	rowsFiltered      []characterOverviewRow
	search            *widget.Entry
	selectAlliance    *kxwidget.FilterChipSelect
	selectCorporation *kxwidget.FilterChipSelect
	selectRegion      *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortButton        *xwidget.SortButton
	u                 baseUI
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

func NewOverview(u baseUI) *Overview {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[characterOverviewRow]{{
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
			return optional.Compare(a.unreadCount, b.unreadCount)
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
			return optional.Compare(a.skillpoints, b.skillpoints)
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
			return optional.Compare(a.walletBalance, b.walletBalance)
		},
	}})

	info := widget.NewLabel("Loading...")
	info.Importance = widget.LowImportance

	a := &Overview{
		footer:       ui.NewLabelWithTruncation(""),
		columnSorter: xwidget.NewColumnSorter(columns, overviewColCharacter, xwidget.SortAsc),
		loadInfo:     info,
		search:       widget.NewEntry(),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRowsAsync(-1)
	})
	a.search.OnChanged = func(_ string) {
		a.filterRowsAsync(-1)
	}
	a.search.PlaceHolder = "Search characters and systems"
	if !a.u.IsMobile() {
		a.main = a.makeGrid()
	} else {
		a.main = a.makeList()
	}

	a.selectAlliance = kxwidget.NewFilterChipSelect("Alliance", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectCorporation = kxwidget.NewFilterChipSelect("Corporation", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectRegion = kxwidget.NewFilterChipSelect("Region", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectSolarSystem = kxwidget.NewFilterChipSelectWithSearch("System", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())

	// Signals
	a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		switch arg.Section {
		case app.SectionEveCharacters:
			characters := set.Collect(xiter.MapSlice(a.rows, func(r characterOverviewRow) int64 {
				return r.characterID
			}))
			for characterID := range set.Intersection(characters, arg.Changed).All() {
				a.updateItem(ctx, characterID)
			}
		}
	})
	a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.Update(ctx)
	})
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.Update(ctx)
	})
	a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, _ struct{}) {
		a.Update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		switch arg.Section {
		case
			app.SectionCharacterLocation,
			app.SectionCharacterMailHeaders,
			app.SectionCharacterSkills,
			app.SectionCharacterWalletBalance:
			a.updateItem(ctx, arg.CharacterID)
		}
	})
	a.u.Signals().CharacterSectionUpdated.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		switch arg.Section {
		case
			app.SectionCharacterSkillqueue:
			a.updateItem(ctx, arg.CharacterID)
		}
	})
	a.u.Signals().CharacterChanged.AddListener(func(ctx context.Context, characterID int64) {
		a.updateItem(ctx, characterID)
	})
	return a
}

func (a *Overview) CreateRenderer() fyne.WidgetRenderer {
	filters := container.NewHBox(
		a.selectAlliance,
		a.selectCorporation,
		a.selectRegion,
		a.selectSolarSystem,
		a.selectTag,
		a.sortButton,
	)
	var topBox *fyne.Container
	if a.u.IsMobile() {
		topBox = container.NewVBox(a.search, container.NewHScroll(filters))
	} else {
		topBox = container.NewBorder(nil, nil, filters, nil, a.search)
	}
	c := container.NewBorder(
		topBox,
		a.footer,
		nil,
		nil,
		container.NewStack(a.loadInfo, a.main),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *Overview) makeGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newCharacterCard(
				a.u.EVEImage().CharacterPortraitAsync,
				a.u.EVEImage().CorporationLogoAsync,
				a.u.EVEImage().AllianceLogoAsync,
				false,
				a.u.InfoViewer().Show,
			)
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
		go a.u.ShowCharacter(context.Background(), r.characterID)

	}
	return g
}

func (a *Overview) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newCharacterCard(
				a.u.EVEImage().CharacterPortraitAsync,
				a.u.EVEImage().CorporationLogoAsync,
				a.u.EVEImage().AllianceLogoAsync,
				true,
				a.u.InfoViewer().Show,
			)
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
		go a.u.ShowCharacter(context.Background(), r.characterID)
	}
	return l
}

func (a *Overview) filterRowsAsync(sortCol int) {
	rows := slices.Clone(a.rows)
	total := len(rows)
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

		footer := fmt.Sprintf("Showing %d / %d characters", len(rows), total)

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
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

func (a *Overview) Update(ctx context.Context) {
	reset := func() {
		fyne.Do(func() {
			a.rows = xslices.Reset(a.rows)
			a.filterRowsAsync(-1)
		})
	}
	setFooter := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.footer.Text = s
			a.footer.Importance = i
			a.footer.Refresh()
		})
	}
	rows, err := a.fetchRows(ctx)
	if err != nil {
		reset()
		setFooter("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		slog.Error("Failed to refresh overview UI", "err", err)
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.loadInfo.Hide()
		a.filterRowsAsync(-1)
		if a.OnUpdate != nil {
			a.OnUpdate(len(rows))
		}
	})
}

func (a *Overview) updateItem(ctx context.Context, characterID int64) {
	logErr := func(err error) {
		slog.Error("characterOverview: Failed to update item", "characterID", characterID, "error", err)
	}
	c, err := a.u.Character().GetCharacter(ctx, characterID)
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
		a.filterRowsAsync(-1)
	})
}

func (a *Overview) fetchRows(ctx context.Context) ([]characterOverviewRow, error) {
	characters, err := a.u.Character().ListCharacters(ctx)
	if err != nil {
		return nil, err
	}
	var rows []characterOverviewRow
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

func (a *Overview) fetchRow(ctx context.Context, c *app.Character) (characterOverviewRow, error) {
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
		skillpoints:   c.TrainedSP,
		walletBalance: c.WalletBalance,
	}
	if el, ok := c.Location.Value(); ok {
		if es, ok := el.SolarSystem.Value(); ok {
			r.regionName = es.Constellation.Region.Name
			r.solarSystemName = es.Name
			r.searchTarget += "~" + strings.ToLower(es.Name)
		}
	}
	total, unread, err := a.u.Character().GetMailCounts(ctx, c.ID)
	if err != nil {
		return r, err
	}
	if total > 0 {
		r.unreadCount = optional.New(unread)
	}
	d, err := a.u.Character().TotalTrainingTime(ctx, c.ID)
	if err != nil {
		return r, err
	}
	if v, ok := d.Value(); ok {
		r.trainingActive.Set(v > 0)
	}
	tags, err := a.u.Character().ListTagsForCharacter(ctx, c.ID)
	if err != nil {
		return r, err
	}
	r.tags = tags
	return r, nil
}

// characterCard is a widget that shows a card for a character.
// It has a large version designed for desktop and a small version designed for mobile.
type characterCard struct {
	widget.BaseWidget

	allianceLogo             *xwidget.TappableImage
	background               *canvas.Rectangle
	border                   *canvas.Rectangle
	characterName            *widget.Label
	corporationLogo          *xwidget.TappableImage
	isSmall                  bool
	mails                    *widget.Label
	portrait                 *canvas.Image
	resourceTrainingActive   fyne.Resource
	resourceTrainingExpired  fyne.Resource
	resourceTrainingInactive fyne.Resource
	resourceTrainingUnknown  fyne.Resource
	ship                     *widget.Label
	showInfo                 func(*app.EveEntity)
	skillpoints              *widget.Label
	solarSystem              *xwidget.RichText
	trainingStatus           *ttwidget.Icon
	wallet                   *widget.Label
	loadCharacter            loadFuncAsync
	loadCorporation          loadFuncAsync
	loadAlliance             loadFuncAsync
}

func newCharacterCard(loadCharacter, loadCorporation, loadAlliance loadFuncAsync, isSmall bool, showInfo func(*app.EveEntity)) *characterCard {
	const numberTemplate = "9.999.999.999"
	makeLabel := func(s string) *widget.Label {
		l := widget.NewLabel(s)
		l.Alignment = fyne.TextAlignTrailing
		l.Truncation = fyne.TextTruncateEllipsis
		return l
	}
	var portrait *canvas.Image
	if isSmall {
		portrait = xwidget.NewImageFromResource(
			icons.Characterplaceholder256Jpeg,
			fyne.NewSquareSize(88),
		)
	} else {
		portrait = xwidget.NewImageFromResource(
			icons.Characterplaceholder512Jpeg,
			fyne.NewSquareSize(200),
		)
	}
	resTraining := theme.MediaRecordIcon()
	trainingUnknown := theme.NewDisabledResource(icons.QuestionmarksmallSvg)
	w := &characterCard{
		allianceLogo:             xwidget.NewTappableImage(icons.Corporationplaceholder64Png, nil),
		background:               canvas.NewRectangle(theme.Color(theme.ColorNameHover)),
		border:                   canvas.NewRectangle(color.Transparent),
		characterName:            widget.NewLabel("Veronica Blomquist"),
		corporationLogo:          xwidget.NewTappableImage(icons.Corporationplaceholder64Png, nil),
		isSmall:                  isSmall,
		loadAlliance:             loadAlliance,
		loadCharacter:            loadCharacter,
		loadCorporation:          loadCorporation,
		mails:                    makeLabel(numberTemplate),
		portrait:                 portrait,
		resourceTrainingActive:   theme.NewSuccessThemedResource(resTraining),
		resourceTrainingExpired:  theme.NewErrorThemedResource(resTraining),
		resourceTrainingInactive: theme.NewDisabledResource(resTraining),
		resourceTrainingUnknown:  trainingUnknown,
		ship:                     makeLabel("Merlin"),
		showInfo:                 showInfo,
		skillpoints:              makeLabel(numberTemplate),
		solarSystem:              xwidget.NewRichText(),
		trainingStatus:           ttwidget.NewIcon(trainingUnknown),
		wallet:                   makeLabel(numberTemplate + " ISK"),
	}
	w.ExtendBaseWidget(w)
	var logoSize float32
	if isSmall {
		logoSize = ui.IconUnitSize
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
				&xlayout.BottomLeftLayout{},
				container.New(logoBorder, w.corporationLogo),
			),
			container.New(
				&xlayout.BottomRightLayout{},
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

	w.loadCharacter(c.characterID, portraitSize, func(r fyne.Resource) {
		w.portrait.Resource = r
		w.portrait.Refresh()
	})

	if !w.isSmall {
		w.corporationLogo.OnTapped = func() {
			w.showInfo(c.corporation)
		}
		w.corporationLogo.SetToolTip(c.corporationName())
	}

	w.loadCorporation(c.corporation.ID, logoSize, func(r fyne.Resource) {
		w.corporationLogo.SetResource(r)
	})

	if alliance, ok := c.alliance.Value(); ok {
		if !w.isSmall {
			w.allianceLogo.OnTapped = func() {
				w.showInfo(alliance)
			}
			w.allianceLogo.SetToolTip(c.allianceName())
		}
		w.allianceLogo.Show()
		w.loadAlliance(alliance.ID, logoSize, func(r fyne.Resource) {
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

	w.skillpoints.SetText(c.skillpoints.StringFunc("?", func(v int64) string {
		return humanize.Comma(int64(v))
	}))

	if v, ok := c.trainingActive.Value(); ok {
		if v {
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
	} else {
		w.trainingStatus.SetResource(w.resourceTrainingUnknown)
		w.trainingStatus.SetToolTip("Training status unknown")
	}

	w.wallet.SetText(c.walletBalance.StringFunc("?", func(v float64) string {
		return humanize.Comma(int64(v)) + " ISK"
	}))

	w.ship.SetText(c.shipName())

	rt := xwidget.RichTextSegmentsFromText("?")
	if el, ok := c.location.Value(); ok {
		if es, ok := el.SolarSystem.Value(); ok {
			rt = es.DisplayRichText()
		}
	}
	rt = xwidget.AlignRichTextSegments(fyne.TextAlignTrailing, rt)
	w.solarSystem.Set(rt)
}
