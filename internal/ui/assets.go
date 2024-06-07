package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/widgets"
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

var defaultAssetIcon = theme.NewDisabledResource(resourceQuestionmarkSvg)

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
	content       fyne.CanvasObject
	assets        *widget.GridWrap
	assetsData    binding.UntypedList
	assetsTop     *widget.Label
	locations     *widget.Tree
	locationsData binding.StringTree
	locationsTop  *widget.Label
	ui            *ui
}

func (u *ui) newAssetsArea() *assetsArea {
	a := assetsArea{
		assetsData:    binding.NewUntypedList(),
		assetsTop:     widget.NewLabel(""),
		locationsData: binding.NewStringTree(),
		locationsTop:  widget.NewLabel(""),
		ui:            u,
	}
	a.locationsTop.TextStyle.Bold = true
	a.locations = a.makeLocationsTree()
	locations := container.NewBorder(container.NewVBox(a.locationsTop, widget.NewSeparator()), nil, nil, nil, a.locations)

	a.assetsTop.TextStyle.Bold = true
	a.assets = u.makeAssetGrid(a.assetsData)
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
			return container.NewHBox(
				widget.NewLabel("Template"),
				widgets.NewTappableIcon(theme.InfoIcon(), func() {}))
		},
		func(di binding.DataItem, branch bool, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			label := row.Objects[0].(*widget.Label)
			icon := row.Objects[1].(*widgets.TappableIcon)
			n, err := treeNodeFromDataItem[locationNode](di)
			if err != nil {
				slog.Error("Failed to render asset location in UI", "err", err)
				label.SetText("ERROR")
				return
			}
			label.SetText(n.Name)
			if n.isBranch() {
				icon.OnTapped = func() {
					a.ui.showLocationInfoWindow(n.LocationID)
				}
				icon.Show()
			} else {
				icon.Hide()
			}
		},
	)
	t.OnSelected = func(uid widget.TreeNodeID) {
		defer t.UnselectAll()
		n, err := treeNodeFromBoundTree[locationNode](a.locationsData, uid)
		if err != nil {
			slog.Error("Failed to select location", "err", err)
			return
		}
		if n.isBranch() {
			t.ToggleBranch(uid)
			return
		}
		if err := a.redrawAssets(n); err != nil {
			slog.Warn("Failed to redraw assets", "err", err)
		}
	}
	return t
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
	characterID := a.ui.characterID()
	locations, err := a.ui.sv.Characters.ListCharacterAssetLocations(context.Background(), characterID)
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
	hasData, err := a.ui.sv.Characters.CharacterSectionWasUpdated(
		context.Background(), a.ui.characterID(), model.CharacterSectionAssets)
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
	var f func(context.Context, int32, int64) ([]*model.CharacterAsset, error)
	switch n.Type {
	case nodeShipHangar:
		f = a.ui.sv.Characters.ListCharacterAssetsInShipHangar
	case nodeItemHangar:
		f = a.ui.sv.Characters.ListCharacterAssetsInItemHangar
	default:
		return fmt.Errorf("invalid node type: %v", n.Type)
	}
	assets, err := f(context.Background(), n.CharacterID, n.LocationID)
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

func (u *ui) showNewAssetWindow(ca *model.CharacterAsset) {
	w := u.app.NewWindow(fmt.Sprintf("%s \"%s\" (%s): Contents", ca.EveType.Name, ca.Name, ca.EveType.Group.Name))
	oo, err := u.sv.Characters.ListCharacterAssetsInLocation(context.Background(), ca.CharacterID, ca.ItemID)
	if err != nil {
		panic(err)
	}
	data := binding.NewUntypedList()
	if err := data.Set(copyToUntypedSlice(oo)); err != nil {
		panic(err)
	}
	top := widget.NewLabel(fmt.Sprintf("%s items", humanize.Commaf(float64(len(oo)))))
	top.TextStyle.Bold = true
	assets := u.makeAssetGrid(data)
	content := container.NewBorder(container.NewVBox(top, widget.NewSeparator()), nil, nil, nil, assets)

	w.SetContent(content)
	w.Resize(fyne.Size{Width: 650, Height: 500})
	w.Show()
}

func (u *ui) makeAssetGrid(assetsData binding.UntypedList) *widget.GridWrap {
	g := widget.NewGridWrapWithData(
		assetsData,
		func() fyne.CanvasObject {
			return NewAssetListWidget(u.sv.EveImage, defaultAssetIcon)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			ca, err := convertDataItem[*model.CharacterAsset](di)
			if err != nil {
				panic(err)
			}
			item := co.(*AssetListWidget)
			item.SetAsset(ca.DisplayName(), ca.Quantity, ca.IsSingleton, ca.EveType.ID, ca.Variant())
		},
	)
	g.OnSelected = func(id widget.GridWrapItemID) {
		ca, err := getItemUntypedList[*model.CharacterAsset](assetsData, id)
		if err != nil {
			slog.Error("failed to access assets in grid", "err", err)
			return
		}
		if ca.IsContainer() {
			u.showNewAssetWindow(ca)
		} else {
			u.showTypeInfoWindow(ca.EveType.ID, u.characterID())
		}
		g.UnselectAll()
	}
	return g
}
