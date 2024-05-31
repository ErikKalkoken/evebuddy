package ui

import (
	"cmp"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/dustin/go-humanize"
)

type locationNodeType uint

const (
	nodeBranch locationNodeType = iota + 1
	nodeShipHangar
	nodeItemHangar
)

type locationNode struct {
	CharacterID int32
	LocationID  int64
	Name        string
	System      string
	Type        locationNodeType
}

func (n locationNode) UID() widget.TreeNodeID {
	if n.CharacterID == 0 || n.LocationID == 0 || n.Type == 0 {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%d-%d", n.CharacterID, n.LocationID, n.Type)
}

func (n locationNode) isBranch() bool {
	return n.Type == nodeBranch
}

// assetsArea is the UI area that shows the skillqueue
type assetsArea struct {
	content          fyne.CanvasObject
	defaultAssetIcon fyne.Resource
	assets           *widget.GridWrap
	assetsData       binding.UntypedList
	assetsTop        *widget.Label
	locations        *widget.Tree
	locationsData    binding.StringTree
	locationsTop     *widget.Label
	ui               *ui
}

func (u *ui) newAssetsArea() *assetsArea {
	a := assetsArea{
		assetsData:       binding.NewUntypedList(),
		assetsTop:        widget.NewLabel(""),
		defaultAssetIcon: theme.NewDisabledResource(resourceQuestionmarkSvg),
		locationsData:    binding.NewStringTree(),
		locationsTop:     widget.NewLabel(""),
		ui:               u,
	}
	a.locationsTop.TextStyle.Bold = true
	a.locations = a.makeLocationsTree()
	locations := container.NewBorder(container.NewVBox(a.locationsTop, widget.NewSeparator()), nil, nil, nil, a.locations)

	a.assetsTop.TextStyle.Bold = true
	a.assets = a.makeAssetGrid()
	assets := container.NewBorder(container.NewVBox(a.assetsTop, widget.NewSeparator()), nil, nil, nil, a.assets)

	main := container.NewHSplit(locations, assets)
	main.SetOffset(0.33)
	a.content = main
	return &a
}

func (a *assetsArea) makeLocationsTree() *widget.Tree {
	t := widget.NewTreeWithData(
		a.locationsData,
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(di binding.DataItem, branch bool, co fyne.CanvasObject) {
			label := co.(*widget.Label)
			n, err := treeNodeFromDataItem[locationNode](di)
			if err != nil {
				slog.Error("Failed to render asset location in UI", "err", err)
				label.SetText("ERROR")
				return
			}
			label.SetText(n.Name)
		},
	)
	t.OnSelected = func(uid widget.TreeNodeID) {
		err := func() error {
			n, err := treeNodeFromBoundTree[locationNode](a.locationsData, uid)
			if err != nil {
				return err
			}
			if n.isBranch() {
				t.ToggleBranch(uid)
				t.UnselectAll()
			} else {
				return a.redrawAssets(n)
			}
			return nil
		}()
		if err != nil {
			t := "Failed to select location"
			slog.Error(t, "err", err)
			a.ui.statusBarArea.SetError(t)
		}
	}
	return t
}

func (a *assetsArea) makeAssetGrid() *widget.GridWrap {
	g := widget.NewGridWrapWithData(
		a.assetsData,
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromResource(a.defaultAssetIcon)
			icon.FillMode = canvas.ImageFillContain
			icon.SetMinSize(fyne.Size{Width: 40, Height: 40})
			name := widget.NewLabel("Asset Template Name XXX")
			quantity := widget.NewLabel("")
			return container.NewBorder(nil, nil, icon, quantity, name)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			box := co.(*fyne.Container)
			name := box.Objects[0].(*widget.Label)
			icon := box.Objects[1].(*canvas.Image)
			quantity := box.Objects[2].(*widget.Label)
			o, err := convertDataItem[*model.CharacterAsset](di)
			if err != nil {
				panic(err)
			}
			icon.Resource = a.defaultAssetIcon
			icon.Refresh()
			refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
				if o.IsSKIN() {
					return resourceSkinicon64pxPng, nil
				} else if o.IsBPO() {
					return a.ui.imageManager.InventoryTypeBPO(o.EveType.ID, 64)
				} else if o.IsBlueprintCopy {
					return a.ui.imageManager.InventoryTypeBPC(o.EveType.ID, 64)
				} else {
					return a.ui.imageManager.InventoryTypeIcon(o.EveType.ID, 64)
				}
			})
			var t string
			if o.Name != "" {
				t = o.Name
			} else {
				t = o.EveType.Name
			}
			if !o.IsSingleton {
				quantity.SetText(humanize.Comma(int64(o.Quantity)))
				quantity.Show()
			} else {
				quantity.Hide()
			}
			name.Wrapping = fyne.TextWrapWord
			name.Text = t
			name.Refresh()
		},
	)
	return g
}

func (a *assetsArea) redraw() {
	a.locations.CloseAllBranches()
	a.locations.ScrollToTop()
	t, i, err := func() (string, widget.Importance, error) {
		if err := a.clearAssets(); err != nil {
			return "", 0, err
		}
		ids, values, total, err := a.updateLocationData()
		if err != nil {
			return "", 0, err
		}
		if err := a.locationsData.Set(ids, values); err != nil {
			return "", 0, err
		}
		return a.makeTopText(total)
	}()
	if err != nil {
		slog.Error("Failed to redraw asset locations UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.locationsTop.Text = t
	a.locationsTop.Importance = i
	a.locationsTop.Refresh()
}

func (a *assetsArea) updateLocationData() (map[string][]string, map[string]string, int, error) {
	values := make(map[string]string)
	ids := make(map[string][]string)
	if !a.ui.hasCharacter() {
		return ids, values, 0, nil
	}
	characterID := a.ui.currentCharID()
	locations, err := a.ui.service.ListCharacterAssetLocations(characterID)
	if err != nil {
		return nil, nil, 0, err
	}
	nodes := make([]locationNode, len(locations))
	for i, l := range locations {
		n := locationNode{CharacterID: characterID, LocationID: l.ID, Type: nodeBranch}
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
	for _, ln := range nodes {
		uid := ln.UID()
		values[uid], err = objectToJSON(ln)
		if err != nil {
			return nil, nil, 0, err
		}
		ids[""] = append(ids[""], uid)
		for _, t := range []locationNodeType{nodeShipHangar, nodeItemHangar} {
			hn := locationNode{
				CharacterID: ln.CharacterID,
				LocationID:  ln.LocationID,
				Type:        t,
			}
			switch t {
			case nodeShipHangar:
				hn.Name = "Ship Hangar"
			case nodeItemHangar:
				hn.Name = "Item Hangar"
			}
			subUID := hn.UID()
			values[subUID], err = objectToJSON(hn)
			if err != nil {
				return nil, nil, 0, err
			}
			ids[uid] = append(ids[uid], subUID)
		}
	}
	return ids, values, len(locations), nil
}

func (a *assetsArea) makeTopText(total int) (string, widget.Importance, error) {
	if !a.ui.hasCharacter() {
		return "No character", widget.LowImportance, nil
	}
	hasData, err := a.ui.service.CharacterSectionWasUpdated(a.ui.currentCharID(), model.CharacterSectionAssets)
	if err != nil {
		return "", 0, err
	}
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	return fmt.Sprintf("%d locations", total), widget.MediumImportance, nil
}

func (a *assetsArea) redrawAssets(n locationNode) error {
	empty := make([]*model.CharacterAsset, 0)
	if err := a.assetsData.Set(copyToUntypedSlice(empty)); err != nil {
		return err
	}
	var f func(int32, int64) ([]*model.CharacterAsset, error)
	switch n.Type {
	case nodeShipHangar:
		f = a.ui.service.ListCharacterAssetsInShipHangar
	case nodeItemHangar:
		f = a.ui.service.ListCharacterAssetsInItemHangar
	default:
		return fmt.Errorf("invalid node type: %v", n.Type)
	}
	assets, err := f(n.CharacterID, n.LocationID)
	if err != nil {
		return err
	}
	if err := a.assetsData.Set(copyToUntypedSlice(assets)); err != nil {
		return err
	}
	a.assetsTop.SetText(fmt.Sprintf("%d items", len(assets)))
	return nil
}

func (a *assetsArea) clearAssets() error {
	empty := make([]*model.CharacterAsset, 0)
	if err := a.assetsData.Set(copyToUntypedSlice(empty)); err != nil {
		return err
	}
	a.assetsTop.SetText("")
	return nil
}
