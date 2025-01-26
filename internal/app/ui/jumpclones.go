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
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
)

type jumpCloneNode struct {
	ImplantCount           int
	ImplantTypeID          int32
	ImplantTypeName        string
	ImplantTypeDescription string
	IsUnknown              bool
	JumpCloneID            int32
	JumpCloneName          string
	LocationID             int64
	LocationName           string
}

func (n jumpCloneNode) IsRoot() bool {
	return n.ImplantTypeID == 0
}

func (n jumpCloneNode) UID() widget.TreeNodeID {
	if n.JumpCloneID == 0 {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%d", n.JumpCloneID, n.ImplantTypeID)
}

// JumpClonesArea is the UI area that shows the skillqueue
type JumpClonesArea struct {
	Content    *fyne.Container
	top        *widget.Label
	treeData   *fynetree.FyneTree[jumpCloneNode]
	treeWidget *widget.Tree
	u          *BaseUI
}

func (u *BaseUI) NewJumpClonesArea() *JumpClonesArea {
	a := JumpClonesArea{
		top:      widget.NewLabel(""),
		treeData: fynetree.New[jumpCloneNode](),
		u:        u,
	}
	a.top.TextStyle.Bold = true
	a.treeWidget = a.makeTree()
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.Content = container.NewBorder(top, nil, nil, nil, a.treeWidget)
	return &a
}

func (a *JumpClonesArea) makeTree() *widget.Tree {
	t := widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return a.treeData.ChildUIDs(uid)
		},
		func(uid widget.TreeNodeID) bool {
			return a.treeData.IsBranch(uid)
		},
		func(branch bool) fyne.CanvasObject {
			icon := canvas.NewImageFromResource(IconCharacterplaceholder32Jpeg)
			icon.FillMode = canvas.ImageFillOriginal
			first := widget.NewLabel("Template")
			second := kwidget.NewTappableIcon(theme.InfoIcon(), nil)
			third := widget.NewLabel("Template")
			return container.NewHBox(icon, first, second, third)
		},
		func(uid widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			hbox := co.(*fyne.Container).Objects
			icon := hbox[0].(*canvas.Image)
			first := hbox[1].(*widget.Label)
			second := hbox[2].(*kwidget.TappableIcon)
			third := hbox[3].(*widget.Label)
			n, ok := a.treeData.Value(uid)
			if !ok {
				return
			}
			if n.IsRoot() {
				icon.Resource = eveicon.GetResourceByName(eveicon.CloningCenter)
				icon.Refresh()
				first.SetText(n.LocationName)
				var t string
				var i widget.Importance
				if n.ImplantCount > 0 {
					t = fmt.Sprintf("%d implants", n.ImplantCount)
					i = widget.MediumImportance
				} else {
					t = "No implants"
					i = widget.LowImportance
				}
				if !n.IsUnknown {
					second.OnTapped = func() {
						a.u.ShowLocationInfoWindow(n.LocationID)
					}
					second.Show()
				} else {
					second.Hide()
				}
				third.Text = t
				third.Importance = i
				third.Refresh()
				third.Show()
			} else {
				RefreshImageResourceAsync(icon, func() (fyne.Resource, error) {
					return a.u.EveImageService.InventoryTypeIcon(n.ImplantTypeID, DefaultIconPixelSize)
				})
				first.SetText(n.ImplantTypeName)
				second.Hide()
				third.Hide()
			}
		},
	)
	t.OnSelected = func(uid widget.TreeNodeID) {
		defer t.UnselectAll()
		n, ok := a.treeData.Value(uid)
		if !ok {
			return
		}
		// if n.IsRoot() && !n.IsUnknown {
		// 	a.ShowLocationInfoWindow(n.LocationID)
		// 	return
		// }
		if !n.IsRoot() {
			a.u.ShowTypeInfoWindow(n.ImplantTypeID, a.u.CharacterID(), DescriptionTab)
		}
	}
	return t
}

func (a *JumpClonesArea) Redraw() {
	var t string
	var i widget.Importance
	tree, err := a.newTreeData()
	if err != nil {
		slog.Error("Failed to refresh jump clones UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		clonesCount := len(tree.ChildUIDs(""))
		t, i = a.makeTopText(clonesCount)
	}
	a.treeData = tree
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.treeWidget.Refresh()
}

func (a *JumpClonesArea) newTreeData() (*fynetree.FyneTree[jumpCloneNode], error) {
	tree := fynetree.New[jumpCloneNode]()
	if !a.u.HasCharacter() {
		return tree, nil
	}
	clones, err := a.u.CharacterService.ListCharacterJumpClones(context.TODO(), a.u.CharacterID())
	if err != nil {
		return tree, err
	}
	for _, c := range clones {
		n := jumpCloneNode{
			ImplantCount:  len(c.Implants),
			JumpCloneID:   c.JumpCloneID,
			JumpCloneName: c.Name,
			LocationID:    c.Location.ID,
		}
		// TODO: Refactor to use same location method for all unknown location cases
		if c.Location.Name != "" {
			n.LocationName = c.Location.Name
		} else {
			n.LocationName = fmt.Sprintf("Unknown location #%d", c.Location.ID)
			n.IsUnknown = true
		}
		uid := tree.MustAdd("", n.UID(), n)
		for _, i := range c.Implants {
			n := jumpCloneNode{
				ImplantTypeDescription: i.EveType.DescriptionPlain(),
				ImplantTypeID:          i.EveType.ID,
				ImplantTypeName:        i.EveType.Name,
				JumpCloneID:            c.JumpCloneID,
			}
			tree.MustAdd(uid, n.UID(), n)
		}
	}
	return tree, err
}

func (a *JumpClonesArea) makeTopText(total int) (string, widget.Importance) {
	if !a.u.HasCharacter() {
		return "No character", widget.LowImportance
	}
	hasData := a.u.StatusCacheService.CharacterSectionExists(a.u.CharacterID(), app.SectionJumpClones)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	return fmt.Sprintf("%d clones", total), widget.MediumImportance
}
