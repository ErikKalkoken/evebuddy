package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/datanodetree"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
)

type jumpCloneNode struct {
	ImplantCount           int
	ImplantTypeID          int32
	ImplantTypeName        string
	ImplantTypeDescription string
	IsUnknown              bool
	JumpCloneID            int32
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

// jumpClonesArea is the UI area that shows the skillqueue
type jumpClonesArea struct {
	content  *fyne.Container
	tree     *widget.Tree
	treeData binding.StringTree
	top      *widget.Label
	ui       *ui
}

func (u *ui) NewJumpClonesArea() *jumpClonesArea {
	a := jumpClonesArea{
		top:      widget.NewLabel(""),
		treeData: binding.NewStringTree(),
		ui:       u,
	}
	a.top.TextStyle.Bold = true
	a.tree = a.makeTree()
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, a.tree)
	return &a
}

func (a *jumpClonesArea) makeTree() *widget.Tree {
	t := widget.NewTreeWithData(
		a.treeData,
		func(branch bool) fyne.CanvasObject {
			icon := canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
			icon.FillMode = canvas.ImageFillOriginal
			first := widget.NewLabel("Template")
			second := widget.NewLabel("Template")
			return container.NewHBox(icon, first, second)
		},
		func(di binding.DataItem, branch bool, co fyne.CanvasObject) {
			hbox := co.(*fyne.Container)
			icon := hbox.Objects[0].(*canvas.Image)
			first := hbox.Objects[1].(*widget.Label)
			second := hbox.Objects[2].(*widget.Label)
			n, err := datanodetree.NodeFromDataItem[jumpCloneNode](di)
			if err != nil {
				slog.Error("Failed to render jump clone item in UI", "err", err)
				first.SetText("ERROR")
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
				second.Text = t
				second.Importance = i
				second.Refresh()
				second.Show()
			} else {
				refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
					return a.ui.EveImageService.InventoryTypeIcon(n.ImplantTypeID, defaultIconSize)
				})
				first.SetText(n.ImplantTypeName)
				second.Hide()
				second.Hide()
			}
		},
	)
	t.OnSelected = func(uid widget.TreeNodeID) {
		defer t.UnselectAll()
		n, err := datanodetree.NodeFromBoundTree[jumpCloneNode](a.treeData, uid)
		if err != nil {
			slog.Error("Failed to select jump clone", "err", err)
			return
		}
		if n.IsRoot() && !n.IsUnknown {
			a.ui.showLocationInfoWindow(n.LocationID)
			return
		}
		if !n.IsRoot() {
			a.ui.showTypeInfoWindow(n.ImplantTypeID, a.ui.characterID())
		}
	}
	return t
}

func (a *jumpClonesArea) redraw() {
	t, i, err := func() (string, widget.Importance, error) {
		tree, total, err := a.updateTreeData()
		if err != nil {
			return "", 0, err
		}
		ids, values, err := tree.StringTree()
		if err != nil {
			return "", 0, err
		}
		if err := a.treeData.Set(ids, values); err != nil {
			return "", 0, err
		}
		t, i := a.makeTopText(total)
		return t, i, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh jump clones UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
}

func (a *jumpClonesArea) updateTreeData() (datanodetree.DataNodeTree[jumpCloneNode], int, error) {
	tree := datanodetree.New[jumpCloneNode]()
	if !a.ui.hasCharacter() {
		return tree, 0, nil
	}
	clones, err := a.ui.CharacterService.ListCharacterJumpClones(context.Background(), a.ui.characterID())
	if err != nil {
		return tree, 0, err
	}
	for _, c := range clones {
		n := jumpCloneNode{
			JumpCloneID:  c.JumpCloneID,
			ImplantCount: len(c.Implants),
			LocationID:   c.Location.ID,
		}
		// TODO: Refactor to use same location method for all unknown location cases
		if c.Location.Name != "" {
			n.LocationName = c.Location.Name
		} else {
			n.LocationName = fmt.Sprintf("Unknown location #%d", c.Location.ID)
			n.IsUnknown = true
		}
		id := tree.Add("", n)
		for _, i := range c.Implants {
			n := jumpCloneNode{
				JumpCloneID:            c.JumpCloneID,
				ImplantTypeName:        i.EveType.Name,
				ImplantTypeID:          i.EveType.ID,
				ImplantTypeDescription: i.EveType.DescriptionPlain(),
			}
			tree.Add(id, n)
		}
	}
	return tree, len(clones), nil
}

func (a *jumpClonesArea) makeTopText(total int) (string, widget.Importance) {
	if !a.ui.hasCharacter() {
		return "No character", widget.LowImportance
	}
	hasData := a.ui.StatusCacheService.CharacterSectionExists(a.ui.characterID(), app.SectionJumpClones)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	return fmt.Sprintf("%d clones", total), widget.MediumImportance
}
