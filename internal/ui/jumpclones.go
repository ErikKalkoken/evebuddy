package ui

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

const jumpClonesUpdateTicker = 10 * time.Second

type jumpCloneNode struct {
	Name                   string
	Region                 string
	ImplantCount           int
	ImplantTypeID          int32
	ImplantTypeName        string
	ImplantTypeDescription string
}

func newJumpCloneNodeFromJSON(s string) jumpCloneNode {
	var n jumpCloneNode
	err := json.Unmarshal([]byte(s), &n)
	if err != nil {
		panic(err)
	}
	return n
}

func (n jumpCloneNode) toJSON() string {
	s, err := json.Marshal(n)
	if err != nil {
		panic(err)
	}
	return string(s)
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

	a.tree = widget.NewTreeWithData(
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
			v, err := di.(binding.String).Get()
			if err != nil {
				panic(err)
			}
			n := newJumpCloneNodeFromJSON(v)
			hbox := co.(*fyne.Container)
			icon := hbox.Objects[0].(*canvas.Image)
			first := hbox.Objects[1].(*widget.Label)
			second := hbox.Objects[2].(*widget.Label)
			third := hbox.Objects[3].(*widget.Label)
			if n.isClone() {
				icon.Resource = resourceClone64Png
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
					return a.ui.imageManager.InventoryTypeIcon(n.ImplantTypeID, defaultIconSize)
				})
				first.SetText(n.ImplantTypeName)
				second.Hide()
				third.Hide()
			}
		},
	)
	a.tree.OnSelected = func(uid widget.TreeNodeID) {
		v, err := a.treeData.GetValue(uid)
		if err != nil {
			slog.Error("Failed to get tree data item", "error", err)
			return
		}
		n := newJumpCloneNodeFromJSON(v)
		if n.isBranch() {
			a.tree.ToggleBranch(uid)
		}
		if n.isClone() {
			a.tree.UnselectAll()
			return
		}
		d := makeImplantDetailDialog(n.ImplantTypeName, n.ImplantTypeDescription, a.ui.window)
		d.SetOnClosed(func() {
			a.tree.UnselectAll()
		})
		d.Show()
	}

	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, container.NewVScroll(a.tree))
	return &a
}

func (a *jumpClonesArea) Redraw() {
	ids, values, total, err := a.updateTreeData()
	if err != nil {
		panic(err)
	}
	if err := a.treeData.Set(ids, values); err != nil {
		panic(err)
	}
	s, i := a.makeTopText(total)
	a.top.Text = s
	a.top.Importance = i
	a.top.Refresh()
}

func (a *jumpClonesArea) updateTreeData() (map[string][]string, map[string]string, int, error) {
	values := make(map[string]string)
	ids := make(map[string][]string)
	if !a.ui.HasCharacter() {
		return ids, values, 0, nil
	}
	clones, err := a.ui.service.ListCharacterJumpClones(a.ui.CurrentCharID())
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
		values[id] = n.toJSON()
		ids[""] = append(ids[""], id)
		for _, i := range c.Implants {
			subID := fmt.Sprintf("%s-%d", id, i.EveType.ID)
			n := jumpCloneNode{
				ImplantTypeName:        i.EveType.Name,
				ImplantTypeID:          i.EveType.ID,
				ImplantTypeDescription: i.EveType.DescriptionPlain(),
			}
			values[subID] = n.toJSON()
			ids[id] = append(ids[id], subID)
		}
	}
	return ids, values, len(clones), nil
}

func (a *jumpClonesArea) makeTopText(total int) (string, widget.Importance) {
	hasData, err := a.ui.service.CharacterSectionWasUpdated(a.ui.CurrentCharID(), model.CharacterSectionJumpClones)
	if err != nil {
		return "ERROR", widget.DangerImportance
	}
	if !hasData {
		return "No data", widget.LowImportance
	}
	return fmt.Sprintf("%d clones", total), widget.MediumImportance
}

func (a *jumpClonesArea) StartUpdateTicker() {
	ticker := time.NewTicker(jumpClonesUpdateTicker)
	go func() {
		for {
			func() {
				cc, err := a.ui.service.ListCharactersShort()
				if err != nil {
					slog.Error("Failed to fetch list of characters", "err", err)
					return
				}
				for _, c := range cc {
					a.MaybeUpdateAndRefresh(c.ID)
				}
			}()
			<-ticker.C
		}
	}()
}

func (a *jumpClonesArea) MaybeUpdateAndRefresh(characterID int32) {
	changed, err := a.ui.service.UpdateCharacterSectionIfExpired(characterID, model.CharacterSectionJumpClones)
	if err != nil {
		slog.Error("Failed to update jump clones", "character", characterID, "err", err)
		return
	}
	if changed && characterID == a.ui.CurrentCharID() {
		a.Redraw()
	}
}
