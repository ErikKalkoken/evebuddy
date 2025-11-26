package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
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
	const openState = "open"
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			open, _, err := xesi.RateLimited("GetCharactersCharacterIdOrders", characterID, func() ([]esi.GetCharactersCharacterIdOrders200Ok, *http.Response, error) {
				return s.esiClient.ESI.MarketApi.GetCharactersCharacterIdOrders(ctx, characterID, nil)
			})
			if err != nil {
				return nil, err
			}
			slog.Debug("Received open orders from ESI", "count", len(open), "characterID", characterID)
			history, _, err := xesi.RateLimited("GetCharactersCharacterIdOrdersHistory", characterID, func() ([]esi.GetCharactersCharacterIdOrdersHistory200Ok, *http.Response, error) {
				return s.esiClient.ESI.MarketApi.GetCharactersCharacterIdOrdersHistory(ctx, characterID, nil)
			})
			if err != nil {
				return nil, err
			}
			slog.Debug("Received history orders from ESI", "count", len(history), "characterID", characterID)
			orders := make(map[int64]esi.GetCharactersCharacterIdOrdersHistory200Ok)
			for _, o := range open {
				if o.Duration == 0 {
					continue
				}
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
					State:         openState,
					TypeId:        o.TypeId,
					VolumeRemain:  o.VolumeRemain,
					VolumeTotal:   o.VolumeTotal,
				}
			}
			for _, o := range history {
				if o.Duration == 0 {
					continue
				}
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
				case openState:
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
			slog.Info("Stored updated market orders", "characterID", characterID, "count", len(orders))

			// Mark orphaned orders
			incoming := set.Collect(maps.Keys(orders))
			current, err := s.st.ListCharacterMarketOrders(ctx, characterID)
			if err != nil {
				return err
			}
			currentActive := set.Collect(xiter.Map(xiter.FilterSlice(current, func(x *app.CharacterMarketOrder) bool {
				return x.State == app.OrderOpen
			}), func(x *app.CharacterMarketOrder) int64 {
				return x.OrderID
			}))
			orphans := set.Difference(currentActive, incoming)
			if orphans.Size() > 0 {
				// Orders might disappear from the "order" response,
				// but not yet appear in the "history" response due to caching.
				// We mark this with unknown state.
				// They should be updated later with the correct state once the history cache has expired.
				err := s.st.UpdateCharacterMarketOrderState(ctx, storage.UpdateCharacterMarketOrderStateParams{
					CharacterID: characterID,
					OrderIDs:    orphans,
					State:       app.OrderUnknown,
				})
				if err != nil {
					return err
				}
				slog.Info("Marked orphaned market orders", "characterID", characterID, "count", orphans.Size())
			}

			// Delete stale orders
			if arg.MarketOrderRetention == 0 {
				return nil
			}
			current2, err := s.st.ListCharacterMarketOrders(ctx, characterID)
			if err != nil {
				return err
			}
			stale := set.Collect(xiter.Map(xiter.FilterSlice(current2, func(x *app.CharacterMarketOrder) bool {
				return x.State != app.OrderOpen && time.Since(x.Issued) > arg.MarketOrderRetention
			}), func(x *app.CharacterMarketOrder) int64 {
				return x.OrderID
			}))
			if stale.Size() > 0 {
				err := s.st.DeleteCharacterMarketOrders(ctx, characterID, stale)
				if err != nil {
					return err
				}
				slog.Info("Deleted stale market orders", "characterID", characterID, "count", stale.Size())
			}
			return nil
		})
}
