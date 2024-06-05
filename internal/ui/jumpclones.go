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

	"github.com/ErikKalkoken/evebuddy/internal/eveonline/icons"
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

type jumpCloneNode struct {
	Name                   string
	Region                 string
	ImplantCount           int
	ImplantTypeID          int32
	ImplantTypeName        string
	ImplantTypeDescription string
}

func (n jumpCloneNode) isClone() bool {
	return n.ImplantTypeID == 0
}

func (n jumpCloneNode) isBranch() bool {
	return n.ImplantTypeID == 0 && n.ImplantCount > 0
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
			third := widget.NewLabel("Template")
			return container.NewHBox(icon, first, second, third)
		},
		func(di binding.DataItem, branch bool, co fyne.CanvasObject) {
			hbox := co.(*fyne.Container)
			icon := hbox.Objects[0].(*canvas.Image)
			first := hbox.Objects[1].(*widget.Label)
			second := hbox.Objects[2].(*widget.Label)
			third := hbox.Objects[3].(*widget.Label)
			n, err := treeNodeFromDataItem[jumpCloneNode](di)
			if err != nil {
				slog.Error("Failed to render jump clone item in UI", "err", err)
				first.SetText("ERROR")
				return
			}
			if n.isClone() {
				icon.Resource, _ = icons.GetResource(icons.CloningCenter)
				icon.Refresh()
				first.SetText(n.Name)
				second.SetText(n.Region)
				second.Show()
				var t string
				var i widget.Importance
				if n.ImplantCount > 0 {
					t = fmt.Sprintf("%d implants", n.ImplantCount)
					i = widget.MediumImportance
				} else {
					t = "No implants"
					i = widget.LowImportance
				}
				third.Text = t
				third.Importance = i
				third.Refresh()
				third.Show()
			} else {
				refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
					return a.ui.sv.EveImage.InventoryTypeIcon(n.ImplantTypeID, defaultIconSize)
				})
				first.SetText(n.ImplantTypeName)
				second.Hide()
				third.Hide()
			}
		},
	)
	t.OnSelected = func(uid widget.TreeNodeID) {
		n, err := treeNodeFromBoundTree[jumpCloneNode](a.treeData, uid)
		if err != nil {
			t := "Failed to select jump clone"
			slog.Error(t, "err", err)
			a.ui.statusBarArea.SetError(t)
		}
		if n.isBranch() {
			t.ToggleBranch(uid)
		}
		if n.isClone() {
			t.UnselectAll()
			return
		}
		a.ui.showTypeInfoWindow(n.ImplantTypeID, a.ui.currentCharID())
		t.UnselectAll()
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
		return a.makeTopText(total)
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
	clones, err := a.ui.sv.Characters.ListCharacterJumpClones(context.Background(), a.ui.currentCharID())
	if err != nil {
		return nil, nil, 0, err
	}
	for _, c := range clones {
		id := fmt.Sprint(c.JumpCloneID)
		n := jumpCloneNode{
			ImplantCount: len(c.Implants),
		}
		// TODO: Refactor to use same location method for all unknown location cases
		if c.Location.Name != "" {
			n.Name = c.Location.Name
		} else {
			n.Name = fmt.Sprintf("Unknown location #%d", c.Location.ID)
		}
		if c.Region != nil {
			n.Region = c.Region.Name
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

func (a *jumpClonesArea) makeTopText(total int) (string, widget.Importance, error) {
	if !a.ui.hasCharacter() {
		return "No character", widget.LowImportance, nil
	}
	hasData, err := a.ui.sv.Characters.CharacterSectionWasUpdated(context.Background(), a.ui.currentCharID(), model.CharacterSectionJumpClones)
	if err != nil {
		return "", 0, err
	}
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	return fmt.Sprintf("%d clones", total), widget.MediumImportance, nil
}
