package characterservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
	"golang.org/x/sync/errgroup"
)

type characterMarketOrdersResult struct {
	open    []esi.GetCharactersCharacterIdOrders200Ok
	history []esi.GetCharactersCharacterIdOrdersHistory200Ok
}

func (s *CharacterService) updateMarketOrdersESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterMarketOrders {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			var r characterMarketOrdersResult
			g := new(errgroup.Group)
			g.Go(func() error {
				orders, _, err := s.esiClient.ESI.MarketApi.GetCharactersCharacterIdOrders(ctx, characterID, nil)
				if err != nil {
					return err
				}
				r.open = orders
				slog.Debug("Received open orders from ESI", "count", len(orders), "characterID", characterID)
				return nil
			})
			g.Go(func() error {
				orders, _, err := s.esiClient.ESI.MarketApi.GetCharactersCharacterIdOrdersHistory(ctx, characterID, nil)
				if err != nil {
					return err
				}
				r.history = orders
				slog.Debug("Received history orders from ESI", "count", len(orders), "characterID", characterID)
				return nil
			})
			err := g.Wait()
			return r, err
		},
		func(ctx context.Context, characterID int32, data any) error {
			r := data.(characterMarketOrdersResult)
			orders := make(map[int64]esi.GetCharactersCharacterIdOrdersHistory200Ok)
			for _, o := range r.open {
				orders[o.OrderId] = esi.GetCharactersCharacterIdOrdersHistory200Ok{
					Duration:      o.Duration,
					Escrow:        o.Escrow,
					IsBuyOrder:    o.IsBuyOrder,
					IsCorporation: o.IsCorporation,
					Issued:        o.Issued,
					LocationId:    o.LocationId,
					MinVolume:     o.MinVolume,
					OrderId:       o.OrderId,
					Price:         o.Price,
					Range_:        o.Range_,
					RegionId:      o.RegionId,
					State:         "open",
					TypeId:        o.TypeId,
					VolumeRemain:  o.VolumeRemain,
					VolumeTotal:   o.VolumeTotal,
				}
			}
			for _, o := range r.history {
				orders[o.OrderId] = o
			}
			var locationIDs set.Set[int64]
			var regionIDs, typeIDs set.Set[int32]
			for _, o := range orders {
				locationIDs.Add(o.LocationId)
				regionIDs.Add(o.RegionId)
				typeIDs.Add(o.TypeId)
			}
			g := new(errgroup.Group)
			g.Go(func() error {
				return s.eus.AddMissingLocations(ctx, locationIDs)
			})
			g.Go(func() error {
				return s.eus.AddMissingRegions(ctx, regionIDs)
			})
			g.Go(func() error {
				return s.eus.AddMissingTypes(ctx, typeIDs)
			})
			if err := g.Wait(); err != nil {
				return err
			}
			for _, o := range orders {
				var state app.MarketOrderState
				switch o.State {
				case "open":
					state = app.OrderOpen
				case "expired":
					state = app.OrderExpired
				case "cancelled":
					state = app.OrderCancelled
				default:
					state = app.OrderUndefined
				}
				arg := storage.UpdateOrCreateCharacterMarketOrderParams{
					CharacterID:   characterID,
					Duration:      int(o.Duration),
					IsBuyOrder:    o.IsBuyOrder,
					IsCorporation: o.IsCorporation,
					Issued:        o.Issued,
					LocationID:    o.LocationId,
					OrderID:       o.OrderId,
					Price:         o.Price,
					Range:         o.Range_,
					RegionID:      o.RegionId,
					State:         state,
					TypeID:        o.TypeId,
					VolumeRemains: int(o.VolumeRemain),
					VolumeTotal:   int(o.VolumeTotal),
				}
				if o.Escrow != 0 {
					arg.Escrow = optional.New(o.Escrow)
				}
				if o.MinVolume != 0 {
					arg.MinVolume = optional.New(int(o.MinVolume))
				}
				if err := s.st.UpdateOrCreateCharacterMarketOrder(ctx, arg); err != nil {
					return err
				}
			}
			slog.Info(
				"Stored updated market orders",
				"characterID", characterID,
				"open", len(r.open),
				"history", len(r.history),
			)
			return nil
		})
}
