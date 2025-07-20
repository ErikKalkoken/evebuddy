package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/dustin/go-humanize"
	"golang.org/x/sync/errgroup"
)

func eveEntityCategoryFromESICategory(c string) app.EveEntityCategory {
	categoryMap := map[string]app.EveEntityCategory{
		"alliance":       app.EveEntityAlliance,
		"character":      app.EveEntityCharacter,
		"corporation":    app.EveEntityCorporation,
		"constellation":  app.EveEntityConstellation,
		"faction":        app.EveEntityFaction,
		"inventory_type": app.EveEntityInventoryType,
		"mailing_list":   app.EveEntityMailList,
		"region":         app.EveEntityRegion,
		"solar_system":   app.EveEntitySolarSystem,
		"station":        app.EveEntityStation,
	}
	c2, ok := categoryMap[c]
	if !ok {
		return app.EveEntityUnknown
	}
	return c2
}

func (s *EveUniverseService) GetType(ctx context.Context, id int32) (*app.EveType, error) {
	return s.st.GetEveType(ctx, id)
}

func (s *EveUniverseService) GetOrCreateCategoryESI(ctx context.Context, id int32) (*app.EveCategory, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateCategoryESI-%d", id), func() (any, error) {
		o1, err := s.st.GetEveCategory(ctx, id)
		if err == nil {
			return o1, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		r, _, err := s.esiClient.ESI.UniverseApi.GetUniverseCategoriesCategoryId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveCategoryParams{
			ID:          id,
			Name:        r.Name,
			IsPublished: r.Published,
		}
		o2, err := s.st.CreateEveCategory(ctx, arg)
		if err != nil {
			return nil, err
		}
		slog.Info("Created eve category", "ID", id)
		return o2, nil
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveCategory), nil
}

func (s *EveUniverseService) GetOrCreateGroupESI(ctx context.Context, id int32) (*app.EveGroup, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateGroupESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveGroup(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		group, _, err := s.esiClient.ESI.UniverseApi.GetUniverseGroupsGroupId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		c, err := s.GetOrCreateCategoryESI(ctx, group.CategoryId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveGroupParams{
			ID:          id,
			Name:        group.Name,
			CategoryID:  c.ID,
			IsPublished: group.Published,
		}
		if err := s.st.CreateEveGroup(ctx, arg); err != nil {
			return nil, err
		}
		slog.Info("Created eve group", "ID", id)
		return s.st.GetEveGroup(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveGroup), nil
}

func (s *EveUniverseService) GetOrCreateTypeESI(ctx context.Context, id int32) (*app.EveType, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateTypeESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveType(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		t, _, err := s.esiClient.ESI.UniverseApi.GetUniverseTypesTypeId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		g, err := s.GetOrCreateGroupESI(ctx, t.GroupId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveTypeParams{
			ID:             id,
			GroupID:        g.ID,
			Capacity:       t.Capacity,
			Description:    t.Description,
			GraphicID:      t.GraphicId,
			IconID:         t.IconId,
			IsPublished:    t.Published,
			MarketGroupID:  t.MarketGroupId,
			Mass:           t.Mass,
			Name:           t.Name,
			PackagedVolume: t.PackagedVolume,
			PortionSize:    int(t.PortionSize),
			Radius:         t.Radius,
			Volume:         t.Volume,
		}
		if err := s.st.CreateEveType(ctx, arg); err != nil {
			return nil, err
		}
		for _, o := range t.DogmaAttributes {
			x, err := s.GetOrCreateDogmaAttributeESI(ctx, o.AttributeId)
			if err != nil {
				return nil, err
			}
			switch x.Unit {
			case app.EveUnitGroupID:
				go func(ctx context.Context, groupID int32) {
					_, err := s.GetOrCreateGroupESI(ctx, groupID)
					if err != nil {
						slog.Error("Failed to fetch eve group %d", "ID", groupID, "err", err)
					}
				}(ctx, int32(o.Value))
			case app.EveUnitTypeID:
				go func(ctx context.Context, typeID int32) {
					_, err := s.GetOrCreateTypeESI(ctx, typeID)
					if err != nil {
						slog.Error("Failed to fetch eve type %d", "ID", typeID, "err", err)
					}
				}(ctx, int32(o.Value))
			}
			arg := storage.CreateEveTypeDogmaAttributeParams{
				DogmaAttributeID: o.AttributeId,
				EveTypeID:        id,
				Value:            o.Value,
			}
			if err := s.st.CreateEveTypeDogmaAttribute(ctx, arg); err != nil {
				return nil, err
			}
		}
		for _, o := range t.DogmaEffects {
			arg := storage.CreateEveTypeDogmaEffectParams{
				DogmaEffectID: o.EffectId,
				EveTypeID:     id,
				IsDefault:     o.IsDefault,
			}
			if err := s.st.CreateEveTypeDogmaEffect(ctx, arg); err != nil {
				return nil, err
			}
		}
		slog.Info("Created eve type", "ID", id)
		return s.st.GetEveType(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveType), nil
}

// AddMissingTypes fetches missing typeIDs from ESI.
// Invalid IDs (e.g. 0) will be ignored
func (s *EveUniverseService) AddMissingTypes(ctx context.Context, ids set.Set[int32]) error {
	ids2 := ids.Clone()
	ids2.Delete(0) // ignore invalid ID
	if ids.Size() == 0 {
		return nil
	}
	missing, err := s.st.MissingEveTypes(ctx, ids2)
	if err != nil {
		return err
	}
	if missing.Size() == 0 {
		return nil
	}
	slog.Debug("Trying to fetch missing EveTypes from ESI", "count", missing.Size())
	g := new(errgroup.Group)
	for id := range missing.All() {
		g.Go(func() error {
			_, err := s.GetOrCreateTypeESI(ctx, id)
			return err
		})
	}
	return g.Wait()
}

func (s *EveUniverseService) UpdateCategoryWithChildrenESI(ctx context.Context, categoryID int32) error {
	_, err, _ := s.sfg.Do(fmt.Sprintf("UpdateCategoryWithChildrenESI-%d", categoryID), func() (any, error) {
		var typeIds set.Set[int32]
		_, err := s.GetOrCreateCategoryESI(ctx, categoryID)
		if err != nil {
			return nil, err
		}
		category, _, err := s.esiClient.ESI.UniverseApi.GetUniverseCategoriesCategoryId(ctx, categoryID, nil)
		if err != nil {
			return nil, err
		}
		g := new(errgroup.Group)
		for _, id := range category.Groups {
			g.Go(func() error {
				_, err := s.GetOrCreateGroupESI(ctx, id)
				return err
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
		groupTypes := make([][]int32, len(category.Groups))
		g = new(errgroup.Group)
		for i, id := range category.Groups {
			g.Go(func() error {
				group, _, err := s.esiClient.ESI.UniverseApi.GetUniverseGroupsGroupId(ctx, id, nil)
				if err != nil {
					return err
				}
				groupTypes[i] = group.Types
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
		for _, ids := range groupTypes {
			typeIds.AddSeq(slices.Values(ids))
		}
		if err := s.AddMissingTypes(ctx, typeIds); err != nil {
			return nil, err
		}
		slog.Info("Updated eve types", "categoryID", categoryID, "count", typeIds.Size())
		return nil, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *EveUniverseService) GetDogmaAttribute(ctx context.Context, id int32) (*app.EveDogmaAttribute, error) {
	return s.st.GetEveDogmaAttribute(ctx, id)
}

func (s *EveUniverseService) GetOrCreateDogmaAttributeESI(ctx context.Context, id int32) (*app.EveDogmaAttribute, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("createDogmaAttributeFromESI-%d", id), func() (any, error) {
		o1, err := s.st.GetEveDogmaAttribute(ctx, id)
		if err == nil {
			return o1, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		d, _, err := s.esiClient.ESI.DogmaApi.GetDogmaAttributesAttributeId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveDogmaAttributeParams{
			ID:           d.AttributeId,
			DefaultValue: d.DefaultValue,
			Description:  d.Description,
			DisplayName:  d.DisplayName,
			IconID:       d.IconId,
			Name:         d.Name,
			IsHighGood:   d.HighIsGood,
			IsPublished:  d.Published,
			IsStackable:  d.Stackable,
			UnitID:       app.EveUnitID(d.UnitId),
		}
		o2, err := s.st.CreateEveDogmaAttribute(ctx, arg)
		if err != nil {
			return nil, err
		}
		slog.Info("Created eve dogma attribute", "ID", id)
		return o2, nil
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveDogmaAttribute), nil
}

// FormatDogmaValue returns a formatted value.
func (s *EveUniverseService) FormatDogmaValue(ctx context.Context, value float32, unitID app.EveUnitID) (string, int32) {
	return formatDogmaValue(ctx, formatDogmaValueParams{
		value:                        value,
		unitID:                       unitID,
		getDogmaAttribute:            s.GetDogmaAttribute,
		getOrCreateDogmaAttributeESI: s.GetOrCreateDogmaAttributeESI,
		getType:                      s.GetType,
		getOrCreateTypeESI:           s.GetOrCreateTypeESI,
	})
}

type formatDogmaValueParams struct {
	value                        float32
	unitID                       app.EveUnitID
	getDogmaAttribute            func(context.Context, int32) (*app.EveDogmaAttribute, error)
	getOrCreateDogmaAttributeESI func(context.Context, int32) (*app.EveDogmaAttribute, error)
	getType                      func(context.Context, int32) (*app.EveType, error)
	getOrCreateTypeESI           func(context.Context, int32) (*app.EveType, error)
}

func formatDogmaValue(ctx context.Context, args formatDogmaValueParams) (string, int32) {
	defaultFormatter := func(v float32) string {
		return humanize.CommafWithDigits(float64(v), 2)
	}
	now := time.Now()
	value := args.value
	switch args.unitID {
	case app.EveUnitAbsolutePercent:
		return fmt.Sprintf("%.0f%%", value*100), 0
	case app.EveUnitAcceleration:
		return fmt.Sprintf("%s m/s²", defaultFormatter(value)), 0
	case app.EveUnitAttributeID:
		da, err := args.getDogmaAttribute(ctx, int32(value))
		if err != nil {
			go func() {
				_, err := args.getOrCreateDogmaAttributeESI(ctx, int32(value))
				if err != nil {
					slog.Error("Failed to fetch dogma attribute from ESI", "ID", value, "err", err)
				}
			}()
			return "?", 0
		}
		return da.DisplayName, da.IconID
	case app.EveUnitAttributePoints:
		return fmt.Sprintf("%s points", defaultFormatter(value)), 0
	case app.EveUnitCapacitorUnits:
		return fmt.Sprintf("%s GJ", humanize.FormatFloat("#,###.#", float64(value))), 0
	case app.EveUnitDroneBandwidth:
		return fmt.Sprintf("%s Mbit/s", defaultFormatter(value)), 0
	case app.EveUnitHitpoints:
		return fmt.Sprintf("%s HP", defaultFormatter(value)), 0
	case app.EveUnitInverseAbsolutePercent:
		return fmt.Sprintf("%.0f%%", (1-value)*100), 0
	case app.EveUnitLength:
		if value > 1000 {
			return fmt.Sprintf("%s km", defaultFormatter(value/float32(1000))), 0
		} else {
			return fmt.Sprintf("%s m", defaultFormatter(value)), 0
		}
	case app.EveUnitLevel:
		return fmt.Sprintf("Level %s", defaultFormatter(value)), 0
	case app.EveUnitLightYear:
		return fmt.Sprintf("%.1f LY", value), 0
	case app.EveUnitMass:
		return fmt.Sprintf("%s kg", defaultFormatter(value)), 0
	case app.EveUnitMegaWatts:
		return fmt.Sprintf("%s MW", defaultFormatter(value)), 0
	case app.EveUnitMillimeters:
		return fmt.Sprintf("%s mm", defaultFormatter(value)), 0
	case app.EveUnitMilliseconds:
		return strings.TrimSpace(humanize.RelTime(now, now.Add(time.Duration(value)*time.Millisecond), "", "")), 0
	case app.EveUnitMultiplier:
		return fmt.Sprintf("%.3f x", value), 0
	case app.EveUnitPercentage:
		return fmt.Sprintf("%.0f%%", value*100), 0
	case app.EveUnitTeraflops:
		return fmt.Sprintf("%s tf", defaultFormatter(value)), 0
	case app.EveUnitVolume:
		return fmt.Sprintf("%s m3", defaultFormatter(value)), 0
	case app.EveUnitWarpSpeed:
		return fmt.Sprintf("%s AU/s", defaultFormatter(value)), 0
	case app.EveUnitTypeID:
		et, err := args.getType(ctx, int32(value))
		if err != nil {
			go func() {
				_, err := args.getOrCreateTypeESI(ctx, int32(value))
				if err != nil {
					slog.Error("Failed to fetch type from ESI", "typeID", value, "err", err)
				}
			}()
			return "?", 0
		}
		return et.Name, et.IconID
	case app.EveUnitUnits:
		return fmt.Sprintf("%s units", defaultFormatter(value)), 0
	case app.EveUnitNone, app.EveUnitHardpoints, app.EveUnitFittingSlots, app.EveUnitSlot:
		return defaultFormatter(value), 0
	}
	return fmt.Sprintf("%s ???", defaultFormatter(value)), 0
}

func (s *EveUniverseService) ListTypeDogmaAttributesForType(
	ctx context.Context,
	typeID int32,
) ([]*app.EveTypeDogmaAttribute, error) {
	return s.st.ListEveTypeDogmaAttributesForType(ctx, typeID)
}

// TODO: Not fully thread safe: Might update for same ID multiple times.

// MarketPrice returns the average market price for a type. Or empty when no price is known for this type.
func (s *EveUniverseService) MarketPrice(ctx context.Context, typeID int32) (optional.Optional[float64], error) {
	var v optional.Optional[float64]
	o, err := s.st.GetEveMarketPrice(ctx, typeID)
	if errors.Is(err, app.ErrNotFound) {
		return v, nil
	} else if err != nil {
		return v, err
	}
	return optional.New(o.AveragePrice), nil
}

// TODO: Change to bulk create

func (s *EveUniverseService) updateMarketPricesESI(ctx context.Context) error {
	_, err, _ := s.sfg.Do("updateMarketPricesESI", func() (any, error) {
		prices, _, err := s.esiClient.ESI.MarketApi.GetMarketsPrices(ctx, nil)
		if err != nil {
			return nil, err
		}
		for _, p := range prices {
			arg := storage.UpdateOrCreateEveMarketPriceParams{
				TypeID:        p.TypeId,
				AdjustedPrice: p.AdjustedPrice,
				AveragePrice:  p.AveragePrice,
			}
			if err := s.st.UpdateOrCreateEveMarketPrice(ctx, arg); err != nil {
				return nil, err
			}
		}
		slog.Info("Updated market prices", "count", len(prices))
		return nil, nil
	})
	return err
}

func (s *EveUniverseService) updateCategories(ctx context.Context) error {
	g := new(errgroup.Group)
	g.Go(func() error {
		return s.UpdateCategoryWithChildrenESI(ctx, app.EveCategorySkill)
	})
	g.Go(func() error {
		return s.UpdateCategoryWithChildrenESI(ctx, app.EveCategoryShip)
	})
	if err := g.Wait(); err != nil {
		return err
	}
	if err := s.UpdateShipSkills(ctx); err != nil {
		return err
	}
	return nil
}

func (s *EveUniverseService) UpdateShipSkills(ctx context.Context) error {
	return s.st.UpdateEveShipSkills(ctx)
}
