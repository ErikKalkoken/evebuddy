package app

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// MarketOrderState represents the current state of a market order.
type MarketOrderState uint

const (
	OrderUndefined MarketOrderState = iota // zero value
	OrderCancelled
	OrderExpired
	OrderOpen
	OrderUnknown // status can not be determined
)

func (mos MarketOrderState) String() string {
	switch mos {
	case OrderCancelled:
		return "cancelled"
	case OrderExpired:
		return "expired"
	case OrderOpen:
		return "open"
	case OrderUndefined:
		return "undefined"
	case OrderUnknown:
		return "unknown"
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
	Location      *EveLocationShort
	MinVolume     optional.Optional[int]
	OrderID       int64
	Owner         *EveEntity
	Price         float64
	Range         string
	Region        *EntityShort[int32]
	State         MarketOrderState
	Type          *EntityShort[int32]
	VolumeRemains int
	VolumeTotal   int
}
