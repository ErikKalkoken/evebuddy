package corporation

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type memberRow struct {
	id           int64
	name         string
	isCEO        bool
	isOwned      bool
	searchTarget string
}

type Members struct {
	widget.BaseWidget

	corporation  atomic.Pointer[app.Corporation]
	footer       *widget.Label
	list         *widget.List
	rows         []memberRow
	rowsFiltered []memberRow
	searchBox    *widget.Entry
	s            ui
}

func NewMembers(s ui) *Members {
	a := &Members{
		footer: awidget.NewLabelWithTruncation(""),
		s:      s,
	}
	a.list = a.makeList()
	a.ExtendBaseWidget(a)

	a.searchBox = widget.NewEntry()
	a.searchBox.SetPlaceHolder("Search members")
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
	a.s.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
		a.corporation.Store(c)
		a.update(ctx)
	})
	a.s.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		if a.corporation.Load().IDOrZero() != arg.CorporationID {
			return
		}
		if arg.Section != app.SectionCorporationMembers {
			return
		}
		a.update(ctx)
	})
	return a
}

func (a *Members) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		a.searchBox,
		a.footer,
		nil,
		nil,
		a.list,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *Members) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newCorporationMemberItem(awidget.LoadEveEntityIconFunc(a.s.EVEImage()))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			co.(*corporationMemberItem).set(a.rowsFiltered[id])
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		a.s.InfoWindow().ShowEntity(&app.EveEntity{ID: r.id, Category: app.EveEntityCharacter})
	}
	return l
}

func (a *Members) filterRowsAsync() {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	search := strings.ToLower(a.searchBox.Text)

	go func() {
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r memberRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		slices.SortFunc(rows, func(a, b memberRow) int {
			return strings.Compare(a.name, b.name)
		})

		footer := fmt.Sprintf("Showing %s / %s members", ihumanize.Comma(len(rows)), ihumanize.Comma(totalRows))

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.rowsFiltered = rows
			a.list.Refresh()
		})
	}()
}

func (a *Members) update(ctx context.Context) {
	var corporationID, ceoID int64
	if c := a.corporation.Load(); c != nil {
		corporationID = c.ID
		ceoID = optional.Map(c.EveCorporation.Ceo, 0, func(x *app.EveEntity) int64 {
			return x.ID
		})
	}
	var rows []memberRow
	var err error
	hasData, err := a.s.Corporation().HasSection(ctx, corporationID, app.SectionCorporationMembers)
	if hasData {
		rows2, err2 := a.fetchRows(ctx, corporationID, ceoID)
		if err2 != nil {
			err = err2
		} else {
			rows = rows2
			hasData = len(rows2) > 0
		}
	}
	t, i := makeTopText(corporationID, hasData, err, nil)
	if t != "" {
		fyne.Do(func() {
			a.footer.Text, a.footer.Importance = t, i
			a.footer.Refresh()
		})
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync()
	})
}

func (a *Members) fetchRows(ctx context.Context, corporationID, ceoID int64) ([]memberRow, error) {
	cc, err := a.s.Character().ListCharacters(ctx)
	if err != nil {
		return nil, err
	}
	var owned set.Set[int64]
	for _, c := range cc {
		if c.EveCharacter.Corporation.ID == corporationID {
			owned.Add(c.ID)
		}
	}
	oo, err := a.s.Corporation().ListMembers(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	var rows []memberRow
	for _, o := range oo {
		rows = append(rows, memberRow{
			id:           o.Character.ID,
			name:         o.Character.Name,
			isCEO:        o.Character.ID == ceoID,
			isOwned:      owned.Contains(o.Character.ID),
			searchTarget: strings.ToLower(o.Character.Name),
		})
	}
	return rows, nil
}

type corporationMemberItem struct {
	widget.BaseWidget

	ceo    *ttwidget.Icon
	owned  *ttwidget.Icon
	member *awidget.EveEntityListItem
}

func newCorporationMemberItem(loadCharacterIcon awidget.EveEntityIconLoader) *corporationMemberItem {
	ceo := ttwidget.NewIcon(theme.NewWarningThemedResource(icons.CrownSvg))
	ceo.SetToolTip("CEO of this corporation")
	owned := ttwidget.NewIcon(theme.NewSuccessThemedResource(icons.CheckDecagramSvg))
	owned.SetToolTip("You own this character")
	w := &corporationMemberItem{
		ceo:    ceo,
		owned:  owned,
		member: awidget.NewEveEntityListItem(loadCharacterIcon),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *corporationMemberItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewPadded(container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(w.owned, w.ceo),
		w.member,
	))
	return widget.NewSimpleRenderer(c)
}

func (w *corporationMemberItem) set(r memberRow) {
	w.member.Set2(r.id, r.name, app.EveEntityCharacter)
	if r.isOwned {
		w.owned.Show()
	} else {
		w.owned.Hide()
	}
	if r.isCEO {
		w.ceo.Show()
	} else {
		w.ceo.Hide()
	}
}
