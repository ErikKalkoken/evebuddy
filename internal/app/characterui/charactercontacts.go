package characterui

import (
	"cmp"
	"context"
	"fmt"
	"image/color"
	"iter"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	awidget "github.com/ErikKalkoken/evebuddy/internal/app/commonui"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/uiservices"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type characterContactRow struct {
	blockedSelect    string
	category         string
	contact          *app.EveEntity
	isBlocked        optional.Optional[bool]
	isNPC            optional.Optional[bool]
	isWatched        optional.Optional[bool]
	labels           set.Set[string]
	labelsDisplay    string
	npcSelect        string
	searchTarget     string
	standing         float64
	standingCategory app.StandingCategory
	watchedSelect    string
}

type CharacterContacts struct {
	widget.BaseWidget

	character      atomic.Pointer[app.Character]
	columnSorter   *xwidget.ColumnSorter[characterContactRow]
	footer         *widget.Label
	list           fyne.CanvasObject
	rows           []characterContactRow
	rowsFiltered   []characterContactRow
	searchBox      *widget.Entry
	selectBlocked  *kxwidget.FilterChipSelect
	selectCategory *kxwidget.FilterChipSelect
	selectLabel    *kxwidget.FilterChipSelect
	selectNPC      *kxwidget.FilterChipSelect
	selectStanding *kxwidget.FilterChipSelect
	selectWatched  *kxwidget.FilterChipSelect
	sortButton     *xwidget.SortButton
	u              uiservices.UIServices
}

func NewCharacterContacts(u uiservices.UIServices) *CharacterContacts {
	columnSorter := xwidget.NewColumnSorter(xwidget.NewDataColumns([]xwidget.DataColumn[characterContactRow]{{
		ID:    1,
		Label: "Name",
		Sort: func(a, b characterContactRow) int {
			return strings.Compare(a.contact.Name, b.contact.Name)
		},
	}, {
		ID:    2,
		Label: "Standing",
		Sort: func(a, b characterContactRow) int {
			return cmp.Compare(a.standing, b.standing)
		},
	}}),
		1,
		xwidget.SortAsc,
	)
	a := &CharacterContacts{
		columnSorter: columnSorter,
		footer:       newLabelWithTruncation(),
		u:            u,
	}
	a.list = a.makeList()
	a.ExtendBaseWidget(a)

	// filters
	a.searchBox = widget.NewEntry()
	a.searchBox.SetPlaceHolder("Search")
	a.searchBox.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.searchBox.SetText("")
	})
	a.searchBox.OnChanged = func(s string) {
		if len(s) == 1 {
			return
		}
		a.filterRowsAsync()
		switch x := a.list.(type) {
		case *widget.List:
			x.ScrollToTop()
		case *xwidget.StripedList:
			x.ScrollToTop()
		}
	}
	a.selectBlocked = kxwidget.NewFilterChipSelect("Blocked", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.selectCategory = kxwidget.NewFilterChipSelect("Category", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.selectLabel = kxwidget.NewFilterChipSelectWithSearch("Label", []string{}, func(string) {
		a.filterRowsAsync()
	}, a.u.MainWindow())
	a.selectNPC = kxwidget.NewFilterChipSelect("NPC", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.selectStanding = kxwidget.NewFilterChipSelect("Standing", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.selectWatched = kxwidget.NewFilterChipSelect("Watched", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync()
	}, a.u.MainWindow())

	// signals
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.CharacterID {
			return
		}
		switch arg.Section {
		case app.SectionCharacterContacts, app.SectionCharacterContactLabels:
			a.update(ctx)
		}
	})
	return a
}

func (a *CharacterContacts) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(
		a.selectCategory,
		a.selectStanding,
		a.selectBlocked,
		a.selectWatched,
		a.selectLabel,
		a.selectNPC,
		a.sortButton,
	)
	var topBox *fyne.Container
	if app.IsMobile() {
		topBox = container.NewVBox(
			container.NewHScroll(filter),
			a.searchBox,
		)
	} else {
		topBox = container.NewBorder(
			nil,
			nil,
			filter,
			nil,
			a.searchBox,
		)
	}
	c := container.NewBorder(
		topBox,
		a.footer,
		nil,
		nil,
		a.list,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterContacts) makeList() fyne.CanvasObject {
	if app.IsMobile() {
		l := xwidget.NewStripedList(
			func() int {
				return len(a.rowsFiltered)
			},
			func() fyne.CanvasObject {
				return newCharacterContactItem(a.u.EVEImage())
			},
			func(id widget.ListItemID, co fyne.CanvasObject) {
				if id >= len(a.rowsFiltered) {
					return
				}
				co.(*characterContactItem).set(a.rowsFiltered[id])

			},
		)
		l.OnSelected = func(id widget.ListItemID) {
			defer l.UnselectAll()
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			a.u.InfoWindow().ShowEntity(r.contact)
		}
		return l
	}
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newCharacterContactItem(a.u.EVEImage())
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			co.(*characterContactItem).set(a.rowsFiltered[id])

		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		a.u.InfoWindow().ShowEntity(r.contact)
	}
	return l
}

func (a *CharacterContacts) filterRowsAsync() {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	blocked := a.selectBlocked.Selected
	category := a.selectCategory.Selected
	label := a.selectLabel.Selected
	npc := a.selectNPC.Selected
	standing := a.selectStanding.Selected
	watched := a.selectWatched.Selected
	search := strings.ToLower(a.searchBox.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(-1)

	go func() {
		var hasNPC bool
		for _, r := range rows {
			if v, ok := r.isNPC.Value(); ok && v {
				hasNPC = true
				break
			}
		}
		var hasBlocked bool
		for _, r := range rows {
			if v, ok := r.isBlocked.Value(); ok && v {
				hasBlocked = true
				break
			}
		}
		var hasWatched bool
		for _, r := range rows {
			if v, ok := r.isWatched.Value(); ok && v {
				hasWatched = true
				break
			}
		}
		var hasLabels bool
		for _, r := range rows {
			if r.labels.Size() > 0 {
				hasLabels = true
				break
			}
		}
		if blocked != "" {
			rows = slices.DeleteFunc(rows, func(r characterContactRow) bool {
				return r.blockedSelect != blocked
			})
		}
		if category != "" {
			rows = slices.DeleteFunc(rows, func(r characterContactRow) bool {
				return r.category != category
			})
		}
		if label != "" {
			rows = slices.DeleteFunc(rows, func(r characterContactRow) bool {
				return !r.labels.Contains(label)
			})
		}
		if npc != "" {
			rows = slices.DeleteFunc(rows, func(r characterContactRow) bool {
				return r.npcSelect != npc
			})
		}
		if standing != "" {
			rows = slices.DeleteFunc(rows, func(r characterContactRow) bool {
				return r.standingCategory.String() != standing
			})
		}
		if watched != "" {
			rows = slices.DeleteFunc(rows, func(r characterContactRow) bool {
				return r.watchedSelect != watched
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r characterContactRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		blockedOptions := slices.Collect(xiter.MapSlice(rows, func(r characterContactRow) string {
			return r.blockedSelect
		}))
		categoryOptions := slices.Collect(xiter.MapSlice(rows, func(r characterContactRow) string {
			return r.category
		}))
		labelOptions := slices.Collect(xiter.Chain(xslices.Map(rows, func(r characterContactRow) iter.Seq[string] {
			return r.labels.All()
		})...))
		npcOptions := slices.Collect(xiter.MapSlice(rows, func(r characterContactRow) string {
			return r.npcSelect
		}))
		standingOptions := slices.Collect(xiter.MapSlice(rows, func(r characterContactRow) string {
			return r.standingCategory.String()
		}))
		watchedOptions := slices.Collect(xiter.MapSlice(rows, func(r characterContactRow) string {
			return r.watchedSelect
		}))

		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		footer := fmt.Sprintf(
			"Showing %s / %s contacts",
			ihumanize.Comma(len(rows)),
			ihumanize.Comma(totalRows),
		)

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectCategory.SetOptions(categoryOptions)
			a.selectStanding.SetOptions(standingOptions)
			if !hasLabels {
				a.selectLabel.Disable()
			} else {
				a.selectLabel.Enable()
				a.selectLabel.SetOptions(labelOptions)
			}
			if !hasBlocked {
				a.selectBlocked.Disable()
			} else {
				a.selectBlocked.Enable()
				a.selectBlocked.SetOptions(blockedOptions)
			}
			if !hasNPC {
				a.selectNPC.Disable()
			} else {
				a.selectNPC.Enable()
				a.selectNPC.SetOptions(npcOptions)
			}
			if !hasWatched {
				a.selectWatched.Disable()
			} else {
				a.selectWatched.Enable()
				a.selectWatched.SetOptions(watchedOptions)
			}
			a.rowsFiltered = rows
			a.list.Refresh()
		})
	}()
}

func (a *CharacterContacts) update(ctx context.Context) {
	clear := func() {
		fyne.Do(func() {
			a.rows = xslices.Reset(a.rows)
			a.filterRowsAsync()
		})
	}
	setFooter := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.footer.Text = s
			a.footer.Importance = i
			a.footer.Refresh()
		})
	}
	characterID := characterIDOrZero(a.character.Load())
	if characterID == 0 {
		clear()
		setFooter("No character", widget.LowImportance)
		return
	}
	if !a.u.StatusCache().HasCharacterSection(characterID, app.SectionCharacterContacts) {
		clear()
		setFooter("Loading data...", widget.WarningImportance)
		return
	}
	rows, err := a.fetchRows(ctx, characterID)
	if err != nil {
		clear()
		setFooter("ERROR: "+app.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync()
	})
}

func (a *CharacterContacts) fetchRows(ctx context.Context, characterID int64) ([]characterContactRow, error) {
	oo, err := a.u.Character().ListContacts(ctx, characterID)
	if err != nil {
		return nil, err
	}
	var rows []characterContactRow
	for _, o := range oo {
		isNPC := o.Contact.IsNPC()
		labelsDisplay := strings.Join(slices.Sorted(o.Labels.All()), ", ")
		blockedSelect := o.IsBlocked.StringFunc("no", func(v bool) string {
			if v {
				return "yes"
			}
			return "no"
		})
		npcSelect := isNPC.StringFunc("no", func(v bool) string {
			if v {
				return "yes"
			}
			return "no"
		})
		watchedSelect := o.IsWatched.StringFunc("no", func(v bool) string {
			if v {
				return "yes"
			}
			return "no"
		})
		rows = append(rows, characterContactRow{
			category:         o.Contact.CategoryDisplay(),
			contact:          o.Contact,
			isBlocked:        o.IsBlocked,
			isWatched:        o.IsWatched,
			labels:           o.Labels,
			isNPC:            isNPC,
			labelsDisplay:    labelsDisplay,
			searchTarget:     strings.ToLower(o.Contact.Name),
			standing:         o.Standing,
			standingCategory: app.NewStandingCategory(o.Standing),
			blockedSelect:    blockedSelect,
			npcSelect:        npcSelect,
			watchedSelect:    watchedSelect,
		})
	}
	return rows, nil
}

type characterContactItem struct {
	widget.BaseWidget

	blocked  *ttwidget.Icon
	eis      awidget.EveEntityEIS
	icon     *canvas.Image
	category *widget.Label
	npc      *widget.Label
	labels   *widget.Label
	name     *widget.Label
	symbol   *standingSymbol
	watched  *ttwidget.Icon
}

func newCharacterContactItem(eis awidget.EveEntityEIS) *characterContactItem {
	icon := xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(32))
	name := widget.NewLabel("")
	name.Truncation = fyne.TextTruncateClip
	labels := widget.NewLabel("")
	labels.Importance = widget.HighImportance
	labels.SizeName = theme.SizeNameCaptionText
	category := widget.NewLabel("")
	category.SizeName = theme.SizeNameCaptionText
	npc := widget.NewLabel("NPC")
	npc.Importance = widget.WarningImportance
	npc.SizeName = theme.SizeNameCaptionText
	watched := ttwidget.NewIcon(theme.NewPrimaryThemedResource(icons.EyeSvg))
	watched.SetToolTip("Watched")
	blocked := ttwidget.NewIcon(theme.NewErrorThemedResource(icons.CancelSvg))
	blocked.SetToolTip("Blocked")
	w := &characterContactItem{
		blocked:  blocked,
		category: category,
		eis:      eis,
		icon:     icon,
		labels:   labels,
		name:     name,
		npc:      npc,
		symbol:   newStandingSymbol(),
		watched:  watched,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *characterContactItem) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			layout.NewSpacer(),
			container.New(layout.NewCustomPaddedLayout(p, p, p, -p), w.icon),
			layout.NewSpacer(),
		),
		container.NewHBox(w.watched, w.symbol),
		container.New(layout.NewCustomPaddedVBoxLayout(-3*p),
			layout.NewSpacer(),
			w.name,
			container.NewHBox(w.category, w.npc, w.labels),
			layout.NewSpacer(),
		),
	)
	return widget.NewSimpleRenderer(container.New(layout.NewCustomPaddedLayout(0, 0, p, p), c))
}

func (w *characterContactItem) set(r characterContactRow) {
	w.name.SetText(r.contact.Name)
	w.labels.SetText(r.labelsDisplay)
	w.category.SetText(r.category)
	w.symbol.set(r.standing, r.standingCategory)
	awidget.LoadEveEntityIconAsync(w.eis, r.contact, func(r fyne.Resource) {
		w.icon.Resource = r
		w.icon.Refresh()
	})
	if v, ok := r.isBlocked.Value(); ok && v {
		w.blocked.Show()
	} else {
		w.blocked.Hide()
	}
	if v, ok := r.isWatched.Value(); ok && v {
		w.watched.Show()
	} else {
		w.watched.Hide()
	}
	if v, ok := r.isNPC.Value(); ok && v {
		w.npc.Show()
	} else {
		w.npc.Hide()
	}
}

type standingSymbol struct {
	ttwidget.ToolTipWidget

	bad       fyne.Resource
	bg        *canvas.Rectangle
	excellent fyne.Resource
	good      fyne.Resource
	icon      *widget.Icon
	neutral   fyne.Resource
	terrible  fyne.Resource
}

func newStandingSymbol() *standingSymbol {
	icon := widget.NewIcon(icons.BlankSvg)
	bg := canvas.NewRectangle(color.Transparent)
	p := theme.Padding()
	bg.SetMinSize(icon.MinSize().Subtract(fyne.NewPos(2*p, 2*p)))
	w := &standingSymbol{
		bad:       theme.NewWarningThemedResource(icons.MinusBoxSvg),
		bg:        bg,
		excellent: theme.NewSuccessThemedResource(icons.PlusBoxSvg),
		good:      theme.NewPrimaryThemedResource(icons.PlusBoxSvg),
		icon:      icon,
		neutral:   theme.NewDisabledResource(icons.EqualBoxSvg),
		terrible:  theme.NewErrorThemedResource(icons.MinusBoxSvg),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *standingSymbol) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewPadded(container.NewCenter(container.NewPadded(w.bg), w.icon))
	return widget.NewSimpleRenderer(c)
}

func (w *standingSymbol) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(theme.ColorNameForegroundOnPrimary, v)
	w.bg.Refresh()
	w.icon.Refresh()
	w.BaseWidget.Refresh()
}

func (w *standingSymbol) set(v float64, sc app.StandingCategory) {
	var r fyne.Resource
	switch sc {
	case app.TerribleStanding:
		r = w.terrible
	case app.BadStanding:
		r = w.bad
	case app.NeutralStanding:
		r = w.neutral
	case app.GoodStanding:
		r = w.good
	case app.ExcellentStanding:
		r = w.excellent
	}
	w.icon.Resource = r
	w.SetToolTip(fmt.Sprintf("%0.1f", v))
	w.Refresh()
}
