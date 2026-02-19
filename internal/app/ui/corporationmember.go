package ui

import (
	"context"
	"fmt"
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
)

type corporationMemberRow struct {
	id           int64
	name         string
	isCEO        bool
	isOwned      bool
	searchTarget string
}

type corporationMember struct {
	widget.BaseWidget

	corporation  atomic.Pointer[app.Corporation]
	footer       *widget.Label
	list         *widget.List
	rows         []corporationMemberRow
	rowsFiltered []corporationMemberRow
	searchBox    *widget.Entry
	u            *baseUI
}

func newCorporationMember(u *baseUI) *corporationMember {
	a := &corporationMember{
		rows:   make([]corporationMemberRow, 0),
		footer: newLabelWithTruncation(),
		u:      u,
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
		a.filterRows()
		a.list.ScrollToTop()
	}
	a.u.currentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
		a.corporation.Store(c)
		a.update(ctx)
	})
	a.u.corporationSectionChanged.AddListener(func(ctx context.Context, arg corporationSectionUpdated) {
		if corporationIDOrZero(a.corporation.Load()) != arg.corporationID {
			return
		}
		if arg.section != app.SectionCorporationMembers {
			return
		}
		a.update(ctx)
	})
	return a
}

func (a *corporationMember) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		a.searchBox,
		a.footer,
		nil,
		nil,
		a.list,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *corporationMember) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			portrait := iwidget.NewImageFromResource(
				icons.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			name := widget.NewLabel("Template")
			owned := ttwidget.NewIcon(theme.NewSuccessThemedResource(icons.CheckDecagramSvg))
			owned.SetToolTip("You own this character")
			ceo := ttwidget.NewIcon(theme.NewWarningThemedResource(icons.CrownSvg))
			ceo.SetToolTip("CEO of this corporation")
			return container.NewHBox(portrait, name, owned, ceo)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			box := co.(*fyne.Container).Objects
			portrait := box[0].(*canvas.Image)
			a.u.eis.CharacterPortraitAsync(r.id, app.IconPixelSize, func(r fyne.Resource) {
				portrait.Resource = r
				portrait.Refresh()
			})
			box[1].(*widget.Label).SetText(r.name)
			owned := box[2]
			if r.isOwned {
				owned.Show()
			} else {
				owned.Hide()
			}
			ceo := box[3]
			if r.isCEO {
				ceo.Show()
			} else {
				ceo.Hide()
			}
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		a.u.ShowEveEntityInfoWindow(&app.EveEntity{ID: r.id, Category: app.EveEntityCharacter})
	}
	return l
}

func (a *corporationMember) filterRows() {
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

func (a *corporationMember) update(ctx context.Context) {
	var corporationID, ceoID int64
	if c := a.corporation.Load(); c != nil {
		corporationID = c.ID
		ceoID = optional.Map(c.EveCorporation.Ceo, 0, func(x *app.EveEntity) int64 {
			return x.ID
		})
	}
	var rows []corporationMemberRow
	var err error
	hasData := a.u.scs.HasCorporationSection(corporationID, app.SectionCorporationMembers)
	if hasData {
		rows2, err2 := a.fetchRows(ctx, corporationID, ceoID)
		if err2 != nil {
			err = err2
		} else {
			rows = rows2
			hasData = len(rows2) > 0
		}
	}
	t, i := a.u.makeTopText(corporationID, hasData, err, nil)
	if t != "" {
		fyne.Do(func() {
			a.footer.Text, a.footer.Importance = t, i
			a.footer.Refresh()
		})
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRows()
	})
}

func (a *corporationMember) fetchRows(ctx context.Context, corporationID, ceoID int64) ([]corporationMemberRow, error) {
	cc, err := a.u.cs.ListCharacters(ctx)
	if err != nil {
		return nil, err
	}
	var owned set.Set[int64]
	for _, c := range cc {
		if c.EveCharacter.Corporation.ID == corporationID {
			owned.Add(c.ID)
		}
	}
	oo, err := a.u.rs.ListMembers(ctx, corporationID)
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
