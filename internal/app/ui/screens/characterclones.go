package screens

import (
	"context"
	"errors"
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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

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

func (n *characterCloneNode) isTop() bool {
	return n.implantTypeID == 0
}

func (n characterCloneNode) UID() widget.TreeNodeID {
	return fmt.Sprintf("%d-%d-%d", n.characterID, n.jumpCloneID, n.implantTypeID)
}

// cloneJumpInfo holds the inputs for the next clone jump of a character.
// The zero value means nothing is loaded.
type cloneJumpInfo struct {
	characterID int64
	lastJumpAt  optional.Optional[time.Time]
	skillLevel  int64 // Infomorph Synchronizing
}

type CharacterClones struct {
	widget.BaseWidget

	character   atomic.Pointer[app.Character]
	cloneTree   *xwidget.Tree[characterCloneNode]
	errorLabel  *widget.Label
	footerLabel *widget.Label
	jumpInfo    cloneJumpInfo // only accessed on the main thread
	topRun      latestRun
	topText     *xwidget.RichText
	u           baseUI
	updateRun   latestRun
}

func NewCharacterClones(u baseUI) *CharacterClones {
	a := &CharacterClones{
		errorLabel:  ui.NewLabelWithWrapping(""),
		footerLabel: ui.NewLabelWithTruncation(""),
		topText:     xwidget.NewRichText(),
		u:           u,
	}
	a.ExtendBaseWidget(a)

	a.topText.Wrapping = fyne.TextWrapWord
	a.errorLabel.Importance = widget.DangerImportance
	a.errorLabel.Hide()

	a.cloneTree = a.makeTree()

	// Signals
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDOrZero() != arg.CharacterID {
			return
		}
		switch arg.Section {
		case app.SectionCharacterJumpClones:
			a.update(ctx)
		case app.SectionCharacterSkills:
			a.loadJumpInfo(ctx) // skill level shortens the jump cooldown
		}
	})
	a.u.Signals().RefreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		a.refreshTop()
	})
	return a
}

func (a *CharacterClones) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.topText, container.NewVBox(a.errorLabel, a.footerLabel), nil, nil, a.cloneTree)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterClones) makeTree() *xwidget.Tree[characterCloneNode] {
	t := xwidget.NewTree(
		func(_ bool) fyne.CanvasObject {
			return newCharacterJumpCloneItem(
				a.u.IsMobile(),
				a.u.EVEImage().InventoryTypeIconAsync,
				a.u.InfoViewer().ShowType,
				a.u.InfoViewer().ShowLocation,
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
	isLatest := a.updateRun.start()
	characterID := a.character.Load().IDOrZero()
	if characterID == 0 {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.errorLabel.Hide()
			a.footerLabel.SetText("")
			a.cloneTree.Clear()
		})
		a.loadJumpInfo(ctx)
		return
	}

	td, err := a.fetchData(ctx, characterID)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh jump clones UI", "err", err)
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footerLabel.SetText("")
			a.cloneTree.Clear()
			a.errorLabel.SetText("ERROR: " + a.u.ErrorDisplay(err))
			a.errorLabel.Show()
			a.Refresh()
		})
		a.loadJumpInfo(ctx)
		return
	}

	footer := fmt.Sprintf("%s clones", ihumanize.Comma(td.ChildrenCount(nil)))
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.errorLabel.Hide()
		a.footerLabel.SetText(footer)
		a.cloneTree.Set(td)
	})
	a.loadJumpInfo(ctx)
}

func (a *CharacterClones) fetchData(ctx context.Context, characterID int64) (*xwidget.TreeData[characterCloneNode], error) {
	clones, err := a.u.Character().ListJumpClones(ctx, characterID)
	if err != nil {
		return nil, err
	}

	td := xwidget.NewTreeData[characterCloneNode]()
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

// loadJumpInfo loads the inputs for the next clone jump and shows them in the header.
func (a *CharacterClones) loadJumpInfo(ctx context.Context) {
	isLatest := a.topRun.start()
	setTop := func(text string, color fyne.ThemeColorName) {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.jumpInfo = cloneJumpInfo{}
			ts := &widget.TextSegment{
				Text: text,
				Style: widget.RichTextStyle{
					ColorName: color,
				},
			}
			a.topText.Set([]widget.RichTextSegment{ts})
		})
	}

	characterID := a.character.Load().IDOrZero()
	if characterID == 0 {
		setTop("No character", theme.ColorNameDisabled)
		return
	}
	reportError := func(err error) {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh jump clones header", "characterID", characterID, "error", err)
		setTop("ERROR: "+a.u.ErrorDisplay(err), theme.ColorNameError)
	}
	hasData, err := a.u.Character().HasSection(ctx, characterID, app.SectionCharacterJumpClones)
	if err != nil {
		reportError(err)
		return
	}
	if !hasData {
		setTop("Waiting for character data to be loaded...", theme.ColorNameWarning)
		return
	}
	c, err := a.u.Character().GetCharacter(ctx, characterID)
	if err != nil {
		reportError(err)
		return
	}
	var skillLevel int64
	sk, err := a.u.Character().GetSkill(ctx, characterID, app.EveTypeInfomorphSynchronizing)
	if errors.Is(err, app.ErrNotFound) {
		skillLevel = 0
	} else if err != nil {
		reportError(err)
		return
	} else {
		skillLevel = sk.ActiveSkillLevel
	}
	info := cloneJumpInfo{
		characterID: characterID,
		lastJumpAt:  c.LastCloneJumpAt,
		skillLevel:  skillLevel,
	}
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.jumpInfo = info
		a.showJumpInfo(info)
	})
}

// refreshTop updates the header from the loaded jump info, e.g. to show elapsed time.
func (a *CharacterClones) refreshTop() {
	fyne.Do(func() {
		id := a.jumpInfo.characterID
		if id == 0 || id != a.character.Load().IDOrZero() {
			return // nothing loaded yet or header shows a status
		}
		a.showJumpInfo(a.jumpInfo)
	})
}

// showJumpInfo shows the next and last clone jump in the header.
// Must be called on the main thread.
func (a *CharacterClones) showJumpInfo(info cloneJumpInfo) {
	var nextJumpColor fyne.ThemeColorName
	var nextJump, lastJump string

	v, ok := app.CalcNextCloneJump(info.lastJumpAt, info.skillLevel, time.Now()).Value()
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

	if v, ok := info.lastJumpAt.Value(); !ok {
		lastJump = "?"
	} else {
		lastJump = humanize.Time(v)
	}

	style := widget.RichTextStyle{
		ColorName: theme.ColorNameForeground,
	}
	styleInline := style
	styleInline.Inline = true
	segs := []widget.RichTextSegment{
		&widget.TextSegment{
			Text:  "Next available jump: ",
			Style: styleInline,
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
			Style: style,
		},
	}
	a.topText.Set(segs)
}

type characterJumpCloneItem struct {
	widget.BaseWidget

	iconInfo     *xwidget.Button
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
	iconMain := xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(ui.IconUnitSize))
	main := ttwidget.NewLabel("Template")
	main.Truncation = fyne.TextTruncateEllipsis
	iconInfo := xwidget.NewIconButton(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil)
	implants := widget.NewLabel("9")
	spacer := xwidget.NewSpacer(fyne.NewSize(40, 10))
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
	w.loadTypeIcon(n.implantTypeID, ui.IconPixelSize, func(r fyne.Resource) {
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
