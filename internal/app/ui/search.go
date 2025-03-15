package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type resultNode struct {
	category string
	ee       *app.EveEntity
}

func (sn resultNode) isCategory() bool {
	return sn.ee == nil
}

func (sn resultNode) UID() widget.TreeNodeID {
	if sn.isCategory() {
		return "C_" + sn.category
	}
	return fmt.Sprintf("EE_%d", sn.ee.ID)
}

func (sn resultNode) String() string {
	if sn.isCategory() {
		return sn.category
	}
	return sn.ee.Name
}

type SearchArea struct {
	Content fyne.CanvasObject

	indicator  *widget.ProgressBarInfinite
	entry      *widget.Entry
	note       *widget.Label
	resultTree *fynetree.FyneTree[resultNode]
	tree       *widget.Tree
	u          *BaseUI
}

func NewSearchArea(u *BaseUI) *SearchArea {
	a := &SearchArea{
		indicator:  widget.NewProgressBarInfinite(),
		note:       widget.NewLabel(""),
		resultTree: fynetree.New[resultNode](),
		entry:      widget.NewEntry(),
		u:          u,
	}
	a.tree = a.makeTree()
	a.entry.ActionItem = iwidget.NewIconButton(theme.CancelIcon(), func() {
		a.entry.SetText("")
		a.clearTree()
	})
	a.entry.PlaceHolder = "Search New Eden"
	a.entry.OnSubmitted = func(s string) {
		go a.doSearch(s)
	}
	a.note.Importance = widget.WarningImportance
	a.note.Wrapping = fyne.TextWrapWord
	a.indicator.Hide()

	c := container.NewBorder(
		container.NewVBox(
			a.entry,
			a.indicator,
			a.note,
		),
		nil,
		nil,
		nil,
		a.tree,
	)
	a.Content = c
	return a
}

func (a *SearchArea) makeTree() *widget.Tree {
	t := widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return a.resultTree.ChildUIDs(uid)
		},
		func(uid widget.TreeNodeID) bool {
			return a.resultTree.IsBranch(uid)
		},
		func(b bool) fyne.CanvasObject {
			name := widget.NewLabel("Template")
			image := container.NewPadded(iwidget.NewImageFromResource(icon.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)))
			return container.NewBorder(
				nil,
				nil,
				image,
				nil,
				name,
			)
		},
		func(uid widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			v, ok := a.resultTree.Value(uid)
			if !ok {
				return
			}
			border := co.(*fyne.Container).Objects
			border[0].(*widget.Label).SetText(v.String())
			if v.isCategory() {
				return
			}
			image := border[1].(*fyne.Container).Objects[0].(*canvas.Image)
			appwidget.RefreshImageResourceAsync(image, func() (fyne.Resource, error) {
				c := v.ee.Category.ToEveImage()
				if c == "" {
					return icon.Questionmark32Png, nil
				}
				return a.u.EveImageService.EntityIcon(v.ee.ID, c, app.IconPixelSize)
			})
		},
	)
	t.OnSelected = func(uid widget.TreeNodeID) {
		defer t.UnselectAll()
		v, ok := a.resultTree.Value(uid)
		if !ok {
			return
		}
		if v.isCategory() {
			return
		}
		a.u.ShowEveEntityInfoWindow(v.ee)
	}
	return t
}

func (a *SearchArea) doSearch(search string) {
	a.clearTree()
	if search == "" {
		return
	}
	a.indicator.Show()
	a.indicator.Start()
	defer func() {
		a.indicator.Stop()
		a.indicator.Hide()
	}()
	results, total, err := a.u.CharacterService.SearchESI(context.Background(), a.u.CurrentCharacterID(), search)
	if err != nil {
		d2 := iwidget.NewErrorDialog("Search failed", err, a.u.Window)
		d2.Show()
		return
	}
	categories := []app.EveEntityCategory{
		app.EveEntityAlliance,
		app.EveEntityCharacter,
		app.EveEntityCorporation,
		app.EveEntityInventoryType,
		app.EveEntitySolarSystem,
		app.EveEntityStation,
	}
	t := fynetree.New[resultNode]()
	var categoriesFound int
	titler := cases.Title(language.English)
	for _, c := range categories {
		_, ok := results[c]
		if !ok {
			continue
		}
		categoriesFound++
		n := resultNode{category: titler.String(fmt.Sprintf("%s (%d)", c.String(), len(results[c])))}
		parentUID, err := t.Add("", n.UID(), n)
		if err != nil {
			panic(err)
		}
		for _, o := range results[c] {
			n := resultNode{ee: o}
			t.Add(parentUID, n.UID(), n)
		}
	}
	a.resultTree = t
	a.tree.Refresh()
	if categoriesFound == 1 {
		a.tree.OpenAllBranches()
	}
	if total < 500 {
		a.note.Hide()
	} else {
		a.note.SetText(fmt.Sprintf(
			"More then 500 sesarch results where returned for \"%s\". Please be more specific.",
			search,
		))
	}
}

func (a *SearchArea) clearTree() {
	a.resultTree = fynetree.New[resultNode]()
	a.tree.Refresh()
}
