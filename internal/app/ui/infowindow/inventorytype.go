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
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/shared"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	ilayout "github.com/ErikKalkoken/evebuddy/internal/layout"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type attributeRow struct {
	icon    fyne.Resource
	label   string
	value   string
	isTitle bool
	action  func(v string)
}

type requiredSkill struct {
	rank          int
	name          string
	typeID        int32
	activeLevel   int
	requiredLevel int
	trainedLevel  int
}

// inventoryTypeInfo displays information about Eve Online inventory types
type inventoryTypeInfo struct {
	widget.BaseWidget

	id             int32
	attributesData []attributeRow
	et             *app.EveType
	fittingData    []attributeRow
	metaLevel      int
	character      *app.EveEntity
	price          *app.EveMarketPrice
	requiredSkills []requiredSkill
	techLevel      int

	iw InfoWindow
	w  fyne.Window
}

func NewInventoryTypeInfo(iw InfoWindow, typeID, characterID int32, w fyne.Window) (*inventoryTypeInfo, error) {
	ctx := context.Background()
	a := &inventoryTypeInfo{iw: iw, w: w, id: typeID}
	a.ExtendBaseWidget(a)
	et, err := iw.u.EveUniverseService().GetOrCreateTypeESI(ctx, typeID)
	if err != nil {
		return nil, err
	}
	a.et = et
	owner, err := iw.u.EveUniverseService().GetOrCreateEntityESI(ctx, characterID)
	if err != nil {
		return nil, err
	}
	a.character = owner
	if a.et == nil {
		return nil, nil
	}
	p, err := iw.u.EveUniverseService().GetMarketPrice(ctx, a.et.ID)
	if errors.Is(err, app.ErrNotFound) {
		p = nil
	} else if err != nil {
		return nil, err
	} else if p.AveragePrice != 0 {
		a.price = p
	} else {
		a.price = nil
	}
	oo, err := iw.u.EveUniverseService().ListTypeDogmaAttributesForType(ctx, a.et.ID)
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
	return a, nil
}

func (a *inventoryTypeInfo) CreateRenderer() fyne.WidgetRenderer {
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
	return widget.NewSimpleRenderer(c)
}

func (a *inventoryTypeInfo) MakeTitle(suffix string) string {
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

func (a *inventoryTypeInfo) calcAttributesData(
	ctx context.Context,
	attributes map[int32]*app.EveTypeDogmaAttribute,
) []attributeRow {
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
			v, substituteIcon := a.iw.u.EveUniverseService().FormatDogmaValue(ctx, value, o.DogmaAttribute.Unit)
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
	rows := make([]attributeRow, 0)
	if a.et.Volume > 0 {
		v, _ := a.iw.u.EveUniverseService().FormatDogmaValue(ctx, a.et.Volume, app.EveUnitVolume)
		if a.et.Volume != a.et.PackagedVolume {
			v2, _ := a.iw.u.EveUniverseService().FormatDogmaValue(ctx, a.et.PackagedVolume, app.EveUnitVolume)
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
				rows = append(rows, attributeRow{label: ag.DisplayName(), isTitle: true})
			}
			rows = append(rows, groupedRows[ag]...)
		}
	}
	if a.iw.u.IsDeveloperMode() {
		rows = append(rows, attributeRow{label: "Developer Mode", isTitle: true})
		rows = append(rows, attributeRow{
			label: "EVE ID",
			value: fmt.Sprint(a.et.ID),
			action: func(v string) {
				a.w.Clipboard().SetContent(v)
			},
		})
	}
	return rows
}

func (a *inventoryTypeInfo) calcFittingData(ctx context.Context, attributes map[int32]*app.EveTypeDogmaAttribute) []attributeRow {
	data := make([]attributeRow, 0)
	for _, da := range attributeGroupsMap[attributeGroupFitting] {
		o, ok := attributes[da]
		if !ok {
			continue
		}
		iconID := o.DogmaAttribute.IconID
		r, _ := eveicon.GetResourceByIconID(iconID)
		v, _ := a.iw.u.EveUniverseService().FormatDogmaValue(ctx, o.Value, o.DogmaAttribute.Unit)
		data = append(data, attributeRow{
			icon:  r,
			label: o.DogmaAttribute.DisplayName,
			value: v,
		})
	}
	return data
}

func (a *inventoryTypeInfo) calcRequiredSkills(ctx context.Context, characterID int32, attributes map[int32]*app.EveTypeDogmaAttribute) ([]requiredSkill, error) {
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
		et, err := a.iw.u.EveUniverseService().GetType(ctx, typeID)
		if err != nil {
			return nil, err
		}
		skill := requiredSkill{
			rank:          i + 1,
			requiredLevel: requiredLevel,
			name:          et.Name,
			typeID:        typeID,
		}
		cs, err := a.iw.u.CharacterService().GetCharacterSkill(ctx, characterID, typeID)
		if errors.Is(err, app.ErrNotFound) {
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

func (a *inventoryTypeInfo) makeTop() fyne.CanvasObject {
	typeIcon := container.New(ilayout.NewTopLeftLayout())
	if a.et.HasRender() {
		size := 128
		r, err := a.iw.u.EveImageService().InventoryTypeRender(a.et.ID, size)
		if err != nil {
			slog.Error("Failed to load inventory type render", "typeID", a.et.ID, "error", err)
			r = theme.BrokenImageIcon()
		}
		render := kxwidget.NewTappableImage(r, func() {
			go a.iw.showZoomWindow(a.et.Name, a.et.ID, a.iw.u.EveImageService().InventoryTypeRender, a.w)
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
		icon := shared.NewImageResourceAsync(icons.QuestionmarkSvg, fyne.NewSquareSize(s), func() (fyne.Resource, error) {
			if a.et.IsSKIN() {
				return a.iw.u.EveImageService().InventoryTypeSKIN(a.et.ID, app.IconPixelSize)
			} else if a.et.IsBlueprint() {
				return a.iw.u.EveImageService().InventoryTypeBPO(a.et.ID, app.IconPixelSize)
			} else {
				return a.iw.u.EveImageService().InventoryTypeIcon(a.et.ID, app.IconPixelSize)
			}
		})
		typeIcon.Add(icon)
	}
	characterIcon := iwidget.NewImageFromResource(icons.QuestionmarkSvg, fyne.NewSquareSize(app.IconUnitSize))
	characterName := kxwidget.NewTappableLabel("", func() {
		a.iw.ShowEveEntity(a.character)
	})
	characterName.Wrapping = fyne.TextWrapWord
	if a.character != nil {
		shared.RefreshImageResourceAsync(characterIcon, func() (fyne.Resource, error) {
			return a.iw.u.EveImageService().CharacterPortrait(a.character.ID, app.IconPixelSize)
		})
		characterName.SetText(a.character.Name)
	} else {
		characterIcon.Hide()
		characterName.Hide()
	}
	hasRequiredSkills := true
	for _, o := range a.requiredSkills {
		if o.requiredLevel > o.activeLevel {
			hasRequiredSkills = false
			break
		}
	}
	checkIcon := widget.NewIcon(boolIconResource(hasRequiredSkills))
	if a.character != nil && !a.character.IsCharacter() || len(a.requiredSkills) == 0 {
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
				container.NewHBox(checkIcon, characterIcon),
				nil,
				characterName,
			)))
}

func (a *inventoryTypeInfo) makeDescriptionTab() fyne.CanvasObject {
	s := a.et.DescriptionPlain()
	if s == "" {
		s = a.et.Name
	}
	description := widget.NewLabel(s)
	description.Wrapping = fyne.TextWrapWord
	return container.NewVScroll(description)
}

func (a *inventoryTypeInfo) makeMarketTab() fyne.CanvasObject {
	c := container.NewHBox(
		widget.NewLabel("Average price"),
		layout.NewSpacer(),
		widget.NewLabel(ihumanize.Number(a.price.AveragePrice, 1)),
	)
	return container.NewVScroll(c)
}

func (a *inventoryTypeInfo) makeAttributesTab() fyne.CanvasObject {
	list := widget.NewList(
		func() int {
			return len(a.attributesData)
		},
		func() fyne.CanvasObject {
			return shared.NewTypeAttributeItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.attributesData) {
				return
			}
			r := a.attributesData[id]
			item := co.(*shared.TypeAttributeItem)
			if r.isTitle {
				item.SetTitle(r.label)
			} else {
				item.SetRegular(r.icon, r.label, r.value)
			}
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if id >= len(a.attributesData) {
			return
		}
		r := a.attributesData[id]
		if r.action != nil {
			r.action(r.value)
		}
	}
	return list
}

func (a *inventoryTypeInfo) makeFittingsTab() fyne.CanvasObject {
	l := widget.NewList(
		func() int {
			return len(a.fittingData)
		},
		func() fyne.CanvasObject {
			return shared.NewTypeAttributeItem()
		},
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			r := a.fittingData[lii]
			item := co.(*shared.TypeAttributeItem)
			item.SetRegular(r.icon, r.label, r.value)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *inventoryTypeInfo) makeRequirementsTab() fyne.CanvasObject {
	l := widget.NewList(
		func() int {
			return len(a.requiredSkills)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Placeholder"),
				layout.NewSpacer(),
				widget.NewLabel("Check"),
				shared.NewSkillLevel(),
				widget.NewIcon(icons.QuestionmarkSvg),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			o := a.requiredSkills[id]
			row := co.(*fyne.Container).Objects
			skill := row[0].(*widget.Label)
			text := row[2].(*widget.Label)
			level := row[3].(*shared.SkillLevel)
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
		a.iw.show(infoInventoryType, int64(r.typeID))
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
