package storage

import (
	"context"
	"database/sql"
	"fmt"
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
	"personal":    app.ContractAvailabilityPersonal,
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
	if arg.CharacterID == 0 || arg.ContractID == 0 {
		return 0, fmt.Errorf("create character contract. Mandatory fields not set: %v", arg)
	}
	if arg.UpdatedAt.IsZero() {
		arg.UpdatedAt = time.Now().UTC()
	}
	arg2 := queries.CreateCharacterContractParams{
		Availability:        contractAvailabilityToDBValue[arg.Availability],
		Buyout:              arg.Buyout,
		CharacterID:         int64(arg.CharacterID),
		Collateral:          arg.Collateral,
		ContractID:          int64(arg.ContractID),
		DateAccepted:        NewNullTimeFromTime(arg.DateAccepted),
		DateCompleted:       NewNullTimeFromTime(arg.DateCompleted),
		DateExpired:         arg.DateExpired,
		DateIssued:          arg.DateIssued,
		DaysToComplete:      int64(arg.DaysToComplete),
		ForCorporation:      arg.ForCorporation,
		IssuerCorporationID: int64(arg.IssuerCorporationID),
		IssuerID:            int64(arg.IssuerID),
		Price:               arg.Price,
		Reward:              arg.Reward,
		Status:              contractStatusToDBValue[arg.Status],
		StatusNotified:      contractStatusToDBValue[arg.StatusNotified],
		Title:               arg.Title,
		Type:                contractTypeToDBValue[arg.Type],
		UpdatedAt:           arg.UpdatedAt,
		Volume:              arg.Volume,
	}
	if arg.AcceptorID != 0 {
		arg2.AcceptorID = NewNullInt64(int64(arg.AcceptorID))
	}
	if arg.AssigneeID != 0 {
		arg2.AssigneeID = NewNullInt64(int64(arg.AssigneeID))
	}
	if arg.EndLocationID != 0 {
		arg2.EndLocationID = NewNullInt64(arg.EndLocationID)
	}
	if arg.StartLocationID != 0 {
		arg2.StartLocationID = NewNullInt64(arg.StartLocationID)
	}
	id, err := st.q.CreateCharacterContract(ctx, arg2)
	if err != nil {
		return 0, fmt.Errorf("create contract: %v: %w", arg, err)
	}
	return id, nil
}

func (st *Storage) GetCharacterContract(ctx context.Context, characterID, contractID int32) (*app.CharacterContract, error) {
	arg := queries.GetCharacterContractParams{
		CharacterID: int64(characterID),
		ContractID:  int64(contractID),
	}
	r, err := st.q.GetCharacterContract(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get contract for character %d: %w", characterID, err)
	}
	o := r.CharacterContract
	acceptor := nullEveEntry{ID: o.AcceptorID, Name: r.AcceptorName, Category: r.AcceptorCategory}
	assignee := nullEveEntry{ID: o.AssigneeID, Name: r.AssigneeName, Category: r.AssigneeCategory}
	o2 := characterContractFromDBModel(
		o,
		r.EveEntity,
		r.EveEntity_2,
		acceptor,
		assignee,
		r.EndLocationName,
		r.StartLocationName,
		r.EndSolarSystemID,
		r.EndSolarSystemName,
		r.StartSolarSystemID,
		r.StartSolarSystemName,
		r.Items,
	)
	return o2, err
}

func (st *Storage) ListCharacterContractIDs(ctx context.Context, characterID int32) ([]int32, error) {
	ids, err := st.q.ListCharacterContractIDs(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list contract ids for character %d: %w", characterID, err)
	}
	return convertNumericSlice[int32](ids), nil
}

func (st *Storage) ListCharacterContracts(ctx context.Context, characterID int32) ([]*app.CharacterContract, error) {
	rows, err := st.q.ListCharacterContracts(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list contracts for character %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterContract, len(rows))
	for i, r := range rows {
		o := r.CharacterContract
		acceptor := nullEveEntry{ID: o.AcceptorID, Name: r.AcceptorName, Category: r.AcceptorCategory}
		assignee := nullEveEntry{ID: o.AssigneeID, Name: r.AssigneeName, Category: r.AssigneeCategory}
		oo[i] = characterContractFromDBModel(
			o,
			r.EveEntity,
			r.EveEntity_2,
			acceptor,
			assignee,
			r.EndLocationName,
			r.StartLocationName,
			r.EndSolarSystemID,
			r.EndSolarSystemName,
			r.StartSolarSystemID,
			r.StartSolarSystemName,
			r.Items,
		)
	}
	return oo, nil
}

func (st *Storage) ListCharacterContractsForNotify(ctx context.Context, characterID int32, earliest time.Time) ([]*app.CharacterContract, error) {
	arg := queries.ListCharacterContractsForNotifyParams{
		CharacterID: int64(characterID),
		UpdatedAt:   earliest,
	}
	rows, err := st.q.ListCharacterContractsForNotify(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list contracts to notify for character %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterContract, len(rows))
	for i, r := range rows {
		o := r.CharacterContract
		acceptor := nullEveEntry{ID: o.AcceptorID, Name: r.AcceptorName, Category: r.AcceptorCategory}
		assignee := nullEveEntry{ID: o.AssigneeID, Name: r.AssigneeName, Category: r.AssigneeCategory}
		oo[i] = characterContractFromDBModel(
			o,
			r.EveEntity,
			r.EveEntity_2,
			acceptor,
			assignee,
			r.EndLocationName,
			r.StartLocationName,
			r.EndSolarSystemID,
			r.EndSolarSystemName,
			r.StartSolarSystemID,
			r.StartSolarSystemName,
			r.Items,
		)
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
	if arg.CharacterID == 0 || arg.ContractID == 0 {
		return fmt.Errorf("update character contract. IDs not set: %v", arg)
	}
	arg2 := queries.UpdateCharacterContractParams{
		CharacterID:   int64(arg.CharacterID),
		ContractID:    int64(arg.ContractID),
		DateAccepted:  NewNullTimeFromTime(arg.DateAccepted),
		DateCompleted: NewNullTimeFromTime(arg.DateCompleted),
		Status:        contractStatusToDBValue[arg.Status],
		UpdatedAt:     time.Now().UTC(),
	}
	if arg.AcceptorID != 0 {
		arg2.AcceptorID.Int64 = int64(arg.AcceptorID)
		arg2.AcceptorID.Valid = true
	}
	err := st.q.UpdateCharacterContract(ctx, arg2)
	if err != nil {
		return fmt.Errorf("update character contract: %w", err)
	}
	return nil
}

func (st *Storage) UpdateCharacterContractNotified(ctx context.Context, id int64, status app.ContractStatus) error {
	if id == 0 {
		return fmt.Errorf("update character contract notified. IDs not set: %d", id)
	}
	var statusNotified string
	for k, v := range contractStatusFromDBValue {
		if v == status {
			statusNotified = k
			break
		}
	}
	arg2 := queries.UpdateCharacterContractNotifiedParams{
		ID:             id,
		StatusNotified: statusNotified,
		UpdatedAt:      time.Now().UTC(),
	}
	err := st.q.UpdateCharacterContractNotified(ctx, arg2)
	if err != nil {
		return fmt.Errorf("update character contract notified: %w", err)
	}
	return nil
}

func characterContractFromDBModel(
	o queries.CharacterContract,
	issuerCorporation queries.EveEntity,
	issuer queries.EveEntity,
	acceptor nullEveEntry,
	assignee nullEveEntry,
	endLocationName sql.NullString,
	startLocationName sql.NullString,
	endSolarSystemID sql.NullInt64,
	endSolarSystemName sql.NullString,
	startSolarSystemID sql.NullInt64,
	startSolarSystemName sql.NullString,
	items any,
) *app.CharacterContract {
	i2, ok := items.(string)
	if !ok {
		i2 = ""
	}
	o2 := &app.CharacterContract{
		ID:                o.ID,
		Acceptor:          eveEntityFromNullableDBModel(acceptor),
		Assignee:          eveEntityFromNullableDBModel(assignee),
		Availability:      contractAvailabilityFromDBValue[o.Availability],
		Buyout:            o.Buyout,
		CharacterID:       int32(o.CharacterID),
		Collateral:        o.Collateral,
		ContractID:        int32(o.ContractID),
		DateAccepted:      optional.FromNullTime(o.DateAccepted),
		DateCompleted:     optional.FromNullTime(o.DateCompleted),
		DateExpired:       o.DateExpired,
		DateIssued:        o.DateIssued,
		DaysToComplete:    int32(o.DaysToComplete),
		ForCorporation:    o.ForCorporation,
		IssuerCorporation: eveEntityFromDBModel(issuerCorporation),
		Issuer:            eveEntityFromDBModel(issuer),
		Items:             strings.Split(i2, ","),
		Price:             o.Price,
		Reward:            o.Reward,
		Status:            contractStatusFromDBValue[o.Status],
		StatusNotified:    contractStatusFromDBValue[o.StatusNotified],
		Title:             o.Title,
		Type:              contractTypeFromDBValue[o.Type],
		UpdatedAt:         o.UpdatedAt,
		Volume:            o.Volume,
	}
	if o.EndLocationID.Valid && endLocationName.Valid {
		o2.EndLocation = &app.EntityShort[int64]{
			ID:   o.EndLocationID.Int64,
			Name: endLocationName.String,
		}
	}
	if o.StartLocationID.Valid && startLocationName.Valid {
		o2.StartLocation = &app.EntityShort[int64]{
			ID:   o.StartLocationID.Int64,
			Name: startLocationName.String,
		}
	}
	if endSolarSystemID.Valid && endSolarSystemName.Valid {
		o2.EndSolarSystem = &app.EntityShort[int32]{
			ID:   int32(endSolarSystemID.Int64),
			Name: endSolarSystemName.String,
		}
	}
	if startSolarSystemID.Valid && startSolarSystemName.Valid {
		o2.StartSolarSystem = &app.EntityShort[int32]{
			ID:   int32(startSolarSystemID.Int64),
			Name: startSolarSystemName.String,
		}
	}
	return o2
}
