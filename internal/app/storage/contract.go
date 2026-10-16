package storage

import (
	"database/sql"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
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

func setContractLocationsFromDBModel(
	o *app.Contract,
	endLocationID sql.NullInt64,
	endLocationName endLocationName,
	endSolarSystemID endSolarSystemID,
	endSolarSystemName endSolarSystemName,
	endSolarSystemSecurity endSolarSystemSecurity,
	startLocationID sql.NullInt64,
	startLocationName startLocationName,
	startSolarSystemID startSolarSystemID,
	startSolarSystemName startSolarSystemName,
	startSolarSystemSecurity startSolarSystemSecurity,
) {
	if endLocationID.Valid {
		o.EndLocation = optional.New(&app.EveLocationShort{
			ID:             endLocationID.Int64,
			Name:           optional.FromNullString(sql.NullString(endLocationName)),
			SecurityStatus: optional.FromNullFloat64ToFloat32(sql.NullFloat64(endSolarSystemSecurity)),
		})
	}
	if startLocationID.Valid {
		o.StartLocation = optional.New(&app.EveLocationShort{
			ID:             startLocationID.Int64,
			Name:           optional.FromNullString(sql.NullString(startLocationName)),
			SecurityStatus: optional.FromNullFloat64ToFloat32(sql.NullFloat64(startSolarSystemSecurity)),
		})
	}
	if endSolarSystemID.Valid && endSolarSystemName.Valid {
		o.EndSolarSystem = optional.New(&app.EntityShort{
			ID:   endSolarSystemID.Int64,
			Name: endSolarSystemName.String,
		})
	}
	if startSolarSystemID.Valid && startSolarSystemName.Valid {
		o.StartSolarSystem = optional.New(&app.EntityShort{
			ID:   startSolarSystemID.Int64,
			Name: startSolarSystemName.String,
		})
	}
}

// contractBidFromDBModel maps bids of both owners, since their rows have identical fields.
func contractBidFromDBModel(o queries.CharacterContractBid, e queries.EveEntity) *app.ContractBid {
	return &app.ContractBid{
		ContractID: o.ContractID,
		Amount:     o.Amount,
		BidID:      o.BidID,
		Bidder:     eveEntityFromDBModel(e),
		DateBid:    o.DateBid,
	}
}

// contractItemFromDBModel maps items of both owners, since their rows have identical fields.
func contractItemFromDBModel(o queries.CharacterContractItem, t queries.EveType, g queries.EveGroup, c queries.EveCategory) *app.ContractItem {
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
