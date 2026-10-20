package storage

import (
	"database/sql"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

var contractAvailabilityFromDBValue = map[string]app.ContractAvailability{
	"":            app.ContractAvailabilityUndefined,
	"alliance":    app.ContractAvailabilityAlliance,
	"corporation": app.ContractAvailabilityCorporation,
	"personal":    app.ContractAvailabilityPrivate,
	"public":      app.ContractAvailabilityPublic,
}

var contractStatusFromDBValue = map[string]app.ContractStatus{
	"":                    app.ContractStatusUndefined,
	"cancelled":           app.ContractStatusCancelled,
	"deleted":             app.ContractStatusDeleted,
	"failed":              app.ContractStatusFailed,
	"finished_contractor": app.ContractStatusFinishedContractor,
	"finished_issuer":     app.ContractStatusFinishedIssuer,
	"finished":            app.ContractStatusFinished,
	"in_progress":         app.ContractStatusInProgress,
	"outstanding":         app.ContractStatusOutstanding,
	"rejected":            app.ContractStatusRejected,
	"reversed":            app.ContractStatusReversed,
}

var contractTypeFromDBValue = map[string]app.ContractType{
	"":              app.ContractTypeUndefined,
	"auction":       app.ContractTypeAuction,
	"courier":       app.ContractTypeCourier,
	"item_exchange": app.ContractTypeItemExchange,
	"loan":          app.ContractTypeLoan,
	"unknown":       app.ContractTypeUnknown,
}

var contractAvailabilityToDBValue = map[app.ContractAvailability]string{}
var contractStatusToDBValue = map[app.ContractStatus]string{}
var contractTypeToDBValue = map[app.ContractType]string{}

func init() {
	for k, v := range contractAvailabilityFromDBValue {
		contractAvailabilityToDBValue[v] = k
	}
	for k, v := range contractStatusFromDBValue {
		contractStatusToDBValue[v] = k
	}
	for k, v := range contractTypeFromDBValue {
		contractTypeToDBValue[v] = k
	}
}

// Contract statuses in which courier collateral is held in escrow.
var contractEscrowStatuses = []app.ContractStatus{app.ContractStatusOutstanding, app.ContractStatusInProgress}

// contractStatusesToDBValues returns the DB values for statuses.
func contractStatusesToDBValues(statuses []app.ContractStatus) []string {
	return xslices.Map(statuses, func(x app.ContractStatus) string {
		return contractStatusToDBValue[x]
	})
}

// contractBidDBModel has the same fields as the bid rows of both owners,
// so they can be converted directly.
type contractBidDBModel struct {
	ID         int64
	ContractID int64
	Amount     float64
	BidID      int64
	BidderID   int64
	DateBid    time.Time
}

// contractItemDBModel has the same fields as the item rows of both owners,
// so they can be converted directly.
type contractItemDBModel struct {
	ID          int64
	ContractID  int64
	IsIncluded  bool
	IsSingleton bool
	Quantity    int64
	RawQuantity int64
	RecordID    int64
	TypeID      int64
}

type CreateContractBidParams struct {
	Amount     float64
	BidderID   int64
	BidID      int64
	ContractID int64
	DateBid    time.Time
}

type CreateContractItemParams struct {
	ContractID  int64
	IsIncluded  bool
	IsSingleton bool
	Quantity    int64
	RawQuantity optional.Optional[int64]
	RecordID    int64
	TypeID      int64
}

func contractItemNamesFromDBModel(items any) []string {
	x, ok := items.(string)
	if !ok || x == "" {
		return nil
	}
	return strings.Split(x, ",")
}

func contractLocationFromDBModel(id sql.NullInt64, name sql.NullString, security sql.NullFloat64) optional.Optional[*app.EveLocationShort] {
	if !id.Valid {
		return optional.Optional[*app.EveLocationShort]{}
	}
	return optional.New(&app.EveLocationShort{
		ID:             id.Int64,
		Name:           optional.FromNullString(name),
		SecurityStatus: optional.FromNullFloat64ToFloat32(security),
	})
}

func contractSolarSystemFromDBModel(id sql.NullInt64, name sql.NullString) optional.Optional[*app.EntityShort] {
	if !id.Valid || !name.Valid {
		return optional.Optional[*app.EntityShort]{}
	}
	return optional.New(&app.EntityShort{
		ID:   id.Int64,
		Name: name.String,
	})
}

// contractBidFromDBModel maps bids of both owners.
func contractBidFromDBModel(o contractBidDBModel, e queries.EveEntity) *app.ContractBid {
	return &app.ContractBid{
		ContractID: o.ContractID,
		Amount:     o.Amount,
		BidID:      o.BidID,
		Bidder:     eveEntityFromDBModel(e),
		DateBid:    o.DateBid,
	}
}

// contractItemFromDBModel maps items of both owners.
func contractItemFromDBModel(o contractItemDBModel, t queries.EveType, g queries.EveGroup, c queries.EveCategory) *app.ContractItem {
	return &app.ContractItem{
		ContractID:  o.ContractID,
		IsIncluded:  o.IsIncluded,
		IsSingleton: o.IsSingleton,
		Quantity:    o.Quantity,
		RawQuantity: optional.FromZeroValue(o.RawQuantity),
		RecordID:    o.RecordID,
		Type:        eveTypeFromDBModel(t, g, c),
	}
}
