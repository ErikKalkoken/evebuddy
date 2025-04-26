package ui

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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/set"
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

	iw *InfoWindow
}

func NewInventoryTypeInfo(iw *InfoWindow, typeID, characterID int32) (*inventoryTypeInfo, error) {
	ctx := context.Background()
	a := &inventoryTypeInfo{iw: iw, id: typeID}
	a.ExtendBaseWidget(a)
	et, err := iw.u.eus.GetOrCreateTypeESI(ctx, typeID)
	if err != nil {
		return nil, err
	}
	a.et = et
	owner, err := iw.u.eus.GetOrCreateEntityESI(ctx, characterID)
	if err != nil {
		return nil, err
	}
	a.character = owner
	if a.et == nil {
		return nil, nil
	}
	p, err := iw.u.eus.GetMarketPrice(ctx, a.et.ID)
	if errors.Is(err, app.ErrNotFound) {
		p = nil
	} else if err != nil {
		return nil, err
	} else if p.AveragePrice != 0 {
		a.price = p
	} else {
		a.price = nil
	}
	oo, err := iw.u.eus.ListTypeDogmaAttributesForType(ctx, a.et.ID)
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
	descriptionTab := container.NewTabItem("Description", a.makeDescriptionTab())
	tabs := container.NewAppTabs(descriptionTab)
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
	marketLabel := widget.NewLabel("Fetching prices...")
	marketTab := container.NewTabItem("Market", marketLabel)
	ctx, cancel := context.WithCancel(context.Background())
	a.iw.onClosedFuncs = append(a.iw.onClosedFuncs, cancel)
	go func() {
		const (
			priceFormat    = "#,###.##"
			currencySuffix = " ISK"
		)
		ticker := time.NewTicker(60 * time.Second)
	L:
		for {
			r, err := a.iw.u.js.FetchPrices(ctx, a.id)
			if err != nil {
				fyne.Do(func() {
					marketLabel.Text = "Error: " + a.iw.u.humanizeError(err)
					marketLabel.Importance = widget.DangerImportance
					marketLabel.Refresh()
				})
			} else {
				c := newAttributeList(a.iw,
					newAttributeItem("Average price", humanize.FormatFloat(priceFormat, a.price.AveragePrice)+currencySuffix),
					newAttributeItem("Jita sell price", humanize.FormatFloat(priceFormat, r.ImmediatePrices.SellPrice)+currencySuffix),
					newAttributeItem("Jita buy price", humanize.FormatFloat(priceFormat, r.ImmediatePrices.BuyPrice)+currencySuffix),
					newAttributeItem("Jita sell volume", ihumanize.Comma(r.SellVolume)),
					newAttributeItem("Jita buy volume", ihumanize.Comma(r.BuyVolume)),
				)
				fyne.Do(func() {
					marketTab.Content = c
					tabs.Refresh()
				})
			}
			select {
			case <-ctx.Done():
				break L
			case <-ticker.C:
			}
		}
		slog.Debug("market update type for canceled", "name", a.et.Name)
	}()
	tabs.Append(marketTab)
	// Set initial tab
	if a.iw.u.Settings().PreferMarketTab() && a.et.IsTradeable() {
		tabs.Select(marketTab)
	} else if requirementsTab != nil && a.et.Group.Category.ID == app.EveCategorySkill {
		tabs.Select(requirementsTab)
	} else if attributeTab != nil &&
		set.NewFromSlice([]int32{
			app.EveCategoryDrone,
			app.EveCategoryFighter,
			app.EveCategoryOrbitals,
			app.EveCategoryShip,
			app.EveCategoryStructure,
		}).Contains(a.et.Group.Category.ID) {
		tabs.Select(attributeTab)
	} else {
		tabs.Select(descriptionTab)
	}
	c := container.NewBorder(top, nil, nil, nil, tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *inventoryTypeInfo) makeTop() fyne.CanvasObject {
	var typeIcon fyne.CanvasObject
	loader := func() (fyne.Resource, error) {
		if a.et.IsSKIN() {
			return a.iw.u.eis.InventoryTypeSKIN(a.et.ID, app.IconPixelSize)
		} else if a.et.IsBlueprint() {
			return a.iw.u.eis.InventoryTypeBPO(a.et.ID, app.IconPixelSize)
		} else {
			return a.iw.u.eis.InventoryTypeIcon(a.et.ID, app.IconPixelSize)
		}
	}
	if !a.et.HasRender() {
		icon := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(logoUnitSize))
		iwidget.RefreshImageAsync(icon, loader)
		typeIcon = icon
	} else {
		icon := kxwidget.NewTappableImage(icons.BlankSvg, nil)
		icon.SetFillMode(canvas.ImageFillContain)
		icon.SetMinSize(fyne.NewSquareSize(logoUnitSize))
		icon.OnTapped = func() {
			go fyne.Do(func() {
				a.iw.showZoomWindow(a.et.Name, a.id, a.iw.u.eis.InventoryTypeRender, a.iw.w)
			})
		}
		iwidget.RefreshTappableImageAsync(icon, loader)
		typeIcon = icon
	}

	characterIcon := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
	characterName := kxwidget.NewTappableLabel("", func() {
		a.iw.showEveEntity(a.character)
	})
	characterName.Wrapping = fyne.TextWrapWord
	if a.character != nil {
		iwidget.RefreshImageAsync(characterIcon, func() (fyne.Resource, error) {
			return a.iw.u.eis.CharacterPortrait(a.character.ID, app.IconPixelSize)
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
	name := makeInfoName()
	name.SetText(a.et.Name)
	emb := iwidget.NewTappableIcon(icons.EvemarketbrowserJpg, func() {
		a.iw.openURL(fmt.Sprintf("https://evemarketbrowser.com/region/0/type/%d", a.id))
	})
	emb.SetToolTip("Show on evemarketbrowser.com")
	evemarketbrowser := container.NewStack(canvas.NewRectangle(theme.Color(theme.ColorNameButton)), emb)
	j := iwidget.NewTappableIcon(icons.JanicePng, func() {
		a.iw.openURL(fmt.Sprintf("https://janice.e-351.com/i/%d", a.id))
	})
	j.SetToolTip("Show on janice.e-351.com")
	janice := container.NewStack(canvas.NewRectangle(color.White), j)
	if !a.et.IsTradeable() {
		evemarketbrowser.Hide()
		janice.Hide()
	}
	return container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			container.NewPadded(typeIcon),
			container.New(
				layout.NewCustomPaddedHBoxLayout(3*theme.Padding()),
				layout.NewSpacer(),
				evemarketbrowser,
				janice,
				layout.NewSpacer(),
			),
		),
		nil,
		container.NewVBox(
			name,
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

func (a *inventoryTypeInfo) makeAttributesTab() fyne.CanvasObject {
	list := widget.NewList(
		func() int {
			return len(a.attributesData)
		},
		func() fyne.CanvasObject {
			return appwidget.NewTypeAttributeItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.attributesData) {
				return
			}
			r := a.attributesData[id]
			item := co.(*appwidget.TypeAttributeItem)
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
				appwidget.NewSkillLevel(),
				widget.NewIcon(icons.QuestionmarkSvg),
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
		a.iw.show(infoInventoryType, int64(r.typeID))
		l.UnselectAll()
	}
	return l
}

func (a *inventoryTypeInfo) title() string {
	return a.et.Group.Name
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
			v, substituteIcon := a.iw.u.eus.FormatDogmaValue(ctx, value, o.DogmaAttribute.Unit)
			var iconID int32
			if substituteIcon != 0 {
				iconID = substituteIcon
			} else {
				iconID = o.DogmaAttribute.IconID
			}
			r, _ := eveicon.FromID(iconID)
			groupedRows[ag] = append(groupedRows[ag], attributeRow{
				icon:  r,
				label: o.DogmaAttribute.DisplayName,
				value: v,
			})
		}
	}
	rows := make([]attributeRow, 0)
	if a.et.Volume > 0 {
		v, _ := a.iw.u.eus.FormatDogmaValue(ctx, a.et.Volume, app.EveUnitVolume)
		if a.et.Volume != a.et.PackagedVolume {
			v2, _ := a.iw.u.eus.FormatDogmaValue(ctx, a.et.PackagedVolume, app.EveUnitVolume)
			v += fmt.Sprintf(" (%s Packaged)", v2)
		}
		r := attributeRow{
			icon:  eveicon.FromName(eveicon.Structure),
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
				a.iw.u.App().Clipboard().SetContent(v)
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
		r, _ := eveicon.FromID(iconID)
		v, _ := a.iw.u.eus.FormatDogmaValue(ctx, o.Value, o.DogmaAttribute.Unit)
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
		et, err := a.iw.u.eus.GetType(ctx, typeID)
		if err != nil {
			return nil, err
		}
		skill := requiredSkill{
			rank:          i + 1,
			requiredLevel: requiredLevel,
			name:          et.Name,
			typeID:        typeID,
		}
		cs, err := a.iw.u.cs.GetSkill(ctx, characterID, typeID)
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

// attributeGroup represents a group of dogma attributes.
//
// Used for rendering the attributes and fitting tabs for inventory type info
type attributeGroup string

func (ag attributeGroup) DisplayName() string {
	titler := cases.Title(language.English)
	return titler.String(string(ag))
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

func boolIconResource(ok bool) fyne.Resource {
	if ok {
		return theme.NewSuccessThemedResource(theme.ConfirmIcon())
	}
	return theme.NewErrorThemedResource(theme.CancelIcon())
}
