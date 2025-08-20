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

type CharacterMarketOrder struct {
	CharacterID   int32
	Duration      int
	Escrow        optional.Optional[float64]
	IsBuyOrder    bool
	IsCorporation bool
	Issued        time.Time
	LocationID    int64
	MinVolume     optional.Optional[int]
	OrderID       int64
	Price         float64
	Range         string
	RegionID      int32
	State         MarketOrderState
	TypeID        int32
	VolumeRemains int
	VolumeTotal   int
}
