package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/eveonline/icons"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
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
}

// Substituting icon ID for missing icons
var iconPatches = map[int32]int32{
	model.EveDogmaAttributeJumpDriveFuelNeed: icons.HeliumIsotopes,
}

type infoWindow struct {
	attributesData []attributesRow
	content        fyne.CanvasObject
	characterID    int32
	et             *model.EveType
	fittingData    []attributesRow
	skills         []*model.CharacterShipSkill
	ui             *ui
	window         fyne.Window
}

func (u *ui) showTypeWindow(typeID int32) {
	iw, err := u.newInfoWindow(typeID)
	if err != nil {
		u.showErrorDialog("Failed to open info window", err)
		return
	}
	w := u.app.NewWindow(iw.makeTitle("Information"))
	w.SetContent(iw.content)
	w.Resize(fyne.Size{Width: 500, Height: 500})
	w.Show()
	iw.window = w
}

func (u *ui) newInfoWindow(typeID int32) (*infoWindow, error) {
	ctx := context.Background()
	et, err := u.sv.EveUniverse.GetEveType(ctx, typeID)
	if err != nil {
		return nil, err
	}
	oo, err := u.sv.EveUniverse.ListEveTypeDogmaAttributesForType(ctx, typeID)
	if err != nil {
		return nil, err
	}
	m := make(map[int32]*model.EveDogmaAttributeForType)
	for _, o := range oo {
		m[o.DogmaAttribute.ID] = o
	}
	attributesData := calcAttributesData(ctx, u.sv, m)
	fittingData := calcFittingData(ctx, u.sv, m)

	characterID := u.currentCharID()
	skills, err := u.sv.Characters.ListCharacterShipSkills(ctx, characterID, et.ID)
	if err != nil {
		return nil, err
	}
	a := &infoWindow{
		attributesData: attributesData,
		characterID:    characterID,
		et:             et,
		fittingData:    fittingData,
		skills:         skills,
		ui:             u,
	}
	a.content = a.makeContent()
	return a, nil
}

func calcAttributesData(ctx context.Context, sv *service.Service, attributes map[int32]*model.EveDogmaAttributeForType) []attributesRow {
	data := make([]attributesRow, 0)

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
			r, _ := icons.GetResource(iconID)
			data = append(data, attributesRow{
				icon:  r,
				label: o.DogmaAttribute.DisplayName,
				value: formatAttributeValue(ctx, sv, value, o.DogmaAttribute.UnitID),
			})
		}
	}
	return data
}

func calcFittingData(ctx context.Context, sv *service.Service, attributes map[int32]*model.EveDogmaAttributeForType) []attributesRow {
	data := make([]attributesRow, 0)
	for _, da := range attributeGroupsMap[attributeGroupFitting] {
		o, ok := attributes[da]
		if !ok {
			continue
		}
		iconID := o.DogmaAttribute.IconID
		r, _ := icons.GetResource(iconID)
		data = append(data, attributesRow{
			icon:  r,
			label: o.DogmaAttribute.DisplayName,
			value: formatAttributeValue(ctx, sv, o.Value, o.DogmaAttribute.UnitID),
		})
	}
	return data
}

func (a *infoWindow) makeTitle(suffix string) string {
	return fmt.Sprintf("%s (%s): %s", a.et.Name, a.et.Group.Name, suffix)
}

func (a *infoWindow) makeContent() fyne.CanvasObject {
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
	if len(a.skills) > 0 {
		tabs.Append(container.NewTabItem("Requirements", a.makeRequirementsTab()))
	}
	c := container.NewBorder(top, nil, nil, nil, tabs)
	return c
}

func (a *infoWindow) makeTop() fyne.CanvasObject {
	image := newImageResourceAsync(resourceQuestionmarkSvg, func() (fyne.Resource, error) {
		if a.et.IsSKIN() {
			return resourceSkinicon64pxPng, nil
		} else if a.et.IsBlueprint() {
			return a.ui.sv.EveImage.InventoryTypeBPO(a.et.ID, 64)
		} else {
			return a.ui.sv.EveImage.InventoryTypeIcon(a.et.ID, 64)
		}
	})
	image.FillMode = canvas.ImageFillOriginal
	b := widget.NewButton("Show", func() {
		w := a.ui.app.NewWindow(a.makeTitle("Render"))
		i := newImageResourceAsync(resourceQuestionmarkSvg, func() (fyne.Resource, error) {
			return a.ui.sv.EveImage.InventoryTypeRender(a.et.ID, 512)
		})
		i.FillMode = canvas.ImageFillContain
		s := float32(512) / w.Canvas().Scale()
		w.Resize(fyne.Size{Width: s, Height: s})
		w.SetContent(i)
		w.Show()
	})
	if a.et.HasRender() {
		b.Enable()
	} else {
		b.Disable()
	}
	canFly := true
	for _, o := range a.skills {
		if !o.ActiveSkillLevel.Valid || o.SkillLevel > uint(o.ActiveSkillLevel.Int) {
			canFly = false
			break
		}
	}
	var icon *widget.Icon
	if canFly {
		icon = widget.NewIcon(theme.NewSuccessThemedResource(theme.ConfirmIcon()))
	} else {
		icon = widget.NewIcon(theme.NewErrorThemedResource(theme.CancelIcon()))
	}
	if len(a.skills) == 0 {
		icon.Hide()
	}
	return container.NewHBox(image, b, icon)
}

func (a *infoWindow) makeDescriptionTab() fyne.CanvasObject {
	description := widget.NewLabel(a.et.DescriptionPlain())
	description.Wrapping = fyne.TextWrapWord
	return container.NewVScroll(description)
}

type attributesRow struct {
	icon    fyne.Resource
	label   string
	value   string
	isTitle bool
}

func (a *infoWindow) makeAttributesTab() fyne.CanvasObject {
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

func (a *infoWindow) makeFittingsTab() fyne.CanvasObject {
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

func (a *infoWindow) makeRequirementsTab() fyne.CanvasObject {
	l := widget.NewList(
		func() int {
			return len(a.skills)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Placeholder"),
				layout.NewSpacer(),
				widget.NewLabel("Check"))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			r := a.skills[id]
			row := co.(*fyne.Container)
			skill := row.Objects[0].(*widget.Label)
			check := row.Objects[2].(*widget.Label)
			skill.SetText(skillDisplayName(r.SkillName, r.SkillLevel))
			var t string
			var i widget.Importance
			if r.ActiveSkillLevel.Valid && uint(r.ActiveSkillLevel.Int) >= r.SkillLevel {
				t = "OK"
				i = widget.SuccessImportance
			} else if !r.ActiveSkillLevel.Valid {
				t = "Skill not injected"
				i = widget.DangerImportance
			} else {
				t = fmt.Sprintf("Current level %s", toRomanLetter(r.ActiveSkillLevel.Int))
				i = widget.WarningImportance
			}
			check.Text = t
			check.Importance = i
			check.Refresh()
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		r := a.skills[id]
		a.ui.showTypeWindow(r.SkillTypeID)
		l.UnselectAll()
	}
	return l
}

// formatAttributeValue returns the formatted value of a dogma attribute.
func formatAttributeValue(ctx context.Context, sv *service.Service, value float32, unit int32) string {
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
		return fmt.Sprintf("%s m", defaultFormatter(value))
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
		et, err := sv.EveUniverse.GetEveType(ctx, int32(value))
		if err != nil {
			go func() {
				_, err := sv.EveUniverse.GetOrCreateEveTypeESI(ctx, int32(value))
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
