package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type characterImplantsNode struct {
	implantCount           int
	implantTypeID          int32
	implantTypeName        string
	implantTypeDescription string
	characterID            int32
	characterName          string
}

func (n characterImplantsNode) IsRoot() bool {
	return n.implantTypeID == 0
}

func (n characterImplantsNode) UID() widget.TreeNodeID {
	if n.characterID == 0 {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%d", n.characterID, n.implantTypeID)
}

type augmentations struct {
	widget.BaseWidget

	top  *widget.Label
	tree *iwidget.Tree[characterImplantsNode]
	u    *baseUI
}

func newAugmentations(u *baseUI) *augmentations {
	top := widget.NewLabel("")
	top.Wrapping = fyne.TextWrapWord
	a := &augmentations{
		top: top,
		u:   u,
	}
	a.ExtendBaseWidget(a)
	a.tree = a.makeTree()
	return a
}

func (a *augmentations) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.tree)
	return widget.NewSimpleRenderer(c)
}

func (a *augmentations) makeTree() *iwidget.Tree[characterImplantsNode] {
	t := iwidget.NewTree(
		func(branch bool) fyne.CanvasObject {
			iconMain := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
			main := ttwidget.NewLabel("Template")
			total := widget.NewLabel("9")
			return container.NewBorder(
				nil,
				nil,
				iconMain,
				nil,
				container.NewHBox(main, total),
			)
		},
		func(n characterImplantsNode, b bool, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			hbox := border[0].(*fyne.Container).Objects
			main := hbox[0].(*ttwidget.Label)
			total := hbox[1].(*widget.Label)
			iconMain := border[1].(*canvas.Image)
			if n.IsRoot() {
				go a.u.updateCharacterAvatar(n.characterID, func(r fyne.Resource) {
					fyne.Do(func() {
						iconMain.Resource = r
						iconMain.Refresh()
					})
				})
				main.SetText(n.characterName)
				main.SetToolTip("")
				if n.implantCount > 0 {
					total.SetText(fmt.Sprint(n.implantCount))
					total.Show()
				} else {
					total.Hide()
				}
			} else {
				iwidget.RefreshImageAsync(iconMain, func() (fyne.Resource, error) {
					return a.u.eis.InventoryTypeIcon(n.implantTypeID, app.IconPixelSize)
				})
				main.SetText(n.implantTypeName)
				main.SetToolTip(n.implantTypeDescription)
				total.Hide()
			}
		},
	)
	t.OnSelectedNode = func(n characterImplantsNode) {
		defer t.UnselectAll()
		if n.IsRoot() {
			a.u.ShowInfoWindow(app.EveEntityCharacter, n.characterID)
		} else {
			a.u.ShowTypeInfoWindowWithCharacter(n.implantTypeID, n.characterID)
		}
	}
	return t
}

func (a *augmentations) update() {
	td, err := a.updateTreeData()
	if err != nil {
		slog.Error("Failed to refresh augmentations UI", "err", err)
		fyne.Do(func() {
			a.top.Text = "ERROR: " + a.u.humanizeError(err)
			a.top.Importance = widget.DangerImportance
			a.top.Refresh()
			a.top.Show()
		})
		return
	}
	fyne.Do(func() {
		a.tree.Set(td)
		a.top.Hide()
	})
}

func (a *augmentations) updateTreeData() (iwidget.TreeNodes[characterImplantsNode], error) {
	var tree iwidget.TreeNodes[characterImplantsNode]
	ctx := context.Background()
	characters, err := a.u.cs.ListCharactersShort(ctx)
	if err != nil {
		return tree, err
	}
	m := make(map[int32][]*app.CharacterImplant)
	for _, c := range characters {
		m[c.ID] = make([]*app.CharacterImplant, 0)
	}
	oo, err := a.u.cs.ListAllImplants(ctx)
	if err != nil {
		return tree, err
	}
	for _, o := range oo {
		_, ok := m[o.CharacterID]
		if !ok {
			slog.Warn("Unknown character for implant", "characterID", o.CharacterID)
			continue
		}
		m[o.CharacterID] = append(m[o.CharacterID], o)
	}
	for k := range m {
		slices.SortFunc(m[k], func(a, b *app.CharacterImplant) int {
			return cmp.Compare(a.SlotNum, b.SlotNum)
		})
	}
	for _, c := range characters {
		n := characterImplantsNode{
			characterID:   c.ID,
			characterName: c.Name,
			implantCount:  len(m[c.ID]),
		}
		uid := tree.MustAdd(iwidget.RootUID, n)
		for _, o := range m[c.ID] {
			n := characterImplantsNode{
				implantTypeDescription: o.EveType.DescriptionPlain(),
				implantTypeID:          o.EveType.ID,
				implantTypeName:        o.EveType.Name,
				characterID:            c.ID,
			}
			tree.MustAdd(uid, n)
		}
	}
	return tree, err
}
