package app

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// MarketOrderState represents the current state of a market order.
type MarketOrderState uint

const (
	OrderUndefined MarketOrderState = iota
	OrderCancelled
	OrderExpired
	OrderOpen
)

func (mos MarketOrderState) String() string {
	switch mos {
	case OrderUndefined:
		return "undefined"
	case OrderCancelled:
		return "cancelled"
	case OrderExpired:
		return "expired"
	case OrderOpen:
		return "open"
	}
	return "?"
}

type CharacterMarketOrder struct {
	CharacterID   int32
	Duration      int
	Escrow        optional.Optional[float64]
	IsBuyOrder    bool
	IsCorporation bool
	Issued        time.Time
	Location      *EntityShort[int64]
	MinVolume     optional.Optional[int]
	OrderID       int64
	Price         float64
	Range         string
	Region        *EntityShort[int32]
	State         MarketOrderState
	Type          *EntityShort[int32]
	VolumeRemains int
	VolumeTotal   int
}
