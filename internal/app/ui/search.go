package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infowindow"
	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
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
	message    *widget.Label
	u          *BaseUI
	w          fyne.Window
}

func NewSearchArea(u *BaseUI) *SearchArea {
	a := &SearchArea{
		entry:      widget.NewEntry(),
		indicator:  widget.NewProgressBarInfinite(),
		message:    widget.NewLabel(""),
		note:       widget.NewLabel(""),
		resultTree: fynetree.New[resultNode](),
		u:          u,
		w:          u.Window,
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
		container.NewStack(a.tree, container.NewCenter(a.message)),
	)
	a.Content = c
	return a
}

func (a *SearchArea) SetWindow(w fyne.Window) {
	a.w = w
}

func (a *SearchArea) Focus() {
	a.w.Canvas().Focus(a.entry)
}

func (a *SearchArea) makeTree() *widget.Tree {
	supportedCategories := infowindow.SupportedEveEntities()
	t := widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return a.resultTree.ChildUIDs(uid)
		},
		func(uid widget.TreeNodeID) bool {
			return a.resultTree.IsBranch(uid)
		},
		func(b bool) fyne.CanvasObject {
			name := widget.NewLabel("Template")
			image := container.NewPadded(iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(app.IconUnitSize)))
			info := iwidget.NewIconButton(theme.InfoIcon(), nil)
			return container.NewBorder(
				nil,
				nil,
				image,
				info,
				name,
			)
		},
		func(uid widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			v, ok := a.resultTree.Node(uid)
			if !ok {
				return
			}
			border := co.(*fyne.Container).Objects
			border[0].(*widget.Label).SetText(v.String())
			image := border[1].(*fyne.Container).Objects[0].(*canvas.Image)
			info := border[2].(*iwidget.IconButton)
			if v.isCategory() {
				info.Hide()
				image.Hide()
				return
			}
			if imageCategory := v.ee.Category.ToEveImage(); imageCategory != "" {
				go func() {
					image.Show()
					ctx := context.Background()
					res, err := func() (fyne.Resource, error) {
						switch v.ee.Category {
						case app.EveEntityInventoryType:
							et, err := a.u.EveUniverseService.GetOrCreateEveTypeESI(ctx, v.ee.ID)
							if err != nil {
								return nil, err
							}
							switch et.Group.Category.ID {
							case app.EveCategorySKINs:
								return a.u.EveImageService.InventoryTypeSKIN(et.ID, app.IconPixelSize)
							case app.EveCategoryBlueprint:
								return a.u.EveImageService.InventoryTypeBPO(et.ID, app.IconPixelSize)
							default:
								return a.u.EveImageService.InventoryTypeIcon(et.ID, app.IconPixelSize)
							}
						default:
							return a.u.EveImageService.EntityIcon(v.ee.ID, imageCategory, app.IconPixelSize)
						}
					}()
					if err != nil {
						res = theme.BrokenImageIcon()
						slog.Error("failed to load image", "error", err)
					}
					image.Resource = res
					image.Refresh()
				}()
			} else {
				image.Hide()
			}
			if supportedCategories.Contains(v.ee.Category) {
				info.OnTapped = func() {
					iw := infowindow.New(a.u.CurrentCharacterID, a.u.CharacterService, a.u.EveUniverseService, a.u.EveImageService, a.w)
					iw.ShowEveEntity(v.ee)
				}
				info.Show()
			} else {
				info.Hide()
			}
		},
	)
	t.OnSelected = func(uid widget.TreeNodeID) {
		defer t.UnselectAll()
		v, ok := a.resultTree.Node(uid)
		if !ok {
			return
		}
		if v.isCategory() {
			t.OpenBranch(uid)
		}
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
	if len(results) == 0 {
		a.message.SetText("No results")
		a.message.Show()
		return
	}
	categories := []character.SearchCategory{
		character.SearchAgent,
		character.SearchAlliance,
		character.SearchCharacter,
		character.SearchCorporation,
		character.SearchFaction,
		character.SearchInventoryType,
		character.SearchSolarSystem,
		character.SearchStation,
	}
	t := fynetree.New[resultNode]()
	var categoriesFound int
	for _, c := range categories {
		_, ok := results[c]
		if !ok {
			continue
		}
		categoriesFound++
		n := resultNode{category: fmt.Sprintf("%s (%d)", c.String(), len(results[c]))}
		parentUID, err := t.Add(fynetree.RootUID, n)
		if err != nil {
			panic(err)
		}
		for _, o := range results[c] {
			n := resultNode{ee: o}
			t.Add(parentUID, n)
		}
	}
	a.resultTree = t
	a.tree.Refresh()
	if categoriesFound == 1 {
		a.tree.OpenAllBranches()
	}
	if total == 500 {
		a.note.SetText(fmt.Sprintf(
			"Search for \"%s\" exceeded the server limit of 500 results "+
				"and may not contain the items you are looking for.",
			search,
		))
		a.note.Show()
	}
}

func (a *SearchArea) clearTree() {
	a.resultTree = fynetree.New[resultNode]()
	a.tree.Refresh()
	a.message.Hide()
	a.note.Hide()
}
