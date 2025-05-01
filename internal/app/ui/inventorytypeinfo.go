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
	baseInfoWidget

	characterIcon    *canvas.Image
	characterID      int32
	characterName    *kxwidget.TappableLabel
	checkIcon        *widget.Icon
	description      *widget.Label
	evemarketbrowser *fyne.Container
	iw               *InfoWindow
	janice           *fyne.Container
	tabs             *container.AppTabs
	typeIcon         *kxwidget.TappableImage
	typeID           int32
}

func newInventoryTypeInfo(iw *InfoWindow, typeID, characterID int32) *inventoryTypeInfo {
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	typeIcon := kxwidget.NewTappableImage(icons.BlankSvg, nil)
	typeIcon.SetFillMode(canvas.ImageFillContain)
	typeIcon.SetMinSize(fyne.NewSquareSize(logoUnitSize))
	a := &inventoryTypeInfo{
		characterIcon: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		characterID:   characterID,
		checkIcon:     widget.NewIcon(icons.BlankSvg),
		description:   description,
		iw:            iw,
		typeIcon:      typeIcon,
		typeID:        typeID,
	}
	a.initBase()
	a.ExtendBaseWidget(a)

	a.checkIcon.Hide()
	a.characterIcon.Hide()
	a.characterName = kxwidget.NewTappableLabel("", nil)
	a.characterName.Wrapping = fyne.TextWrapWord
	a.characterName.Hide()

	emb := iwidget.NewTappableIcon(icons.EvemarketbrowserJpg, func() {
		a.iw.openURL(fmt.Sprintf("https://evemarketbrowser.com/region/0/type/%d", a.typeID))
	})
	emb.SetToolTip("Show on evemarketbrowser.com")
	a.evemarketbrowser = container.NewStack(canvas.NewRectangle(theme.Color(theme.ColorNameButton)), emb)
	a.evemarketbrowser.Hide()

	j := iwidget.NewTappableIcon(icons.JanicePng, func() {
		a.iw.openURL(fmt.Sprintf("https://janice.e-351.com/i/%d", a.typeID))
	})
	j.SetToolTip("Show on janice.e-351.com")
	a.janice = container.NewStack(canvas.NewRectangle(color.White), j)
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
				a.evemarketbrowser,
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

func (a *inventoryTypeInfo) update() error {
	ctx := context.Background()
	et, err := a.iw.u.eus.GetOrCreateTypeESI(ctx, a.typeID)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(et.Name)
		if et.IsTradeable() {
			a.evemarketbrowser.Show()
			a.janice.Show()
		}
		s := et.DescriptionPlain()
		if s == "" {
			s = et.Name
		}
		a.description.SetText(s)
	})

	iwidget.RefreshTappableImageAsync(a.typeIcon, func() (fyne.Resource, error) {
		if et.IsSKIN() {
			return a.iw.u.eis.InventoryTypeSKIN(et.ID, app.IconPixelSize)
		} else if et.IsBlueprint() {
			return a.iw.u.eis.InventoryTypeBPO(et.ID, app.IconPixelSize)
		} else {
			return a.iw.u.eis.InventoryTypeIcon(et.ID, app.IconPixelSize)
		}
	})
	if et.HasRender() {
		a.typeIcon.OnTapped = func() {
			fyne.Do(func() {
				a.iw.showZoomWindow(et.Name, a.typeID, a.iw.u.eis.InventoryTypeRender, a.iw.w)
			})
		}
	}

	var character *app.EveEntity
	if a.characterID != 0 {
		ee, err := a.iw.u.eus.GetOrCreateEntityESI(ctx, a.characterID)
		if err != nil {
			return err
		}
		character = ee
		iwidget.RefreshImageAsync(a.characterIcon, func() (fyne.Resource, error) {
			return a.iw.u.eis.CharacterPortrait(character.ID, app.IconPixelSize)
		})
		fyne.Do(func() {
			a.characterIcon.Show()
			a.characterName.OnTapped = func() {
				a.iw.showEveEntity(character)
			}
			a.characterName.SetText(character.Name)
			a.characterName.Show()
		})
	}

	dogmaAttributes, err := a.fetchDogmaAttributes(ctx, et.ID)
	if err != nil {
		return err
	}

	var requiredSkills []requiredSkill
	if a.characterID != 0 {
		skills, err := a.calcRequiredSkills(ctx, a.characterID, dogmaAttributes, a.iw.u)
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
	var attributeTab *container.TabItem
	attributes := a.calcAttributesData(ctx, et, dogmaAttributes, a.iw.u)
	if len(attributes) > 0 {
		list := widget.NewList(
			func() int {
				return len(attributes)
			},
			func() fyne.CanvasObject {
				return appwidget.NewTypeAttributeItem()
			},
			func(id widget.ListItemID, co fyne.CanvasObject) {
				if id >= len(attributes) {
					return
				}
				r := attributes[id]
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
			if id >= len(attributes) {
				return
			}
			r := attributes[id]
			if r.action != nil {
				r.action(r.value)
			}
		}
		attributeTab = container.NewTabItem("Attributes", list)
		fyne.Do(func() {
			a.tabs.Append(attributeTab)
		})
	}

	fittingData := a.calcFittingData(ctx, dogmaAttributes, a.iw.u)
	if len(fittingData) > 0 {
		list := widget.NewList(
			func() int {
				return len(fittingData)
			},
			func() fyne.CanvasObject {
				return appwidget.NewTypeAttributeItem()
			},
			func(lii widget.ListItemID, co fyne.CanvasObject) {
				r := fittingData[lii]
				item := co.(*appwidget.TypeAttributeItem)
				item.SetRegular(r.icon, r.label, r.value)
			},
		)
		list.OnSelected = func(id widget.ListItemID) {
			list.UnselectAll()
		}
		fyne.Do(func() {
			a.tabs.Append(container.NewTabItem("Fittings", list))
		})
	}

	var requirementsTab *container.TabItem
	if len(requiredSkills) > 0 {
		list := widget.NewList(
			func() int {
				return len(requiredSkills)
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
				o := requiredSkills[id]
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
		list.OnSelected = func(id widget.ListItemID) {
			r := requiredSkills[id]
			a.iw.show(infoInventoryType, int64(r.typeID))
			list.UnselectAll()
		}
		requirementsTab = container.NewTabItem("Requirements", list)
		fyne.Do(func() {
			a.tabs.Append(requirementsTab)
		})
	}

	var marketTab *container.TabItem
	if et.IsTradeable() {
		ctx, cancel := context.WithCancel(context.Background())
		a.iw.onClosedFuncs = append(a.iw.onClosedFuncs, cancel)
		marketTab = container.NewTabItem("Market", widget.NewLabel("Fetching prices..."))
		fyne.DoAndWait(func() {
			a.tabs.Append(marketTab)
			a.tabs.Refresh()
		})
		go func() {
			const (
				priceFormat    = "#,###.##"
				currencySuffix = " ISK"
			)
			ticker := time.NewTicker(60 * time.Second)
		L:
			for {
				var items []attributeItem

				var averagePrice string
				p, err := a.iw.u.eus.MarketPrice(ctx, et.ID)
				if err != nil {
					slog.Error("average price", "typeID", et.ID, "error", err)
					averagePrice = "ERROR: " + a.iw.u.humanizeError(err)
				} else {
					averagePrice = p.StringFunc("?", func(v float64) string {
						return humanize.FormatFloat(priceFormat, v) + currencySuffix
					})
				}
				items = append(items, newAttributeItem("Average price", averagePrice))

				j, err := a.iw.u.js.FetchPrices(ctx, a.typeID)
				if err != nil {
					slog.Error("janice pricer", "typeID", et.ID, "error", err)
					s := "ERROR: " + a.iw.u.humanizeError(err)
					items = append(items, newAttributeItem("Janice prices", s))
				} else {
					items2 := []attributeItem{
						newAttributeItem("Jita sell price", humanize.FormatFloat(priceFormat, j.ImmediatePrices.SellPrice)+currencySuffix),
						newAttributeItem("Jita buy price", humanize.FormatFloat(priceFormat, j.ImmediatePrices.BuyPrice)+currencySuffix),
						newAttributeItem("Jita sell volume", ihumanize.Comma(j.SellVolume)),
						newAttributeItem("Jita buy volume", ihumanize.Comma(j.BuyVolume)),
					}
					items = slices.Concat(items, items2)
				}
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
	}

	// Set initial tab
	fyne.Do(func() {
		if marketTab != nil && a.iw.u.settings.PreferMarketTab() {
			a.tabs.Select(marketTab)
		} else if requirementsTab != nil && et.Group.Category.ID == app.EveCategorySkill {
			a.tabs.Select(requirementsTab)
		} else if attributeTab != nil &&
			set.Of[int32](
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

func (a *inventoryTypeInfo) fetchDogmaAttributes(ctx context.Context, id int32) (map[int32]*app.EveTypeDogmaAttribute, error) {
	oo, err := a.iw.u.eus.ListTypeDogmaAttributesForType(ctx, id)
	if err != nil {
		return nil, err
	}
	dogmaAttributes := make(map[int32]*app.EveTypeDogmaAttribute)
	for _, o := range oo {
		dogmaAttributes[o.DogmaAttribute.ID] = o
	}
	return dogmaAttributes, nil
}

// TODO: Re-enable
// func (a *inventoryTypeInfo) title() string {
// 	return a.et.Group.Name
// }

// func calcLevels(attributes map[int32]*app.EveTypeDogmaAttribute) (int, int) {
// 	var tech, meta int
// 	x, ok := attributes[app.EveDogmaAttributeTechLevel]
// 	if ok {
// 		tech = int(x.Value)
// 	}
// 	x, ok = attributes[app.EveDogmaAttributeMetaLevel]
// 	if ok {
// 		meta = int(x.Value)
// 	}
// 	return tech, meta
// }

func (*inventoryTypeInfo) calcAttributesData(ctx context.Context, et *app.EveType, attributes map[int32]*app.EveTypeDogmaAttribute, u *BaseUI) []attributeRow {
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
			v, substituteIcon := u.eus.FormatDogmaValue(ctx, value, o.DogmaAttribute.Unit)
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
	if et.Volume > 0 {
		v, _ := u.eus.FormatDogmaValue(ctx, et.Volume, app.EveUnitVolume)
		if et.Volume != et.PackagedVolume {
			v2, _ := u.eus.FormatDogmaValue(ctx, et.PackagedVolume, app.EveUnitVolume)
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
	if u.IsDeveloperMode() {
		rows = append(rows, attributeRow{label: "Developer Mode", isTitle: true})
		rows = append(rows, attributeRow{
			label: "EVE ID",
			value: fmt.Sprint(et.ID),
			action: func(v string) {
				u.App().Clipboard().SetContent(v)
			},
		})
	}
	return rows
}

func (*inventoryTypeInfo) calcFittingData(ctx context.Context, attributes map[int32]*app.EveTypeDogmaAttribute, u *BaseUI) []attributeRow {
	data := make([]attributeRow, 0)
	for _, da := range attributeGroupsMap[attributeGroupFitting] {
		o, ok := attributes[da]
		if !ok {
			continue
		}
		iconID := o.DogmaAttribute.IconID
		r, _ := eveicon.FromID(iconID)
		v, _ := u.eus.FormatDogmaValue(ctx, o.Value, o.DogmaAttribute.Unit)
		data = append(data, attributeRow{
			icon:  r,
			label: o.DogmaAttribute.DisplayName,
			value: v,
		})
	}
	return data
}

func (*inventoryTypeInfo) calcRequiredSkills(ctx context.Context, characterID int32, attributes map[int32]*app.EveTypeDogmaAttribute, u *BaseUI) ([]requiredSkill, error) {
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
		et, err := u.eus.GetType(ctx, typeID)
		if err != nil {
			return nil, err
		}
		skill := requiredSkill{
			rank:          i + 1,
			requiredLevel: requiredLevel,
			name:          et.Name,
			typeID:        typeID,
		}
		cs, err := u.cs.GetSkill(ctx, characterID, typeID)
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
