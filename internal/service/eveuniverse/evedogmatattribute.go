package eveuniverse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/dustin/go-humanize"
)

func (eu *EveUniverseService) GetEveDogmaAttribute(ctx context.Context, id int32) (*model.EveDogmaAttribute, error) {
	o, err := eu.st.GetEveDogmaAttribute(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return o, nil
}

func (eu *EveUniverseService) GetOrCreateEveDogmaAttributeESI(ctx context.Context, id int32) (*model.EveDogmaAttribute, error) {
	o, err := eu.st.GetEveDogmaAttribute(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveDogmaAttributeFromESI(ctx, id)
	} else if err != nil {
		return o, err
	}
	return o, nil
}

func (eu *EveUniverseService) createEveDogmaAttributeFromESI(ctx context.Context, id int32) (*model.EveDogmaAttribute, error) {
	key := fmt.Sprintf("createEveDogmaAttributeFromESI-%d", id)
	x, err, _ := eu.sfg.Do(key, func() (any, error) {
		o, _, err := eu.esiClient.ESI.DogmaApi.GetDogmaAttributesAttributeId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveDogmaAttributeParams{
			ID:           o.AttributeId,
			DefaultValue: o.DefaultValue,
			Description:  o.Description,
			DisplayName:  o.DisplayName,
			IconID:       o.IconId,
			Name:         o.Name,
			IsHighGood:   o.HighIsGood,
			IsPublished:  o.Published,
			IsStackable:  o.Stackable,
			UnitID:       o.UnitId,
		}
		return eu.st.CreateEveDogmaAttribute(ctx, arg)
	})
	if err != nil {
		return nil, err
	}
	return x.(*model.EveDogmaAttribute), nil
}

// FormatValue returns a formatted value.
func (eu *EveUniverseService) FormatValue(ctx context.Context, value float32, unitID int32) (string, int32) {
	defaultFormatter := func(v float32) string {
		return humanize.CommafWithDigits(float64(v), 2)
	}
	now := time.Now()
	switch unitID {
	case model.EveUnitAbsolutePercent:
		return fmt.Sprintf("%.0f%%", value*100), 0
	case model.EveUnitAcceleration:
		return fmt.Sprintf("%s m/sec", defaultFormatter(value)), 0
	case model.EveUnitAttributeID:
		da, err := eu.GetEveDogmaAttribute(ctx, int32(value))
		if err != nil {
			go func() {
				_, err := eu.GetOrCreateEveDogmaAttributeESI(ctx, int32(value))
				if err != nil {
					slog.Error("Failed to fetch dogma attribute from ESI", "ID", value, "err", err)
				}
			}()
			return "?", 0
		}
		return da.DisplayName, da.IconID
	case model.EveUnitAttributePoints:
		return fmt.Sprintf("%s points", defaultFormatter(value)), 0
	case model.EveUnitCapacitorUnits:
		return fmt.Sprintf("%.1f GJ", value), 0
	case model.EveUnitDroneBandwidth:
		return fmt.Sprintf("%s Mbit/s", defaultFormatter(value)), 0
	case model.EveUnitHitpoints:
		return fmt.Sprintf("%s HP", defaultFormatter(value)), 0
	case model.EveUnitInverseAbsolutePercent:
		return fmt.Sprintf("%.0f%%", (1-value)*100), 0
	case model.EveUnitLength:
		if value > 1000 {
			return fmt.Sprintf("%s km", defaultFormatter(value/float32(1000))), 0
		} else {
			return fmt.Sprintf("%s m", defaultFormatter(value)), 0
		}
	case model.EveUnitLevel:
		return fmt.Sprintf("Level %s", defaultFormatter(value)), 0
	case model.EveUnitLightYear:
		return fmt.Sprintf("%.1f LY", value), 0
	case model.EveUnitMass:
		return fmt.Sprintf("%s kg", defaultFormatter(value)), 0
	case model.EveUnitMegaWatts:
		return fmt.Sprintf("%s MW", defaultFormatter(value)), 0
	case model.EveUnitMillimeters:
		return fmt.Sprintf("%s mm", defaultFormatter(value)), 0
	case model.EveUnitMilliseconds:
		return humanize.RelTime(now, now.Add(time.Duration(value)*time.Millisecond), "", ""), 0
	case model.EveUnitMultiplier:
		return fmt.Sprintf("%.3f x", value), 0
	case model.EveUnitPercentage:
		return fmt.Sprintf("%.0f%%", value*100), 0
	case model.EveUnitTeraflops:
		return fmt.Sprintf("%s tf", defaultFormatter(value)), 0
	case model.EveUnitVolume:
		return fmt.Sprintf("%s m3", defaultFormatter(value)), 0
	case model.EveUnitWarpSpeed:
		return fmt.Sprintf("%s AU/s", defaultFormatter(value)), 0
	case model.EveUnitTypeID:
		et, err := eu.GetEveType(ctx, int32(value))
		if err != nil {
			go func() {
				_, err := eu.GetOrCreateEveTypeESI(ctx, int32(value))
				if err != nil {
					slog.Error("Failed to fetch type from ESI", "typeID", value, "err", err)
				}
			}()
			return "?", 0
		}
		return et.Name, et.IconID
	case model.EveUnitUnits:
		return fmt.Sprintf("%s units", defaultFormatter(value)), 0
	case model.EveUnitNone, model.EveUnitHardpoints, model.EveUnitFittingSlots, model.EveUnitSlot:
		return defaultFormatter(value), 0
	}
	return fmt.Sprintf("%s ???", defaultFormatter(value)), 0
}
