package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

var corporationContractAvailabilityFromDBValue = map[string]app.ContractAvailability{
	"":            app.ContractAvailabilityUndefined,
	"alliance":    app.ContractAvailabilityAlliance,
	"corporation": app.ContractAvailabilityCorporation,
	"personal":    app.ContractAvailabilityPrivate,
	"public":      app.ContractAvailabilityPublic,
}

var corporationContractStatusFromDBValue = map[string]app.ContractStatus{
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

var corporationContractTypeFromDBValue = map[string]app.ContractType{
	"":              app.ContractTypeUndefined,
	"auction":       app.ContractTypeAuction,
	"courier":       app.ContractTypeCourier,
	"item_exchange": app.ContractTypeItemExchange,
	"loan":          app.ContractTypeLoan,
	"unknown":       app.ContractTypeUnknown,
}

var corporationContractAvailabilityToDBValue = map[app.ContractAvailability]string{}
var corporationContractStatusToDBValue = map[app.ContractStatus]string{}
var corporationContractTypeToDBValue = map[app.ContractType]string{}

func init() {
	for k, v := range corporationContractAvailabilityFromDBValue {
		corporationContractAvailabilityToDBValue[v] = k
	}
	for k, v := range corporationContractStatusFromDBValue {
		corporationContractStatusToDBValue[v] = k
	}
	for k, v := range corporationContractTypeFromDBValue {
		corporationContractTypeToDBValue[v] = k
	}
}

type CreateCorporationContractParams struct {
	AcceptorID          int32
	AssigneeID          int32
	Availability        app.ContractAvailability
	Buyout              float64
	CorporationID       int32
	Collateral          float64
	ContractID          int32
	DateAccepted        time.Time
	DateCompleted       time.Time
	DateExpired         time.Time
	DateIssued          time.Time
	DaysToComplete      int32
	EndLocationID       int64
	ForCorporation      bool
	IssuerCorporationID int32
	IssuerID            int32
	Price               float64
	Reward              float64
	StartLocationID     int64
	Status              app.ContractStatus
	StatusNotified      app.ContractStatus
	Title               string
	Type                app.ContractType
	UpdatedAt           time.Time
	Volume              float64
}

func (st *Storage) CreateCorporationContract(ctx context.Context, arg CreateCorporationContractParams) (int64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCorporationContract: %+v: %w", arg, err)
	}
	if arg.CorporationID == 0 || arg.ContractID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	if arg.UpdatedAt.IsZero() {
		arg.UpdatedAt = time.Now().UTC()
	}
	id, err := st.qRW.CreateCorporationContract(ctx, queries.CreateCorporationContractParams{
		AcceptorID:          NewNullInt64(int64(arg.AcceptorID)),
		AssigneeID:          NewNullInt64(int64(arg.AssigneeID)),
		Availability:        corporationContractAvailabilityToDBValue[arg.Availability],
		Buyout:              arg.Buyout,
		CorporationID:       int64(arg.CorporationID),
		Collateral:          arg.Collateral,
		ContractID:          int64(arg.ContractID),
		DateAccepted:        NewNullTimeFromTime(arg.DateAccepted),
		DateCompleted:       NewNullTimeFromTime(arg.DateCompleted),
		DateExpired:         arg.DateExpired,
		DateIssued:          arg.DateIssued,
		DaysToComplete:      int64(arg.DaysToComplete),
		EndLocationID:       NewNullInt64(arg.EndLocationID),
		ForCorporation:      arg.ForCorporation,
		IssuerCorporationID: int64(arg.IssuerCorporationID),
		IssuerID:            int64(arg.IssuerID),
		Price:               arg.Price,
		Reward:              arg.Reward,
		StartLocationID:     NewNullInt64(arg.StartLocationID),
		Status:              corporationContractStatusToDBValue[arg.Status],
		StatusNotified:      corporationContractStatusToDBValue[arg.StatusNotified],
		Title:               arg.Title,
		Type:                corporationContractTypeToDBValue[arg.Type],
		UpdatedAt:           arg.UpdatedAt,
		Volume:              arg.Volume,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	return id, nil
}

func (st *Storage) GetCorporationContract(ctx context.Context, corporationID, contractID int32) (*app.CorporationContract, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCorporationContract for corporation %d: %w", corporationID, err)
	}
	r, err := st.qRO.GetCorporationContract(ctx, queries.GetCorporationContractParams{
		CorporationID: int64(corporationID),
		ContractID:    int64(contractID),
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	c := r.CorporationContract
	acceptor := nullEveEntry{id: c.AcceptorID, name: r.AcceptorName, category: r.AcceptorCategory}
	assignee := nullEveEntry{id: c.AssigneeID, name: r.AssigneeName, category: r.AssigneeCategory}
	o := corporationContractFromDBModel(corporationContractFromDBModelParams{
		acceptor:                 acceptor,
		assignee:                 assignee,
		contract:                 c,
		endLocationName:          r.EndLocationName,
		endSolarSystemID:         r.EndSolarSystemID,
		endSolarSystemName:       r.EndSolarSystemName,
		endSolarSystemSecurity:   r.EndSolarSystemSecurityStatus,
		issuer:                   r.EveEntity_2,
		issuerCorporation:        r.EveEntity,
		items:                    r.Items,
		startLocationName:        r.StartLocationName,
		startSolarSystemID:       r.StartSolarSystemID,
		startSolarSystemName:     r.StartSolarSystemName,
		startSolarSystemSecurity: r.StartSolarSystemSecurityStatus,
	})
	return o, nil
}

func (st *Storage) ListCorporationContractIDs(ctx context.Context, corporationID int32) ([]int32, error) {
	ids, err := st.qRO.ListCorporationContractIDs(ctx, int64(corporationID))
	if err != nil {
		return nil, fmt.Errorf("list contract ids for corporation %d: %w", corporationID, err)
	}
	return convertNumericSlice[int32](ids), nil
}

func (st *Storage) ListCorporationContracts(ctx context.Context, corporationID int32) ([]*app.CorporationContract, error) {
	rows, err := st.qRO.ListCorporationContracts(ctx, int64(corporationID))
	if err != nil {
		return nil, fmt.Errorf("ListCorporationContracts: %w", err)
	}
	oo := make([]*app.CorporationContract, len(rows))
	for i, r := range rows {
		c := r.CorporationContract
		acceptor := nullEveEntry{id: c.AcceptorID, name: r.AcceptorName, category: r.AcceptorCategory}
		assignee := nullEveEntry{id: c.AssigneeID, name: r.AssigneeName, category: r.AssigneeCategory}
		oo[i] = corporationContractFromDBModel(corporationContractFromDBModelParams{
			acceptor:                 acceptor,
			assignee:                 assignee,
			contract:                 c,
			endLocationName:          r.EndLocationName,
			endSolarSystemID:         r.EndSolarSystemID,
			endSolarSystemName:       r.EndSolarSystemName,
			endSolarSystemSecurity:   r.EndSolarSystemSecurityStatus,
			issuer:                   r.EveEntity_2,
			issuerCorporation:        r.EveEntity,
			items:                    r.Items,
			startLocationName:        r.StartLocationName,
			startSolarSystemID:       r.StartSolarSystemID,
			startSolarSystemName:     r.StartSolarSystemName,
			startSolarSystemSecurity: r.StartSolarSystemSecurityStatus,
		})
	}
	return oo, nil
}

type UpdateCorporationContractParams struct {
	AcceptorID    int32
	DateAccepted  time.Time
	DateCompleted time.Time
	CorporationID int32
	ContractID    int32
	Status        app.ContractStatus
}

func (st *Storage) UpdateCorporationContract(ctx context.Context, arg UpdateCorporationContractParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCorporationContract %+v: %w", arg, err)
	}
	if arg.CorporationID == 0 || arg.ContractID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateCorporationContract(ctx, queries.UpdateCorporationContractParams{
		CorporationID: int64(arg.CorporationID),
		ContractID:    int64(arg.ContractID),
		DateAccepted:  NewNullTimeFromTime(arg.DateAccepted),
		DateCompleted: NewNullTimeFromTime(arg.DateCompleted),
		Status:        corporationContractStatusToDBValue[arg.Status],
		UpdatedAt:     time.Now().UTC(),
		AcceptorID:    NewNullInt64(arg.AcceptorID),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) UpdateCorporationContractNotified(ctx context.Context, id int64, status app.ContractStatus) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCorporationContractNotified %d %s: %w", id, status, err)
	}
	if id == 0 {
		return wrapErr(app.ErrInvalid)
	}
	var statusNotified string
	for k, v := range corporationContractStatusFromDBValue {
		if v == status {
			statusNotified = k
			break
		}
	}
	err := st.qRW.UpdateCorporationContractNotified(ctx, queries.UpdateCorporationContractNotifiedParams{
		ID:             id,
		StatusNotified: statusNotified,
		UpdatedAt:      time.Now().UTC(),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

type corporationContractFromDBModelParams struct {
	acceptor                 nullEveEntry
	assignee                 nullEveEntry
	contract                 queries.CorporationContract
	endLocationName          sql.NullString
	endSolarSystemID         sql.NullInt64
	endSolarSystemName       sql.NullString
	endSolarSystemSecurity   sql.NullFloat64
	issuer                   queries.EveEntity
	issuerCorporation        queries.EveEntity
	items                    any
	startLocationName        sql.NullString
	startSolarSystemID       sql.NullInt64
	startSolarSystemName     sql.NullString
	startSolarSystemSecurity sql.NullFloat64
}

func corporationContractFromDBModel(arg corporationContractFromDBModelParams) *app.CorporationContract {
	i2, ok := arg.items.(string)
	if !ok {
		i2 = ""
	}
	o2 := &app.CorporationContract{
		Acceptor:          eveEntityFromNullableDBModel(arg.acceptor),
		Assignee:          eveEntityFromNullableDBModel(arg.assignee),
		Availability:      corporationContractAvailabilityFromDBValue[arg.contract.Availability],
		Buyout:            arg.contract.Buyout,
		CorporationID:     int32(arg.contract.CorporationID),
		Collateral:        arg.contract.Collateral,
		ContractID:        int32(arg.contract.ContractID),
		DateAccepted:      optional.FromNullTime(arg.contract.DateAccepted),
		DateCompleted:     optional.FromNullTime(arg.contract.DateCompleted),
		DateExpired:       arg.contract.DateExpired,
		DateIssued:        arg.contract.DateIssued,
		DaysToComplete:    int32(arg.contract.DaysToComplete),
		ForCorporation:    arg.contract.ForCorporation,
		ID:                arg.contract.ID,
		Issuer:            eveEntityFromDBModel(arg.issuer),
		IssuerCorporation: eveEntityFromDBModel(arg.issuerCorporation),
		Items:             strings.Split(i2, ","),
		Price:             arg.contract.Price,
		Reward:            arg.contract.Reward,
		Status:            corporationContractStatusFromDBValue[arg.contract.Status],
		StatusNotified:    corporationContractStatusFromDBValue[arg.contract.StatusNotified],
		Title:             arg.contract.Title,
		Type:              corporationContractTypeFromDBValue[arg.contract.Type],
		UpdatedAt:         arg.contract.UpdatedAt,
		Volume:            arg.contract.Volume,
	}
	if arg.contract.EndLocationID.Valid {
		o2.EndLocation = &app.EveLocationShort{
			ID:             arg.contract.EndLocationID.Int64,
			Name:           optional.FromNullString(arg.endLocationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.endSolarSystemSecurity),
		}
	}
	if arg.contract.StartLocationID.Valid {
		o2.StartLocation = &app.EveLocationShort{
			ID:             arg.contract.StartLocationID.Int64,
			Name:           optional.FromNullString(arg.startLocationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.startSolarSystemSecurity),
		}
	}
	if arg.endSolarSystemID.Valid && arg.endSolarSystemName.Valid {
		o2.EndSolarSystem = &app.EntityShort[int32]{
			ID:   int32(arg.endSolarSystemID.Int64),
			Name: arg.endSolarSystemName.String,
		}
	}
	if arg.startSolarSystemID.Valid && arg.startSolarSystemName.Valid {
		o2.StartSolarSystem = &app.EntityShort[int32]{
			ID:   int32(arg.startSolarSystemID.Int64),
			Name: arg.startSolarSystemName.String,
		}
	}
	return o2
}

type CreateCorporationContractBidParams struct {
	Amount     float32
	BidderID   int32
	BidID      int32
	ContractID int64
	DateBid    time.Time
}

func (st *Storage) CreateCorporationContractBid(ctx context.Context, arg CreateCorporationContractBidParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCorporationContractBid %+v: %w", arg, err)
	}
	if arg.ContractID == 0 || arg.BidID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if err := st.qRW.CreateCorporationContractBid(ctx, queries.CreateCorporationContractBidParams{
		Amount:     float64(arg.Amount),
		BidderID:   int64(arg.BidderID),
		BidID:      int64(arg.BidID),
		ContractID: arg.ContractID,
		DateBid:    arg.DateBid,
	}); err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetCorporationContractBid(ctx context.Context, contractID int64, bidID int32) (*app.CorporationContractBid, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCorporationContractBid %d %d: %w", contractID, bidID, err)
	}
	if contractID == 0 || bidID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCorporationContractBid(ctx, queries.GetCorporationContractBidParams{
		ContractID: contractID,
		BidID:      int64(bidID),
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	o := corporationContractBidFromDBModel(r.CorporationContractBid, r.EveEntity)
	return o, err
}

func (st *Storage) ListCorporationContractBids(ctx context.Context, contractID int64) ([]*app.CorporationContractBid, error) {
	rows, err := st.qRO.ListCorporationContractBids(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("list bids for contract %d: %w", contractID, err)
	}
	oo := make([]*app.CorporationContractBid, len(rows))
	for i, r := range rows {
		oo[i] = corporationContractBidFromDBModel(r.CorporationContractBid, r.EveEntity)
	}
	return oo, nil
}

func (st *Storage) ListCorporationContractBidIDs(ctx context.Context, contractID int64) (set.Set[int32], error) {
	ids, err := st.qRO.ListCorporationContractBidIDs(ctx, contractID)
	if err != nil {
		return set.Set[int32]{}, fmt.Errorf("list bid IDs for contract %d: %w", contractID, err)
	}
	return set.Of(convertNumericSlice[int32](ids)...), err
}

func corporationContractBidFromDBModel(o queries.CorporationContractBid, e queries.EveEntity) *app.CorporationContractBid {
	o2 := &app.CorporationContractBid{
		ContractID: o.ContractID,
		Amount:     float32(o.Amount),
		BidID:      int32(o.BidID),
		Bidder:     eveEntityFromDBModel(e),
		DateBid:    o.DateBid,
	}
	return o2
}

type CreateCorporationContractItemParams struct {
	ContractID  int64
	IsIncluded  bool
	IsSingleton bool
	Quantity    int32
	RawQuantity int32
	RecordID    int64
	TypeID      int32
}

func (st *Storage) CreateCorporationContractItem(ctx context.Context, arg CreateCorporationContractItemParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCorporationContractItem %+v: %w", arg, err)
	}
	if arg.ContractID == 0 || arg.TypeID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if err := st.qRW.CreateCorporationContractItem(ctx, queries.CreateCorporationContractItemParams{
		ContractID:  arg.ContractID,
		IsIncluded:  arg.IsIncluded,
		IsSingleton: arg.IsSingleton,
		Quantity:    int64(arg.Quantity),
		RawQuantity: int64(arg.RawQuantity),
		RecordID:    arg.RecordID,
		TypeID:      int64(arg.TypeID),
	}); err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetCorporationContractItem(ctx context.Context, contractID, recordID int64) (*app.CorporationContractItem, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCorporationContractItem %d %d: %w", contractID, recordID, err)
	}
	if contractID == 0 || recordID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCorporationContractItem(ctx, queries.GetCorporationContractItemParams{
		ContractID: contractID,
		RecordID:   recordID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	o := corporationContractItemFromDBModel(r.CorporationContractItem, r.EveType, r.EveGroup, r.EveCategory)
	return o, err
}

func (st *Storage) ListCorporationContractItems(ctx context.Context, contractID int64) ([]*app.CorporationContractItem, error) {
	rows, err := st.qRO.ListCorporationContractItems(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("list items for contract %d: %w", contractID, err)
	}
	oo := make([]*app.CorporationContractItem, len(rows))
	for i, r := range rows {
		oo[i] = corporationContractItemFromDBModel(r.CorporationContractItem, r.EveType, r.EveGroup, r.EveCategory)
	}
	return oo, nil
}

func corporationContractItemFromDBModel(o queries.CorporationContractItem, t queries.EveType, g queries.EveGroup, c queries.EveCategory) *app.CorporationContractItem {
	o2 := &app.CorporationContractItem{
		ContractID:  o.ContractID,
		IsIncluded:  o.IsIncluded,
		IsSingleton: o.IsSingleton,
		Quantity:    int(o.Quantity),
		RawQuantity: int(o.RawQuantity),
		RecordID:    o.RecordID,
		Type:        eveTypeFromDBModel(t, g, c),
	}
	return o2
}
