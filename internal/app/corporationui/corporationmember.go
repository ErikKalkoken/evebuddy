// Package corporationui provides UI elements related to corporations.
package corporationui

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
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/infowindow"
	awidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type UI interface {
	InfoWindow() *infowindow.InfoWindow
}

type Params struct {
	CharacterService   *characterservice.CharacterService
	CorporationService *corporationservice.CorporationService
	EveImageService    *eveimageservice.EveImageService
	Signals            *app.Signals
	UI                 UI
}

type corporationMemberRow struct {
	id           int64
	name         string
	isCEO        bool
	isOwned      bool
	searchTarget string
}

type CorporationMember struct {
	widget.BaseWidget

	corporation  atomic.Pointer[app.Corporation]
	cs           *characterservice.CharacterService
	eis          *eveimageservice.EveImageService
	footer       *widget.Label
	list         *widget.List
	rows         []corporationMemberRow
	rowsFiltered []corporationMemberRow
	rs           *corporationservice.CorporationService
	searchBox    *widget.Entry
	signals      *app.Signals
	u            UI
}

func NewCorporationMember(arg Params) *CorporationMember {
	a := &CorporationMember{
		cs:      arg.CharacterService,
		eis:     arg.EveImageService,
		footer:  newLabelWithTruncation(),
		rs:      arg.CorporationService,
		signals: arg.Signals,
		u:       arg.UI,
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
	a.signals.CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
		a.corporation.Store(c)
		a.update(ctx)
	})
	a.signals.CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		if corporationIDOrZero(a.corporation.Load()) != arg.CorporationID {
			return
		}
		if arg.Section != app.SectionCorporationMembers {
			return
		}
		a.update(ctx)
	})
	return a
}

func (a *CorporationMember) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		a.searchBox,
		a.footer,
		nil,
		nil,
		a.list,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *CorporationMember) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newCorporationMemberItem(a.eis.CharacterPortraitAsync)
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
		a.u.InfoWindow().ShowEntity(&app.EveEntity{ID: r.id, Category: app.EveEntityCharacter})
	}
	return l
}

func (a *CorporationMember) filterRowsAsync() {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	search := strings.ToLower(a.searchBox.Text)

	go func() {
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r corporationMemberRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		slices.SortFunc(rows, func(a, b corporationMemberRow) int {
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

func (a *CorporationMember) update(ctx context.Context) {
	var corporationID, ceoID int64
	if c := a.corporation.Load(); c != nil {
		corporationID = c.ID
		ceoID = optional.Map(c.EveCorporation.Ceo, 0, func(x *app.EveEntity) int64 {
			return x.ID
		})
	}
	var rows []corporationMemberRow
	var err error
	hasData, err := a.rs.HasSection(ctx, corporationID, app.SectionCorporationMembers)
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

func (a *CorporationMember) fetchRows(ctx context.Context, corporationID, ceoID int64) ([]corporationMemberRow, error) {
	cc, err := a.cs.ListCharacters(ctx)
	if err != nil {
		return nil, err
	}
	var owned set.Set[int64]
	for _, c := range cc {
		if c.EveCharacter.Corporation.ID == corporationID {
			owned.Add(c.ID)
		}
	}
	oo, err := a.rs.ListMembers(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	var rows []corporationMemberRow
	for _, o := range oo {
		rows = append(rows, corporationMemberRow{
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
	member *awidget.EntityListItem
}

func newCorporationMemberItem(loadCharacterIcon loadFuncAsync) *corporationMemberItem {
	ceo := ttwidget.NewIcon(theme.NewWarningThemedResource(icons.CrownSvg))
	ceo.SetToolTip("CEO of this corporation")
	owned := ttwidget.NewIcon(theme.NewSuccessThemedResource(icons.CheckDecagramSvg))
	owned.SetToolTip("You own this character")
	w := &corporationMemberItem{
		ceo:    ceo,
		owned:  owned,
		member: awidget.NewEntityListItem(false, loadCharacterIcon),
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

func (w *corporationMemberItem) set(r corporationMemberRow) {
	w.member.Set(r.id, r.name)
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
