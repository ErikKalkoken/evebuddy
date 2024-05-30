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
)

type locationNode struct {
	CharacterID int32
	LocationID  int64
	Name        string
	System      string
}

// func (n locationNode) isBranch() bool {
// 	return n.ImplantTypeID == 0 && n.ImplantCount > 0
// }

// assetsArea is the UI area that shows the skillqueue
type assetsArea struct {
	content          fyne.CanvasObject
	defaultAssetIcon fyne.Resource
	assets           *widget.GridWrap
	assetsData       binding.UntypedList
	locations        *widget.Tree
	locationsData    binding.StringTree
	top              *widget.Label
	ui               *ui
}

func (u *ui) newAssetsArea() *assetsArea {
	a := assetsArea{
		top:              widget.NewLabel(""),
		defaultAssetIcon: theme.NewThemedResource(resourceQuestionmark64dpSvg),
		assetsData:       binding.NewUntypedList(),
		locationsData:    binding.NewStringTree(),
		ui:               u,
	}
	a.top.TextStyle.Bold = true

	a.locations = widget.NewTreeWithData(
		a.locationsData,
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
	a.locations.OnSelected = func(uid widget.TreeNodeID) {
		err := func() error {
			n, err := fetchTreeNode[locationNode](a.locationsData, uid)
			if err != nil {
				return err
			}
			return a.updateAssetData(n.CharacterID, n.LocationID)
		}()
		if err != nil {
			t := "Failed to select location"
			slog.Error(t, "err", err)
			a.ui.statusBarArea.SetError(t)
		}
		// if n.isBranch() {
		// 	a.tree.ToggleBranch(uid)
		// }
		// if n.isClone() {
		// 	a.tree.UnselectAll()
		// 	return
		// }
	}

	a.assets = widget.NewGridWrapWithData(
		a.assetsData,
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromResource(a.defaultAssetIcon)
			icon.FillMode = canvas.ImageFillContain
			icon.SetMinSize(fyne.Size{Width: 70, Height: 70})
			name := widget.NewLabel("First Line\nSecond Line")
			name.Wrapping = fyne.TextWrapBreak
			return container.NewBorder(icon, nil, nil, nil, name)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			box := co.(*fyne.Container)
			icon := box.Objects[1].(*canvas.Image)
			name := box.Objects[0].(*widget.Label)
			o, err := convertDataItem[*model.CharacterAsset](di)
			if err != nil {
				panic(err)
			}
			icon.Resource = a.defaultAssetIcon
			icon.Refresh()
			refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
				return a.ui.imageManager.InventoryTypeIcon(o.EveType.ID, 64)
			})
			name.SetText(o.EveType.Name)
		},
	)
	top := container.NewVBox(a.top, widget.NewSeparator())
	main := container.NewHSplit(a.locations, a.assets)
	a.content = container.NewBorder(top, nil, nil, nil, main)
	return &a
}

func (a *assetsArea) redraw() {
	t, i, err := func() (string, widget.Importance, error) {
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
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
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
		n := locationNode{CharacterID: characterID, LocationID: l.ID}
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
		id := fmt.Sprint(n.LocationID)
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

func (a *assetsArea) updateAssetData(characterID int32, locationID int64) error {
	empty := make([]*model.CharacterAsset, 0)
	if err := a.assetsData.Set(copyToUntypedSlice(empty)); err != nil {
		return err
	}
	assets, err := a.ui.service.ListCharacterAssetsAtLocation(characterID, locationID)
	if err != nil {
		return err
	}
	return a.assetsData.Set(copyToUntypedSlice(assets))
}
