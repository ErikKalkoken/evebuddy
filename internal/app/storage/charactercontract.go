package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func (st *Storage) CalculateCharacterContractItemsValue(ctx context.Context, characterID int64) (float64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CalculateCharacterContractItemsValue: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	v, err := st.qRO.CalculateCharacterContractItemsValue(ctx, queries.CalculateCharacterContractItemsValueParams{
		CharacterID:   characterID,
		EveCategoryID: app.EveCategoryBlueprint,
		Status: []string{
			contractStatusToDBValue[app.ContractStatusOutstanding],
			contractStatusToDBValue[app.ContractStatusInProgress],
		},
		Types: []string{
			contractTypeToDBValue[app.ContractTypeAuction],
			contractTypeToDBValue[app.ContractTypeCourier],
		},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, wrapErr(err)
	}
	v2 := optional.FromNullFloat64(v)
	return v2.ValueOrZero(), nil
}

// CalculateCharacterContractsEscrow returns the ISK held in escrow for a character's
// courier collateral and winning auction bids.
func (st *Storage) CalculateCharacterContractsEscrow(ctx context.Context, characterID int64) (float64, error) {
	v1, err := st.CalculateCharacterContractsCourierEscrow(ctx, characterID)
	if err != nil {
		return 0, err
	}
	v2, err := st.CalculateCharacterContractsAuctionEscrow(ctx, characterID)
	if err != nil {
		return 0, err
	}
	return v1 + v2, nil
}

func (st *Storage) CalculateCharacterContractsCourierEscrow(ctx context.Context, characterID int64) (float64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CalculateCharacterContractsCourierEscrow: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	v, err := st.qRO.CalculateCharacterContractsCourierEscrow(ctx, queries.CalculateCharacterContractsCourierEscrowParams{
		CharacterID: characterID,
		Type:        contractTypeToDBValue[app.ContractTypeCourier],
		Status:      contractStatusesToDBValues(contractEscrowCourierStatuses),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, wrapErr(err)
	}
	v2 := optional.FromNullFloat64(v)
	return v2.ValueOrZero(), nil
}

func (st *Storage) CalculateCharacterContractsAuctionEscrow(ctx context.Context, characterID int64) (float64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CalculateCharacterContractsAuctionEscrow: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	v, err := st.qRO.CalculateCharacterContractsAuctionEscrow(ctx, queries.CalculateCharacterContractsAuctionEscrowParams{
		CharacterID: characterID,
		Status:      contractStatusesToDBValues(contractEscrowAuctionStatuses),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, wrapErr(err)
	}
	v2 := optional.FromNullFloat64(v.TotalWinningBids)
	return v2.ValueOrZero(), nil
}

func (st *Storage) CountCharactersOutstandingPersonalContracts(ctx context.Context) (map[int64]int, error) {
	rows, err := st.qRO.CountCharactersOutstandingPersonalContracts(
		ctx,
		contractStatusToDBValue[app.ContractStatusOutstanding],
	)
	if err != nil {
		return nil, fmt.Errorf("CountCharactersOutstandingPersonalContracts: %w", err)
	}
	m := make(map[int64]int)
	for _, r := range rows {
		m[r.CharacterID] = int(r.Number)
	}
	return m, nil
}

func (st *Storage) CountCharactersOutstandingCorporationContracts(ctx context.Context) (map[int64]int, error) {
	rows, err := st.qRO.CountCharactersOutstandingCorporationContracts(
		ctx,
		contractStatusToDBValue[app.ContractStatusOutstanding],
	)
	if err != nil {
		return nil, fmt.Errorf("CountCharactersOutstandingCorporationContracts: %w", err)
	}
	m := make(map[int64]int)
	for _, r := range rows {
		m[r.CharacterID] = int(r.Number)
	}
	return m, nil
}

type CreateCharacterContractParams struct {
	AcceptorID          int64
	AssigneeID          int64
	Availability        app.ContractAvailability
	Buyout              optional.Optional[float64]
	CharacterID         int64
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
		AcceptorID:          NewNullInt64(arg.AcceptorID),
		AssigneeID:          NewNullInt64(arg.AssigneeID),
		Availability:        contractAvailabilityToDBValue[arg.Availability],
		Buyout:              arg.Buyout.ValueOrZero(),
		CharacterID:         arg.CharacterID,
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

func (st *Storage) DeleteCharacterContracts(ctx context.Context, characterID int64, contractIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCharacterContracts for character %d and contract IDs: %s: %w", characterID, contractIDs, err)
	}
	if characterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if contractIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.DeleteCharacterContracts(ctx, queries.DeleteCharacterContractsParams{
		CharacterID: characterID,
		ContractIds: slices.Collect(contractIDs.All()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Contracts deleted for character", "characterID", characterID, "contractIDs", contractIDs)
	return nil
}

func (st *Storage) GetCharacterContract(ctx context.Context, characterID, contractID int64) (*app.CharacterContract, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterContract for character %d: %w", characterID, err)
	}
	r, err := st.qRO.GetCharacterContract(ctx, queries.GetCharacterContractParams{
		CharacterID: characterID,
		ContractID:  contractID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	o := characterContractFromDBModel(r)
	return o, nil
}

func (st *Storage) ListCharacterContractIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterContractIDs: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return set.Set[int64]{}, wrapErr(app.ErrInvalid)
	}
	ids, err := st.qRO.ListCharacterContractIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) ListAllCharacterContracts(ctx context.Context) ([]*app.CharacterContract, error) {
	rows, err := st.qRO.ListAllCharacterContracts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list character contracts: %w", err)
	}
	oo := make([]*app.CharacterContract, len(rows))
	for i, r := range rows {
		oo[i] = characterContractFromDBModel(queries.GetCharacterContractRow(r))
	}
	return oo, nil
}

func (st *Storage) ListCharacterContracts(ctx context.Context, characterID int64) ([]*app.CharacterContract, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterContracts: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterContracts(ctx, characterID)
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]*app.CharacterContract, len(rows))
	for i, r := range rows {
		oo[i] = characterContractFromDBModel(queries.GetCharacterContractRow(r))
	}
	return oo, nil
}

type UpdateCharacterContractParams struct {
	AcceptorID    int64
	DateAccepted  optional.Optional[time.Time]
	DateCompleted optional.Optional[time.Time]
	CharacterID   int64
	ContractID    int64
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
		CharacterID:   arg.CharacterID,
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

func (st *Storage) UpdateCharacterContractNotified(ctx context.Context, id int64, status app.ContractStatus) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterContractNotified %d %s: %w", id, status, err)
	}
	if id == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateCharacterContractNotified(ctx, queries.UpdateCharacterContractNotifiedParams{
		ID:             id,
		StatusNotified: contractStatusToDBValue[status],
		UpdatedAt:      time.Now().UTC(),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func characterContractFromDBModel(r queries.GetCharacterContractRow) *app.CharacterContract {
	cc := r.CharacterContract
	o := &app.CharacterContract{
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
		CharacterID: cc.CharacterID,
	}
	return o
}

func (st *Storage) CreateCharacterContractBid(ctx context.Context, arg CreateContractBidParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharacterContractBid %+v: %w", arg, err)
	}
	if arg.ContractID == 0 || arg.BidID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if err := st.qRW.CreateCharacterContractBid(ctx, queries.CreateCharacterContractBidParams{
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

func (st *Storage) GetCharacterContractBid(ctx context.Context, contractID int64, bidID int64) (*app.ContractBid, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterContractBid %d %d: %w", contractID, bidID, err)
	}
	if contractID == 0 || bidID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCharacterContractBid(ctx, queries.GetCharacterContractBidParams{
		ContractID: contractID,
		BidID:      bidID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	o := contractBidFromDBModel(r.CharacterContractBid, r.EveEntity)
	return o, err
}

func (st *Storage) ListCharacterContractBids(ctx context.Context, contractID int64) ([]*app.ContractBid, error) {
	rows, err := st.qRO.ListCharacterContractBids(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("list bids for contract %d: %w", contractID, err)
	}
	oo := make([]*app.ContractBid, len(rows))
	for i, r := range rows {
		oo[i] = contractBidFromDBModel(r.CharacterContractBid, r.EveEntity)
	}
	return oo, nil
}

func (st *Storage) ListCharacterContractBidIDs(ctx context.Context, contractID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterContractBidIDs(ctx, contractID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list bid IDs for contract %d: %w", contractID, err)
	}
	return set.Collect(slices.Values(ids)), err
}

func (st *Storage) CreateCharacterContractItem(ctx context.Context, arg CreateContractItemParams) error {
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
		Quantity:    arg.Quantity,
		RawQuantity: arg.RawQuantity.ValueOrZero(),
		RecordID:    arg.RecordID,
		TypeID:      arg.TypeID,
	}); err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetCharacterContractItem(ctx context.Context, contractID, recordID int64) (*app.ContractItem, error) {
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
	o := contractItemFromDBModel(r.CharacterContractItem, r.EveType, r.EveGroup, r.EveCategory)
	return o, err
}

func (st *Storage) ListCharacterContractItems(ctx context.Context, contractID int64) ([]*app.ContractItem, error) {
	rows, err := st.qRO.ListCharacterContractItems(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("list items for contract %d: %w", contractID, err)
	}
	oo := make([]*app.ContractItem, len(rows))
	for i, r := range rows {
		oo[i] = contractItemFromDBModel(r.CharacterContractItem, r.EveType, r.EveGroup, r.EveCategory)
	}
	return oo, nil
}
