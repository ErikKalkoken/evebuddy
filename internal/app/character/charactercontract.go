package character

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
)

func (s *CharacterService) CountCharacterContractBids(ctx context.Context, contractID int64) (int, error) {
	x, err := s.st.ListCharacterContractBidIDs(ctx, contractID)
	if err != nil {
		return 0, err
	}
	return x.Size(), nil
}

func (s *CharacterService) GetCharacterContractTopBid(ctx context.Context, contractID int64) (*app.CharacterContractBid, error) {
	bids, err := s.st.ListCharacterContractBids(ctx, contractID)
	if err != nil {
		return nil, err
	}
	if len(bids) == 0 {
		return nil, ErrNotFound
	}
	var max float32
	var top *app.CharacterContractBid
	for _, b := range bids {
		if top == nil || b.Amount > max {
			top = b
		}
	}
	return top, nil
}

func (s *CharacterService) ListCharacterContracts(ctx context.Context, characterID int32) ([]*app.CharacterContract, error) {
	return s.st.ListCharacterContracts(ctx, characterID)
}

func (s *CharacterService) ListCharacterContractItems(ctx context.Context, contractID int64) ([]*app.CharacterContractItem, error) {
	return s.st.ListCharacterContractItems(ctx, contractID)
}

func (s *CharacterService) UpdateCharacterContractNotified(ctx context.Context, id int64, status app.ContractStatus) error {
	return s.st.UpdateCharacterContractNotified(ctx, id, status)
}

// updateCharacterContractsESI updates the wallet journal from ESI and reports wether it has changed.
func (s *CharacterService) updateCharacterContractsESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionContracts {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			contracts, err := fetchFromESIWithPaging(
				func(pageNum int) ([]esi.GetCharactersCharacterIdContracts200Ok, *http.Response, error) {
					arg := &esi.GetCharactersCharacterIdContractsOpts{
						Page: esioptional.NewInt32(int32(pageNum)),
					}
					return s.esiClient.ESI.ContractsApi.GetCharactersCharacterIdContracts(ctx, characterID, arg)
				})
			if err != nil {
				return false, err
			}
			slog.Info("Received contracts from ESI", "count", len(contracts), "characterID", characterID)
			return contracts, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			contracts := data.([]esi.GetCharactersCharacterIdContracts200Ok)
			// fetch missing eve entities
			ids := set.New[int32]()
			for _, c := range contracts {
				ids.Add(c.IssuerId)
				ids.Add(c.IssuerCorporationId)
				if c.AcceptorId != 0 {
					ids.Add(c.AcceptorId)
				}
				if c.AssigneeId != 0 {
					ids.Add(c.AssigneeId)
				}
			}
			_, err := s.EveUniverseService.AddMissingEveEntities(ctx, ids.ToSlice())
			if err != nil {
				return err
			}
			// identify new contracts
			ii, err := s.st.ListCharacterContractIDs(ctx, characterID)
			if err != nil {
				return err
			}
			existingIDs := set.NewFromSlice(ii)
			var existingContracts, newContracts []esi.GetCharactersCharacterIdContracts200Ok
			for _, c := range contracts {
				if existingIDs.Contains(c.ContractId) {
					existingContracts = append(existingContracts, c)
				} else {
					newContracts = append(newContracts, c)
				}
			}
			slog.Debug("contracts", "existing", existingIDs, "entries", contracts)
			// create new entries
			if len(newContracts) > 0 {
				for _, o := range newContracts {
					err := s.createNewContract(ctx, characterID, o)
					if err != nil {
						slog.Error("create contract", "contract", o, "error", err)
						continue
					}
				}
				slog.Info("Stored new contracts", "characterID", characterID, "count", len(newContracts))
			}
			if len(existingContracts) > 0 {
				for _, c := range existingContracts {
					arg := storage.UpdateCharacterContractParams{
						AcceptorID:    c.AcceptorId,
						AssigneeID:    c.AssigneeId,
						DateAccepted:  c.DateAccepted,
						DateCompleted: c.DateCompleted,
						CharacterID:   characterID,
						ContractID:    c.ContractId,
						Status:        c.Status,
					}
					if err := s.st.UpdateCharacterContract(ctx, arg); err != nil {
						return err
					}
				}
				slog.Info("Updated contracts", "characterID", characterID, "count", len(existingContracts))
			}
			// add new bids for auctions
			for _, c := range contracts {
				if c.Type_ != "auction" {
					continue
				}
				err := s.updateContractBids(ctx, characterID, c.ContractId)
				if err != nil {
					slog.Error("update contract bids", "contract", c, "error", err)
					continue
				}
			}
			return nil
		})
}

func (s *CharacterService) createNewContract(ctx context.Context, characterID int32, o esi.GetCharactersCharacterIdContracts200Ok) error {
	if o.StartLocationId != 0 {
		_, err := s.EveUniverseService.GetOrCreateEveLocationESI(ctx, o.StartLocationId)
		if err != nil {
			return err
		}
	}
	if o.EndLocationId != 0 {
		_, err := s.EveUniverseService.GetOrCreateEveLocationESI(ctx, o.EndLocationId)
		if err != nil {
			return err
		}
	}
	arg := storage.CreateCharacterContractParams{
		AcceptorID:          o.AcceptorId,
		AssigneeID:          o.AssigneeId,
		Availability:        o.Availability,
		Buyout:              o.Buyout,
		CharacterID:         characterID,
		Collateral:          o.Collateral,
		ContractID:          o.ContractId,
		DateAccepted:        o.DateAccepted,
		DateCompleted:       o.DateCompleted,
		DateExpired:         o.DateExpired,
		DateIssued:          o.DateIssued,
		DaysToComplete:      o.DaysToComplete,
		EndLocationID:       o.EndLocationId,
		ForCorporation:      o.ForCorporation,
		IssuerCorporationID: o.IssuerCorporationId,
		IssuerID:            o.IssuerId,
		Price:               o.Price,
		Reward:              o.Reward,
		StartLocationID:     o.StartLocationId,
		Status:              o.Status,
		Title:               o.Title,
		Type:                o.Type_,
		Volume:              o.Volume,
	}
	id, err := s.st.CreateCharacterContract(ctx, arg)
	if err != nil {
		return err
	}
	items, _, err := s.esiClient.ESI.ContractsApi.GetCharactersCharacterIdContractsContractIdItems(ctx, characterID, o.ContractId, nil)
	if err != nil {
		return err
	}
	for _, it := range items {
		et, err := s.EveUniverseService.GetOrCreateEveTypeESI(ctx, it.TypeId)
		if err != nil {
			return err
		}
		arg := storage.CreateCharacterContractItemParams{
			ContractID:  id,
			IsIncluded:  it.IsIncluded,
			IsSingleton: it.IsSingleton,
			Quantity:    it.Quantity,
			RawQuantity: it.RawQuantity,
			RecordID:    it.RecordId,
			TypeID:      et.ID,
		}
		if err := s.st.CreateCharacterContractItem(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func (s *CharacterService) updateContractBids(ctx context.Context, characterID, contractID int32) error {
	c, err := s.st.GetCharacterContract(ctx, characterID, contractID)
	if err != nil {
		return err
	}
	existingBidIDs, err := s.st.ListCharacterContractBidIDs(ctx, c.ID)
	if err != nil {
		return err
	}
	bids, _, err := s.esiClient.ESI.ContractsApi.GetCharactersCharacterIdContractsContractIdBids(ctx, characterID, contractID, nil)
	if err != nil {
		return err
	}
	newBids := make([]esi.GetCharactersCharacterIdContractsContractIdBids200Ok, 0)
	for _, b := range bids {
		if !existingBidIDs.Contains(b.BidId) {
			newBids = append(newBids, b)
		}
	}
	if len(newBids) == 0 {
		return nil
	}
	eeIDs := make([]int32, 0)
	for _, b := range newBids {
		if b.BidderId != 0 {
			eeIDs = append(eeIDs, b.BidderId)
		}
	}
	if len(eeIDs) > 0 {
		if _, err = s.EveUniverseService.AddMissingEveEntities(ctx, eeIDs); err != nil {
			return err
		}
	}
	for _, b := range newBids {
		arg := storage.CreateCharacterContractBidParams{
			ContractID: c.ID,
			Amount:     b.Amount,
			BidID:      b.BidId,
			BidderID:   b.BidderId,
			DateBid:    b.DateBid,
		}
		if err := s.st.CreateCharacterContractBid(ctx, arg); err != nil {
			return err
		}
	}
	slog.Info("created contract bids", "characterID", characterID, "contract", contractID, "count", len(newBids))
	return nil
}
