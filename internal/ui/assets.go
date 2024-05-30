package ui

import (
	"cmp"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

type locationNode struct {
	ID     int64
	Name   string
	System string
}

// func (n locationNode) isBranch() bool {
// 	return n.ImplantTypeID == 0 && n.ImplantCount > 0
// }

// assetsArea is the UI area that shows the skillqueue
type assetsArea struct {
	content          fyne.CanvasObject
	assetGrid        *widget.GridWrap
	assetGridData    binding.UntypedList
	locationTree     *widget.Tree
	locationTreeData binding.StringTree
	top              *widget.Label
	ui               *ui
}

func (u *ui) newAssetsArea() *assetsArea {
	a := assetsArea{
		top:              widget.NewLabel(""),
		assetGridData:    binding.NewUntypedList(),
		locationTreeData: binding.NewStringTree(),
		ui:               u,
	}
	a.top.TextStyle.Bold = true

	a.locationTree = widget.NewTreeWithData(
		a.locationTreeData,
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(di binding.DataItem, branch bool, co fyne.CanvasObject) {
			label := co.(*widget.Label)
			n, err := func() (locationNode, error) {
				v, err := di.(binding.String).Get()
				if err != nil {
					return locationNode{}, err
				}
				n, err := newObjectFromJSON[locationNode](v)
				if err != nil {
					return locationNode{}, err
				}
				return n, nil
			}()
			if err != nil {
				slog.Error("Failed to render asset location in UI", "err", err)
				label.SetText("ERROR")
				return
			}
			label.SetText(n.Name)
		},
	)
	// a.tree.OnSelected = func(uid widget.TreeNodeID) {
	// 	n, err := func() (locationNode, error) {
	// 		v, err := a.treeData.GetValue(uid)
	// 		if err != nil {
	// 			return locationNode{}, fmt.Errorf("failed to get tree data item: %w", err)
	// 		}
	// 		n, err := newObjectFromJSON[locationNode](v)
	// 		if err != nil {
	// 			return locationNode{}, err
	// 		}
	// 		return n, nil
	// 	}()
	// 	if err != nil {
	// 		t := "Failed to select jump clone"
	// 		slog.Error(t, "err", err)
	// 		a.ui.statusBarArea.SetError(t)
	// 	}
	// 	if n.isBranch() {
	// 		a.tree.ToggleBranch(uid)
	// 	}
	// 	if n.isClone() {
	// 		a.tree.UnselectAll()
	// 		return
	// 	}
	// 	d := makeTypeDetailDialog(n.ImplantTypeName, n.ImplantTypeDescription, a.ui.window)
	// 	d.SetOnClosed(func() {
	// 		a.tree.UnselectAll()
	// 	})
	// 	d.Show()
	// }

	a.assetGrid = widget.NewGridWrapWithData(
		a.assetGridData,
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			//
		},
	)

	top := container.NewVBox(a.top, widget.NewSeparator())
	main := container.NewHSplit(a.locationTree, a.assetGrid)
	a.content = container.NewBorder(top, nil, nil, nil, main)
	return &a
}

func (a *assetsArea) redraw() {
	t, i, err := func() (string, widget.Importance, error) {
		ids, values, total, err := a.updateTreeData()
		if err != nil {
			return "", 0, err
		}
		if err := a.locationTreeData.Set(ids, values); err != nil {
			return "", 0, err
		}
		return a.makeTopText(total)
	}()
	if err != nil {
		slog.Error("Failed to redraw asset locations UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
}

func (a *assetsArea) updateTreeData() (map[string][]string, map[string]string, int, error) {
	values := make(map[string]string)
	ids := make(map[string][]string)
	if !a.ui.hasCharacter() {
		return ids, values, 0, nil
	}
	locations, err := a.ui.service.ListCharacterAssetLocations(a.ui.currentCharID())
	if err != nil {
		return nil, nil, 0, err
	}
	nodes := make([]locationNode, len(locations))
	for i, l := range locations {
		n := locationNode{ID: l.ID}
		// TODO: Refactor to use same location method for all unknown location cases
		if l.Location.Name != "" {
			n.Name = l.Location.Name
		} else {
			n.Name = fmt.Sprintf("Unknown location #%d", l.Location.ID)
		}
		if l.SolarSystem != nil {
			n.System = l.SolarSystem.Name
		}
		nodes[i] = n
	}
	slices.SortFunc(nodes, func(a, b locationNode) int {
		return cmp.Compare(a.Name, b.Name)
	})
	for _, n := range nodes {
		id := fmt.Sprint(n.ID)
		values[id], err = objectToJSON(n)
		if err != nil {
			return nil, nil, 0, err
		}
		ids[""] = append(ids[""], id)
		// 	for _, i := range l.Implants {
		// 		subID := fmt.Sprintf("%s-%d", id, i.EveType.ID)
		// 		n := locationNode{
		// 			ImplantTypeName:        i.EveType.Name,
		// 			ImplantTypeID:          i.EveType.ID,
		// 			ImplantTypeDescription: i.EveType.DescriptionPlain(),
		// 		}
		// 		values[subID], err = objectToJSON(n)
		// 		if err != nil {
		// 			return nil, nil, 0, err
		// 		}
		// 		ids[id] = append(ids[id], subID)
		// 	}
	}

	return ids, values, len(locations), nil
}

func (a *assetsArea) makeTopText(total int) (string, widget.Importance, error) {
	if !a.ui.hasCharacter() {
		return "No character", widget.LowImportance, nil
	}
	hasData, err := a.ui.service.CharacterSectionWasUpdated(a.ui.currentCharID(), model.CharacterSectionJumpClones)
	if err != nil {
		return "", 0, err
	}
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	return fmt.Sprintf("%d locations", total), widget.MediumImportance, nil
}
