package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type searchNode struct {
	category string
	ee       *app.EveEntity
}

func (sn searchNode) isCategory() bool {
	return sn.ee == nil
}

func (sn searchNode) UID() widget.TreeNodeID {
	if sn.isCategory() {
		return "C_" + sn.category
	}
	return fmt.Sprintf("EE_%d", sn.ee.ID)
}

func (sn searchNode) String() string {
	if sn.isCategory() {
		return sn.category
	}
	return sn.ee.Name
}

func (u *BaseUI) ShowSearchDialog() {
	entry := widget.NewEntry()
	entry.ActionItem = widget.NewIcon(theme.SearchIcon())
	d := dialog.NewCustomConfirm("Search New Eden", "Search", "Close", entry, func(b bool) {
		go func() {
			search := entry.Text
			if search == "" {
				return
			}
			ee, total, err := u.CharacterService.SearchESI(context.Background(), u.CurrentCharacterID(), search)
			if err != nil {
				d2 := iwidget.NewErrorDialog("Search failed", err, u.Window)
				d2.Show()
				return
			}
			t := fynetree.New[searchNode]()
			categories := []app.EveEntityCategory{
				app.EveEntityAlliance,
				app.EveEntityCharacter,
				app.EveEntityCorporation,
				app.EveEntityInventoryType,
				app.EveEntitySolarSystem,
				app.EveEntityStation,
			}
			var categoriesFound int
			for _, c := range categories {
				_, ok := ee[c]
				if !ok {
					continue
				}
				categoriesFound++
				n := searchNode{category: app.Titler.String(fmt.Sprintf("%s (%d)", c.String(), len(ee[c])))}
				parentUID, err := t.Add("", n.UID(), n)
				if err != nil {
					panic(err)
				}
				for _, o := range ee[c] {
					n := searchNode{ee: o}
					t.Add(parentUID, n.UID(), n)
				}
			}
			tree := widget.NewTree(
				func(uid widget.TreeNodeID) []widget.TreeNodeID {
					return t.ChildUIDs(uid)
				},
				func(uid widget.TreeNodeID) bool {
					return t.IsBranch(uid)
				},
				func(b bool) fyne.CanvasObject {
					return widget.NewLabel("Template")
				},
				func(uid widget.TreeNodeID, b bool, co fyne.CanvasObject) {
					v, ok := t.Value(uid)
					if !ok {
						return
					}
					co.(*widget.Label).SetText(v.String())
				},
			)
			tree.OnSelected = func(uid widget.TreeNodeID) {
				defer tree.UnselectAll()
				v, ok := t.Value(uid)
				if !ok {
					return
				}
				if v.isCategory() {
					return
				}
				u.ShowEveEntityInfoWindow(v.ee)
			}
			if categoriesFound == 1 {
				tree.OpenAllBranches()
			}
			w := u.FyneApp.NewWindow(fmt.Sprintf("Search results (%d)", total))
			l := widget.NewLabel(fmt.Sprintf(
				"More then 500 sesarch results where returned for \"%s\". PLease be more specific.",
				search,
			))
			l.Importance = widget.WarningImportance
			note := container.NewVBox(l, widget.NewSeparator())
			if total < 500 {
				note.Hide()
			}
			c := container.NewBorder(
				note,
				container.NewCenter(widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
					w.Hide()
				})),
				nil,
				nil,
				tree,
			)
			w.SetContent(c)
			w.Resize(fyne.NewSize(600, 400))
			w.Show()
		}()
	}, u.Window)
	d.Show()
	d.Resize(fyne.NewSize(500, 100))
}
