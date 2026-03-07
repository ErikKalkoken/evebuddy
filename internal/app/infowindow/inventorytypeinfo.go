package infowindow

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"log/slog"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/commonui"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// inventoryTypeInfo displays information about Eve Online inventory types.
type inventoryTypeInfo struct {
	widget.BaseWidget
	baseInfo

	characterIcon    *canvas.Image
	characterID      int64
	characterName    *widget.Hyperlink
	checkIcon        *widget.Icon
	description      *widget.Label
	eveMarketBrowser *fyne.Container
	janice           *fyne.Container
	setTitle         func(string) // for setting the title during update
	tabs             *container.AppTabs
	typeIcon         *xwidget.TappableImage
	typeID           int64
}

func newInventoryTypeInfo(iw *InfoWindow, typeID, characterID int64) *inventoryTypeInfo {
	typeIcon := xwidget.NewTappableImage(icons.BlankSvg, nil)
	typeIcon.SetFillMode(canvas.ImageFillContain)
	typeIcon.SetMinSize(fyne.NewSquareSize(logoUnitSize))
	a := &inventoryTypeInfo{
		characterIcon: xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		characterID:   characterID,
		checkIcon:     widget.NewIcon(icons.BlankSvg),
		description:   newLabelWithWrapAndSelectable(""),
		typeIcon:      typeIcon,
		typeID:        typeID,
	}
	a.initBase(iw)
	a.ExtendBaseWidget(a)

	a.checkIcon.Hide()
	a.characterIcon.Hide()
	a.characterName = widget.NewHyperlink("", nil)
	a.characterName.Wrapping = fyne.TextWrapWord
	a.characterName.Hide()

	emb := xwidget.NewTappableIcon(icons.EvemarketbrowserJpg, func() {
		a.iw.openURL(fmt.Sprintf("https://evemarketbrowser.com/region/0/type/%d", a.typeID))
	})
	emb.SetToolTip("Show on evemarketbrowser.com")
	a.eveMarketBrowser = container.NewStack(canvas.NewRectangle(theme.Color(theme.ColorNameButton)), emb)
	a.eveMarketBrowser.Hide()

	janice := xwidget.NewTappableIcon(icons.JanicePng, func() {
		a.iw.openURL(fmt.Sprintf("https://janice.e-351.com/i/%d", a.typeID))
	})
	janice.SetToolTip("Show on janice.e-351.com")
	a.janice = container.NewStack(canvas.NewRectangle(color.White), janice)
	a.janice.Hide()

	a.tabs = container.NewAppTabs(container.NewTabItem("Description", container.NewVScroll(a.description)))
	return a
}

func (a *inventoryTypeInfo) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			container.NewPadded(a.typeIcon),
			container.New(
				layout.NewCustomPaddedHBoxLayout(3*theme.Padding()),
				layout.NewSpacer(),
				a.eveMarketBrowser,
				a.janice,
				layout.NewSpacer(),
			),
		),
		nil,
		container.NewVBox(
			a.name,
			container.NewBorder(
				nil,
				nil,
				container.NewHBox(a.checkIcon, a.characterIcon),
				nil,
				a.characterName,
			)),
	)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *inventoryTypeInfo) update(ctx context.Context) error {
	et, err := a.iw.s.EVEUniverse().GetOrCreateTypeESI(ctx, a.typeID)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(et.Name)
		a.setTitle(et.Group.Name)
		if et.IsTradeable() {
			a.eveMarketBrowser.Show()
			a.janice.Show()
		}
		s := et.DescriptionPlain()
		if s == "" {
			s = et.Name
		}
		a.description.SetText(s)
	})
	fyne.Do(func() {
		if et.IsSKIN() {
			a.iw.s.EVEImage().InventoryTypeSKINAsync(et.ID, app.IconPixelSize, func(r fyne.Resource) {
				a.typeIcon.SetResource(r)
			})
		} else if et.IsBlueprint() {
			a.iw.s.EVEImage().InventoryTypeBPOAsync(et.ID, app.IconPixelSize, func(r fyne.Resource) {
				a.typeIcon.SetResource(r)
			})
		} else {
			a.iw.s.EVEImage().InventoryTypeIconAsync(et.ID, app.IconPixelSize, func(r fyne.Resource) {
				a.typeIcon.SetResource(r)
			})
		}
	})
	if et.HasRender() {
		a.typeIcon.OnTapped = func() {
			a.iw.showZoomWindow(et.Name, a.typeID, a.iw.s.EVEImage().InventoryTypeRenderAsync, a.iw.w)
		}
	}

	var character *app.EveEntity
	if a.characterID != 0 {
		ee, err := a.iw.s.EVEUniverse().GetOrCreateEntityESI(ctx, a.characterID)
		if err != nil {
			return err
		}
		character = ee
		fyne.Do(func() {
			a.iw.s.EVEImage().CharacterPortraitAsync(character.ID, app.IconPixelSize, func(r fyne.Resource) {
				a.characterIcon.Resource = r
				a.characterIcon.Refresh()
			})
			a.characterIcon.Show()
			a.characterName.OnTapped = func() {
				a.iw.ShowEntity(character)
			}
			a.characterName.SetText(character.Name)
			a.characterName.Show()
		})
	}

	oo, err := a.iw.s.EVEUniverse().ListTypeDogmaAttributesForType(ctx, et.ID)
	if err != nil {
		return err
	}
	dogmaAttributes := make(map[int64]*app.EveTypeDogmaAttribute)
	for _, o := range oo {
		dogmaAttributes[o.DogmaAttribute.ID] = o
	}

	var requiredSkills []requiredSkill
	if a.characterID != 0 {
		skills, err := a.calcRequiredSkills(ctx, a.characterID, dogmaAttributes)
		if err != nil {
			return err
		}
		requiredSkills = skills
	}
	hasRequiredSkills := true
	for _, o := range requiredSkills {
		if o.requiredLevel > o.activeLevel {
			hasRequiredSkills = false
			break
		}
	}
	if character != nil && character.IsCharacter() && len(requiredSkills) > 0 {
		fyne.Do(func() {
			a.checkIcon.SetResource(boolIconResource(hasRequiredSkills))
			a.checkIcon.Show()
		})
	}

	// tabs
	attributeTab := a.makeAttributeTab(ctx, dogmaAttributes, et)
	if attributeTab != nil {
		fyne.Do(func() {
			a.tabs.Append(attributeTab)
		})
	}
	fittingTab := a.makeFittingTab(ctx, dogmaAttributes)
	if fittingTab != nil {
		fyne.Do(func() {
			a.tabs.Append(fittingTab)
		})
	}
	requirementsTab := a.makeRequirementsTab(requiredSkills)
	if requirementsTab != nil {
		fyne.Do(func() {
			a.tabs.Append(requirementsTab)
		})
	}
	marketTab := a.makeMarketTab(ctx, et)
	if marketTab != nil {
		fyne.Do(func() {
			a.tabs.Append(marketTab)
		})
	}

	// Set initial tab
	fyne.Do(func() {
		if marketTab != nil && a.iw.s.Settings().PreferMarketTab() {
			a.tabs.Select(marketTab)
		} else if requirementsTab != nil && et.Group.Category.ID == app.EveCategorySkill {
			a.tabs.Select(requirementsTab)
		} else if attributeTab != nil &&
			set.Of[int64](
				app.EveCategoryDrone,
				app.EveCategoryFighter,
				app.EveCategoryOrbitals,
				app.EveCategoryShip,
				app.EveCategoryStructure,
			).Contains(et.Group.Category.ID) {
			a.tabs.Select(attributeTab)
		}
		a.tabs.Refresh()
	})
	return nil
}

func (a *inventoryTypeInfo) makeAttributeTab(ctx context.Context, dogmaAttributes map[int64]*app.EveTypeDogmaAttribute, et *app.EveType) *container.TabItem {
	attributes := a.calcAttributesData(ctx, et, dogmaAttributes)
	if len(attributes) == 0 {
		return nil
	}
	list := widget.NewList(
		func() int {
			return len(attributes)
		},
		func() fyne.CanvasObject {
			return newTypeAttributeItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(attributes) {
				return
			}
			r := attributes[id]
			item := co.(*typeAttributeItem)
			if r.isTitle {
				item.SetTitle(r.label)
			} else {
				item.SetRegular(r.icon, r.label, r.value)
			}
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if id >= len(attributes) {
			return
		}
		r := attributes[id]
		if r.action != nil {
			r.action(r.value)
		}
	}
	return container.NewTabItem("Attributes", list)
}

// attributeGroup represents a group of dogma attributes.
//
// Used for rendering the attributes and fitting tabs for inventory type info
type attributeGroup string

func (ag attributeGroup) DisplayName() string {
	return xstrings.Title(string(ag))
}

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
var attributeGroupsMap = map[attributeGroup][]int64{
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
		app.EveDogmaAttributeServiceSlots,
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

type typeAttributeRow struct {
	icon    fyne.Resource
	label   string
	value   string
	isTitle bool
	action  func(v string)
}

func (a *inventoryTypeInfo) calcAttributesData(ctx context.Context, et *app.EveType, attributes map[int64]*app.EveTypeDogmaAttribute) []typeAttributeRow {
	droneCapacity, ok := attributes[app.EveDogmaAttributeDroneCapacity]
	hasDrones := ok && droneCapacity.Value > 0

	jumpDrive, ok := attributes[app.EveDogmaAttributeOnboardJumpDrive]
	hasJumpDrive := ok && jumpDrive.Value == 1.0

	groupedRows := make(map[attributeGroup][]typeAttributeRow)

	for _, ag := range attributeGroups {
		var attributeSelection []*app.EveTypeDogmaAttribute
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
			v, substituteIcon := a.iw.s.EVEUniverse().FormatDogmaValue(ctx, value, o.DogmaAttribute.Unit)
			var iconID int64
			if substituteIcon != 0 {
				iconID = substituteIcon
			} else {
				iconID = o.DogmaAttribute.IconID.ValueOrZero()
			}
			r, _ := eveicon.FromID(iconID)
			groupedRows[ag] = append(groupedRows[ag], typeAttributeRow{
				icon:  r,
				label: o.DogmaAttribute.DisplayName.ValueOrZero(),
				value: v,
			})
		}
	}
	var rows []typeAttributeRow
	if v, ok := et.Volume.Value(); ok {
		value, _ := a.iw.s.EVEUniverse().FormatDogmaValue(ctx, v, app.EveUnitVolume)
		if pv, ok := et.PackagedVolume.Value(); ok && !optional.Equal(et.Volume, et.PackagedVolume) {
			s, _ := a.iw.s.EVEUniverse().FormatDogmaValue(ctx, pv, app.EveUnitVolume)
			value += fmt.Sprintf(" (%s Packaged)", s)
		}
		r := typeAttributeRow{
			icon:  eveicon.FromName(eveicon.Structure),
			label: "Volume",
			value: value,
		}
		var ag attributeGroup
		if len(groupedRows[attributeGroupStructure]) > 0 {
			ag = attributeGroupStructure
		} else {
			ag = attributeGroupMiscellaneous
		}
		groupedRows[ag] = append([]typeAttributeRow{r}, groupedRows[ag]...)
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
				rows = append(rows, typeAttributeRow{label: ag.DisplayName(), isTitle: true})
			}
			rows = append(rows, groupedRows[ag]...)
		}
	}
	if app.IsDeveloperMode() {
		rows = append(rows, typeAttributeRow{label: "Developer Mode", isTitle: true})
		rows = append(rows, typeAttributeRow{
			label: "EVE ID",
			value: fmt.Sprint(et.ID),
			action: func(v string) {
				fyne.CurrentApp().Clipboard().SetContent(v)
			},
		})
	}
	return rows
}

func (a *inventoryTypeInfo) makeFittingTab(ctx context.Context, dogmaAttributes map[int64]*app.EveTypeDogmaAttribute) *container.TabItem {
	fittingData := a.calcFittingData(ctx, dogmaAttributes)
	if len(fittingData) == 0 {
		return nil
	}
	list := widget.NewList(
		func() int {
			return len(fittingData)
		},
		func() fyne.CanvasObject {
			return newTypeAttributeItem()
		},
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			r := fittingData[lii]
			item := co.(*typeAttributeItem)
			item.SetRegular(r.icon, r.label, r.value)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		list.UnselectAll()
	}
	return container.NewTabItem("Fittings", list)
}

func (a *inventoryTypeInfo) calcFittingData(ctx context.Context, dogmaAttributes map[int64]*app.EveTypeDogmaAttribute) []typeAttributeRow {
	var data []typeAttributeRow
	for _, da := range attributeGroupsMap[attributeGroupFitting] {
		o, ok := dogmaAttributes[da]
		if !ok {
			continue
		}
		r, _ := eveicon.FromID(o.DogmaAttribute.IconID.ValueOrZero())
		v, _ := a.iw.s.EVEUniverse().FormatDogmaValue(ctx, o.Value, o.DogmaAttribute.Unit)
		data = append(data, typeAttributeRow{
			icon:  r,
			label: o.DogmaAttribute.DisplayName.ValueOrZero(),
			value: v,
		})
	}
	return data
}

func (a *inventoryTypeInfo) makeRequirementsTab(requiredSkills []requiredSkill) *container.TabItem {
	if len(requiredSkills) == 0 {
		return nil
	}
	list := widget.NewList(
		func() int {
			return len(requiredSkills)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Placeholder"),
				layout.NewSpacer(),
				widget.NewLabel("Check"),
				commonui.NewSkillLevel(),
				widget.NewIcon(icons.QuestionmarkSvg),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			o := requiredSkills[id]
			row := co.(*fyne.Container).Objects
			skill := row[0].(*widget.Label)
			text := row[2].(*widget.Label)
			level := row[3].(*commonui.SkillLevel)
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
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		r := requiredSkills[id]
		a.iw.show(infoInventoryType, r.typeID)
	}
	return container.NewTabItem("Requirements", list)
}

const (
	priceFormat    = "#,###.##"
	currencySuffix = " ISK"
)

func (a *inventoryTypeInfo) makeMarketTab(ctx context.Context, et *app.EveType) *container.TabItem {
	if !et.IsTradeable() {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	a.iw.onClosedFuncs = append(a.iw.onClosedFuncs, cancel)
	marketTab := container.NewTabItem("Market", widget.NewLabel("Fetching prices..."))
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
	L:
		for {
			var items []attributeItem

			var averagePrice string
			p, err := a.iw.s.EVEUniverse().MarketPrice(ctx, et.ID)
			if err != nil {
				slog.Error("average price", "typeID", et.ID, "error", err)
				averagePrice = "ERROR: " + app.ErrorDisplay(err)
			} else {
				averagePrice = p.StringFunc("?", func(v float64) string {
					return humanize.FormatFloat(priceFormat, v) + currencySuffix
				})
			}
			items = append(items, newAttributeItem("Average price", averagePrice))
			it, err := a.addJanicePriceItems(ctx, et.ID)
			if err != nil {
				slog.Error("janice pricer", "typeID", et.ID, "error", err)
				s := "ERROR: " + app.ErrorDisplay(err)
				items = append(items, newAttributeItem("Janice prices", s))
			} else {
				items = slices.Concat(items, it)
			}
			items = slices.Concat(items)

			c := newAttributeList(a.iw, items...)
			fyne.Do(func() {
				marketTab.Content = c
				a.tabs.Refresh()
			})
			select {
			case <-ctx.Done():
				break L
			case <-ticker.C:
			}
		}
		slog.Debug("market update type for canceled", "name", et.Name)
	}()
	return marketTab
}

func (a *inventoryTypeInfo) addJanicePriceItems(ctx context.Context, typeID int64) ([]attributeItem, error) {
	j, err := a.iw.s.Janice().FetchPrices(ctx, typeID)
	if err != nil {
		return nil, fmt.Errorf("fetch prices from janice: %w", err)
	}

	items := []attributeItem{
		newAttributeItem("Jita sell price", humanize.FormatFloat(
			priceFormat,
			j.ImmediatePrices.SellPrice)+currencySuffix,
		),
		newAttributeItem("Jita buy price", humanize.FormatFloat(
			priceFormat,
			j.ImmediatePrices.BuyPrice)+currencySuffix,
		),
		newAttributeItem("Jita sell volume", ihumanize.Comma(j.SellVolume)),
		newAttributeItem("Jita buy volume", ihumanize.Comma(j.BuyVolume)),
	}
	return items, nil
}

type requiredSkill struct {
	rank          int
	name          string
	typeID        int64
	activeLevel   int64
	requiredLevel int64
	trainedLevel  int64
}

func (a *inventoryTypeInfo) calcRequiredSkills(ctx context.Context, characterID int64, attributes map[int64]*app.EveTypeDogmaAttribute) ([]requiredSkill, error) {
	var skills []requiredSkill
	skillAttributes := []struct {
		id    int64
		level int64
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
		typeID := int64(daID.Value)
		daLevel, ok := attributes[x.level]
		if !ok {
			continue
		}
		requiredLevel := int64(daLevel.Value)
		et, err := a.iw.s.EVEUniverse().GetType(ctx, typeID)
		if err != nil {
			return nil, err
		}
		skill := requiredSkill{
			rank:          i + 1,
			requiredLevel: requiredLevel,
			name:          et.Name,
			typeID:        typeID,
		}
		cs, err := a.iw.s.Character().GetSkill(ctx, characterID, typeID)
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

// The typeAttributeItem widget is used to render items on the type info window.
type typeAttributeItem struct {
	widget.BaseWidget
	icon  *widget.Icon
	label *widget.Label
	value *widget.Label
}

func newTypeAttributeItem() *typeAttributeItem {
	w := &typeAttributeItem{
		icon:  widget.NewIcon(theme.QuestionIcon()),
		label: widget.NewLabel(""),
		value: widget.NewLabel(""),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *typeAttributeItem) SetRegular(icon fyne.Resource, label, value string) {
	w.label.TextStyle.Bold = false
	w.label.Importance = widget.MediumImportance
	w.label.Text = label
	w.label.Refresh()
	w.icon.SetResource(icon)
	w.icon.Show()
	w.value.SetText(value)
	w.value.Show()
}

func (w *typeAttributeItem) SetTitle(label string) {
	w.label.TextStyle.Bold = true
	w.label.Text = label
	w.label.Refresh()
	w.icon.Hide()
	w.value.Hide()
}

func (w *typeAttributeItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewHBox(w.icon, w.label, layout.NewSpacer(), w.value)
	return widget.NewSimpleRenderer(c)
}
