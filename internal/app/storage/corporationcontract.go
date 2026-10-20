package storage

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

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

// CalculateCorporationContractsEscrow returns the ISK a corporation holds in escrow for open contracts:
// collateral of accepted couriers and rewards of couriers and item exchanges issued for the corporation.
// Auction bids are not included: ESI names the bidding character, not the corporation.
func (st *Storage) CalculateCorporationContractsEscrow(ctx context.Context, corporationID int64) (float64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CalculateCorporationContractsEscrow: %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	status := contractStatusesToDBValues(contractEscrowStatuses)
	collateral, err := st.qRO.CalculateCorporationContractsCollateralEscrow(ctx, queries.CalculateCorporationContractsCollateralEscrowParams{
		CorporationID: corporationID,
		Type:          contractTypeToDBValue[app.ContractTypeCourier],
		Status:        status,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	reward, err := st.qRO.CalculateCorporationContractsRewardEscrow(ctx, queries.CalculateCorporationContractsRewardEscrowParams{
		CorporationID: corporationID,
		Types: []string{
			contractTypeToDBValue[app.ContractTypeCourier],
			contractTypeToDBValue[app.ContractTypeItemExchange],
		},
		Status: status,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	return collateral.Float64 + reward.Float64, nil
}

func (st *Storage) CountCorporationContractsWithStatus(ctx context.Context, corporationID int64, statuses set.Set[app.ContractStatus]) (int64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CountCorporationContractsWithStatus: %d %v: %w", corporationID, statuses, err)
	}
	if statuses.Size() == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	var statuses2 []string
	for s := range statuses.All() {
		statuses2 = append(statuses2, contractStatusToDBValue[s])
	}
	v, err := st.qRO.CountCorporationContractsWithStatus(ctx, queries.CountCorporationContractsWithStatusParams{
		CorporationID: corporationID,
		Statuses:      statuses2,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	return v, nil
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
		Availability:        contractAvailabilityToDBValue[arg.Availability],
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
		Status:              contractStatusToDBValue[arg.Status],
		StatusNotified:      contractStatusToDBValue[arg.StatusNotified],
		Title:               arg.Title.ValueOrZero(),
		Type:                contractTypeToDBValue[arg.Type],
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
		return fmt.Errorf("DeleteCorporationContracts for corporation %d and contract IDs: %s: %w", corporationID, contractIDs, err)
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
	o := corporationContractFromDBModel(r)
	return o, nil
}

func (st *Storage) ListCorporationContractIDs(ctx context.Context, corporationID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCorporationContractIDs(ctx, corporationID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list contract ids for corporation %d: %w", corporationID, err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) ListCorporationContracts(ctx context.Context, corporationID int64) ([]*app.CorporationContract, error) {
	rows, err := st.qRO.ListCorporationContracts(ctx, corporationID)
	if err != nil {
		return nil, fmt.Errorf("ListCorporationContracts: %w", err)
	}
	oo := make([]*app.CorporationContract, len(rows))
	for i, r := range rows {
		oo[i] = corporationContractFromDBModel(queries.GetCorporationContractRow(r))
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
		Status:        contractStatusToDBValue[arg.Status],
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
	err := st.qRW.UpdateCorporationContractNotified(ctx, queries.UpdateCorporationContractNotifiedParams{
		ID:             id,
		StatusNotified: contractStatusToDBValue[status],
		UpdatedAt:      time.Now().UTC(),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func corporationContractFromDBModel(r queries.GetCorporationContractRow) *app.CorporationContract {
	cc := r.CorporationContract
	o2 := &app.CorporationContract{
		Contract: app.Contract{
			Acceptor:          eveEntityFromNullableDBModel(nullEveEntity{id: cc.AcceptorID, name: r.AcceptorName, category: r.AcceptorCategory}),
			Assignee:          eveEntityFromNullableDBModel(nullEveEntity{id: cc.AssigneeID, name: r.AssigneeName, category: r.AssigneeCategory}),
			Availability:      contractAvailabilityFromDBValue[cc.Availability],
			Buyout:            optional.FromZeroValue(cc.Buyout),
			Collateral:        optional.FromZeroValue(cc.Collateral),
			ContractID:        cc.ContractID,
			DateAccepted:      optional.FromNullTime(cc.DateAccepted),
			DateCompleted:     optional.FromNullTime(cc.DateCompleted),
			DateExpired:       cc.DateExpired,
			DateIssued:        cc.DateIssued,
			DaysToComplete:    optional.FromZeroValue(cc.DaysToComplete),
			EndLocation:       contractLocationFromDBModel(cc.EndLocationID, r.EndLocationName, r.EndSolarSystemSecurityStatus),
			EndSolarSystem:    contractSolarSystemFromDBModel(r.EndSolarSystemID, r.EndSolarSystemName),
			ForCorporation:    cc.ForCorporation,
			ID:                cc.ID,
			Issuer:            eveEntityFromDBModel(r.EveEntity_2),
			IssuerCorporation: eveEntityFromDBModel(r.EveEntity),
			Items:             contractItemNamesFromDBModel(r.Items),
			Price:             optional.FromZeroValue(cc.Price),
			Reward:            optional.FromZeroValue(cc.Reward),
			StartLocation:     contractLocationFromDBModel(cc.StartLocationID, r.StartLocationName, r.StartSolarSystemSecurityStatus),
			StartSolarSystem:  contractSolarSystemFromDBModel(r.StartSolarSystemID, r.StartSolarSystemName),
			Status:            contractStatusFromDBValue[cc.Status],
			StatusNotified:    contractStatusFromDBValue[cc.StatusNotified],
			Title:             optional.FromZeroValue(cc.Title),
			Type:              contractTypeFromDBValue[cc.Type],
			UpdatedAt:         cc.UpdatedAt,
			Volume:            optional.FromZeroValue(cc.Volume),
		},
		CorporationID: cc.CorporationID,
	}
	return o2
}

func (st *Storage) CreateCorporationContractBid(ctx context.Context, arg CreateContractBidParams) error {
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

func (st *Storage) GetCorporationContractBid(ctx context.Context, contractID int64, bidID int64) (*app.ContractBid, error) {
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
	o := contractBidFromDBModel(contractBidDBModel(r.CorporationContractBid), r.EveEntity)
	return o, err
}

func (st *Storage) ListCorporationContractBids(ctx context.Context, contractID int64) ([]*app.ContractBid, error) {
	rows, err := st.qRO.ListCorporationContractBids(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("list bids for contract %d: %w", contractID, err)
	}
	oo := make([]*app.ContractBid, len(rows))
	for i, r := range rows {
		oo[i] = contractBidFromDBModel(contractBidDBModel(r.CorporationContractBid), r.EveEntity)
	}
	return oo, nil
}

func (st *Storage) ListCorporationContractBidIDs(ctx context.Context, contractID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCorporationContractBidIDs(ctx, contractID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list bid IDs for contract %d: %w", contractID, err)
	}
	return set.Collect(slices.Values(ids)), err
}

func (st *Storage) CreateCorporationContractItem(ctx context.Context, arg CreateContractItemParams) error {
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

func (st *Storage) GetCorporationContractItem(ctx context.Context, contractID, recordID int64) (*app.ContractItem, error) {
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
	o := contractItemFromDBModel(contractItemDBModel(r.CorporationContractItem), r.EveType, r.EveGroup, r.EveCategory)
	return o, err
}

func (st *Storage) ListCorporationContractItems(ctx context.Context, contractID int64) ([]*app.ContractItem, error) {
	rows, err := st.qRO.ListCorporationContractItems(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("list items for contract %d: %w", contractID, err)
	}
	oo := make([]*app.ContractItem, len(rows))
	for i, r := range rows {
		oo[i] = contractItemFromDBModel(contractItemDBModel(r.CorporationContractItem), r.EveType, r.EveGroup, r.EveCategory)
	}
	return oo, nil
}
