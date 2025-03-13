package ui

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
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type attributeGroup string

func (ag attributeGroup) DisplayName() string {
	return Titler.String(string(ag))
}

// groups of attributes to display on the attributes and fitting tab
const (
	attributeGroupArmor                 attributeGroup = "armor"
	attributeGroupCapacitor             attributeGroup = "capacitor"
	attributeGroupElectronicResistances attributeGroup = "electronic resistances"
	attributeGroupFitting               attributeGroup = "fitting"
	attributeGroupFighter               attributeGroup = "fighter squadron facilities"
	attributeGroupJumpDrive             attributeGroup = "jump drive systems"
	attributeGroupMiscellaneous         attributeGroup = "miscellaneous"
	attributeGroupPropulsion            attributeGroup = "propulsion"
	attributeGroupShield                attributeGroup = "shield"
	attributeGroupStructure             attributeGroup = "structure"
	attributeGroupTargeting             attributeGroup = "targeting"
)

// attribute groups to show in order on attributes tab
var attributeGroups = []attributeGroup{
	attributeGroupStructure,
	attributeGroupArmor,
	attributeGroupShield,
	attributeGroupElectronicResistances,
	attributeGroupCapacitor,
	attributeGroupTargeting,
	attributeGroupFighter,
	attributeGroupJumpDrive,
	attributeGroupPropulsion,
	attributeGroupMiscellaneous,
}

// assignment of attributes to groups
var attributeGroupsMap = map[attributeGroup][]int32{
	attributeGroupStructure: {
		app.EveDogmaAttributeStructureHitpoints,
		app.EveDogmaAttributeCapacity,
		app.EveDogmaAttributeDroneCapacity,
		app.EveDogmaAttributeDroneBandwidth,
		app.EveDogmaAttributeMass,
		app.EveDogmaAttributeInertiaModifier,
		app.EveDogmaAttributeStructureEMDamageResistance,
		app.EveDogmaAttributeStructureThermalDamageResistance,
		app.EveDogmaAttributeStructureKineticDamageResistance,
		app.EveDogmaAttributeStructureExplosiveDamageResistance,
	},
	attributeGroupArmor: {
		app.EveDogmaAttributeArmorHitpoints,
		app.EveDogmaAttributeArmorEMDamageResistance,
		app.EveDogmaAttributeArmorThermalDamageResistance,
		app.EveDogmaAttributeArmorKineticDamageResistance,
		app.EveDogmaAttributeArmorExplosiveDamageResistance,
	},
	attributeGroupShield: {
		app.EveDogmaAttributeShieldCapacity,
		app.EveDogmaAttributeShieldRechargeTime,
		app.EveDogmaAttributeShieldEMDamageResistance,
		app.EveDogmaAttributeShieldThermalDamageResistance,
		app.EveDogmaAttributeShieldKineticDamageResistance,
		app.EveDogmaAttributeShieldExplosiveDamageResistance,
	},
	attributeGroupElectronicResistances: {
		app.EveDogmaAttributeCargoScanResistance,
		app.EveDogmaAttributeCapacitorWarfareResistance,
		app.EveDogmaAttributeSensorWarfareResistance,
		app.EveDogmaAttributeWeaponDisruptionResistance,
		app.EveDogmaAttributeTargetPainterResistance,
		app.EveDogmaAttributeStasisWebifierResistance,
		app.EveDogmaAttributeRemoteLogisticsImpedance,
		app.EveDogmaAttributeRemoteElectronicAssistanceImpedance,
		app.EveDogmaAttributeECMResistance,
		app.EveDogmaAttributeCapacitorWarfareResistanceBonus,
		app.EveDogmaAttributeStasisWebifierResistanceBonus,
	},
	attributeGroupCapacitor: {
		app.EveDogmaAttributeCapacitorCapacity,
		app.EveDogmaAttributeCapacitorRechargeTime,
	},
	attributeGroupTargeting: {
		app.EveDogmaAttributeMaximumTargetingRange,
		app.EveDogmaAttributeMaximumLockedTargets,
		app.EveDogmaAttributeSignatureRadius,
		app.EveDogmaAttributeScanResolution,
		app.EveDogmaAttributeRADARSensorStrength,
		app.EveDogmaAttributeLadarSensorStrength,
		app.EveDogmaAttributeMagnetometricSensorStrength,
		app.EveDogmaAttributeGravimetricSensorStrength,
	},
	attributeGroupPropulsion: {
		app.EveDogmaAttributeMaxVelocity,
		app.EveDogmaAttributeShipWarpSpeed,
	},
	attributeGroupJumpDrive: {
		app.EveDogmaAttributeJumpDriveCapacitorNeed,
		app.EveDogmaAttributeMaximumJumpRange,
		app.EveDogmaAttributeJumpDriveFuelNeed,
		app.EveDogmaAttributeJumpDriveConsumptionAmount,
		app.EveDogmaAttributeFuelBayCapacity,
	},
	attributeGroupFighter: {
		app.EveDogmaAttributeFighterHangarCapacity,
		app.EveDogmaAttributeFighterSquadronLaunchTubes,
		app.EveDogmaAttributeLightFighterSquadronLimit,
		app.EveDogmaAttributeSupportFighterSquadronLimit,
		app.EveDogmaAttributeHeavyFighterSquadronLimit,
	},
	attributeGroupFitting: {
		app.EveDogmaAttributeCPUOutput,
		app.EveDogmaAttributeCPUusage,
		app.EveDogmaAttributePowergridOutput,
		app.EveDogmaAttributeCalibration,
		app.EveDogmaAttributeRigSlots,
		app.EveDogmaAttributeLauncherHardpoints,
		app.EveDogmaAttributeTurretHardpoints,
		app.EveDogmaAttributeHighSlots,
		app.EveDogmaAttributeMediumSlots,
		app.EveDogmaAttributeLowSlots,
		app.EveDogmaAttributeRigSlots,
	},
	attributeGroupMiscellaneous: {
		app.EveDogmaAttributeImplantSlot,
		app.EveDogmaAttributeCharismaModifier,
		app.EveDogmaAttributeIntelligenceModifier,
		app.EveDogmaAttributeMemoryModifier,
		app.EveDogmaAttributePerceptionModifier,
		app.EveDogmaAttributeWillpowerModifier,
		app.EveDogmaAttributePrimaryAttribute,
		app.EveDogmaAttributeSecondaryAttribute,
		app.EveDogmaAttributeTrainingTimeMultiplier,
		app.EveDogmaAttributeTechLevel,
	},
}

type requiredSkill struct {
	rank          int
	name          string
	typeID        int32
	activeLevel   int
	requiredLevel int
	trainedLevel  int
}

type attributeRow struct {
	icon    fyne.Resource
	label   string
	value   string
	isTitle bool
}

// ItemInfoArea represents a UI component to display information about Eve Online items;
// similar to the info window in the game client.
type ItemInfoArea struct {
	Content fyne.CanvasObject

	attributesData []attributeRow
	et             *app.EveType
	fittingData    []attributeRow
	metaLevel      int
	owner          *app.EveEntity
	price          *app.EveMarketPrice
	requiredSkills []requiredSkill
	techLevel      int

	eis app.EveImageService
	eus *eveuniverse.EveUniverseService
	cs  *character.CharacterService
	w   fyne.Window
}

// TODO: Restructure, so that window is first drawn empty and content loaded in background (same as character info windo)
func NewItemInfoArea(
	cs *character.CharacterService,
	eis app.EveImageService,
	eus *eveuniverse.EveUniverseService,
	typeID, characterID int32,
	w fyne.Window,
) (*ItemInfoArea, error) {
	ctx := context.Background()
	a := &ItemInfoArea{
		cs:  cs,
		eis: eis,
		eus: eus,
		w:   w,
	}
	et, err := eus.GetEveType(ctx, typeID)
	if err != nil {
		return nil, err
	}
	a.et = et
	owner, err := eus.GetOrCreateEveEntityESI(ctx, characterID)
	if err != nil {
		return nil, err
	}
	a.owner = owner
	if a.et == nil {
		return nil, nil
	}
	p, err := eus.GetEveMarketPrice(ctx, a.et.ID)
	if errors.Is(err, eveuniverse.ErrNotFound) {
		p = nil
	} else if err != nil {
		return nil, err
	} else if p.AveragePrice != 0 {
		a.price = p
	} else {
		a.price = nil
	}
	oo, err := eus.ListEveTypeDogmaAttributesForType(ctx, a.et.ID)
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

func (a *ItemInfoArea) MakeTitle(suffix string) string {
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

func (a *ItemInfoArea) calcAttributesData(ctx context.Context, attributes map[int32]*app.EveTypeDogmaAttribute) []attributeRow {
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
			v, substituteIcon := a.eus.FormatValue(ctx, value, o.DogmaAttribute.Unit)
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
		v, _ := a.eus.FormatValue(ctx, a.et.Volume, app.EveUnitVolume)
		if a.et.Volume != a.et.PackagedVolume {
			v2, _ := a.eus.FormatValue(ctx, a.et.PackagedVolume, app.EveUnitVolume)
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

func (a *ItemInfoArea) calcFittingData(ctx context.Context, attributes map[int32]*app.EveTypeDogmaAttribute) []attributeRow {
	data := make([]attributeRow, 0)
	for _, da := range attributeGroupsMap[attributeGroupFitting] {
		o, ok := attributes[da]
		if !ok {
			continue
		}
		iconID := o.DogmaAttribute.IconID
		r, _ := eveicon.GetResourceByIconID(iconID)
		v, _ := a.eus.FormatValue(ctx, o.Value, o.DogmaAttribute.Unit)
		data = append(data, attributeRow{
			icon:  r,
			label: o.DogmaAttribute.DisplayName,
			value: v,
		})
	}
	return data
}

func (a *ItemInfoArea) calcRequiredSkills(ctx context.Context, characterID int32, attributes map[int32]*app.EveTypeDogmaAttribute) ([]requiredSkill, error) {
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
		et, err := a.eus.GetEveType(ctx, typeID)
		if err != nil {
			return nil, err
		}
		skill := requiredSkill{
			rank:          i + 1,
			requiredLevel: requiredLevel,
			name:          et.Name,
			typeID:        typeID,
		}
		cs, err := a.cs.GetCharacterSkill(ctx, characterID, typeID)
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

func (a *ItemInfoArea) makeContent() fyne.CanvasObject {
	top := a.makeTop()
	t := container.NewTabItem("Description", a.makeDescriptionTab())
	tabs := container.NewAppTabs(t)
	if len(a.attributesData) > 0 && a.et.Group.Category.ID != app.EveCategoryStation {
		tabs.Append(container.NewTabItem("Attributes", a.makeAttributesTab()))
	}
	if len(a.fittingData) > 0 {
		tabs.Append(container.NewTabItem("Fittings", a.makeFittingsTab()))
	}
	if len(a.requiredSkills) > 0 {
		t := container.NewTabItem("Requirements", a.makeRequirementsTab())
		tabs.Append(t)
	}
	if a.price != nil {
		tabs.Append(container.NewTabItem("Market", a.makeMarketTab()))
	}
	c := container.NewBorder(top, nil, nil, nil, tabs)
	return c
}

func (a *ItemInfoArea) makeTop() fyne.CanvasObject {
	typeIcon := container.New(&topLeftLayout{})
	if a.et.HasRender() {
		size := 128
		r, err := a.eis.InventoryTypeRender(a.et.ID, size)
		if err != nil {
			slog.Error("Failed to load inventory type render", "typeID", a.et.ID, "error", err)
			r = theme.BrokenImageIcon()
		}
		render := kxwidget.NewTappableImage(r, func() {
			w := fyne.CurrentApp().NewWindow(a.MakeTitle("Render"))
			size := 512
			s := float32(size) / w.Canvas().Scale()
			i := appwidget.NewImageResourceAsync(icon.QuestionmarkSvg, fyne.NewSquareSize(s), func() (fyne.Resource, error) {
				return a.eis.InventoryTypeRender(a.et.ID, size)
			})
			p := theme.Padding()
			w.SetContent(container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), i))
			w.Show()
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
		s := float32(DefaultIconPixelSize) * 1.3
		icon := appwidget.NewImageResourceAsync(icon.QuestionmarkSvg, fyne.NewSquareSize(s), func() (fyne.Resource, error) {
			if a.et.IsSKIN() {
				return a.eis.InventoryTypeSKIN(a.et.ID, DefaultIconPixelSize)
			} else if a.et.IsBlueprint() {
				return a.eis.InventoryTypeBPO(a.et.ID, DefaultIconPixelSize)
			} else {
				return a.eis.InventoryTypeIcon(a.et.ID, DefaultIconPixelSize)
			}
		})
		typeIcon.Add(icon)
	}
	ownerIcon := iwidget.NewImageFromResource(icon.QuestionmarkSvg, fyne.NewSquareSize(DefaultIconUnitSize))
	ownerName := widget.NewLabel("")
	ownerName.Wrapping = fyne.TextWrapWord
	if a.owner != nil {
		appwidget.RefreshImageResourceAsync(ownerIcon, func() (fyne.Resource, error) {
			switch a.owner.Category {
			case app.EveEntityCharacter:
				return a.eis.CharacterPortrait(a.owner.ID, DefaultIconPixelSize)
			case app.EveEntityCorporation:
				return a.eis.CorporationLogo(a.owner.ID, DefaultIconPixelSize)
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
	checkIcon := widget.NewIcon(BoolIconResource(hasRequiredSkills))
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

func (a *ItemInfoArea) makeDescriptionTab() fyne.CanvasObject {
	description := widget.NewLabel(a.et.DescriptionPlain())
	description.Wrapping = fyne.TextWrapWord
	return container.NewVScroll(description)
}

func (a *ItemInfoArea) makeMarketTab() fyne.CanvasObject {
	c := container.NewHBox(
		widget.NewLabel("Average price"),
		layout.NewSpacer(),
		widget.NewLabel(humanize.Number(a.price.AveragePrice, 1)),
	)
	return container.NewVScroll(c)
}

func (a *ItemInfoArea) makeAttributesTab() fyne.CanvasObject {
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

func (a *ItemInfoArea) makeFittingsTab() fyne.CanvasObject {
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

func (a *ItemInfoArea) makeRequirementsTab() fyne.CanvasObject {
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
			skill.SetText(SkillDisplayName(o.name, o.requiredLevel))
			if o.activeLevel == 0 && o.trainedLevel == 0 {
				text.Text = "Skill not injected"
				text.Importance = widget.DangerImportance
				text.Refresh()
				text.Show()
				level.Hide()
				icon.Hide()
			} else if o.activeLevel >= o.requiredLevel {
				icon.SetResource(BoolIconResource(true))
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
		// r := a.requiredSkills[id]
		// a.u.ShowTypeInfoWindow(r.typeID, a.owner.ID)
		l.UnselectAll()
	}
	return l
}
