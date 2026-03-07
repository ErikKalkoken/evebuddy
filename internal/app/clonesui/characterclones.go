package clonesui

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/uiservices"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type hasCharacterSection interface {
	HasCharacterSection(characterID int64, section app.CharacterSection) bool
}

type characterCloneNode struct {
	characterID            int64
	implantCount           int
	implantTypeDescription string
	implantTypeID          int64
	implantTypeName        string
	isUnknown              bool
	jumpCloneID            int64
	jumpCloneName          string
	locationID             int64
	locationName           string
	systemSecurityType     app.SolarSystemSecurityType
	systemSecurityValue    float32
}

func (n characterCloneNode) isTop() bool {
	return n.implantTypeID == 0
}

func (n characterCloneNode) UID() widget.TreeNodeID {
	if n.jumpCloneID == 0 {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%d", n.jumpCloneID, n.implantTypeID)
}

type CharacterClones struct {
	widget.BaseWidget

	character atomic.Pointer[app.Character]
	top       *xwidget.RichText
	tree      *xwidget.Tree[characterCloneNode]
	u         uiservices.UIServices
}

func NewCharacterClones(u uiservices.UIServices) *CharacterClones {
	top := xwidget.NewRichText()
	top.Wrapping = fyne.TextWrapWord
	a := &CharacterClones{
		top: top,
		u:   u,
	}
	a.ExtendBaseWidget(a)
	a.tree = a.makeTree()

	// Signals
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDorZero() != arg.CharacterID {
			return
		}
		if arg.Section == app.SectionCharacterJumpClones {
			a.update(ctx)
		}
	})
	a.u.Signals().RefreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			n := a.tree.Data().ChildrenCount(nil)
			a.refreshTop(n)
		})
	})
	return a
}

func (a *CharacterClones) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.tree)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterClones) makeTree() *xwidget.Tree[characterCloneNode] {
	t := xwidget.NewTree(
		func(_ bool) fyne.CanvasObject {
			return newCharacterJumpCloneItem(
				app.IsMobile(),
				a.u.EVEImage().InventoryTypeIconAsync,
				a.u.InfoWindow().ShowTypeWithCharacter,
				a.u.InfoWindow().ShowLocation,
			)
		},
		func(n *characterCloneNode, _ bool, co fyne.CanvasObject) {
			co.(*characterJumpCloneItem).set(n)
		},
	)
	t.OnSelectedNode = func(n *characterCloneNode) {
		defer t.UnselectAll()
		if n.isTop() {
			t.ToggleBranchNode(n)
		}
	}
	return t
}

func (a *CharacterClones) update(ctx context.Context) {
	characterID := a.character.Load().IDorZero()
	if characterID == 0 {
		fyne.Do(func() {
			a.top.SetWithText("No character", widget.RichTextStyle{
				ColorName: theme.ColorNameDisabled,
			})
			a.tree.Clear()
		})
		return
	}

	td, err := a.fetchData(ctx, characterID)
	if err != nil {
		slog.Error("Failed to refresh jump clones UI", "err", err)
		fyne.Do(func() {
			a.top.Set(xwidget.RichTextSegmentsFromText("ERROR: "+app.ErrorDisplay(err), widget.RichTextStyle{
				ColorName: theme.ColorNameError,
			}))
		})
		return
	}

	fyne.Do(func() {
		n := td.ChildrenCount(nil)
		a.refreshTop(n)
		a.tree.Set(td)
	})
}

func (a *CharacterClones) fetchData(ctx context.Context, characterID int64) (xwidget.TreeData[characterCloneNode], error) {
	var td xwidget.TreeData[characterCloneNode]
	clones, err := a.u.Character().ListJumpClones(ctx, characterID)
	if err != nil {
		return td, err
	}
	for _, c := range clones {
		clone := &characterCloneNode{
			characterID:   characterID,
			implantCount:  len(c.Implants),
			jumpCloneID:   c.CloneID,
			jumpCloneName: c.Name.ValueOrZero(),
			locationID:    c.Location.ID,
		}
		// TODO: Refactor to use same location method for all unknown location cases
		// if v, ok := c.Location
		if v, ok := c.Location.Name.Value(); ok {
			clone.locationName = v
		} else {
			clone.locationName = fmt.Sprintf("Unknown location #%d", c.Location.ID)
			clone.isUnknown = true
		}
		if v, ok := c.Location.SecurityStatus.Value(); ok {
			clone.systemSecurityValue = v
			clone.systemSecurityType = app.NewSolarSystemSecurityTypeFromValue(v)
		}
		err := td.Add(nil, clone, len(c.Implants) > 0)
		if err != nil {
			return td, err
		}
		for _, i := range c.Implants {
			implant := &characterCloneNode{
				characterID:            characterID,
				implantTypeDescription: i.EveType.DescriptionPlain(),
				implantTypeID:          i.EveType.ID,
				implantTypeName:        i.EveType.Name,
				jumpCloneID:            c.CloneID,
			}
			err := td.Add(clone, implant, false)
			if err != nil {
				return td, err
			}
		}
	}
	return td, err
}

func (a *CharacterClones) refreshTop(cloneCount int) {
	segs := a.makeTopText(cloneCount, a.character.Load(), a.u.StatusCache())
	a.top.Set(segs)
}

func (*CharacterClones) makeTopText(cloneCount int, c *app.Character, s hasCharacterSection) []widget.RichTextSegment {
	defaultStyle := widget.RichTextStyle{
		ColorName: theme.ColorNameForeground,
	}
	ts := &widget.TextSegment{
		Text:  "",
		Style: defaultStyle,
	}
	if c == nil {
		ts.Text = "No character"
		ts.Style.ColorName = theme.ColorNameDisabled
		return []widget.RichTextSegment{ts}
	}
	hasData := s.HasCharacterSection(c.ID, app.SectionCharacterJumpClones)
	if !hasData {
		ts.Text = "Waiting for character data to be loaded..."
		ts.Style.ColorName = theme.ColorNameWarning
		return []widget.RichTextSegment{ts}
	}

	var nextJumpColor fyne.ThemeColorName
	var nextJump, lastJump string

	v, ok := c.NextCloneJump.Value()
	if !ok {
		nextJump = "?"
		nextJumpColor = theme.ColorNameForeground
	} else if v.IsZero() {
		nextJump = "NOW"
		nextJumpColor = theme.ColorNameSuccess
	} else {
		nextJump = ihumanize.Duration(time.Until(v))
		nextJumpColor = theme.ColorNameError
	}

	if v, ok := c.LastCloneJumpAt.Value(); !ok {
		lastJump = "?"
	} else {
		lastJump = humanize.Time(v)
	}

	defaultStyleInline := defaultStyle
	defaultStyleInline.Inline = true
	segs := []widget.RichTextSegment{
		&widget.TextSegment{
			Text:  fmt.Sprintf("%d clones • Next available jump: ", cloneCount),
			Style: defaultStyleInline,
		},
		&widget.TextSegment{
			Text: nextJump,
			Style: widget.RichTextStyle{
				ColorName: nextJumpColor,
				Inline:    true,
			},
		},
		&widget.TextSegment{
			Text:  fmt.Sprintf(" • Last jump: %s", lastJump),
			Style: defaultStyle,
		},
	}
	return segs
}

type characterJumpCloneItem struct {
	widget.BaseWidget

	iconInfo     *xwidget.TappableIcon
	iconMain     *canvas.Image
	implants     *widget.Label
	isMobile     bool
	main         *ttwidget.Label
	prefix       *widget.Label
	spacer       fyne.CanvasObject
	loadTypeIcon loadFuncAsync
	showType     func(int64, int64)
	showLocation func(int64)
}

func newCharacterJumpCloneItem(isMobile bool, loadTypeIcon loadFuncAsync, showType func(int64, int64), showLocation func(int64)) *characterJumpCloneItem {
	iconMain := xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
	main := ttwidget.NewLabel("Template")
	main.Truncation = fyne.TextTruncateEllipsis
	iconInfo := xwidget.NewTappableIcon(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil)
	implants := widget.NewLabel("9")
	spacer :=xwidget.NewSpacer(fyne.NewSize(40, 10))
	prefix := widget.NewLabel("-9.9")
	prefix.Alignment = fyne.TextAlignTrailing
	w := &characterJumpCloneItem{
		iconInfo:     iconInfo,
		iconMain:     iconMain,
		implants:     implants,
		isMobile:     isMobile,
		main:         main,
		prefix:       prefix,
		spacer:       spacer,
		loadTypeIcon: loadTypeIcon,
		showType:     showType,
		showLocation: showLocation,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *characterJumpCloneItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		nil,
		nil,
		container.NewHBox(w.iconMain, container.NewStack(w.spacer, w.prefix)),
		container.NewHBox(w.implants, w.iconInfo),
		w.main,
	)
	return widget.NewSimpleRenderer(c)
}

func (w *characterJumpCloneItem) set(n *characterCloneNode) {
	if n.isTop() {
		if w.isMobile {
			w.iconMain.Hide()
		} else {
			w.iconMain.Resource = eveicon.FromName(eveicon.CloningCenter)
			w.iconMain.Refresh()
		}
		if !n.isUnknown {
			w.prefix.Text = fmt.Sprintf("%.1f", n.systemSecurityValue)
			w.prefix.Importance = n.systemSecurityType.ToImportance()
			w.iconInfo.OnTapped = func() {
				w.showLocation(n.locationID)
			}
			w.iconInfo.SetToolTip("Show location")
			w.iconInfo.Show()
		} else {
			w.prefix.Text = "?"
			w.prefix.Importance = widget.LowImportance
			w.iconInfo.Hide()
		}
		if n.implantCount > 0 {
			w.implants.SetText(fmt.Sprint(n.implantCount))
			w.implants.Show()
		} else {
			w.implants.Hide()
		}
		w.main.SetText(n.locationName)
		w.main.SetToolTip("")
		w.prefix.Show()
		w.spacer.Show()
		return
	}
	if w.isMobile {
		w.iconMain.Show()
	}
	w.implants.Hide()
	w.loadTypeIcon(n.implantTypeID, app.IconPixelSize, func(r fyne.Resource) {
		w.iconMain.Resource = r
		w.iconMain.Refresh()
	})
	w.main.SetText(n.implantTypeName)
	w.main.SetToolTip(n.implantTypeDescription)
	w.prefix.Hide()
	w.spacer.Hide()
	w.iconInfo.OnTapped = func() {
		w.showType(n.implantTypeID, n.characterID)
	}
	w.iconInfo.SetToolTip("Show implant")
	w.iconInfo.Show()
}
