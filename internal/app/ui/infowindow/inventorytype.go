package infowindow

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type attributeRow struct {
	icon    fyne.Resource
	label   string
	value   string
	isTitle bool
}

type requiredSkill struct {
	rank          int
	name          string
	typeID        int32
	activeLevel   int
	requiredLevel int
	trainedLevel  int
}

// inventoryTypeArea represents a UI component to display information about Eve Online inventory types
type inventoryTypeArea struct {
	Content fyne.CanvasObject

	attributesData []attributeRow
	et             *app.EveType
	fittingData    []attributeRow
	metaLevel      int
	owner          *app.EveEntity
	price          *app.EveMarketPrice
	requiredSkills []requiredSkill
	techLevel      int

	iw InfoWindow
	w  fyne.Window
}

func NewInventoryTypeArea(iw InfoWindow, typeID, characterID int32, w fyne.Window) (*inventoryTypeArea, error) {
	ctx := context.Background()
	a := &inventoryTypeArea{iw: iw, w: w}
	et, err := iw.eus.GetEveType(ctx, typeID)
	if err != nil {
		return nil, err
	}
	a.et = et
	owner, err := iw.eus.GetOrCreateEveEntityESI(ctx, characterID)
	if err != nil {
		return nil, err
	}
	a.owner = owner
	if a.et == nil {
		return nil, nil
	}
	p, err := iw.eus.GetEveMarketPrice(ctx, a.et.ID)
	if errors.Is(err, eveuniverse.ErrNotFound) {
		p = nil
	} else if err != nil {
		return nil, err
	} else if p.AveragePrice != 0 {
		a.price = p
	} else {
		a.price = nil
	}
	oo, err := iw.eus.ListEveTypeDogmaAttributesForType(ctx, a.et.ID)
	if err != nil {
		return nil, err
	}
	attributes := make(map[int32]*app.EveTypeDogmaAttribute)
	for _, o := range oo {
		attributes[o.DogmaAttribute.ID] = o
	}
	a.attributesData = a.calcAttributesData(ctx, attributes)
	a.fittingData = a.calcFittingData(ctx, attributes)
	if characterID != 0 {
		skills, err := a.calcRequiredSkills(ctx, characterID, attributes)
		if err != nil {
			return nil, err
		}
		a.requiredSkills = skills
	}
	a.techLevel, a.metaLevel = calcLevels(attributes)
	a.Content = a.makeContent()
	return a, nil
}

func (a *inventoryTypeArea) MakeTitle(suffix string) string {
	return fmt.Sprintf("%s: %s", a.et.Group.Name, suffix)
}

func calcLevels(attributes map[int32]*app.EveTypeDogmaAttribute) (int, int) {
	var tech, meta int
	x, ok := attributes[app.EveDogmaAttributeTechLevel]
	if ok {
		tech = int(x.Value)
	}
	x, ok = attributes[app.EveDogmaAttributeMetaLevel]
	if ok {
		meta = int(x.Value)
	}
	return tech, meta
}

func (a *inventoryTypeArea) calcAttributesData(ctx context.Context, attributes map[int32]*app.EveTypeDogmaAttribute) []attributeRow {
	droneCapacity, ok := attributes[app.EveDogmaAttributeDroneCapacity]
	hasDrones := ok && droneCapacity.Value > 0

	jumpDrive, ok := attributes[app.EveDogmaAttributeOnboardJumpDrive]
	hasJumpDrive := ok && jumpDrive.Value == 1.0

	groupedRows := make(map[attributeGroup][]attributeRow)

	for _, ag := range attributeGroups {
		attributeSelection := make([]*app.EveTypeDogmaAttribute, 0)
		for _, da := range attributeGroupsMap[ag] {
			o, ok := attributes[da]
			if !ok {
				continue
			}
			if ag == attributeGroupElectronicResistances {
				s := attributeGroupsMap[ag]
				found := slices.Index(s, o.DogmaAttribute.ID) == -1
				if found && o.Value == 0 {
					continue
				}
			}
			switch o.DogmaAttribute.ID {
			case app.EveDogmaAttributeCapacity, app.EveDogmaAttributeMass:
				if o.Value == 0 {
					continue
				}
			case app.EveDogmaAttributeDroneCapacity,
				app.EveDogmaAttributeDroneBandwidth:
				if !hasDrones {
					continue
				}
			case app.EveDogmaAttributeMaximumJumpRange,
				app.EveDogmaAttributeJumpDriveFuelNeed:
				if !hasJumpDrive {
					continue
				}
			case app.EveDogmaAttributeSupportFighterSquadronLimit:
				if o.Value == 0 {
					continue
				}
			}
			attributeSelection = append(attributeSelection, o)
		}
		if len(attributeSelection) == 0 {
			continue
		}
		for _, o := range attributeSelection {
			value := o.Value
			switch o.DogmaAttribute.ID {
			case app.EveDogmaAttributeShipWarpSpeed:
				x := attributes[app.EveDogmaAttributeWarpSpeedMultiplier]
				value = value * x.Value
			}
			v, substituteIcon := a.iw.eus.FormatValue(ctx, value, o.DogmaAttribute.Unit)
			var iconID int32
			if substituteIcon != 0 {
				iconID = substituteIcon
			} else {
				iconID = o.DogmaAttribute.IconID
			}
			r, _ := eveicon.GetResourceByIconID(iconID)
			groupedRows[ag] = append(groupedRows[ag], attributeRow{
				icon:  r,
				label: o.DogmaAttribute.DisplayName,
				value: v,
			})
		}
	}
	data := make([]attributeRow, 0)
	if a.et.Volume > 0 {
		v, _ := a.iw.eus.FormatValue(ctx, a.et.Volume, app.EveUnitVolume)
		if a.et.Volume != a.et.PackagedVolume {
			v2, _ := a.iw.eus.FormatValue(ctx, a.et.PackagedVolume, app.EveUnitVolume)
			v += fmt.Sprintf(" (%s Packaged)", v2)
		}
		r := attributeRow{
			icon:  eveicon.GetResourceByName(eveicon.Structure),
			label: "Volume",
			value: v,
		}
		var ag attributeGroup
		if len(groupedRows[attributeGroupStructure]) > 0 {
			ag = attributeGroupStructure
		} else {
			ag = attributeGroupMiscellaneous
		}
		groupedRows[ag] = append([]attributeRow{r}, groupedRows[ag]...)
	}
	usedGroupsCount := 0
	for _, ag := range attributeGroups {
		if len(groupedRows[ag]) > 0 {
			usedGroupsCount++
		}
	}
	for _, ag := range attributeGroups {
		if len(groupedRows[ag]) > 0 {
			if usedGroupsCount > 1 {
				data = append(data, attributeRow{label: ag.DisplayName(), isTitle: true})
			}
			data = append(data, groupedRows[ag]...)
		}
	}
	return data
}

func (a *inventoryTypeArea) calcFittingData(ctx context.Context, attributes map[int32]*app.EveTypeDogmaAttribute) []attributeRow {
	data := make([]attributeRow, 0)
	for _, da := range attributeGroupsMap[attributeGroupFitting] {
		o, ok := attributes[da]
		if !ok {
			continue
		}
		iconID := o.DogmaAttribute.IconID
		r, _ := eveicon.GetResourceByIconID(iconID)
		v, _ := a.iw.eus.FormatValue(ctx, o.Value, o.DogmaAttribute.Unit)
		data = append(data, attributeRow{
			icon:  r,
			label: o.DogmaAttribute.DisplayName,
			value: v,
		})
	}
	return data
}

func (a *inventoryTypeArea) calcRequiredSkills(ctx context.Context, characterID int32, attributes map[int32]*app.EveTypeDogmaAttribute) ([]requiredSkill, error) {
	skills := make([]requiredSkill, 0)
	skillAttributes := []struct {
		id    int32
		level int32
	}{
		{app.EveDogmaAttributePrimarySkillID, app.EveDogmaAttributePrimarySkillLevel},
		{app.EveDogmaAttributeSecondarySkillID, app.EveDogmaAttributeSecondarySkillLevel},
		{app.EveDogmaAttributeTertiarySkillID, app.EveDogmaAttributeTertiarySkillLevel},
		{app.EveDogmaAttributeQuaternarySkillID, app.EveDogmaAttributeQuaternarySkillLevel},
		{app.EveDogmaAttributeQuinarySkillID, app.EveDogmaAttributeQuinarySkillLevel},
		{app.EveDogmaAttributeSenarySkillID, app.EveDogmaAttributeSenarySkillLevel},
	}
	for i, x := range skillAttributes {
		daID, ok := attributes[x.id]
		if !ok {
			continue
		}
		typeID := int32(daID.Value)
		daLevel, ok := attributes[x.level]
		if !ok {
			continue
		}
		requiredLevel := int(daLevel.Value)
		et, err := a.iw.eus.GetEveType(ctx, typeID)
		if err != nil {
			return nil, err
		}
		skill := requiredSkill{
			rank:          i + 1,
			requiredLevel: requiredLevel,
			name:          et.Name,
			typeID:        typeID,
		}
		cs, err := a.iw.cs.GetCharacterSkill(ctx, characterID, typeID)
		if errors.Is(err, character.ErrNotFound) {
			// do nothing
		} else if err != nil {
			return nil, err
		} else {
			skill.activeLevel = cs.ActiveSkillLevel
			skill.trainedLevel = cs.TrainedSkillLevel
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

func (a *inventoryTypeArea) makeContent() fyne.CanvasObject {
	top := a.makeTop()
	t := container.NewTabItem("Description", a.makeDescriptionTab())
	tabs := container.NewAppTabs(t)
	var attributeTab, requirementsTab *container.TabItem
	if len(a.attributesData) > 0 {
		attributeTab = container.NewTabItem("Attributes", a.makeAttributesTab())
		tabs.Append(attributeTab)
	}
	if len(a.fittingData) > 0 {
		tabs.Append(container.NewTabItem("Fittings", a.makeFittingsTab()))
	}
	if len(a.requiredSkills) > 0 {
		requirementsTab = container.NewTabItem("Requirements", a.makeRequirementsTab())
		tabs.Append(requirementsTab)
	}
	if a.price != nil {
		tabs.Append(container.NewTabItem("Market", a.makeMarketTab()))
	}
	// Select selected tab
	if requirementsTab != nil && a.et.Group.Category.ID == app.EveCategorySkill {
		tabs.Select(requirementsTab)
	} else if attributeTab != nil {
		tabs.Select(attributeTab)
	}
	c := container.NewBorder(top, nil, nil, nil, tabs)
	return c
}

func (a *inventoryTypeArea) makeTop() fyne.CanvasObject {
	typeIcon := container.New(&topLeftLayout{})
	if a.et.HasRender() {
		size := 128
		r, err := a.iw.eis.InventoryTypeRender(a.et.ID, size)
		if err != nil {
			slog.Error("Failed to load inventory type render", "typeID", a.et.ID, "error", err)
			r = theme.BrokenImageIcon()
		}
		render := kxwidget.NewTappableImage(r, func() {
			go a.iw.showZoomWindow(a.et.Name, a.et.ID, a.iw.eis.InventoryTypeRender, a.w)
		})
		render.SetFillMode(canvas.ImageFillContain)
		s := float32(size)
		render.SetMinSize(fyne.Size{Width: s, Height: s})
		typeIcon.Add(render)
		if a.metaLevel > 4 {
			var n eveicon.Name
			switch a.techLevel {
			case 2:
				n = eveicon.Tech2
			case 3:
				n = eveicon.Tech3
			default:
				n = eveicon.Faction
			}
			marker := iwidget.NewImageFromResource(
				eveicon.GetResourceByName(n),
				fyne.NewSquareSize(render.MinSize().Width*0.2),
			)
			typeIcon.Add(container.NewPadded(marker))
		}
	} else {
		s := float32(app.IconPixelSize) * logoZoomFactor
		icon := appwidget.NewImageResourceAsync(icon.QuestionmarkSvg, fyne.NewSquareSize(s), func() (fyne.Resource, error) {
			if a.et.IsSKIN() {
				return a.iw.eis.InventoryTypeSKIN(a.et.ID, app.IconPixelSize)
			} else if a.et.IsBlueprint() {
				return a.iw.eis.InventoryTypeBPO(a.et.ID, app.IconPixelSize)
			} else {
				return a.iw.eis.InventoryTypeIcon(a.et.ID, app.IconPixelSize)
			}
		})
		typeIcon.Add(icon)
	}
	ownerIcon := iwidget.NewImageFromResource(icon.QuestionmarkSvg, fyne.NewSquareSize(app.IconUnitSize))
	ownerName := widget.NewLabel("")
	ownerName.Wrapping = fyne.TextWrapWord
	if a.owner != nil {
		appwidget.RefreshImageResourceAsync(ownerIcon, func() (fyne.Resource, error) {
			switch a.owner.Category {
			case app.EveEntityCharacter:
				return a.iw.eis.CharacterPortrait(a.owner.ID, app.IconPixelSize)
			case app.EveEntityCorporation:
				return a.iw.eis.CorporationLogo(a.owner.ID, app.IconPixelSize)
			default:
				panic("Unexpected owner type")
			}
		})
		ownerName.SetText(a.owner.Name)
	} else {
		ownerIcon.Hide()
		ownerName.Hide()
	}
	hasRequiredSkills := true
	for _, o := range a.requiredSkills {
		if o.requiredLevel > o.activeLevel {
			hasRequiredSkills = false
			break
		}
	}
	checkIcon := widget.NewIcon(boolIconResource(hasRequiredSkills))
	if a.owner != nil && !a.owner.IsCharacter() || len(a.requiredSkills) == 0 {
		checkIcon.Hide()
	}
	title := widget.NewLabel("")
	title.Wrapping = fyne.TextWrapWord
	title.SetText(a.et.Name)
	return container.NewBorder(
		nil,
		nil,
		typeIcon,
		nil,
		container.NewVBox(
			title,
			container.NewBorder(
				nil,
				nil,
				container.NewHBox(checkIcon, ownerIcon),
				nil,
				ownerName,
			)))
}

func (a *inventoryTypeArea) makeDescriptionTab() fyne.CanvasObject {
	s := a.et.DescriptionPlain()
	if s == "" {
		s = a.et.Name
	}
	description := widget.NewLabel(s)
	description.Wrapping = fyne.TextWrapWord
	return container.NewVScroll(description)
}

func (a *inventoryTypeArea) makeMarketTab() fyne.CanvasObject {
	c := container.NewHBox(
		widget.NewLabel("Average price"),
		layout.NewSpacer(),
		widget.NewLabel(ihumanize.Number(a.price.AveragePrice, 1)),
	)
	return container.NewVScroll(c)
}

func (a *inventoryTypeArea) makeAttributesTab() fyne.CanvasObject {
	list := widget.NewList(
		func() int {
			return len(a.attributesData)
		},
		func() fyne.CanvasObject {
			return appwidget.NewTypeAttributeItem()
		},
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			r := a.attributesData[lii]
			item := co.(*appwidget.TypeAttributeItem)
			if r.isTitle {
				item.SetTitle(r.label)
			} else {
				item.SetRegular(r.icon, r.label, r.value)
			}
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		list.UnselectAll()
	}
	return list
}

func (a *inventoryTypeArea) makeFittingsTab() fyne.CanvasObject {
	l := widget.NewList(
		func() int {
			return len(a.fittingData)
		},
		func() fyne.CanvasObject {
			return appwidget.NewTypeAttributeItem()
		},
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			r := a.fittingData[lii]
			item := co.(*appwidget.TypeAttributeItem)
			item.SetRegular(r.icon, r.label, r.value)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *inventoryTypeArea) makeRequirementsTab() fyne.CanvasObject {
	l := widget.NewList(
		func() int {
			return len(a.requiredSkills)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Placeholder"),
				layout.NewSpacer(),
				widget.NewLabel("Check"),
				appwidget.NewSkillLevel(),
				widget.NewIcon(icon.QuestionmarkSvg),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			o := a.requiredSkills[id]
			row := co.(*fyne.Container).Objects
			skill := row[0].(*widget.Label)
			text := row[2].(*widget.Label)
			level := row[3].(*appwidget.SkillLevel)
			icon := row[4].(*widget.Icon)
			skill.SetText(app.SkillDisplayName(o.name, o.requiredLevel))
			if o.activeLevel == 0 && o.trainedLevel == 0 {
				text.Text = "Skill not injected"
				text.Importance = widget.DangerImportance
				text.Refresh()
				text.Show()
				level.Hide()
				icon.Hide()
			} else if o.activeLevel >= o.requiredLevel {
				icon.SetResource(boolIconResource(true))
				icon.Show()
				text.Hide()
				level.Hide()
			} else {
				level.Set(o.activeLevel, o.trainedLevel, o.requiredLevel)
				text.Refresh()
				text.Hide()
				icon.Hide()
				level.Show()
			}
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		r := a.requiredSkills[id]
		a.iw.Show(InventoryType, int64(r.typeID))
		l.UnselectAll()
	}
	return l
}

func boolIconResource(ok bool) fyne.Resource {
	if ok {
		return theme.NewSuccessThemedResource(theme.ConfirmIcon())
	}
	return theme.NewErrorThemedResource(theme.CancelIcon())
}
