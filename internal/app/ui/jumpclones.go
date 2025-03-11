package ui

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"

	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
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
	SystemSecurityValue    float32
	SystemSecurityType     app.SolarSystemSecurityType
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
	Content  *fyne.Container
	OnReDraw func(clonesCount int)

	top        *widget.Label
	treeData   *fynetree.FyneTree[jumpCloneNode]
	treeWidget *widget.Tree
	u          *BaseUI
}

func NewJumpClonesArea(u *BaseUI) *JumpClonesArea {
	a := JumpClonesArea{
		top:      makeTopLabel(),
		treeData: fynetree.New[jumpCloneNode](),
		u:        u,
	}
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
			iconMain := iwidget.NewImageFromResource(
				icon.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(DefaultIconUnitSize),
			)
			main := widget.NewLabel("Template")
			main.Truncation = fyne.TextTruncateEllipsis
			iconInfo := kxwidget.NewTappableIcon(theme.InfoIcon(), nil)
			spacer := canvas.NewRectangle(color.Transparent)
			spacer.SetMinSize(fyne.NewSize(40, 10))
			prefix := widget.NewLabel("-9.9")
			return container.NewBorder(
				nil,
				nil,
				container.NewHBox(iconMain, container.NewStack(spacer, prefix)),
				iconInfo,
				main,
			)
		},
		func(uid widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			n, ok := a.treeData.Value(uid)
			if !ok {
				return
			}
			border := co.(*fyne.Container).Objects
			main := border[0].(*widget.Label)
			hbox := border[1].(*fyne.Container).Objects
			iconMain := hbox[0].(*canvas.Image)
			spacer := hbox[1].(*fyne.Container).Objects[0]
			prefix := hbox[1].(*fyne.Container).Objects[1].(*widget.Label)
			iconInfo := border[2].(*kxwidget.TappableIcon)
			if n.IsRoot() {
				iconMain.Resource = eveicon.GetResourceByName(eveicon.CloningCenter)
				iconMain.Refresh()
				if !n.IsUnknown {
					iconInfo.OnTapped = func() {
						a.u.ShowLocationInfoWindow(n.LocationID)
					}
					iconInfo.Show()
				} else {
					iconInfo.Hide()
				}
				main.SetText(n.LocationName)
				if !n.IsUnknown {
					prefix.Text = fmt.Sprintf("%.1f", n.SystemSecurityValue)
					prefix.Importance = n.SystemSecurityType.ToImportance()
				} else {
					prefix.Text = "?"
					prefix.Importance = widget.LowImportance
				}
				prefix.Show()
				spacer.Show()
			} else {
				appwidget.RefreshImageResourceAsync(iconMain, func() (fyne.Resource, error) {
					return a.u.EveImageService.InventoryTypeIcon(n.ImplantTypeID, DefaultIconPixelSize)
				})
				main.SetText(n.ImplantTypeName)
				iconInfo.OnTapped = func() {
					a.u.ShowTypeInfoWindow(n.ImplantTypeID, a.u.CharacterID(), DescriptionTab)
				}
				prefix.Hide()
				spacer.Hide()
			}
		},
	)
	t.OnSelected = func(uid widget.TreeNodeID) {
		defer t.UnselectAll()
	}
	return t
}

func (a *JumpClonesArea) Redraw() {
	var t string
	var i widget.Importance
	var clonesCount int
	tree, err := a.newTreeData()
	if err != nil {
		slog.Error("Failed to refresh jump clones UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		clonesCount = len(tree.ChildUIDs(""))
		t, i = a.makeTopText(clonesCount)
	}
	a.treeData = tree
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.treeWidget.Refresh()
	if a.OnReDraw != nil {
		a.OnReDraw(clonesCount)
	}
}

func (a *JumpClonesArea) newTreeData() (*fynetree.FyneTree[jumpCloneNode], error) {
	tree := fynetree.New[jumpCloneNode]()
	if !a.u.HasCharacter() {
		return tree, nil
	}
	ctx := context.Background()
	clones, err := a.u.CharacterService.ListCharacterJumpClones(ctx, a.u.CharacterID())
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
		if c.Location != nil {
			loc, err := a.u.EveUniverseService.GetEveLocation(ctx, c.Location.ID)
			if err != nil {
				slog.Error("get location for jump clone", "error", err)
			} else {
				n.LocationName = loc.Name
				n.SystemSecurityValue = float32(loc.SolarSystem.SecurityStatus)
				n.SystemSecurityType = loc.SolarSystem.SecurityType()
			}
		}
		if n.LocationName == "" {
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
