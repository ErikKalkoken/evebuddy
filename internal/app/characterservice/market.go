package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func (s *CharacterService) ListAllMarketOrder(ctx context.Context, isBuyOrders bool) ([]*app.CharacterMarketOrder, error) {
	return s.st.ListAllCharacterMarketOrders(ctx, isBuyOrders)
}

func (s *CharacterService) updateMarketOrdersESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterMarketOrders {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	const openState = "open"
	changed, err := s.updateSectionIfChanged(
		ctx, arg, false,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdOrders")
			open, _, err := s.esiClient.MarketAPI.GetCharactersCharacterIdOrders(ctx, characterID).Execute()
			if err != nil {
				return nil, err
			}
			slog.Debug("Received open orders from ESI", "count", len(open), "characterID", characterID)
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdOrdersHistory")
			history, _, err := s.esiClient.MarketAPI.GetCharactersCharacterIdOrdersHistory(ctx, characterID).Execute()
			if err != nil {
				return nil, err
			}
			slog.Debug("Received history orders from ESI", "count", len(history), "characterID", characterID)
			orders := make(map[int64]esi.CharactersCharacterIdOrdersHistoryGetInner)
			for _, o := range open {
				if o.Duration == 0 {
					continue
				}
				orders[o.OrderId] = esi.CharactersCharacterIdOrdersHistoryGetInner{
					Duration:      o.Duration,
					Escrow:        o.Escrow,
					IsBuyOrder:    o.IsBuyOrder,
					IsCorporation: o.IsCorporation,
					Issued:        o.Issued,
					LocationId:    o.LocationId,
					MinVolume:     o.MinVolume,
					OrderId:       o.OrderId,
					Price:         o.Price,
					Range:         o.Range,
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
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			orders := data.(map[int64]esi.CharactersCharacterIdOrdersHistoryGetInner)
			var locationIDs set.Set[int64]
			var regionIDs, typeIDs set.Set[int64]
			for _, o := range orders {
				locationIDs.Add(o.LocationId)
				regionIDs.Add(o.RegionId)
				typeIDs.Add(o.TypeId)
			}
			ec, err := s.st.GetEveCharacter(ctx, characterID)
			if err != nil {
				return false, err
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
				return false, err
			}
			args := make([]storage.CharacterMarketOrderParams, 0, len(orders))
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
				var ownerID int64
				if o.IsCorporation {
					ownerID = ec.Corporation.ID
				} else {
					ownerID = characterID
				}
				args = append(args, storage.CharacterMarketOrderParams{
					Duration:      o.Duration,
					Escrow:        optional.FromPtr(o.Escrow),
					IsBuyOrder:    optional.FromPtr(o.IsBuyOrder),
					IsCorporation: o.IsCorporation,
					Issued:        o.Issued,
					LocationID:    o.LocationId,
					MinVolume:     optional.FromPtr(o.MinVolume),
					OrderID:       o.OrderId,
					OwnerID:       ownerID,
					Price:         o.Price,
					Range:         o.Range,
					RegionID:      o.RegionId,
					State:         state,
					TypeID:        o.TypeId,
					VolumeRemains: o.VolumeRemain,
					VolumeTotal:   o.VolumeTotal,
				})
			}
			retention := time.Duration(s.settings.MarketOrderRetentionDays()) * time.Hour * 24
			return s.st.ReplaceCharacterMarketOrders(ctx, characterID, retention, args)
		},
	)
	if err != nil {
		return false, err
	}

	// update calculated values
	c, err := s.st.GetCharacter(ctx, arg.characterID)
	if err != nil {
		return false, err
	}
	if arg.forceUpdate || changed || c.OrdersEscrow.IsEmpty() {
		if err := s.updateOrdersEscrow(ctx, arg.characterID); err != nil {
			return false, err
		}
	}
	if arg.forceUpdate || changed || c.OrderItemsValue.IsEmpty() {
		if err := s.updateOrderItemValue(ctx, arg.characterID); err != nil {
			return false, err
		}
	}
	return changed, nil
}

func (s *CharacterService) updateOrderItemValue(ctx context.Context, characterID int64) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("updateOrderItemValue: %d: %w", characterID, err)
	}
	v, err := s.st.CalculateCharacterOrderItemsValue(ctx, characterID)
	if err != nil {
		return wrapErr(err)
	}
	err = s.st.UpdateCharacterOrderItemsValue(ctx, characterID, optional.New(v))
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (s *CharacterService) updateOrdersEscrow(ctx context.Context, characterID int64) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("updateOrdersEscrow: %d: %w", characterID, err)
	}
	v, err := s.st.CalculateCharacterOrdersEscrow(ctx, characterID)
	if err != nil {
		return wrapErr(err)
	}
	err = s.st.UpdateCharacterOrdersEscrow(ctx, characterID, optional.New(v))
	if err != nil {
		return wrapErr(err)
	}
	return nil
}
