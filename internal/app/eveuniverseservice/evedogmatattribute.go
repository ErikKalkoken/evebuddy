package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/dustin/go-humanize"
)

func (eu *EveUniverseService) GetDogmaAttribute(ctx context.Context, id int32) (*app.EveDogmaAttribute, error) {
	o, err := eu.st.GetEveDogmaAttribute(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, app.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return o, nil
}

func (eu *EveUniverseService) GetOrCreateDogmaAttributeESI(ctx context.Context, id int32) (*app.EveDogmaAttribute, error) {
	o, err := eu.st.GetEveDogmaAttribute(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveDogmaAttributeFromESI(ctx, id)
	} else if err != nil {
		return o, err
	}
	return o, nil
}

func (eu *EveUniverseService) createEveDogmaAttributeFromESI(ctx context.Context, id int32) (*app.EveDogmaAttribute, error) {
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
			UnitID:       app.EveUnitID(o.UnitId),
		}
		return eu.st.CreateEveDogmaAttribute(ctx, arg)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveDogmaAttribute), nil
}

// FormatDogmaValue returns a formatted value.
func (eu *EveUniverseService) FormatDogmaValue(ctx context.Context, value float32, unitID app.EveUnitID) (string, int32) {
	defaultFormatter := func(v float32) string {
		return humanize.CommafWithDigits(float64(v), 2)
	}
	now := time.Now()
	switch unitID {
	case app.EveUnitAbsolutePercent:
		return fmt.Sprintf("%.0f%%", value*100), 0
	case app.EveUnitAcceleration:
		return fmt.Sprintf("%s m/sec", defaultFormatter(value)), 0
	case app.EveUnitAttributeID:
		da, err := eu.GetDogmaAttribute(ctx, int32(value))
		if err != nil {
			go func() {
				_, err := eu.GetOrCreateDogmaAttributeESI(ctx, int32(value))
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
		return fmt.Sprintf("%.1f GJ", value), 0
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
		return humanize.RelTime(now, now.Add(time.Duration(value)*time.Millisecond), "", ""), 0
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
		et, err := eu.GetType(ctx, int32(value))
		if err != nil {
			go func() {
				_, err := eu.GetOrCreateTypeESI(ctx, int32(value))
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
