package ui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type corporationMemberRow struct {
	id           int32
	name         string
	isCEO        bool
	searchTarget string
}

type corporationMember struct {
	widget.BaseWidget

	rows         []corporationMemberRow
	rowsFiltered []corporationMemberRow
	list         *widget.List
	searchBox    *widget.Entry
	top          *widget.Label
	u            *baseUI
}

func newCorporationMember(u *baseUI) *corporationMember {
	a := &corporationMember{
		rows: make([]corporationMemberRow, 0),
		top:  widget.NewLabel(""),
		u:    u,
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
	return a
}

func (a *corporationMember) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(a.top, a.searchBox),
		nil,
		nil,
		nil,
		a.list,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *corporationMember) makeList() *widget.List {
	blankIcon := theme.NewThemedResource(icons.BlankSvg)
	ceoIcon := theme.NewThemedResource(icons.CrownSvg)
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
			icon := widget.NewIcon(blankIcon)
			return container.NewHBox(portrait, name, icon)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			box := co.(*fyne.Container).Objects
			portrait := box[0].(*canvas.Image)
			iwidget.RefreshImageAsync(portrait, func() (fyne.Resource, error) {
				return a.u.eis.CharacterPortrait(r.id, app.IconPixelSize)
			})
			box[1].(*widget.Label).SetText(r.name)
			icon := box[2].(*widget.Icon)
			if r.isCEO {
				icon.SetResource(ceoIcon)
			} else {
				icon.SetResource(blankIcon)
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
	rows := slices.Clone(a.rows)
	// search filter
	if search := strings.ToLower(a.searchBox.Text); search != "" {
		rows2 := make([]corporationMemberRow, 0)
		for _, r := range rows {
			var matches bool
			if search == "" {
				matches = true
			} else {
				matches = strings.Contains(r.searchTarget, search)
			}
			if matches {
				rows2 = append(rows2, r)
			}
		}
		rows = rows2
	}
	slices.SortFunc(rows, func(a, b corporationMemberRow) int {
		return strings.Compare(a.name, b.name)
	})
	a.rowsFiltered = rows
	a.list.Refresh()
}

func (a *corporationMember) update() {
	var corporationID, ceoID int32
	if c := a.u.currentCorporation(); c != nil {
		corporationID = c.ID
		ceoID = c.EveCorporation.Ceo.ID
	}
	var rows []corporationMemberRow
	var err error
	hasData := a.u.scs.HasCorporationSection(corporationID, app.SectionCorporationMembers)
	if hasData {
		rows2, err2 := a.fetchRows(corporationID, ceoID)
		if err2 != nil {
			err = err2
		} else {
			rows = rows2
			hasData = len(rows2) > 0
		}
	}
	t, i := a.u.makeTopText(corporationID, hasData, err, func() (string, widget.Importance) {
		return fmt.Sprintf("Members: %d", len(rows)), widget.MediumImportance
	})
	fyne.Do(func() {
		a.top.Text, a.top.Importance = t, i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRows()
	})
}

func (a *corporationMember) fetchRows(corporationID, ceoID int32) ([]corporationMemberRow, error) {
	oo, err := a.u.rs.ListMembers(context.Background(), corporationID)
	if err != nil {
		return nil, err
	}
	rows := make([]corporationMemberRow, 0)
	for _, o := range oo {
		rows = append(rows, corporationMemberRow{
			id:           o.Character.ID,
			name:         o.Character.Name,
			isCEO:        o.Character.ID == ceoID,
			searchTarget: strings.ToLower(o.Character.Name),
		})
	}
	return rows, nil
}
