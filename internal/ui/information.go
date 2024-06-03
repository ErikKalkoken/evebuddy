package ui

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/eveonline/icons"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/dustin/go-humanize"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type attributeCategory string

func (ac attributeCategory) DisplayName() string {
	c := cases.Title(language.English)
	return c.String(string(ac))
}

// categories of attributes to display on the attributes tab
const (
	attributeCategoryArmor                 attributeCategory = "armor"
	attributeCategoryCapacitor             attributeCategory = "capacitor"
	attributeCategoryElectronicResistances attributeCategory = "electronic resistances"
	attributeCategoryFitting               attributeCategory = "fitting"
	attributeCategoryMiscellaneous         attributeCategory = "miscellaneous"
	attributeCategoryPropulsion            attributeCategory = "propulsion"
	attributeCategoryShield                attributeCategory = "shield"
	attributeCategoryStructure             attributeCategory = "structure"
	attributeCategoryTargeting             attributeCategory = "targeting"
)

// assignment of attributes to categories
var attributeCategoriesMap = map[attributeCategory][]int32{
	attributeCategoryStructure: {
		model.EveDogmaAttributeStructureHitpoints,
		model.EveDogmaAttributeDroneCapacity,
		model.EveDogmaAttributeDroneBandwidth,
		model.EveDogmaAttributeMass,
		model.EveDogmaAttributeInertiaModifier,
		model.EveDogmaAttributeStructureEMDamageResistance,
		model.EveDogmaAttributeStructureThermalDamageResistance,
		model.EveDogmaAttributeStructureKineticDamageResistance,
		model.EveDogmaAttributeStructureExplosiveDamageResistance,
	},
	attributeCategoryArmor: {
		model.EveDogmaAttributeArmorHitpoints,
		model.EveDogmaAttributeArmorEMDamageResistance,
		model.EveDogmaAttributeArmorThermalDamageResistance,
		model.EveDogmaAttributeArmorKineticDamageResistance,
		model.EveDogmaAttributeArmorExplosiveDamageResistance,
	},
	attributeCategoryShield: {
		model.EveDogmaAttributeShieldCapacity,
		model.EveDogmaAttributeShieldRechargeTime,
		model.EveDogmaAttributeShieldEMDamageResistance,
		model.EveDogmaAttributeShieldThermalDamageResistance,
		model.EveDogmaAttributeShieldKineticDamageResistance,
		model.EveDogmaAttributeShieldExplosiveDamageResistance,
	},
	attributeCategoryElectronicResistances: {
		model.EveDogmaAttributeCapacitorWarfareResistance,
		model.EveDogmaAttributeStasisWebifierResistance,
		model.EveDogmaAttributeWeaponDisruptionResistance,
	},
	attributeCategoryCapacitor: {
		model.EveDogmaAttributeCapacitorCapacity,
		model.EveDogmaAttributeCapacitorRechargeTime,
	},
	attributeCategoryTargeting: {
		model.EveDogmaAttributeMaximumTargetingRange,
		model.EveDogmaAttributeMaximumLockedTargets,
		model.EveDogmaAttributeSignatureRadius,
		model.EveDogmaAttributeScanResolution,
		model.EveDogmaAttributeRADARSensorStrength,
		model.EveDogmaAttributeLadarSensorStrength,
		model.EveDogmaAttributeMagnetometricSensorStrength,
		model.EveDogmaAttributeGravimetricSensorStrength,
	},
	attributeCategoryPropulsion: {
		model.EveDogmaAttributeMaxVelocity,
		model.EveDogmaAttributeWarpSpeedMultiplier,
	},
}

// Substituting icon ID for missing icons
var iconPatches = map[int32]int32{
	model.EveDogmaAttributeWarpSpeedMultiplier: 97,
}

// attribute categories to show for item category
var attributeCategoriesForItemCategory = map[int32][]attributeCategory{
	model.EveCategoryShip: {
		attributeCategoryStructure,
		attributeCategoryArmor,
		attributeCategoryShield,
		attributeCategoryElectronicResistances,
		attributeCategoryCapacitor,
		attributeCategoryTargeting,
		attributeCategoryPropulsion,
	},
}

type infoWindow struct {
	content fyne.CanvasObject
	ui      *ui
	et      *model.EveType
	window  fyne.Window
}

func (u *ui) showTypeWindow(typeID int32) {
	iw, err := u.newInfoWindow(typeID)
	if err != nil {
		panic(err)
	}
	w := u.app.NewWindow(iw.makeTitle("Information"))
	w.SetContent(iw.content)
	w.Resize(fyne.Size{Width: 500, Height: 500})
	w.Show()
	iw.window = w
}

func (u *ui) newInfoWindow(typeID int32) (*infoWindow, error) {
	et, err := u.sv.EveUniverse.GetEveType(context.Background(), typeID)
	if err != nil {
		return nil, err
	}
	a := &infoWindow{ui: u, et: et}
	a.content = a.makeContent()
	return a, nil
}

func (a *infoWindow) makeTitle(suffix string) string {
	return fmt.Sprintf("%s (%s): %s", a.et.Name, a.et.Group.Name, suffix)
}

func (a *infoWindow) makeContent() fyne.CanvasObject {
	top := a.makeTop()

	description := widget.NewLabel(a.et.DescriptionPlain())
	description.Wrapping = fyne.TextWrapWord
	tabs := container.NewAppTabs(
		container.NewTabItem("Traits", widget.NewLabel("PLACEHOLDER")),
		container.NewTabItem("Description", container.NewVScroll(description)),
		container.NewTabItem("Attributes", a.makeAttributesTab()),
	)
	tabs.SelectIndex(2)

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
	return container.NewHBox(image, b)
}

type row struct {
	icon    fyne.Resource
	label   string
	value   string
	isTitle bool
}

func (a *infoWindow) makeAttributesTab() fyne.CanvasObject {
	data, err := a.prepareData()
	if err != nil {
		panic(err)
	}
	list := widget.NewList(
		func() int {
			return len(data)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(resourceCharacterplaceholder32Jpeg),
				widget.NewLabel("Placeholder"),
				layout.NewSpacer(),
				widget.NewLabel("999.999 m3"))
		},
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			r := data[lii]
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

func (a *infoWindow) prepareData() ([]row, error) {
	data := make([]row, 0)
	oo, err := a.ui.sv.EveUniverse.ListEveTypeDogmaAttributesForType(context.Background(), a.et.ID)
	if err != nil {
		return nil, err
	}
	m := make(map[int32]*model.EveDogmaAttributeForType)
	for _, o := range oo {
		m[o.DogmaAttribute.ID] = o
	}

	droneCapacity, ok := m[model.EveDogmaAttributeDroneCapacity]
	hasDrones := ok && droneCapacity.Value > 0

	acs, ok := attributeCategoriesForItemCategory[a.et.Group.Category.ID]
	if ok {
		for _, ac := range acs {
			data = append(data, row{label: ac.DisplayName(), isTitle: true})
			for _, a := range attributeCategoriesMap[ac] {
				o, ok := m[a]
				if !ok {
					continue
				}
				switch a {
				case model.EveDogmaAttributeDroneCapacity, model.EveDogmaAttributeDroneBandwidth:
					if !hasDrones {
						continue
					}
				}
				iconID := o.DogmaAttribute.IconID
				newIconID, ok := iconPatches[o.DogmaAttribute.ID]
				if ok {
					iconID = newIconID
					fmt.Println(iconID)
				}
				r, _ := icons.GetResource(iconID)
				data = append(data, row{
					icon:  r,
					label: o.DogmaAttribute.DisplayName,
					value: formatAttributeValue(o.Value, o.DogmaAttribute.UnitID),
				})
			}
		}
	}
	return data, nil
}

// formatAttributeValue returns the formatted value of a dogma attribute.
func formatAttributeValue(value float32, unit int32) string {
	defaultFormatter := func(v float32) string {
		return humanize.Commaf(float64(v))
	}
	now := time.Now()
	switch unit {
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
	case model.EveUnitLength:
		return fmt.Sprintf("%s m", defaultFormatter(value))
	case model.EveUnitMass:
		return fmt.Sprintf("%s kg", defaultFormatter(value))
	case model.EveUnitMillimeters:
		return fmt.Sprintf("%s mm", defaultFormatter(value))
	case model.EveUnitMilliseconds:
		return humanize.RelTime(now, now.Add(time.Duration(value)*time.Millisecond), "", "")
	case model.EveUnitMultiplier:
		return fmt.Sprintf("%.3f x", value)
	case model.EveUnitPercentage:
		return fmt.Sprintf("%.0f%%", value*100)
	case model.EveUnitInverseAbsolutePercent:
		return fmt.Sprintf("%.0f%%", (1-value)*100)
	case model.EveUnitVolume:
		return fmt.Sprintf("%s m3", defaultFormatter(value))
	case model.EveUnitNone:
		return defaultFormatter(value)
	}
	return fmt.Sprintf("%s ???", defaultFormatter(value))
}
