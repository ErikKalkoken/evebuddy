package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/eveonline/icons"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service/character"
	"github.com/dustin/go-humanize"
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
		model.EveDogmaAttributeTechLevel,
	},
}

// Substituting icon ID for missing icons
var iconPatches = map[int32]int32{
	model.EveDogmaAttributeJumpDriveFuelNeed: icons.IDHeliumIsotopes,
}

type requiredSkill struct {
	rank          int
	name          string
	typeID        int32
	activeLevel   int
	requiredLevel int
	trainedLevel  int
}

type attributesRow struct {
	icon    fyne.Resource
	label   string
	value   string
	isTitle bool
}

type typeInfoWindow struct {
	attributesData []attributesRow
	content        fyne.CanvasObject
	characterID    int32
	et             *model.EveType
	fittingData    []attributesRow
	requiredSkills []requiredSkill
	ui             *ui
	window         fyne.Window
}

func (u *ui) showTypeInfoWindow(typeID, characterID int32) {
	iw, err := u.newTypeInfoWindow(typeID, characterID)
	if err != nil {
		t := "Failed to open type info window"
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

func (u *ui) newTypeInfoWindow(typeID, characterID int32) (*typeInfoWindow, error) {
	ctx := context.Background()
	et, err := u.sv.EveUniverse.GetEveType(ctx, typeID)
	if err != nil {
		return nil, err
	}
	oo, err := u.sv.EveUniverse.ListEveTypeDogmaAttributesForType(ctx, typeID)
	if err != nil {
		return nil, err
	}
	a := &typeInfoWindow{
		characterID: characterID,
		et:          et,
		ui:          u,
	}
	attributes := make(map[int32]*model.EveDogmaAttributeForType)
	for _, o := range oo {
		attributes[o.DogmaAttribute.ID] = o
	}
	a.attributesData = a.calcAttributesData(ctx, attributes)
	a.fittingData = a.calcFittingData(ctx, attributes)
	skills, err := a.calcRequiredSkills(ctx, characterID, attributes)
	if err != nil {
		return nil, err
	}
	a.requiredSkills = skills
	a.content = a.makeContent()
	return a, nil
}

func (a *typeInfoWindow) calcAttributesData(ctx context.Context, attributes map[int32]*model.EveDogmaAttributeForType) []attributesRow {
	data := make([]attributesRow, 0)

	if a.ui.isDebug {
		data = append(data, attributesRow{label: "DEBUG", isTitle: true})
		data = append(data, attributesRow{label: "Character ID", value: fmt.Sprint(a.characterID)})
		data = append(data, attributesRow{label: "Type ID", value: fmt.Sprint(a.et.ID)})
	}

	droneCapacity, ok := attributes[model.EveDogmaAttributeDroneCapacity]
	hasDrones := ok && droneCapacity.Value > 0

	jumpDrive, ok := attributes[model.EveDogmaAttributeOnboardJumpDrive]
	hasJumpDrive := ok && jumpDrive.Value == 1.0

	for _, ag := range attributeGroups {
		hasData := false
		for _, da := range attributeGroupsMap[ag] {
			_, ok := attributes[da]
			if ok {
				hasData = true
				break
			}
		}
		if !hasData {
			continue
		}
		data = append(data, attributesRow{label: ag.DisplayName(), isTitle: true})
		for _, da := range attributeGroupsMap[ag] {
			o, ok := attributes[da]
			if !ok {
				continue
			}
			value := o.Value
			if ag == attributeGroupElectronicResistances && value == 0 {
				continue
			}
			switch da {
			case model.EveDogmaAttributeCapacity:
				if value == 0 {
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
			case model.EveDogmaAttributeShipWarpSpeed:
				x := attributes[model.EveDogmaAttributeWarpSpeedMultiplier]
				value = value * x.Value
			case model.EveDogmaAttributeSupportFighterSquadronLimit:
				if value == 0 {
					continue
				}
			}
			iconID := o.DogmaAttribute.IconID
			newIconID, ok := iconPatches[o.DogmaAttribute.ID]
			if ok {
				iconID = newIconID
			}
			r, _ := icons.GetResourceByIconID(iconID)
			data = append(data, attributesRow{
				icon:  r,
				label: o.DogmaAttribute.DisplayName,
				value: a.formatAttributeValue(ctx, value, o.DogmaAttribute.UnitID),
			})
		}
	}
	headers := make([]int, 0)
	for i, r := range data {
		if r.isTitle {
			headers = append(headers, i)
		}
	}
	if len(headers) == 1 {
		data = slices.Delete(data, headers[0], headers[0]+1)
	}
	return data
}

func (a *typeInfoWindow) calcFittingData(ctx context.Context, attributes map[int32]*model.EveDogmaAttributeForType) []attributesRow {
	data := make([]attributesRow, 0)
	for _, da := range attributeGroupsMap[attributeGroupFitting] {
		o, ok := attributes[da]
		if !ok {
			continue
		}
		iconID := o.DogmaAttribute.IconID
		r, _ := icons.GetResourceByIconID(iconID)
		data = append(data, attributesRow{
			icon:  r,
			label: o.DogmaAttribute.DisplayName,
			value: a.formatAttributeValue(ctx, o.Value, o.DogmaAttribute.UnitID),
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
	tabs := container.NewAppTabs(
		container.NewTabItem("Description", a.makeDescriptionTab()),
	)
	if len(a.attributesData) > 0 {
		tabs.Append(container.NewTabItem("Attributes", a.makeAttributesTab()))
	}
	if len(a.fittingData) > 0 {
		tabs.Append(container.NewTabItem("Fittings", a.makeFittingsTab()))
	}
	if len(a.requiredSkills) > 0 {
		tabs.Append(container.NewTabItem("Requirements", a.makeRequirementsTab()))
	}
	c := container.NewBorder(top, nil, nil, nil, tabs)
	return c
}

func (a *typeInfoWindow) makeTop() fyne.CanvasObject {
	size := 64
	typeIcon := newImageResourceAsync(resourceQuestionmarkSvg, func() (fyne.Resource, error) {
		if a.et.IsSKIN() {
			return resourceSkinicon64pxPng, nil
		} else if a.et.IsBlueprint() {
			return a.ui.sv.EveImage.InventoryTypeBPO(a.et.ID, size)
		} else {
			return a.ui.sv.EveImage.InventoryTypeIcon(a.et.ID, size)
		}
	})
	typeIcon.FillMode = canvas.ImageFillOriginal

	renderButton := widget.NewButton("Show", func() {
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
	if a.et.HasRender() {
		renderButton.Enable()
	} else {
		renderButton.Disable()
	}
	characterIcon := canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
	characterIcon.FillMode = canvas.ImageFillOriginal
	if a.characterID != 0 {
		refreshImageResourceAsync(characterIcon, func() (fyne.Resource, error) {
			return a.ui.sv.EveImage.CharacterPortrait(a.characterID, 32)
		})
	} else {
		characterIcon.Hide()
	}
	hasRequiredSkills := true
	for _, o := range a.requiredSkills {
		if o.requiredLevel > o.activeLevel {
			hasRequiredSkills = false
			break
		}
	}
	var checkIcon *widget.Icon
	if hasRequiredSkills {
		checkIcon = widget.NewIcon(theme.NewSuccessThemedResource(theme.ConfirmIcon()))
	} else {
		checkIcon = widget.NewIcon(theme.NewErrorThemedResource(theme.CancelIcon()))
	}
	if a.characterID == 0 || len(a.requiredSkills) == 0 {
		checkIcon.Hide()
	}
	return container.NewHBox(typeIcon, renderButton, characterIcon, checkIcon)
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
			return container.NewHBox(
				widget.NewIcon(resourceCharacterplaceholder32Jpeg),
				widget.NewLabel("Placeholder"),
				layout.NewSpacer(),
				widget.NewLabel("999.999 m3"))
		},
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			r := a.attributesData[lii]
			row := co.(*fyne.Container)
			icon := row.Objects[0].(*widget.Icon)
			label := row.Objects[1].(*widget.Label)
			value := row.Objects[3].(*widget.Label)
			if r.isTitle {
				label.TextStyle.Bold = true
				label.Importance = widget.HighImportance
				label.Text = r.label
				label.Refresh()
				icon.Hide()
				value.Hide()
			} else {
				label.TextStyle.Bold = false
				label.Importance = widget.MediumImportance
				label.Text = r.label
				label.Refresh()
				icon.SetResource(r.icon)
				icon.Show()
				value.SetText(r.value)
				value.Show()
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
			return container.NewHBox(
				widget.NewIcon(resourceCharacterplaceholder32Jpeg),
				widget.NewLabel("Placeholder"),
				layout.NewSpacer(),
				widget.NewLabel("999.999 m3"))
		},
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			r := a.fittingData[lii]
			row := co.(*fyne.Container)
			icon := row.Objects[0].(*widget.Icon)
			label := row.Objects[1].(*widget.Label)
			value := row.Objects[3].(*widget.Label)
			label.TextStyle.Bold = false
			label.Importance = widget.MediumImportance
			label.Text = r.label
			label.Refresh()
			icon.SetResource(r.icon)
			value.SetText(r.value)
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
				widget.NewLabel("Check"))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			o := a.requiredSkills[id]
			row := co.(*fyne.Container)
			skill := row.Objects[0].(*widget.Label)
			check := row.Objects[2].(*widget.Label)
			skill.SetText(skillDisplayName(o.name, o.requiredLevel))
			var t string
			var i widget.Importance
			if o.activeLevel == 0 {
				t = "Skill not injected"
				i = widget.DangerImportance

			} else if o.activeLevel >= o.requiredLevel {
				t = "OK"
				i = widget.SuccessImportance
			} else {
				t = fmt.Sprintf("Current level %s", toRomanLetter(o.activeLevel))
				i = widget.WarningImportance
			}
			check.Text = t
			check.Importance = i
			check.Refresh()
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		r := a.requiredSkills[id]
		a.ui.showTypeInfoWindow(r.typeID, a.ui.currentCharID())
		l.UnselectAll()
	}
	return l
}

// formatAttributeValue returns the formatted value of a dogma attribute.
func (a *typeInfoWindow) formatAttributeValue(ctx context.Context, value float32, unit int32) string {
	if a.ui.isDebug {
		return fmt.Sprintf("%v", value)
	}
	defaultFormatter := func(v float32) string {
		return humanize.Commaf(float64(v))
	}
	now := time.Now()
	switch unit {
	case model.EveUnitAbsolutePercent:
		return fmt.Sprintf("%.0f%%", value*100)
	case model.EveUnitAcceleration:
		return fmt.Sprintf("%s m/sec", defaultFormatter(value))
	case model.EveUnitAttributePoints:
		return fmt.Sprintf("%s points", defaultFormatter(value))
	case model.EveUnitCapacitorUnits:
		return fmt.Sprintf("%.1f GJ", value)
	case model.EveUnitDroneBandwidth:
		return fmt.Sprintf("%s Mbit/s", defaultFormatter(value))
	case model.EveUnitHitpoints:
		return fmt.Sprintf("%s HP", defaultFormatter(value))
	case model.EveUnitInverseAbsolutePercent:
		return fmt.Sprintf("%.0f%%", (1-value)*100)
	case model.EveUnitLength:
		if value > 1000 {
			return fmt.Sprintf("%s km", defaultFormatter(value/float32(1000)))
		} else {
			return fmt.Sprintf("%s m", defaultFormatter(value))
		}
	case model.EveUnitLevel:
		return fmt.Sprintf("Level %s", defaultFormatter(value))
	case model.EveUnitLightYear:
		return fmt.Sprintf("%.1f LY", value)
	case model.EveUnitMass:
		return fmt.Sprintf("%s kg", defaultFormatter(value))
	case model.EveUnitMegaWatts:
		return fmt.Sprintf("%s MW", defaultFormatter(value))
	case model.EveUnitMillimeters:
		return fmt.Sprintf("%s mm", defaultFormatter(value))
	case model.EveUnitMilliseconds:
		return humanize.RelTime(now, now.Add(time.Duration(value)*time.Millisecond), "", "")
	case model.EveUnitMultiplier:
		return fmt.Sprintf("%.3f x", value)
	case model.EveUnitPercentage:
		return fmt.Sprintf("%.0f%%", value*100)
	case model.EveUnitTeraflops:
		return fmt.Sprintf("%s tf", defaultFormatter(value))
	case model.EveUnitVolume:
		return fmt.Sprintf("%s m3", defaultFormatter(value))
	case model.EveUnitWarpSpeed:
		return fmt.Sprintf("%s AU/s", defaultFormatter(value))
	case model.EveUnitTypeID:
		et, err := a.ui.sv.EveUniverse.GetEveType(ctx, int32(value))
		if err != nil {
			go func() {
				_, err := a.ui.sv.EveUniverse.GetOrCreateEveTypeESI(ctx, int32(value))
				if err != nil {
					slog.Error("Failed to fetch type from ESI", "typeID", value, "err", err)
				}
			}()
			return "?"
		}
		return et.Name
	case model.EveUnitUnits:
		return fmt.Sprintf("%s units", defaultFormatter(value))
	case model.EveUnitNone, model.EveUnitHardpoints, model.EveUnitFittingSlots:
		return defaultFormatter(value)
	}
	return fmt.Sprintf("%s ???", defaultFormatter(value))
}
