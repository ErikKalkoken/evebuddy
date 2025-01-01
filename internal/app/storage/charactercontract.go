package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateCharacterContractParams struct {
	AcceptorID          int32
	AssigneeID          int32
	Availability        app.CharacterContractAvailability
	Buyout              float64
	CharacterID         int32
	Collateral          float64
	ContractID          int32
	DateAccepted        optional.Optional[time.Time]
	DateCompleted       optional.Optional[time.Time]
	DateExpired         time.Time
	DateIssued          time.Time
	DaysToComplete      int32
	EndLocationID       optional.Optional[int64]
	ForCorporation      bool
	IssuerCorporationID int32
	IssuerID            int32
	Price               float64
	Reward              float64
	StartLocationID     optional.Optional[int64]
	Status              app.CharacterContractStatus
	Title               string
	Type                app.CharacterContractType
	Volume              float64
}

func (st *Storage) CreateCharacterContract(ctx context.Context, arg CreateCharacterContractParams) (int64, error) {
	if arg.CharacterID == 0 || arg.ContractID == 0 || arg.Status == "" {
		return 0, fmt.Errorf("create character contract. Mandatory fields not set: %v", arg)
	}
	arg2 := queries.CreateCharacterContractParams{
		Availability:        string(arg.Availability),
		Buyout:              arg.Buyout,
		CharacterID:         int64(arg.CharacterID),
		Collateral:          arg.Collateral,
		ContractID:          int64(arg.ContractID),
		DateAccepted:        optional.ToNullTime(arg.DateAccepted),
		DateCompleted:       optional.ToNullTime(arg.DateCompleted),
		DateExpired:         arg.DateExpired,
		DateIssued:          arg.DateIssued,
		DaysToComplete:      int64(arg.DaysToComplete),
		EndLocationID:       optional.ToNullInt64(arg.EndLocationID),
		ForCorporation:      arg.ForCorporation,
		IssuerCorporationID: int64(arg.IssuerCorporationID),
		IssuerID:            int64(arg.IssuerID),
		Price:               arg.Price,
		Reward:              arg.Reward,
		StartLocationID:     optional.ToNullInt64(arg.StartLocationID),
		Status:              string(arg.Status),
		Title:               arg.Title,
		Type:                string(arg.Type),
		Volume:              arg.Volume,
	}
	if arg.AcceptorID != 0 {
		arg2.AcceptorID.Int64 = int64(arg.AcceptorID)
		arg2.AcceptorID.Valid = true
	}
	if arg.AssigneeID != 0 {
		arg2.AssigneeID.Int64 = int64(arg.AssigneeID)
		arg2.AssigneeID.Valid = true
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
	return characterContractFromDBModel(o, r.EveEntity, r.EveEntity_2, acceptor, assignee), err
}

func (st *Storage) ListCharacterContractIDs(ctx context.Context, characterID int32) ([]int32, error) {
	ids, err := st.q.ListCharacterContractIDs(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list contract ids for character %d: %w", characterID, err)
	}
	return convertNumericSlice[int64, int32](ids), nil
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
		oo[i] = characterContractFromDBModel(o, r.EveEntity, r.EveEntity_2, acceptor, assignee)
	}
	return oo, nil
}

type UpdateCharacterContractParams struct {
	AcceptorID    int32
	AssigneeID    int32
	DateAccepted  optional.Optional[time.Time]
	DateCompleted optional.Optional[time.Time]
	CharacterID   int32
	ContractID    int32
	Status        app.CharacterContractStatus
}

func (st *Storage) UpdateCharacterContract(ctx context.Context, arg UpdateCharacterContractParams) error {
	if arg.CharacterID == 0 || arg.ContractID == 0 {
		return fmt.Errorf("update character contract. IDs not set: %v", arg)
	}
	arg2 := queries.UpdateCharacterContractParams{
		CharacterID:   int64(arg.CharacterID),
		ContractID:    int64(arg.ContractID),
		DateAccepted:  optional.ToNullTime(arg.DateAccepted),
		DateCompleted: optional.ToNullTime(arg.DateCompleted),
		Status:        string(arg.Status),
	}
	if arg.AcceptorID != 0 {
		arg2.AcceptorID.Int64 = int64(arg.AcceptorID)
		arg2.AcceptorID.Valid = true
	}
	if arg.AssigneeID != 0 {
		arg2.AssigneeID.Int64 = int64(arg.AssigneeID)
		arg2.AssigneeID.Valid = true
	}
	err := st.q.UpdateCharacterContract(ctx, arg2)
	if err != nil {
		return fmt.Errorf("update character contract: %w", err)
	}
	return nil
}

// func (st *Storage) ListCharacterWalletJournalEntries(ctx context.Context, id int32) ([]*app.CharacterContract, error) {
// 	rows, err := st.q.ListCharacterWalletJournalEntries(ctx, int64(id))
// 	if err != nil {
// 		return nil, fmt.Errorf("list wallet journal entries for character %d: %w", id, err)
// 	}
// 	ee := make([]*app.CharacterContract, len(rows))
// 	for i, r := range rows {
// 		o := r.CharacterContract
// 		firstParty := nullEveEntry{ID: o.FirstPartyID, Name: r.FirstName, Category: r.FirstCategory}
// 		secondParty := nullEveEntry{ID: o.SecondPartyID, Name: r.SecondName, Category: r.SecondCategory}
// 		taxReceiver := nullEveEntry{ID: o.TaxReceiverID, Name: r.TaxName, Category: r.TaxCategory}
// 		ee[i] = characterContractFromDBModel(o, firstParty, secondParty, taxReceiver)
// 	}
// 	return ee, nil
// }

func characterContractFromDBModel(
	o queries.CharacterContract,
	issuerCorporation queries.EveEntity,
	issuer queries.EveEntity,
	acceptor nullEveEntry,
	assignee nullEveEntry,
) *app.CharacterContract {
	o2 := &app.CharacterContract{
		ID:                o.ID,
		Acceptor:          eveEntityFromNullableDBModel(acceptor),
		Assignee:          eveEntityFromNullableDBModel(assignee),
		Availability:      app.CharacterContractAvailability(o.Availability),
		Buyout:            o.Buyout,
		CharacterID:       int32(o.CharacterID),
		Collateral:        o.Collateral,
		ContractID:        int32(o.ContractID),
		DateAccepted:      optional.FromNullTime(o.DateAccepted),
		DateCompleted:     optional.FromNullTime(o.DateCompleted),
		DateExpired:       o.DateExpired,
		DateIssued:        o.DateIssued,
		DaysToComplete:    int32(o.DaysToComplete),
		EndLocationID:     optional.FromNullInt64(o.EndLocationID),
		ForCorporation:    o.ForCorporation,
		IssuerCorporation: eveEntityFromDBModel(issuerCorporation),
		Issuer:            eveEntityFromDBModel(issuer),
		Price:             o.Price,
		Reward:            o.Reward,
		StartLocationID:   optional.FromNullInt64(o.StartLocationID),
		Status:            app.CharacterContractStatus(o.Status),
		Title:             o.Title,
		Type:              app.CharacterContractType(o.Type),
		Volume:            o.Volume,
	}
	return o2
}
