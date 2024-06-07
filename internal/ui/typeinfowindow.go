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
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/eveonline/icons"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service/character"
	"github.com/ErikKalkoken/evebuddy/internal/widgets"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type attributeGroup string

func (ag attributeGroup) DisplayName() string {
	c := cases.Title(language.English)
	return c.String(string(ag))
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
		model.EveDogmaAttributeStructureHitpoints,
		model.EveDogmaAttributeCapacity,
		model.EveDogmaAttributeDroneCapacity,
		model.EveDogmaAttributeDroneBandwidth,
		model.EveDogmaAttributeMass,
		model.EveDogmaAttributeInertiaModifier,
		model.EveDogmaAttributeStructureEMDamageResistance,
		model.EveDogmaAttributeStructureThermalDamageResistance,
		model.EveDogmaAttributeStructureKineticDamageResistance,
		model.EveDogmaAttributeStructureExplosiveDamageResistance,
	},
	attributeGroupArmor: {
		model.EveDogmaAttributeArmorHitpoints,
		model.EveDogmaAttributeArmorEMDamageResistance,
		model.EveDogmaAttributeArmorThermalDamageResistance,
		model.EveDogmaAttributeArmorKineticDamageResistance,
		model.EveDogmaAttributeArmorExplosiveDamageResistance,
	},
	attributeGroupShield: {
		model.EveDogmaAttributeShieldCapacity,
		model.EveDogmaAttributeShieldRechargeTime,
		model.EveDogmaAttributeShieldEMDamageResistance,
		model.EveDogmaAttributeShieldThermalDamageResistance,
		model.EveDogmaAttributeShieldKineticDamageResistance,
		model.EveDogmaAttributeShieldExplosiveDamageResistance,
	},
	attributeGroupElectronicResistances: {
		model.EveDogmaAttributeCargoScanResistance,
		model.EveDogmaAttributeCapacitorWarfareResistance,
		model.EveDogmaAttributeSensorWarfareResistance,
		model.EveDogmaAttributeWeaponDisruptionResistance,
		model.EveDogmaAttributeTargetPainterResistance,
		model.EveDogmaAttributeStasisWebifierResistance,
		model.EveDogmaAttributeRemoteLogisticsImpedance,
		model.EveDogmaAttributeRemoteElectronicAssistanceImpedance,
		model.EveDogmaAttributeECMResistance,
		model.EveDogmaAttributeCapacitorWarfareResistanceBonus,
		model.EveDogmaAttributeStasisWebifierResistanceBonus,
	},
	attributeGroupCapacitor: {
		model.EveDogmaAttributeCapacitorCapacity,
		model.EveDogmaAttributeCapacitorRechargeTime,
	},
	attributeGroupTargeting: {
		model.EveDogmaAttributeMaximumTargetingRange,
		model.EveDogmaAttributeMaximumLockedTargets,
		model.EveDogmaAttributeSignatureRadius,
		model.EveDogmaAttributeScanResolution,
		model.EveDogmaAttributeRADARSensorStrength,
		model.EveDogmaAttributeLadarSensorStrength,
		model.EveDogmaAttributeMagnetometricSensorStrength,
		model.EveDogmaAttributeGravimetricSensorStrength,
	},
	attributeGroupPropulsion: {
		model.EveDogmaAttributeMaxVelocity,
		model.EveDogmaAttributeShipWarpSpeed,
	},
	attributeGroupJumpDrive: {
		model.EveDogmaAttributeJumpDriveCapacitorNeed,
		model.EveDogmaAttributeMaximumJumpRange,
		model.EveDogmaAttributeJumpDriveFuelNeed,
		model.EveDogmaAttributeJumpDriveConsumptionAmount,
		model.EveDogmaAttributeFuelBayCapacity,
	},
	attributeGroupFighter: {
		model.EveDogmaAttributeFighterHangarCapacity,
		model.EveDogmaAttributeFighterSquadronLaunchTubes,
		model.EveDogmaAttributeLightFighterSquadronLimit,
		model.EveDogmaAttributeSupportFighterSquadronLimit,
		model.EveDogmaAttributeHeavyFighterSquadronLimit,
	},
	attributeGroupFitting: {
		model.EveDogmaAttributeCPUOutput,
		model.EveDogmaAttributeCPUusage,
		model.EveDogmaAttributePowergridOutput,
		model.EveDogmaAttributeCalibration,
		model.EveDogmaAttributeRigSlots,
		model.EveDogmaAttributeLauncherHardpoints,
		model.EveDogmaAttributeTurretHardpoints,
		model.EveDogmaAttributeHighSlots,
		model.EveDogmaAttributeMediumSlots,
		model.EveDogmaAttributeLowSlots,
		model.EveDogmaAttributeRigSlots,
	},
	attributeGroupMiscellaneous: {
		model.EveDogmaAttributeImplantSlot,
		model.EveDogmaAttributeCharismaModifier,
		model.EveDogmaAttributeIntelligenceModifier,
		model.EveDogmaAttributeMemoryModifier,
		model.EveDogmaAttributePerceptionModifier,
		model.EveDogmaAttributeWillpowerModifier,
		model.EveDogmaAttributePrimaryAttribute,
		model.EveDogmaAttributeSecondaryAttribute,
		model.EveDogmaAttributeTrainingTimeMultiplier,
		model.EveDogmaAttributeTechLevel,
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

type typeInfoWindow struct {
	attributesData []attributeRow
	content        fyne.CanvasObject
	location       *model.EveLocation
	owner          *model.EveEntity
	et             *model.EveType
	fittingData    []attributeRow
	requiredSkills []requiredSkill
	techLevel      int
	metaLevel      int
	ui             *ui
	window         fyne.Window
}

func (u *ui) showTypeInfoWindow(typeID, characterID int32) {
	u.showInfoWindow(u.newTypeInfoWindow(typeID, characterID, 0))
}

func (u *ui) showLocationInfoWindow(locationID int64) {
	u.showInfoWindow(u.newTypeInfoWindow(0, 0, locationID))
}

func (u *ui) showInfoWindow(iw *typeInfoWindow, err error) {
	if err != nil {
		t := "Failed to open info window"
		slog.Error(t, "err", err)
		u.showErrorDialog(t, err)
		return
	}
	w := u.app.NewWindow(iw.makeTitle("Information"))
	iw.window = w
	w.SetContent(iw.content)
	w.Resize(fyne.Size{Width: 500, Height: 500})
	w.Show()
}

func (u *ui) newTypeInfoWindow(typeID, characterID int32, locationID int64) (*typeInfoWindow, error) {
	ctx := context.Background()
	a := &typeInfoWindow{
		ui: u,
	}
	if locationID != 0 {
		location, err := u.sv.EveUniverse.GetEveLocation(ctx, locationID)
		if err != nil {
			return nil, err
		}
		a.location = location
		a.et = location.Type
		a.owner = a.location.Owner
	} else {
		et, err := u.sv.EveUniverse.GetEveType(ctx, typeID)
		if err != nil {
			return nil, err
		}
		a.et = et
		owner, err := u.sv.EveUniverse.GetOrCreateEveEntityESI(ctx, characterID)
		if err != nil {
			return nil, err
		}
		a.owner = owner
	}
	oo, err := u.sv.EveUniverse.ListEveTypeDogmaAttributesForType(ctx, a.et.ID)
	if err != nil {
		return nil, err
	}
	attributes := make(map[int32]*model.EveDogmaAttributeForType)
	for _, o := range oo {
		attributes[o.DogmaAttribute.ID] = o
	}
	a.attributesData = a.calcAttributesData(ctx, attributes)
	a.fittingData = a.calcFittingData(ctx, attributes)
	if !a.isLocation() {
		skills, err := a.calcRequiredSkills(ctx, characterID, attributes)
		if err != nil {
			return nil, err
		}
		a.requiredSkills = skills
	}
	a.techLevel, a.metaLevel = calcLevels(attributes)
	a.content = a.makeContent()
	return a, nil
}

func (a *typeInfoWindow) isLocation() bool {
	return a.location != nil
}

func calcLevels(attributes map[int32]*model.EveDogmaAttributeForType) (int, int) {
	var tech, meta int
	x, ok := attributes[model.EveDogmaAttributeTechLevel]
	if ok {
		tech = int(x.Value)
	}
	x, ok = attributes[model.EveDogmaAttributeMetaLevel]
	if ok {
		meta = int(x.Value)
	}
	return tech, meta
}

func (a *typeInfoWindow) calcAttributesData(ctx context.Context, attributes map[int32]*model.EveDogmaAttributeForType) []attributeRow {
	droneCapacity, ok := attributes[model.EveDogmaAttributeDroneCapacity]
	hasDrones := ok && droneCapacity.Value > 0

	jumpDrive, ok := attributes[model.EveDogmaAttributeOnboardJumpDrive]
	hasJumpDrive := ok && jumpDrive.Value == 1.0

	groupedRows := make(map[attributeGroup][]attributeRow)

	for _, ag := range attributeGroups {
		attributeSelection := make([]*model.EveDogmaAttributeForType, 0)
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
			case model.EveDogmaAttributeCapacity, model.EveDogmaAttributeMass:
				if o.Value == 0 {
					continue
				}
			case model.EveDogmaAttributeDroneCapacity,
				model.EveDogmaAttributeDroneBandwidth:
				if !hasDrones {
					continue
				}
			case model.EveDogmaAttributeMaximumJumpRange,
				model.EveDogmaAttributeJumpDriveFuelNeed:
				if !hasJumpDrive {
					continue
				}
			case model.EveDogmaAttributeSupportFighterSquadronLimit:
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
			case model.EveDogmaAttributeShipWarpSpeed:
				x := attributes[model.EveDogmaAttributeWarpSpeedMultiplier]
				value = value * x.Value
			}
			v, substituteIcon := a.ui.sv.EveUniverse.FormatValue(ctx, value, o.DogmaAttribute.Unit)
			var iconID int32
			if substituteIcon != 0 {
				iconID = substituteIcon
			} else {
				iconID = o.DogmaAttribute.IconID
			}
			r, _ := icons.GetResourceByIconID(iconID)
			groupedRows[ag] = append(groupedRows[ag], attributeRow{
				icon:  r,
				label: o.DogmaAttribute.DisplayName,
				value: v,
			})
		}
	}
	data := make([]attributeRow, 0)
	if a.et.Volume > 0 {
		v, _ := a.ui.sv.EveUniverse.FormatValue(ctx, a.et.Volume, model.EveUnitVolume)
		if a.et.Volume != a.et.PackagedVolume {
			v2, _ := a.ui.sv.EveUniverse.FormatValue(ctx, a.et.PackagedVolume, model.EveUnitVolume)
			v += fmt.Sprintf(" (%s Packaged)", v2)
		}
		r := attributeRow{
			icon:  icons.GetResourceByName(icons.Structure),
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
	if a.ui.isDebug {
		data = append(data, attributeRow{label: "DEBUG", isTitle: true})
		data = append(data, attributeRow{label: "Owner", value: fmt.Sprint(a.owner)})
		data = append(data, attributeRow{label: "Type ID", value: fmt.Sprint(a.et.ID)})
	}
	return data
}

func (a *typeInfoWindow) calcFittingData(ctx context.Context, attributes map[int32]*model.EveDogmaAttributeForType) []attributeRow {
	data := make([]attributeRow, 0)
	for _, da := range attributeGroupsMap[attributeGroupFitting] {
		o, ok := attributes[da]
		if !ok {
			continue
		}
		iconID := o.DogmaAttribute.IconID
		r, _ := icons.GetResourceByIconID(iconID)
		v, _ := a.ui.sv.EveUniverse.FormatValue(ctx, o.Value, o.DogmaAttribute.Unit)
		data = append(data, attributeRow{
			icon:  r,
			label: o.DogmaAttribute.DisplayName,
			value: v,
		})
	}
	return data
}

func (a *typeInfoWindow) calcRequiredSkills(ctx context.Context, characterID int32, attributes map[int32]*model.EveDogmaAttributeForType) ([]requiredSkill, error) {
	skills := make([]requiredSkill, 0)
	skillAttributes := []struct {
		id    int32
		level int32
	}{
		{model.EveDogmaAttributePrimarySkillID, model.EveDogmaAttributePrimarySkillLevel},
		{model.EveDogmaAttributeSecondarySkillID, model.EveDogmaAttributeSecondarySkillLevel},
		{model.EveDogmaAttributeTertiarySkillID, model.EveDogmaAttributeTertiarySkillLevel},
		{model.EveDogmaAttributeQuaternarySkillID, model.EveDogmaAttributeQuaternarySkillLevel},
		{model.EveDogmaAttributeQuinarySkillID, model.EveDogmaAttributeQuinarySkillLevel},
		{model.EveDogmaAttributeSenarySkillID, model.EveDogmaAttributeSenarySkillLevel},
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
		et, err := a.ui.sv.EveUniverse.GetEveType(ctx, typeID)
		if err != nil {
			return nil, err
		}
		skill := requiredSkill{
			rank:          i + 1,
			requiredLevel: requiredLevel,
			name:          et.Name,
			typeID:        typeID,
		}
		cs, err := a.ui.sv.Characters.GetCharacterSkill(ctx, characterID, typeID)
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

func (a *typeInfoWindow) makeTitle(suffix string) string {
	s := fmt.Sprintf("%s (%s): %s", a.et.Name, a.et.Group.Name, suffix)
	if a.ui.isDebug {
		s += " DEBUG"
	}
	return s
}

func (a *typeInfoWindow) makeContent() fyne.CanvasObject {
	top := a.makeTop()
	description := container.NewTabItem("Description", a.makeDescriptionTab())
	tabs := container.NewAppTabs(description)
	if len(a.attributesData) > 0 && a.et.Group.Category.ID != model.EveCategoryStation {
		tabs.Append(container.NewTabItem("Attributes", a.makeAttributesTab()))
	}
	if len(a.fittingData) > 0 {
		tabs.Append(container.NewTabItem("Fittings", a.makeFittingsTab()))
	}
	if len(a.requiredSkills) > 0 {
		tabs.Append(container.NewTabItem("Requirements", a.makeRequirementsTab()))
	}
	if a.isLocation() {
		location := container.NewTabItem("Location", a.makeLocationTab())
		tabs.Append(location)
		tabs.Select(location)
	}
	c := container.NewBorder(top, nil, nil, nil, tabs)
	return c
}

func (a *typeInfoWindow) makeTop() fyne.CanvasObject {
	typeIcon := container.New(&topLeftLayout{})
	if a.et.HasRender() {
		size := 128
		r, err := a.ui.sv.EveImage.InventoryTypeRender(a.et.ID, size)
		if err != nil {
			panic(err)
		}
		render := widgets.NewTappableImage(r, canvas.ImageFillContain, func() {
			w := a.ui.app.NewWindow(a.makeTitle("Render"))
			size := 512
			i := newImageResourceAsync(resourceQuestionmarkSvg, func() (fyne.Resource, error) {
				return a.ui.sv.EveImage.InventoryTypeRender(a.et.ID, size)
			})
			i.FillMode = canvas.ImageFillContain
			s := float32(size) / w.Canvas().Scale()
			w.Resize(fyne.Size{Width: s, Height: s})
			w.SetContent(i)
			w.Show()
		})
		s := float32(size) * 1.3 / a.ui.window.Canvas().Scale()
		render.SetMinSize(fyne.Size{Width: s, Height: s})
		typeIcon.Add(render)
		if a.metaLevel > 4 {
			var n icons.Name
			if a.techLevel == 2 {
				n = icons.Tech2
			} else if a.techLevel == 3 {
				n = icons.Tech3
			} else {
				n = icons.Faction
			}
			marker := canvas.NewImageFromResource(icons.GetResourceByName(n))
			marker.FillMode = canvas.ImageFillOriginal
			typeIcon.Add(marker)
		}
	} else {
		size := 64
		icon := newImageResourceAsync(resourceQuestionmarkSvg, func() (fyne.Resource, error) {
			if a.et.IsSKIN() {
				return resourceSkinicon64pxPng, nil
			} else if a.et.IsBlueprint() {
				return a.ui.sv.EveImage.InventoryTypeBPO(a.et.ID, size)
			} else {
				return a.ui.sv.EveImage.InventoryTypeIcon(a.et.ID, size)
			}
		})
		icon.FillMode = canvas.ImageFillContain
		s := float32(size) * 1.3 / a.ui.window.Canvas().Scale()
		icon.SetMinSize(fyne.Size{Width: s, Height: s})
		typeIcon.Add(icon)
	}
	ownerIcon := canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
	ownerIcon.FillMode = canvas.ImageFillOriginal
	ownerName := widget.NewLabel("")
	if a.owner != nil {
		refreshImageResourceAsync(ownerIcon, func() (fyne.Resource, error) {
			switch a.owner.Category {
			case model.EveEntityCharacter:
				return a.ui.sv.EveImage.CharacterPortrait(a.owner.ID, 32)
			case model.EveEntityCorporation:
				return a.ui.sv.EveImage.CorporationLogo(a.owner.ID, 32)
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
	if a.isLocation() {
		title.SetText(a.location.Name)
	} else {
		title.Hide()
	}
	return container.NewHBox(
		typeIcon,
		container.NewVBox(
			title,
			container.NewHBox(ownerIcon, ownerName, checkIcon)))
}

func (a *typeInfoWindow) makeDescriptionTab() fyne.CanvasObject {
	description := widget.NewLabel(a.et.DescriptionPlain())
	description.Wrapping = fyne.TextWrapWord
	return container.NewVScroll(description)
}

func (a *typeInfoWindow) makeAttributesTab() fyne.CanvasObject {
	list := widget.NewList(
		func() int {
			return len(a.attributesData)
		},
		func() fyne.CanvasObject {
			return widgets.NewTypeAttributeItem()
		},
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			r := a.attributesData[lii]
			item := co.(*widgets.TypeAttributeItem)
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

func (a *typeInfoWindow) makeFittingsTab() fyne.CanvasObject {
	l := widget.NewList(
		func() int {
			return len(a.fittingData)
		},
		func() fyne.CanvasObject {
			return widgets.NewTypeAttributeItem()
		},
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			r := a.fittingData[lii]
			item := co.(*widgets.TypeAttributeItem)
			item.SetRegular(r.icon, r.label, r.value)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *typeInfoWindow) makeRequirementsTab() fyne.CanvasObject {
	l := widget.NewList(
		func() int {
			return len(a.requiredSkills)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Placeholder"),
				layout.NewSpacer(),
				widget.NewLabel("Check"),
				widgets.NewSkillLevel(),
				widget.NewIcon(resourceCharacterplaceholder32Jpeg),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			o := a.requiredSkills[id]
			row := co.(*fyne.Container)
			skill := row.Objects[0].(*widget.Label)
			text := row.Objects[2].(*widget.Label)
			level := row.Objects[3].(*widgets.SkillLevel)
			icon := row.Objects[4].(*widget.Icon)
			skill.SetText(skillDisplayName(o.name, o.requiredLevel))
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
		a.ui.showTypeInfoWindow(r.typeID, a.ui.characterID())
		l.UnselectAll()
	}
	return l
}

type infoRow struct {
	label      string
	importance widget.Importance
	value      string
}

func (a *typeInfoWindow) makeLocationTab() fyne.CanvasObject {
	i := systemSecurity2Importance(a.location.SolarSystem.SecurityStatus)
	data := []infoRow{
		{
			label: "Region",
			value: a.location.SolarSystem.Constellation.Region.Name,
		},
		{
			label: "Constellation",
			value: a.location.SolarSystem.Constellation.Name},
		{
			label: "Solar System",
			value: a.location.SolarSystem.Name,
		},
		{
			label:      "Security",
			value:      fmt.Sprintf("%.1f", a.location.SolarSystem.SecurityStatus),
			importance: i,
		},
	}

	l := widget.NewList(
		func() int {
			return len(data)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Label"),
				layout.NewSpacer(),
				widget.NewLabel("Value"),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			o := data[id]
			row := co.(*fyne.Container)
			label := row.Objects[0].(*widget.Label)
			value := row.Objects[2].(*widget.Label)
			label.SetText(o.label)
			value.Importance = o.importance
			value.Text = o.value
			value.Refresh()
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}
