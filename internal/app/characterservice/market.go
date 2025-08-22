package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/antihax/goesi/esi"
	"golang.org/x/sync/errgroup"
)

func (s *CharacterService) ListAllMarketOrder(ctx context.Context, isBuyOrders bool) ([]*app.CharacterMarketOrder, error) {
	return s.st.ListAllCharacterMarketOrders(ctx, isBuyOrders)
}

func (s *CharacterService) updateMarketOrdersESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterMarketOrders {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			var (
				open    []esi.GetCharactersCharacterIdOrders200Ok
				history []esi.GetCharactersCharacterIdOrdersHistory200Ok
			)
			g := new(errgroup.Group)
			g.Go(func() error {
				orders, _, err := s.esiClient.ESI.MarketApi.GetCharactersCharacterIdOrders(ctx, characterID, nil)
				if err != nil {
					return err
				}
				open = orders
				slog.Debug("Received open orders from ESI", "count", len(orders), "characterID", characterID)
				return nil
			})
			g.Go(func() error {
				orders, _, err := s.esiClient.ESI.MarketApi.GetCharactersCharacterIdOrdersHistory(ctx, characterID, nil)
				if err != nil {
					return err
				}
				history = orders
				slog.Debug("Received history orders from ESI", "count", len(orders), "characterID", characterID)
				return nil
			})
			if err := g.Wait(); err != nil {
				return nil, err
			}

			orders := make(map[int64]esi.GetCharactersCharacterIdOrdersHistory200Ok)
			for _, o := range open {
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
			for _, o := range history {
				orders[o.OrderId] = o
			}
			return orders, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			orders := data.(map[int64]esi.GetCharactersCharacterIdOrdersHistory200Ok)
			var locationIDs set.Set[int64]
			var regionIDs, typeIDs set.Set[int32]
			for _, o := range orders {
				locationIDs.Add(o.LocationId)
				regionIDs.Add(o.RegionId)
				typeIDs.Add(o.TypeId)
			}
			ec, err := s.st.GetEveCharacter(ctx, characterID)
			if err != nil {
				return err
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
			g.Go(func() error {
				_, err := s.eus.AddMissingEntities(ctx, set.Of(characterID))
				return err
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
				var ownerID int32
				if o.IsCorporation {
					ownerID = ec.Corporation.ID
				} else {
					ownerID = characterID
				}
				arg := storage.UpdateOrCreateCharacterMarketOrderParams{
					CharacterID:   characterID,
					Duration:      int(o.Duration),
					IsBuyOrder:    o.IsBuyOrder,
					IsCorporation: o.IsCorporation,
					Issued:        o.Issued,
					LocationID:    o.LocationId,
					OrderID:       o.OrderId,
					OwnerID:       ownerID,
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
				"count", len(orders),
			)

			incoming := set.Collect(maps.Keys(orders))
			current, err := s.st.ListCharacterMarketOrders(ctx, characterID)
			if err != nil {
				return err
			}
			running := set.Collect(xiter.Map(xiter.FilterSlice(current, func(x *app.CharacterMarketOrder) bool {
				return x.State == app.OrderOpen
			}), func(x *app.CharacterMarketOrder) int64 {
				return x.OrderID
			}))
			orphans := set.Difference(running, incoming)
			if orphans.Size() > 0 {
				// The ESI response only returns orders from the last 90 days.
				// It can therefore happen that a long running job vanishes from the response,
				// without the app having received a final status (e.g. expired or canceled).
				// Since the status of the job is undetermined we can only delete it.
				err := s.st.DeleteCharacterMarketOrdersByID(ctx, characterID, orphans)
				if err != nil {
					return err
				}
				slog.Info("Deleted orphaned market orders", "characterID", characterID, "count", orphans.Size())
			}
			return nil
		})
}
