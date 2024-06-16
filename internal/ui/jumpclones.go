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

	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

type jumpCloneNode struct {
	LocationID             int64
	LocationName           string
	ImplantCount           int
	ImplantTypeID          int32
	ImplantTypeName        string
	ImplantTypeDescription string
}

func (n jumpCloneNode) isBranch() bool {
	return n.ImplantTypeID == 0 && n.ImplantCount > 0
}

func (n jumpCloneNode) isClone() bool {
	return n.ImplantTypeID == 0
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
			n, err := treeNodeFromDataItem[jumpCloneNode](di)
			if err != nil {
				slog.Error("Failed to render jump clone item in UI", "err", err)
				first.SetText("ERROR")
				return
			}
			if n.isClone() {
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
					return a.ui.sv.EveImage.InventoryTypeIcon(n.ImplantTypeID, defaultIconSize)
				})
				first.SetText(n.ImplantTypeName)
				second.Hide()
				second.Hide()
			}
		},
	)
	t.OnSelected = func(uid widget.TreeNodeID) {
		defer t.UnselectAll()
		n, err := treeNodeFromBoundTree[jumpCloneNode](a.treeData, uid)
		if err != nil {
			slog.Error("Failed to select jump clone", "err", err)
			t.UnselectAll()
			return
		}
		if n.isBranch() {
			t.ToggleBranch(uid)
			return
		}
		if !n.isClone() {
			a.ui.showTypeInfoWindow(n.ImplantTypeID, a.ui.characterID())
		}
	}
	return t
}

func (a *jumpClonesArea) redraw() {
	t, i, err := func() (string, widget.Importance, error) {
		ids, values, total, err := a.updateTreeData()
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

func (a *jumpClonesArea) updateTreeData() (map[string][]string, map[string]string, int, error) {
	values := make(map[string]string)
	ids := make(map[string][]string)
	if !a.ui.hasCharacter() {
		return ids, values, 0, nil
	}
	clones, err := a.ui.sv.Character.ListCharacterJumpClones(context.Background(), a.ui.characterID())
	if err != nil {
		return nil, nil, 0, err
	}
	for _, c := range clones {
		id := fmt.Sprint(c.JumpCloneID)
		n := jumpCloneNode{
			ImplantCount: len(c.Implants),
			LocationID:   c.Location.ID,
		}
		// TODO: Refactor to use same location method for all unknown location cases
		if c.Location.Name != "" {
			n.LocationName = c.Location.Name
		} else {
			n.LocationName = fmt.Sprintf("Unknown location #%d", c.Location.ID)
		}
		values[id], err = objectToJSON(n)
		if err != nil {
			return nil, nil, 0, err
		}
		ids[""] = append(ids[""], id)
		for _, i := range c.Implants {
			subID := fmt.Sprintf("%s-%d", id, i.EveType.ID)
			n := jumpCloneNode{
				ImplantTypeName:        i.EveType.Name,
				ImplantTypeID:          i.EveType.ID,
				ImplantTypeDescription: i.EveType.DescriptionPlain(),
			}
			values[subID], err = objectToJSON(n)
			if err != nil {
				return nil, nil, 0, err
			}
			ids[id] = append(ids[id], subID)
		}
	}
	return ids, values, len(clones), nil
}

func (a *jumpClonesArea) makeTopText(total int) (string, widget.Importance) {
	if !a.ui.hasCharacter() {
		return "No character", widget.LowImportance
	}
	hasData := a.ui.sv.StatusCache.CharacterSectionExists(a.ui.characterID(), model.SectionJumpClones)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	return fmt.Sprintf("%d clones", total), widget.MediumImportance
}
