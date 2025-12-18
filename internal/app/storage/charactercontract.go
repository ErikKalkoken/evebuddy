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

var characterContractAvailabilityFromDBValue = map[string]app.ContractAvailability{
	"":            app.ContractAvailabilityUndefined,
	"alliance":    app.ContractAvailabilityAlliance,
	"corporation": app.ContractAvailabilityCorporation,
	"personal":    app.ContractAvailabilityPrivate,
	"public":      app.ContractAvailabilityPublic,
}

var characterContractStatusFromDBValue = map[string]app.ContractStatus{
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

var characterContractTypeFromDBValue = map[string]app.ContractType{
	"":              app.ContractTypeUndefined,
	"auction":       app.ContractTypeAuction,
	"courier":       app.ContractTypeCourier,
	"item_exchange": app.ContractTypeItemExchange,
	"loan":          app.ContractTypeLoan,
	"unknown":       app.ContractTypeUnknown,
}

var characterContractAvailabilityToDBValue = map[app.ContractAvailability]string{}
var characterContractStatusToDBValue = map[app.ContractStatus]string{}
var characterContractTypeToDBValue = map[app.ContractType]string{}

func init() {
	for k, v := range characterContractAvailabilityFromDBValue {
		characterContractAvailabilityToDBValue[v] = k
	}
	for k, v := range characterContractStatusFromDBValue {
		characterContractStatusToDBValue[v] = k
	}
	for k, v := range characterContractTypeFromDBValue {
		characterContractTypeToDBValue[v] = k
	}
}

type CreateCharacterContractParams struct {
	AcceptorID          int32
	AssigneeID          int32
	Availability        app.ContractAvailability
	Buyout              float64
	CharacterID         int32
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

func (st *Storage) CreateCharacterContract(ctx context.Context, arg CreateCharacterContractParams) (int64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharacterContract: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.ContractID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	if arg.UpdatedAt.IsZero() {
		arg.UpdatedAt = time.Now().UTC()
	}
	id, err := st.qRW.CreateCharacterContract(ctx, queries.CreateCharacterContractParams{
		AcceptorID:          NewNullInt64(int64(arg.AcceptorID)),
		AssigneeID:          NewNullInt64(int64(arg.AssigneeID)),
		Availability:        characterContractAvailabilityToDBValue[arg.Availability],
		Buyout:              arg.Buyout,
		CharacterID:         int64(arg.CharacterID),
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
		Status:              characterContractStatusToDBValue[arg.Status],
		StatusNotified:      characterContractStatusToDBValue[arg.StatusNotified],
		Title:               arg.Title,
		Type:                characterContractTypeToDBValue[arg.Type],
		UpdatedAt:           arg.UpdatedAt,
		Volume:              arg.Volume,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	return id, nil
}

func (st *Storage) GetCharacterContract(ctx context.Context, characterID, contractID int32) (*app.CharacterContract, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterContract for character %d: %w", characterID, err)
	}
	r, err := st.qRO.GetCharacterContract(ctx, queries.GetCharacterContractParams{
		CharacterID: int64(characterID),
		ContractID:  int64(contractID),
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	c := r.CharacterContract
	acceptor := nullEveEntry{id: c.AcceptorID, name: r.AcceptorName, category: r.AcceptorCategory}
	assignee := nullEveEntry{id: c.AssigneeID, name: r.AssigneeName, category: r.AssigneeCategory}
	o := characterContractFromDBModel(characterContractFromDBModelParams{
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

func (st *Storage) ListCharacterContractIDs(ctx context.Context, characterID int32) ([]int32, error) {
	ids, err := st.qRO.ListCharacterContractIDs(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list contract ids for character %d: %w", characterID, err)
	}
	return convertNumericSlice[int32](ids), nil
}

func (st *Storage) ListAllCharacterContracts(ctx context.Context) ([]*app.CharacterContract, error) {
	rows, err := st.qRO.ListAllCharacterContracts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list character contracts: %w", err)
	}
	oo := make([]*app.CharacterContract, len(rows))
	for i, r := range rows {
		c := r.CharacterContract
		acceptor := nullEveEntry{id: c.AcceptorID, name: r.AcceptorName, category: r.AcceptorCategory}
		assignee := nullEveEntry{id: c.AssigneeID, name: r.AssigneeName, category: r.AssigneeCategory}
		oo[i] = characterContractFromDBModel(characterContractFromDBModelParams{
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

func (st *Storage) ListCharacterContractsForNotify(ctx context.Context, characterID int32, earliest time.Time) ([]*app.CharacterContract, error) {
	rows, err := st.qRO.ListCharacterContractsForNotify(ctx, queries.ListCharacterContractsForNotifyParams{
		CharacterID: int64(characterID),
		UpdatedAt:   earliest,
	})
	if err != nil {
		return nil, fmt.Errorf("list contracts to notify for character %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterContract, len(rows))
	for i, r := range rows {
		c := r.CharacterContract
		acceptor := nullEveEntry{id: c.AcceptorID, name: r.AcceptorName, category: r.AcceptorCategory}
		assignee := nullEveEntry{id: c.AssigneeID, name: r.AssigneeName, category: r.AssigneeCategory}
		oo[i] = characterContractFromDBModel(characterContractFromDBModelParams{
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

type UpdateCharacterContractParams struct {
	AcceptorID    int32
	DateAccepted  time.Time
	DateCompleted time.Time
	CharacterID   int32
	ContractID    int32
	Status        app.ContractStatus
}

func (st *Storage) UpdateCharacterContract(ctx context.Context, arg UpdateCharacterContractParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterContract %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.ContractID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateCharacterContract(ctx, queries.UpdateCharacterContractParams{
		CharacterID:   int64(arg.CharacterID),
		ContractID:    int64(arg.ContractID),
		DateAccepted:  NewNullTimeFromTime(arg.DateAccepted),
		DateCompleted: NewNullTimeFromTime(arg.DateCompleted),
		Status:        characterContractStatusToDBValue[arg.Status],
		UpdatedAt:     time.Now().UTC(),
		AcceptorID:    NewNullInt64(arg.AcceptorID),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) UpdateCharacterContractNotified(ctx context.Context, id int64, status app.ContractStatus) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterContractNotified %d %s: %w", id, status, err)
	}
	if id == 0 {
		return wrapErr(app.ErrInvalid)
	}
	var statusNotified string
	for k, v := range characterContractStatusFromDBValue {
		if v == status {
			statusNotified = k
			break
		}
	}
	err := st.qRW.UpdateCharacterContractNotified(ctx, queries.UpdateCharacterContractNotifiedParams{
		ID:             id,
		StatusNotified: statusNotified,
		UpdatedAt:      time.Now().UTC(),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

type characterContractFromDBModelParams struct {
	acceptor                 nullEveEntry
	assignee                 nullEveEntry
	contract                 queries.CharacterContract
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

func characterContractFromDBModel(arg characterContractFromDBModelParams) *app.CharacterContract {
	i2, ok := arg.items.(string)
	if !ok {
		i2 = ""
	}
	o2 := &app.CharacterContract{
		Acceptor:          eveEntityFromNullableDBModel(arg.acceptor),
		Assignee:          eveEntityFromNullableDBModel(arg.assignee),
		Availability:      characterContractAvailabilityFromDBValue[arg.contract.Availability],
		Buyout:            arg.contract.Buyout,
		CharacterID:       int32(arg.contract.CharacterID),
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
		Status:            characterContractStatusFromDBValue[arg.contract.Status],
		StatusNotified:    characterContractStatusFromDBValue[arg.contract.StatusNotified],
		Title:             arg.contract.Title,
		Type:              characterContractTypeFromDBValue[arg.contract.Type],
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

type CreateCharacterContractBidParams struct {
	Amount     float32
	BidderID   int32
	BidID      int32
	ContractID int64
	DateBid    time.Time
}

func (st *Storage) CreateCharacterContractBid(ctx context.Context, arg CreateCharacterContractBidParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharacterContractBid %+v: %w", arg, err)
	}
	if arg.ContractID == 0 || arg.BidID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if err := st.qRW.CreateCharacterContractBid(ctx, queries.CreateCharacterContractBidParams{
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

func (st *Storage) GetCharacterContractBid(ctx context.Context, contractID int64, bidID int32) (*app.CharacterContractBid, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterContractBid %d %d: %w", contractID, bidID, err)
	}
	if contractID == 0 || bidID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCharacterContractBid(ctx, queries.GetCharacterContractBidParams{
		ContractID: contractID,
		BidID:      int64(bidID),
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	o := characterContractBidFromDBModel(r.CharacterContractBid, r.EveEntity)
	return o, err
}

func (st *Storage) ListCharacterContractBids(ctx context.Context, contractID int64) ([]*app.CharacterContractBid, error) {
	rows, err := st.qRO.ListCharacterContractBids(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("list bids for contract %d: %w", contractID, err)
	}
	oo := make([]*app.CharacterContractBid, len(rows))
	for i, r := range rows {
		oo[i] = characterContractBidFromDBModel(r.CharacterContractBid, r.EveEntity)
	}
	return oo, nil
}

func (st *Storage) ListCharacterContractBidIDs(ctx context.Context, contractID int64) (set.Set[int32], error) {
	ids, err := st.qRO.ListCharacterContractBidIDs(ctx, contractID)
	if err != nil {
		return set.Set[int32]{}, fmt.Errorf("list bid IDs for contract %d: %w", contractID, err)
	}
	return set.Of(convertNumericSlice[int32](ids)...), err
}

func characterContractBidFromDBModel(o queries.CharacterContractBid, e queries.EveEntity) *app.CharacterContractBid {
	o2 := &app.CharacterContractBid{
		ContractID: o.ContractID,
		Amount:     float32(o.Amount),
		BidID:      int32(o.BidID),
		Bidder:     eveEntityFromDBModel(e),
		DateBid:    o.DateBid,
	}
	return o2
}

type CreateCharacterContractItemParams struct {
	ContractID  int64
	IsIncluded  bool
	IsSingleton bool
	Quantity    int32
	RawQuantity int32
	RecordID    int64
	TypeID      int32
}

func (st *Storage) CreateCharacterContractItem(ctx context.Context, arg CreateCharacterContractItemParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharacterContractItem %+v: %w", arg, err)
	}
	if arg.ContractID == 0 || arg.TypeID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if err := st.qRW.CreateCharacterContractItem(ctx, queries.CreateCharacterContractItemParams{
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

func (st *Storage) GetCharacterContractItem(ctx context.Context, contractID, recordID int64) (*app.CharacterContractItem, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterContractItem %d %d: %w", contractID, recordID, err)
	}
	if contractID == 0 || recordID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCharacterContractItem(ctx, queries.GetCharacterContractItemParams{
		ContractID: contractID,
		RecordID:   recordID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	o := characterContractItemFromDBModel(r.CharacterContractItem, r.EveType, r.EveGroup, r.EveCategory)
	return o, err
}

func (st *Storage) ListCharacterContractItems(ctx context.Context, contractID int64) ([]*app.CharacterContractItem, error) {
	rows, err := st.qRO.ListCharacterContractItems(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("list items for contract %d: %w", contractID, err)
	}
	oo := make([]*app.CharacterContractItem, len(rows))
	for i, r := range rows {
		oo[i] = characterContractItemFromDBModel(r.CharacterContractItem, r.EveType, r.EveGroup, r.EveCategory)
	}
	return oo, nil
}

func characterContractItemFromDBModel(o queries.CharacterContractItem, t queries.EveType, g queries.EveGroup, c queries.EveCategory) *app.CharacterContractItem {
	o2 := &app.CharacterContractItem{
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
