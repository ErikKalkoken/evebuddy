package app

import "time"

type CharacterContractBid struct {
	ContractID int64
	Amount     float64
	BidID      int32
	Bidder     *EveEntity
	DateBid    time.Time
}
