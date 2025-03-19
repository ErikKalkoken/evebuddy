package ui

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type jumpCloneNode struct {
	implantCount           int
	implantTypeID          int32
	implantTypeName        string
	implantTypeDescription string
	isUnknown              bool
	jumpCloneID            int32
	jumpCloneName          string
	locationID             int64
	locationName           string
	systemSecurityValue    float32
	systemSecurityType     app.SolarSystemSecurityType
}

func (n jumpCloneNode) IsRoot() bool {
	return n.implantTypeID == 0
}

func (n jumpCloneNode) UID() widget.TreeNodeID {
	if n.jumpCloneID == 0 {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%d", n.jumpCloneID, n.implantTypeID)
}

type CharacterJumpClones struct {
	widget.BaseWidget

	OnReDraw func(clonesCount int)

	top  *widget.RichText
	tree *iwidget.Tree[jumpCloneNode]
	u    *BaseUI
}

func NewCharacterJumpClones(u *BaseUI) *CharacterJumpClones {
	ntop := widget.NewRichText()
	ntop.Wrapping = fyne.TextWrapWord
	a := &CharacterJumpClones{
		top: ntop,
		u:   u,
	}
	a.ExtendBaseWidget(a)
	a.tree = a.makeTree()
	return a
}

func (a *CharacterJumpClones) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewVBox(a.top, widget.NewSeparator())
	c := container.NewBorder(top, nil, nil, nil, a.tree)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterJumpClones) makeTree() *iwidget.Tree[jumpCloneNode] {
	t := iwidget.NewTree(
		func(branch bool) fyne.CanvasObject {
			iconMain := iwidget.NewImageFromResource(
				icons.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(app.IconUnitSize),
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
		func(n jumpCloneNode, b bool, co fyne.CanvasObject) {
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
				if !n.isUnknown {
					iconInfo.OnTapped = func() {
						a.u.ShowLocationInfoWindow(n.locationID)
					}
					iconInfo.Show()
				} else {
					iconInfo.Hide()
				}
				main.SetText(n.locationName)
				if !n.isUnknown {
					prefix.Text = fmt.Sprintf("%.1f", n.systemSecurityValue)
					prefix.Importance = n.systemSecurityType.ToImportance()
				} else {
					prefix.Text = "?"
					prefix.Importance = widget.LowImportance
				}
				prefix.Show()
				spacer.Show()
			} else {
				appwidget.RefreshImageResourceAsync(iconMain, func() (fyne.Resource, error) {
					return a.u.EveImageService().InventoryTypeIcon(n.implantTypeID, app.IconPixelSize)
				})
				main.SetText(n.implantTypeName)
				iconInfo.OnTapped = func() {
					a.u.ShowTypeInfoWindow(n.implantTypeID)
				}
				prefix.Hide()
				spacer.Hide()
			}
		},
	)
	t.OnSelected = func(n jumpCloneNode) {
		defer t.UnselectAll()
	}
	return t
}

func (a *CharacterJumpClones) Update() {
	tree, err := a.newTreeData()
	if err != nil {
		slog.Error("Failed to refresh jump clones UI", "err", err)
		iwidget.SetRichText(a.top, widget.TextSegment{
			Text: "ERROR: " + ihumanize.Error(err),
			Style: widget.RichTextStyle{
				ColorName: theme.ColorNameError,
				TextStyle: fyne.TextStyle{Bold: true},
			}})

	} else {
		a.RefreshTop()
	}
	a.tree.Set(tree)
	if a.OnReDraw != nil {
		a.OnReDraw(a.ClonesCount())
	}
}

func (a *CharacterJumpClones) newTreeData() (*iwidget.TreeData[jumpCloneNode], error) {
	tree := iwidget.NewTreeData[jumpCloneNode]()
	if !a.u.HasCharacter() {
		return tree, nil
	}
	ctx := context.Background()
	clones, err := a.u.CharacterService().ListCharacterJumpClones(ctx, a.u.CurrentCharacterID())
	if err != nil {
		return tree, err
	}
	for _, c := range clones {
		n := jumpCloneNode{
			implantCount:  len(c.Implants),
			jumpCloneID:   c.JumpCloneID,
			jumpCloneName: c.Name,
			locationID:    c.Location.ID,
		}
		// TODO: Refactor to use same location method for all unknown location cases
		if c.Location != nil {
			loc, err := a.u.EveUniverseService().GetLocation(ctx, c.Location.ID)
			if err != nil {
				slog.Error("get location for jump clone", "error", err)
			} else {
				n.locationName = loc.Name
				n.systemSecurityValue = float32(loc.SolarSystem.SecurityStatus)
				n.systemSecurityType = loc.SolarSystem.SecurityType()
			}
		}
		if n.locationName == "" {
			n.locationName = fmt.Sprintf("Unknown location #%d", c.Location.ID)
			n.isUnknown = true
		}
		uid := tree.MustAdd(iwidget.RootUID, n)
		for _, i := range c.Implants {
			n := jumpCloneNode{
				implantTypeDescription: i.EveType.DescriptionPlain(),
				implantTypeID:          i.EveType.ID,
				implantTypeName:        i.EveType.Name,
				jumpCloneID:            c.JumpCloneID,
			}
			tree.MustAdd(uid, n)
		}
	}
	return tree, err
}

func (a *CharacterJumpClones) RefreshTop() {
	boldTextStyle := fyne.TextStyle{Bold: true}
	defaultStyle := widget.RichTextStyle{
		ColorName: theme.ColorNameForeground,
		TextStyle: boldTextStyle,
	}
	defaultStyleInline := defaultStyle
	defaultStyleInline.Inline = true
	s := widget.TextSegment{
		Text:  "",
		Style: defaultStyle,
	}
	c := a.u.CurrentCharacter()
	if c == nil {
		s.Text = "No character"
		s.Style.ColorName = theme.ColorNameDisabled
		iwidget.SetRichText(a.top, s)
		return
	}
	hasData := a.u.StatusCacheService().CharacterSectionExists(c.ID, app.SectionJumpClones)
	if !hasData {
		s.Text = "Waiting for character data to be loaded..."
		s.Style.ColorName = theme.ColorNameWarning
		iwidget.SetRichText(a.top, s)
	}
	var nextJumpColor fyne.ThemeColorName
	var nextJump, lastJump string
	if c.NextCloneJump.IsEmpty() {
		nextJump = "?"
		nextJumpColor = theme.ColorNameForeground
	} else if c.NextCloneJump.MustValue().IsZero() {
		nextJump = "NOW"
		nextJumpColor = theme.ColorNameSuccess
	} else {
		nextJump = ihumanize.Duration(time.Until(c.NextCloneJump.MustValue()))
		nextJumpColor = theme.ColorNameError
	}
	if x := c.LastCloneJumpAt.ValueOrZero(); x.IsZero() {
		lastJump = "?"
	} else {
		lastJump = humanize.Time(x)
	}
	iwidget.SetRichText(
		a.top,
		widget.TextSegment{
			Text:  fmt.Sprintf("%d clones • Next available jump: ", a.ClonesCount()),
			Style: defaultStyleInline,
		},
		widget.TextSegment{
			Text: nextJump,
			Style: widget.RichTextStyle{
				ColorName: nextJumpColor,
				TextStyle: boldTextStyle,
				Inline:    true,
			},
		},
		widget.TextSegment{
			Text:  fmt.Sprintf(" • Last jump: %s", lastJump),
			Style: defaultStyle,
		},
	)
}

func (a *CharacterJumpClones) ClonesCount() int {
	return len(a.tree.Data().ChildUIDs(""))
}

func (a *CharacterJumpClones) StartUpdateTicker() {
	ticker := time.NewTicker(time.Second * 15)
	go func() {
		for {
			a.RefreshTop()
			<-ticker.C
		}
	}()
}
