package ui

import (
	"cmp"
	"context"
	"fmt"
	"image/color"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

// TODO: Add contact labels

type characterContactRow struct {
	blocked          string
	category         string
	contact          *app.EveEntity
	isBlocked        optional.Optional[bool]
	isWatched        optional.Optional[bool]
	searchTarget     string
	standing         float64
	standingCategory app.StandingCategory
	watched          string
}

type characterContacts struct {
	widget.BaseWidget

	character      atomic.Pointer[app.Character]
	columnSorter   *iwidget.ColumnSorter[characterContactRow]
	footer         *widget.Label
	list           *iwidget.StripedList
	rows           []characterContactRow
	rowsFiltered   []characterContactRow
	searchBox      *widget.Entry
	selectBlocked  *kxwidget.FilterChipSelect
	selectCategory *kxwidget.FilterChipSelect
	selectStanding *kxwidget.FilterChipSelect
	selectWatched  *kxwidget.FilterChipSelect
	sortButton     *iwidget.SortButton
	u              *baseUI
}

func newCharacterContacts(u *baseUI) *characterContacts {
	columnSorter := iwidget.NewColumnSorter(iwidget.NewDataColumns([]iwidget.DataColumn[characterContactRow]{{
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
		iwidget.SortAsc,
	)
	a := &characterContacts{
		columnSorter: columnSorter,
		rows:         make([]characterContactRow, 0),
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
		a.list.ScrollToTop()
	}
	a.selectCategory = kxwidget.NewFilterChipSelect("Faction", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.selectStanding = kxwidget.NewFilterChipSelect("Standing", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.selectBlocked = kxwidget.NewFilterChipSelect("Blocked", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.selectWatched = kxwidget.NewFilterChipSelect("Watched", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync()
	}, a.u.window)

	// signals
	a.u.currentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.characterID {
			return
		}
		if arg.section == app.SectionCharacterContacts {
			a.update(ctx)
		}
	})
	return a
}

func (a *characterContacts) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(
		a.selectCategory,
		a.selectStanding,
		a.selectBlocked,
		a.selectWatched,
		a.sortButton,
	)
	var topBox *fyne.Container
	if a.u.isMobile {
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

func (a *characterContacts) makeList() *iwidget.StripedList {
	l := iwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newCharacterContactItem(a.u.eis)
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
		a.u.ShowEveEntityInfoWindow(r.contact)
	}
	return l
}

func (a *characterContacts) filterRowsAsync() {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	blocked := a.selectBlocked.Selected
	category := a.selectCategory.Selected
	standing := a.selectStanding.Selected
	watched := a.selectWatched.Selected
	search := strings.ToLower(a.searchBox.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(-1)

	go func() {
		if category != "" {
			rows = slices.DeleteFunc(rows, func(r characterContactRow) bool {
				return r.category != category
			})
		}
		if blocked != "" {
			rows = slices.DeleteFunc(rows, func(r characterContactRow) bool {
				return r.blocked != blocked
			})
		}
		if watched != "" {
			rows = slices.DeleteFunc(rows, func(r characterContactRow) bool {
				return r.watched != watched
			})
		}
		if standing != "" {
			rows = slices.DeleteFunc(rows, func(r characterContactRow) bool {
				return r.standingCategory.String() != standing
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r characterContactRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		blockedOptions := set.Collect(xiter.MapSlice(rows, func(r characterContactRow) string {
			return r.blocked
		}))
		categoryOptions := set.Collect(xiter.MapSlice(rows, func(r characterContactRow) string {
			return r.category
		}))
		standingOptions := set.Collect(xiter.MapSlice(rows, func(r characterContactRow) string {
			return r.standingCategory.String()
		}))
		watchedOptions := set.Collect(xiter.MapSlice(rows, func(r characterContactRow) string {
			return r.watched
		}))

		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		footer := fmt.Sprintf("Showing %s / %s members", ihumanize.Comma(len(rows)), ihumanize.Comma(totalRows))

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			if blockedOptions.Equal(set.Of("no")) {
				a.selectBlocked.Disable()
			} else {
				a.selectBlocked.Enable()
				a.selectBlocked.SetOptions(slices.Collect(blockedOptions.All()))
			}
			a.selectCategory.SetOptions(slices.Collect(categoryOptions.All()))
			a.selectStanding.SetOptions(slices.Collect(standingOptions.All()))
			if watchedOptions.Equal(set.Of("no")) {
				a.selectWatched.Disable()
			} else {
				a.selectWatched.Enable()
				a.selectWatched.SetOptions(slices.Collect(watchedOptions.All()))
			}
			a.rowsFiltered = rows
			a.list.Refresh()
		})
	}()
}

func (a *characterContacts) update(ctx context.Context) {
	characterID := characterIDOrZero(a.character.Load())
	if characterID == 0 {
		return
	}
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionCharacterContacts)
	if !hasData {
		return
	}
	rows, err := a.fetchRows(ctx, characterID)
	if err != nil {
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync()
	})
}

func (a *characterContacts) fetchRows(ctx context.Context, characterID int64) ([]characterContactRow, error) {
	oo, err := a.u.cs.ListContacts(ctx, characterID)
	if err != nil {
		return nil, err
	}
	var rows []characterContactRow
	for _, o := range oo {
		rows = append(rows, characterContactRow{
			category:         o.Contact.CategoryDisplay(),
			contact:          o.Contact,
			isBlocked:        o.IsBlocked,
			isWatched:        o.IsWatched,
			searchTarget:     strings.ToLower(o.Contact.Name),
			standing:         o.Standing,
			standingCategory: app.NewStandingCategory(o.Standing),
			blocked: o.IsBlocked.StringFunc("no", func(v bool) string {
				if v {
					return "yes"
				}
				return "no"
			}),
			watched: o.IsWatched.StringFunc("no", func(v bool) string {
				if v {
					return "yes"
				}
				return "no"
			}),
		})
	}
	return rows, nil
}

type characterContactItem struct {
	widget.BaseWidget

	eis     eveEntityEIS
	icon    *canvas.Image
	name    *widget.Label
	symbol  *standingSymbol
	watched *ttwidget.Icon
	blocked *ttwidget.Icon
}

func newCharacterContactItem(eis eveEntityEIS) *characterContactItem {
	icon := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
	name := widget.NewLabel("Template")
	name.Truncation = fyne.TextTruncateClip
	watched := ttwidget.NewIcon(theme.NewPrimaryThemedResource(icons.EyeSvg))
	watched.SetToolTip("Watched")
	blocked := ttwidget.NewIcon(theme.NewErrorThemedResource(icons.CancelSvg))
	blocked.SetToolTip("Blocked")
	w := &characterContactItem{
		eis:     eis,
		icon:    icon,
		name:    name,
		blocked: blocked,
		watched: watched,
		symbol:  newStandingSymbol(),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *characterContactItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		nil,
		nil,
		container.NewCenter(w.icon),
		container.NewHBox(w.watched, w.symbol),
		w.name,
	)
	return widget.NewSimpleRenderer(c)
}

func (w *characterContactItem) set(r characterContactRow) {
	w.name.SetText(r.contact.Name)
	// var i widget.Importance
	// switch r.contact.Category {
	// case app.EveEntityAlliance:
	// 	i = widget.WarningImportance
	// case app.EveEntityCharacter:
	// 	i = widget.MediumImportance
	// case app.EveEntityCorporation:
	// 	i = widget.HighImportance
	// case app.EveEntityFaction:
	// 	i = widget.SuccessImportance
	// default:
	// 	i = widget.LowImportance
	// }
	// w.category.Text, w.category.Importance = r.contact.CategoryDisplay(), i
	// w.category.Refresh()

	// if v, ok := r.contact.IsNPC().Value(); ok && v {
	// 	w.npc.Show()
	// } else {
	// 	w.npc.Hide()
	// }
	w.symbol.set(r.standing, r.standingCategory)
	loadEveEntityIconAsync(w.eis, r.contact, func(r fyne.Resource) {
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
}

type standingSymbol struct {
	ttwidget.ToolTipWidget

	icon      *widget.Icon
	bg        *canvas.Rectangle
	terrible  fyne.Resource
	bad       fyne.Resource
	neutral   fyne.Resource
	good      fyne.Resource
	excellent fyne.Resource
}

func newStandingSymbol() *standingSymbol {
	icon := widget.NewIcon(icons.BlankSvg)
	bg := canvas.NewRectangle(color.Transparent)
	p := theme.Padding()
	bg.SetMinSize(icon.MinSize().Subtract(fyne.NewPos(2*p, 2*p)))
	w := &standingSymbol{
		bg:        bg,
		icon:      icon,
		terrible:  theme.NewErrorThemedResource(icons.MinusBoxSvg),
		bad:       theme.NewWarningThemedResource(icons.MinusBoxSvg),
		neutral:   theme.NewDisabledResource(icons.EqualBoxSvg),
		good:      theme.NewColoredResource(icons.PlusBoxSvg, colorNameInfo),
		excellent: theme.NewPrimaryThemedResource(icons.PlusBoxSvg),
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
