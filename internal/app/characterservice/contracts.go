package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// GetContract fetches and returns a contract from the database.
func (s *CharacterService) GetContract(ctx context.Context, characterID, contractID int32) (*app.CharacterContract, error) {
	return s.st.GetCharacterContract(ctx, characterID, contractID)
}

func (s *CharacterService) CountContractBids(ctx context.Context, contractID int64) (int, error) {
	x, err := s.st.ListCharacterContractBidIDs(ctx, contractID)
	if err != nil {
		return 0, err
	}
	return x.Size(), nil
}

func (s *CharacterService) GetContractTopBid(ctx context.Context, contractID int64) (*app.CharacterContractBid, error) {
	bids, err := s.st.ListCharacterContractBids(ctx, contractID)
	if err != nil {
		return nil, err
	}
	if len(bids) == 0 {
		return nil, app.ErrNotFound
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

func (s *CharacterService) NotifyUpdatedContracts(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error {
	_, err, _ := s.sfg.Do(fmt.Sprintf("NotifyUpdatedContracts-%d", characterID), func() (any, error) {
		cc, err := s.st.ListCharacterContractsForNotify(ctx, characterID, earliest)
		if err != nil {
			return nil, err
		}
		characterName, err := s.getCharacterName(ctx, characterID)
		if err != nil {
			return nil, err
		}
		for _, c := range cc {
			if c.Status == c.StatusNotified {
				continue
			}
			if c.Acceptor != nil && c.Acceptor.ID == characterID {
				continue // ignore status changed caused by the current character
			}
			var content string
			name := "'" + c.NameDisplay() + "'"
			switch c.Type {
			case app.ContractTypeCourier:
				switch c.Status {
				case app.ContractStatusInProgress:
					content = fmt.Sprintf("Contract %s has been accepted by %s", name, c.AcceptorDisplay())
				case app.ContractStatusFinished:
					content = fmt.Sprintf("Contract %s has been delivered", name)
				case app.ContractStatusFailed:
					content = fmt.Sprintf("Contract %s has been failed by %s", name, c.AcceptorDisplay())
				}
			case app.ContractTypeItemExchange:
				switch c.Status {
				case app.ContractStatusFinished:
					content = fmt.Sprintf("Contract %s has been accepted by %s", name, c.AcceptorDisplay())
				}
			}
			if content == "" {
				continue
			}
			title := fmt.Sprintf("%s: Contract updated", characterName)
			notify(title, content)
			if err := s.st.UpdateCharacterContractNotified(ctx, c.ID, c.Status); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("NotifyUpdatedContracts for character %d: %w", characterID, err)
	}
	return nil
}

func (s *CharacterService) ListAllContracts(ctx context.Context) ([]*app.CharacterContract, error) {
	return s.st.ListAllCharacterContracts(ctx)
}

func (s *CharacterService) ListContractItems(ctx context.Context, contractID int64) ([]*app.CharacterContractItem, error) {
	return s.st.ListCharacterContractItems(ctx, contractID)
}

var contractAvailabilityFromESIValue = map[string]app.ContractAvailability{
	"alliance":    app.ContractAvailabilityAlliance,
	"corporation": app.ContractAvailabilityCorporation,
	"personal":    app.ContractAvailabilityPrivate,
	"public":      app.ContractAvailabilityPublic,
}

var contractStatusFromESIValue = map[string]app.ContractStatus{
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

var contractTypeFromESIValue = map[string]app.ContractType{
	"auction":       app.ContractTypeAuction,
	"courier":       app.ContractTypeCourier,
	"item_exchange": app.ContractTypeItemExchange,
	"loan":          app.ContractTypeLoan,
	"unknown":       app.ContractTypeUnknown,
}

// updateContractsESI updates the wallet journal from ESI and reports whether it has changed.
func (s *CharacterService) updateContractsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionCharacterContracts {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			contracts, err := xesi.FetchWithPaging(
				func(pageNum int) ([]esi.GetCharactersCharacterIdContracts200Ok, *http.Response, error) {
					return s.esiClient.ESI.ContractsApi.GetCharactersCharacterIdContracts(
						ctx, characterID, &esi.GetCharactersCharacterIdContractsOpts{
							Page: esioptional.NewInt32(int32(pageNum)),
						},
					)
				},
			)
			if err != nil {
				return false, err
			}
			slog.Debug("Received contracts from ESI", "characterID", characterID, "count", len(contracts))
			return contracts, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			contracts := data.([]esi.GetCharactersCharacterIdContracts200Ok)
			// fetch missing eve entities
			var entityIDs set.Set[int32]
			var locationIDs set.Set[int64]
			for _, c := range contracts {
				entityIDs.Add(c.AcceptorId, c.AssigneeId, c.IssuerId, c.IssuerCorporationId)
				locationIDs.Add(c.StartLocationId, c.EndLocationId)
			}
			err := s.addMissingEveEntitiesAndLocations(ctx, entityIDs, locationIDs)
			if err != nil {
				return err
			}
			// identify new contracts
			ii, err := s.st.ListCharacterContractIDs(ctx, characterID)
			if err != nil {
				return err
			}
			existingIDs := set.Of(ii...)
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
				var count int
				for _, c := range newContracts {
					if err := s.createNewContract(ctx, characterID, c); err != nil {
						slog.Error("create contract", "contract", c, "error", err)
						continue
					} else {
						count++
					}
				}
				slog.Info("Stored new contracts", "characterID", characterID, "count", count)
			}
			if len(existingContracts) > 0 {
				var count int
				for _, c := range existingContracts {
					if err := s.updateContract(ctx, characterID, c); err != nil {
						slog.Error("update contract", "contract", c, "error", err)
						continue
					} else {
						count++
					}
				}
				slog.Info("Updated contracts", "characterID", characterID, "count", count)
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

func (s *CharacterService) createNewContract(ctx context.Context, characterID int32, c esi.GetCharactersCharacterIdContracts200Ok) error {
	// Ensuring again all related objects are created to prevent occasional FK constraint error
	entityIDs := set.Of(c.AcceptorId, c.AssigneeId, c.IssuerId, c.IssuerCorporationId)
	locationIDs := set.Of(c.StartLocationId, c.EndLocationId)
	err := s.addMissingEveEntitiesAndLocations(ctx, entityIDs, locationIDs)
	if err != nil {
		return err
	}
	availability, ok := contractAvailabilityFromESIValue[c.Availability]
	if !ok {
		return fmt.Errorf("unknown availability: %s", c.Availability)
	}
	status, ok := contractStatusFromESIValue[c.Status]
	if !ok {
		return fmt.Errorf("unknown status: %s", c.Status)
	}
	typ, ok := contractTypeFromESIValue[c.Type_]
	if !ok {
		return fmt.Errorf("unknown type: %s", c.Type_)
	}
	arg := storage.CreateCharacterContractParams{
		AcceptorID:          c.AcceptorId,
		AssigneeID:          c.AssigneeId,
		Availability:        availability,
		Buyout:              c.Buyout,
		CharacterID:         characterID,
		Collateral:          c.Collateral,
		ContractID:          c.ContractId,
		DateAccepted:        c.DateAccepted,
		DateCompleted:       c.DateCompleted,
		DateExpired:         c.DateExpired,
		DateIssued:          c.DateIssued,
		DaysToComplete:      c.DaysToComplete,
		EndLocationID:       c.EndLocationId,
		ForCorporation:      c.ForCorporation,
		IssuerCorporationID: c.IssuerCorporationId,
		IssuerID:            c.IssuerId,
		Price:               c.Price,
		Reward:              c.Reward,
		StartLocationID:     c.StartLocationId,
		Status:              status,
		Title:               c.Title,
		Type:                typ,
		Volume:              c.Volume,
	}
	id, err := s.st.CreateCharacterContract(ctx, arg)
	if err != nil {
		return err
	}
	items, _, err := s.esiClient.ESI.ContractsApi.GetCharactersCharacterIdContractsContractIdItems(ctx, characterID, c.ContractId, nil)
	if err != nil {
		return err
	}
	typeIDs := set.Of(xslices.Map(items, func(x esi.GetCharactersCharacterIdContractsContractIdItems200Ok) int32 {
		return x.TypeId

	})...)
	if err := s.eus.AddMissingTypes(ctx, typeIDs); err != nil {
		return err
	}
	for _, it := range items {
		arg := storage.CreateCharacterContractItemParams{
			ContractID:  id,
			IsIncluded:  it.IsIncluded,
			IsSingleton: it.IsSingleton,
			Quantity:    it.Quantity,
			RawQuantity: it.RawQuantity,
			RecordID:    it.RecordId,
			TypeID:      it.TypeId,
		}
		if err := s.st.CreateCharacterContractItem(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func (s *CharacterService) updateContract(ctx context.Context, characterID int32, c esi.GetCharactersCharacterIdContracts200Ok) error {
	status, ok := contractStatusFromESIValue[c.Status]
	if !ok {
		return fmt.Errorf("unknown status: %s", c.Status)
	}
	o, err := s.st.GetCharacterContract(ctx, characterID, c.ContractId)
	if err != nil {
		return err
	}
	var acceptorID int32
	if o.Acceptor != nil {
		acceptorID = o.Acceptor.ID
	}
	if c.AcceptorId == acceptorID &&
		c.DateAccepted.Equal(o.DateAccepted.ValueOrZero()) &&
		c.DateCompleted.Equal(o.DateCompleted.ValueOrZero()) &&
		o.Status == contractStatusFromESIValue[c.Status] {
		return nil
	}
	arg := storage.UpdateCharacterContractParams{
		AcceptorID:    c.AcceptorId,
		DateAccepted:  c.DateAccepted,
		DateCompleted: c.DateCompleted,
		CharacterID:   characterID,
		ContractID:    c.ContractId,
		Status:        status,
	}
	if err := s.st.UpdateCharacterContract(ctx, arg); err != nil {
		return err
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
	var eeIDs set.Set[int32]
	for _, b := range newBids {
		if b.BidderId != 0 {
			eeIDs.Add(b.BidderId)
		}
	}
	if eeIDs.Size() > 0 {
		if _, err = s.eus.AddMissingEntities(ctx, eeIDs); err != nil {
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
