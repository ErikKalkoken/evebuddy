package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
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
	AcceptorID          int64
	AssigneeID          int64
	Availability        app.ContractAvailability
	Buyout              optional.Optional[float64]
	CorporationID       int64
	Collateral          optional.Optional[float64]
	ContractID          int64
	DateAccepted        optional.Optional[time.Time]
	DateCompleted       optional.Optional[time.Time]
	DateExpired         time.Time
	DateIssued          time.Time
	DaysToComplete      optional.Optional[int64]
	EndLocationID       optional.Optional[int64]
	ForCorporation      bool
	IssuerCorporationID int64
	IssuerID            int64
	Price               optional.Optional[float64]
	Reward              optional.Optional[float64]
	StartLocationID     optional.Optional[int64]
	Status              app.ContractStatus
	StatusNotified      app.ContractStatus
	Title               optional.Optional[string]
	Type                app.ContractType
	UpdatedAt           time.Time
	Volume              optional.Optional[float64]
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
		AcceptorID:          NewNullInt64(arg.AcceptorID),
		AssigneeID:          NewNullInt64(arg.AssigneeID),
		Availability:        corporationContractAvailabilityToDBValue[arg.Availability],
		Buyout:              arg.Buyout.ValueOrZero(),
		CorporationID:       arg.CorporationID,
		Collateral:          arg.Collateral.ValueOrZero(),
		ContractID:          arg.ContractID,
		DateAccepted:        optional.ToNullTime(arg.DateAccepted),
		DateCompleted:       optional.ToNullTime(arg.DateCompleted),
		DateExpired:         arg.DateExpired,
		DateIssued:          arg.DateIssued,
		DaysToComplete:      arg.DaysToComplete.ValueOrZero(),
		EndLocationID:       optional.ToNullInt64(arg.EndLocationID),
		ForCorporation:      arg.ForCorporation,
		IssuerCorporationID: arg.IssuerCorporationID,
		IssuerID:            arg.IssuerID,
		Price:               arg.Price.ValueOrZero(),
		Reward:              arg.Reward.ValueOrZero(),
		StartLocationID:     optional.ToNullInt64(arg.StartLocationID),
		Status:              corporationContractStatusToDBValue[arg.Status],
		StatusNotified:      corporationContractStatusToDBValue[arg.StatusNotified],
		Title:               arg.Title.ValueOrZero(),
		Type:                corporationContractTypeToDBValue[arg.Type],
		UpdatedAt:           arg.UpdatedAt,
		Volume:              arg.Volume.ValueOrZero(),
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	return id, nil
}

func (st *Storage) DeleteCorporationContracts(ctx context.Context, corporationID int64, contractIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCorporationContracts for character %d and contract IDs: %s: %w", corporationID, contractIDs, err)
	}
	if corporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if contractIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.DeleteCorporationContracts(ctx, queries.DeleteCorporationContractsParams{
		CorporationID: corporationID,
		ContractIds:   slices.Collect(contractIDs.All()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Contracts deleted for corporation", "corporationID", corporationID, "contractIDs", contractIDs)
	return nil
}

func (st *Storage) GetCorporationContract(ctx context.Context, corporationID, contractID int64) (*app.CorporationContract, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCorporationContract for corporation %d: %w", corporationID, err)
	}
	r, err := st.qRO.GetCorporationContract(ctx, queries.GetCorporationContractParams{
		CorporationID: corporationID,
		ContractID:    contractID,
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

func (st *Storage) ListCorporationContractIDs(ctx context.Context, corporationID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCorporationContractIDs(ctx, corporationID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list contract ids for corporation %d: %w", corporationID, err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListCorporationContracts(ctx context.Context, corporationID int64) ([]*app.CorporationContract, error) {
	rows, err := st.qRO.ListCorporationContracts(ctx, corporationID)
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
	AcceptorID    int64
	DateAccepted  optional.Optional[time.Time]
	DateCompleted optional.Optional[time.Time]
	CorporationID int64
	ContractID    int64
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
		CorporationID: arg.CorporationID,
		ContractID:    arg.ContractID,
		DateAccepted:  optional.ToNullTime(arg.DateAccepted),
		DateCompleted: optional.ToNullTime(arg.DateCompleted),
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
		Buyout:            optional.FromZeroValue(arg.contract.Buyout),
		CorporationID:     arg.contract.CorporationID,
		Collateral:        optional.FromZeroValue(arg.contract.Collateral),
		ContractID:        arg.contract.ContractID,
		DateAccepted:      optional.FromNullTime(arg.contract.DateAccepted),
		DateCompleted:     optional.FromNullTime(arg.contract.DateCompleted),
		DateExpired:       arg.contract.DateExpired,
		DateIssued:        arg.contract.DateIssued,
		DaysToComplete:    optional.FromZeroValue(arg.contract.DaysToComplete),
		ForCorporation:    arg.contract.ForCorporation,
		ID:                arg.contract.ID,
		Issuer:            eveEntityFromDBModel(arg.issuer),
		IssuerCorporation: eveEntityFromDBModel(arg.issuerCorporation),
		Items:             strings.Split(i2, ","),
		Price:             optional.FromZeroValue(arg.contract.Price),
		Reward:            optional.FromZeroValue(arg.contract.Reward),
		Status:            corporationContractStatusFromDBValue[arg.contract.Status],
		StatusNotified:    corporationContractStatusFromDBValue[arg.contract.StatusNotified],
		Title:             optional.FromZeroValue(arg.contract.Title),
		Type:              corporationContractTypeFromDBValue[arg.contract.Type],
		UpdatedAt:         arg.contract.UpdatedAt,
		Volume:            optional.FromZeroValue(arg.contract.Volume),
	}
	if arg.contract.EndLocationID.Valid {
		o2.EndLocation = optional.New(&app.EveLocationShort{
			ID:             arg.contract.EndLocationID.Int64,
			Name:           optional.FromNullString(arg.endLocationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.endSolarSystemSecurity),
		})
	}
	if arg.contract.StartLocationID.Valid {
		o2.StartLocation = optional.New(&app.EveLocationShort{
			ID:             arg.contract.StartLocationID.Int64,
			Name:           optional.FromNullString(arg.startLocationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.startSolarSystemSecurity),
		})
	}
	if arg.endSolarSystemID.Valid && arg.endSolarSystemName.Valid {
		o2.EndSolarSystem = optional.New(&app.EntityShort[int64]{
			ID:   arg.endSolarSystemID.Int64,
			Name: arg.endSolarSystemName.String,
		})
	}
	if arg.startSolarSystemID.Valid && arg.startSolarSystemName.Valid {
		o2.StartSolarSystem = optional.New(&app.EntityShort[int64]{
			ID:   arg.startSolarSystemID.Int64,
			Name: arg.startSolarSystemName.String,
		})
	}
	return o2
}

type CreateCorporationContractBidParams struct {
	Amount     float64
	BidderID   int64
	BidID      int64
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
		BidderID:   arg.BidderID,
		BidID:      arg.BidID,
		ContractID: arg.ContractID,
		DateBid:    arg.DateBid,
	}); err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetCorporationContractBid(ctx context.Context, contractID int64, bidID int64) (*app.CorporationContractBid, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCorporationContractBid %d %d: %w", contractID, bidID, err)
	}
	if contractID == 0 || bidID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCorporationContractBid(ctx, queries.GetCorporationContractBidParams{
		ContractID: contractID,
		BidID:      bidID,
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

func (st *Storage) ListCorporationContractBidIDs(ctx context.Context, contractID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCorporationContractBidIDs(ctx, contractID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list bid IDs for contract %d: %w", contractID, err)
	}
	return set.Of(ids...), err
}

func corporationContractBidFromDBModel(o queries.CorporationContractBid, e queries.EveEntity) *app.CorporationContractBid {
	o2 := &app.CorporationContractBid{
		ContractID: o.ContractID,
		Amount:     o.Amount,
		BidID:      o.BidID,
		Bidder:     eveEntityFromDBModel(e),
		DateBid:    o.DateBid,
	}
	return o2
}

type CreateCorporationContractItemParams struct {
	ContractID  int64
	IsIncluded  bool
	IsSingleton bool
	Quantity    int64
	RawQuantity optional.Optional[int64]
	RecordID    int64
	TypeID      int64
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
		Quantity:    arg.Quantity,
		RawQuantity: arg.RawQuantity.ValueOrZero(),
		RecordID:    arg.RecordID,
		TypeID:      arg.TypeID,
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
		Quantity:    o.Quantity,
		RawQuantity: optional.FromZeroValue(o.RawQuantity),
		RecordID:    o.RecordID,
		Type:        eveTypeFromDBModel(t, g, c),
	}
	return o2
}
