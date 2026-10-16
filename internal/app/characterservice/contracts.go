package characterservice

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	contractsync "github.com/ErikKalkoken/evebuddy/internal/app/contractsync"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// GetContract fetches and returns a contract from the database.
func (s *CharacterService) GetContract(ctx context.Context, characterID, contractID int64) (*app.CharacterContract, error) {
	return s.st.GetCharacterContract(ctx, characterID, contractID)
}

// GetContractBidSummary returns the number of bids on a contract and its top bid, if any.
func (s *CharacterService) GetContractBidSummary(ctx context.Context, contractID int64) (int, optional.Optional[*app.ContractBid], error) {
	bids, err := s.st.ListCharacterContractBids(ctx, contractID)
	if err != nil {
		return 0, optional.Optional[*app.ContractBid]{}, err
	}
	top, ok := app.ContractTopBid(bids)
	if !ok {
		return 0, optional.Optional[*app.ContractBid]{}, nil
	}
	return len(bids), optional.New(top), nil
}

func (s *CharacterService) NotifyUpdatedContracts(ctx context.Context, characterID int64, earliest time.Time, notify func(title, content string)) error {
	_, err, _ := s.sfg.Do(fmt.Sprintf("NotifyUpdatedContracts-%d", characterID), func() (any, error) {
		cc, err := s.st.ListCharacterContracts(ctx, characterID)
		if err != nil {
			return nil, err
		}
		cc = slices.DeleteFunc(cc, func(x *app.CharacterContract) bool {
			return x.UpdatedAt.Before(earliest)
		})
		characterName, err := s.getCharacterName(ctx, characterID)
		if err != nil {
			return nil, err
		}
		for _, c := range cc {
			if c.Status == c.StatusNotified {
				continue
			}
			if v, ok := c.Acceptor.Value(); ok && v.ID == characterID {
				continue // ignore status changed caused by the current character
			}
			content := c.UpdateNotification()
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

// ListAllContracts returns the contracts of all characters.
// A contract seen by several characters is returned only once:
// the copy of its issuer or else the copy of the character with the lowest ID.
func (s *CharacterService) ListAllContracts(ctx context.Context) ([]*app.CharacterContract, error) {
	oo, err := s.st.ListAllCharacterContracts(ctx)
	if err != nil {
		return nil, err
	}
	// sort preferred copies first, so they are kept
	rank := func(o *app.CharacterContract) int {
		if o.Issuer.ID == o.CharacterID {
			return 0
		}
		return 1
	}
	slices.SortStableFunc(oo, func(a, b *app.CharacterContract) int {
		return cmp.Or(cmp.Compare(rank(a), rank(b)), cmp.Compare(a.CharacterID, b.CharacterID))
	})
	oo = xslices.DeduplicateFunc(oo, func(a, b *app.CharacterContract) bool {
		return a.ContractID == b.ContractID &&
			a.Type == b.Type &&
			a.Issuer.ID == b.Issuer.ID &&
			optional.EqualFunc(a.Assignee, b.Assignee, func(x, y *app.EveEntity) bool { return x.ID == y.ID }) &&
			a.DateIssued.Equal(b.DateIssued)
	})
	return oo, nil
}

func (s *CharacterService) ListContractItems(ctx context.Context, contractID int64) ([]*app.ContractItem, error) {
	return s.st.ListCharacterContractItems(ctx, contractID)
}

func (s *CharacterService) ListAllCharacterContractSlotsPersonal(ctx context.Context) ([]app.CharacterContractSlots, error) {
	slots := make(map[int64]app.CharacterContractSlots)
	characters, err := s.st.ListCharacterEveCharacters(ctx)
	if err != nil {
		return nil, err
	}
	characterCorporations := make(map[int64]int64)
	for _, c := range characters {
		x := slots[c.ID]
		x.CharacterID = c.ID
		x.CharacterName = c.Name
		x.CorporationID = c.Corporation.ID
		x.CorporationName = c.Corporation.Name
		x.Total = 1 // capacity at level 0
		slots[c.ID] = x
		characterCorporations[c.ID] = c.Corporation.ID
	}
	counts, err := s.st.CountCharactersOutstandingPersonalContracts(ctx)
	if err != nil {
		return nil, err
	}
	for characterID, used := range counts {
		x := slots[characterID]
		x.Used = used
		slots[characterID] = x
	}

	contracting, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeContracting)
	if err != nil {
		return nil, err
	}
	for _, r := range contracting {
		if r.Level > 0 {
			x := slots[r.CharacterID]
			x.Total += r.Level * 4
			slots[r.CharacterID] = x
		}
	}

	advancedContracting, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeAdvancedContracting)
	if err != nil {
		return nil, err
	}
	for _, r := range advancedContracting {
		if r.Level > 0 {
			x := slots[r.CharacterID]
			x.Total += r.Level * 4
			slots[r.CharacterID] = x
		}
	}

	for _, c := range characters {
		x := slots[c.ID]
		x.Free = x.Total - x.Used
		slots[c.ID] = x
	}

	slots2 := slices.Collect(maps.Values(slots))
	return slots2, nil
}

func (s *CharacterService) ListAllCharacterContractSlotsCorporation(ctx context.Context) ([]app.CharacterContractSlots, error) {
	slots := make(map[int64]app.CharacterContractSlots)
	characters, err := s.st.ListCharacterEveCharacters(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range characters {
		x := slots[c.ID]
		x.CharacterID = c.ID
		x.CharacterName = c.Name
		x.CorporationID = c.Corporation.ID
		x.CorporationName = c.Corporation.Name
		x.IsCorporation = true
		x.Total = 10 // capacity at level 0
		slots[c.ID] = x
	}
	counts, err := s.st.CountCharactersOutstandingCorporationContracts(ctx)
	if err != nil {
		return nil, err
	}
	for characterID, used := range counts {
		x := slots[characterID]
		x.Used = used
		slots[characterID] = x
	}

	contracting, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeCorporationContracting)
	if err != nil {
		return nil, err
	}
	for _, r := range contracting {
		if r.Level > 0 {
			x := slots[r.CharacterID]
			x.Total += r.Level * 10
			slots[r.CharacterID] = x
		}
	}

	for _, c := range characters {
		x := slots[c.ID]
		x.Free = x.Total - x.Used
		slots[c.ID] = x
	}

	slots2 := slices.Collect(maps.Values(slots))
	return slots2, nil
}

// updateContractsESI updates the contracts from ESI and reports whether they have changed.
func (s *CharacterService) updateContractsESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterContracts {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	changed, err := s.updateSectionIfChanged(
		ctx, arg, false,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdContracts")
			contracts, err := xgoesi.FetchPages(
				func(page int32) ([]esi.CharactersCharacterIdContractsGetInner, *http.Response, error) {
					return s.esiClient.ContractsAPI.GetCharactersCharacterIdContracts(ctx, characterID).Page(page).Execute()
				},
			)
			if err != nil {
				return false, err
			}
			slices.SortFunc(contracts, func(a, b esi.CharactersCharacterIdContractsGetInner) int {
				return cmp.Compare(a.ContractId, b.ContractId)
			})
			slog.Debug("Received contracts from ESI", "characterID", characterID, "count", len(contracts))
			return contracts, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			contracts := xslices.Map(data.([]esi.CharactersCharacterIdContractsGetInner), func(x esi.CharactersCharacterIdContractsGetInner) contractsync.ESIContract {
				return contractsync.ESIContract(x)
			})
			contracts = contractsync.RemoveDeleted(contracts)
			entityIDs, locationIDs := contractsync.ExtractEntityAndLocationIDs(contracts)
			err := s.eus.AddMissingEntitiesAndLocations(ctx, entityIDs, locationIDs)
			if err != nil {
				return false, err
			}
			oo, err := s.st.ListCharacterContracts(ctx, characterID)
			if err != nil {
				return false, err
			}
			current := xslices.Map(oo, func(x *app.CharacterContract) *app.Contract {
				return &x.Contract
			})
			staleIDs, err := contractsync.Sync(
				ctx,
				contracts,
				current,
				func(ctx context.Context, c contractsync.ESIContract) (int64, error) {
					return s.createNewContract(ctx, characterID, c)
				},
				func(ctx context.Context, o *app.Contract, c contractsync.ESIContract) error {
					return s.updateContract(ctx, characterID, o, c)
				},
				func(ctx context.Context, contractPK, contractID int64) error {
					return s.updateContractBids(ctx, characterID, contractPK, contractID)
				},
				slog.Int64("characterID", characterID),
			)
			if err != nil {
				return false, err
			}
			if err := s.st.DeleteCharacterContracts(ctx, characterID, staleIDs); err != nil {
				return false, err
			}
			return true, nil
		},
	)
	if err != nil {
		return false, err
	}

	// update calculated values
	c, err := s.st.GetCharacter(ctx, arg.characterID)
	if err != nil {
		return false, err
	}
	if arg.forceUpdate || changed || c.ContractItemsValue.IsEmpty() {
		if err := s.updateContractItemsValue(ctx, arg.characterID); err != nil {
			return false, err
		}
	}
	if arg.forceUpdate || changed || c.ContractsEscrow.IsEmpty() {
		if err := s.updateContractsEscrow(ctx, arg.characterID); err != nil {
			return false, err
		}
	}
	return changed, nil
}

func (s *CharacterService) createNewContract(ctx context.Context, characterID int64, c contractsync.ESIContract) (int64, error) {
	// Ensuring again all related objects are created to prevent occasional FK constraint error
	entityIDs, locationIDs := contractsync.ExtractEntityAndLocationIDs([]contractsync.ESIContract{c})
	err := s.eus.AddMissingEntitiesAndLocations(ctx, entityIDs, locationIDs)
	if err != nil {
		return 0, err
	}
	availability, status, typ, err := contractsync.ParseESIContract(c)
	if err != nil {
		return 0, err
	}
	ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdContractsContractIdItems")
	items2, _, err := s.esiClient.ContractsAPI.GetCharactersCharacterIdContractsContractIdItems(ctx, characterID, c.ContractId).Execute()
	if err != nil {
		return 0, err
	}
	items := xslices.Map(items2, func(x esi.CharactersCharacterIdContractsContractIdItemsGetInner) contractsync.ESIContractItem {
		return contractsync.ESIContractItem(x)
	})
	typeIDs := set.Of(xslices.Map(items, func(x contractsync.ESIContractItem) int64 {
		return x.TypeId
	})...)
	if err := s.eus.AddMissingTypes(ctx, typeIDs); err != nil {
		return 0, err
	}
	id, err := s.st.CreateCharacterContract(ctx, storage.CreateCharacterContractParams{
		AcceptorID:          c.AcceptorId,
		AssigneeID:          c.AssigneeId,
		Availability:        availability,
		Buyout:              optional.FromPtr(c.Buyout),
		CharacterID:         characterID,
		Collateral:          optional.FromPtr(c.Collateral),
		ContractID:          c.ContractId,
		DateAccepted:        optional.FromPtr(c.DateAccepted),
		DateCompleted:       optional.FromPtr(c.DateCompleted),
		DateExpired:         c.DateExpired,
		DateIssued:          c.DateIssued,
		DaysToComplete:      optional.FromPtr(c.DaysToComplete),
		EndLocationID:       optional.FromPtr(c.EndLocationId),
		ForCorporation:      c.ForCorporation,
		IssuerCorporationID: c.IssuerCorporationId,
		IssuerID:            c.IssuerId,
		Price:               optional.FromPtr(c.Price),
		Reward:              optional.FromPtr(c.Reward),
		StartLocationID:     optional.FromPtr(c.StartLocationId),
		Status:              status,
		Title:               optional.FromPtr(c.Title),
		Type:                typ,
		Volume:              optional.FromPtr(c.Volume),
	})
	if err != nil {
		return 0, err
	}
	for _, it := range items {
		err := s.st.CreateCharacterContractItem(ctx, storage.CreateContractItemParams{
			ContractID:  id,
			IsIncluded:  it.IsIncluded,
			IsSingleton: it.IsSingleton,
			Quantity:    it.Quantity,
			RawQuantity: optional.FromPtr(it.RawQuantity),
			RecordID:    it.RecordId,
			TypeID:      it.TypeId,
		})
		if err != nil {
			return id, err
		}
	}
	return id, nil
}

func (s *CharacterService) updateContract(ctx context.Context, characterID int64, o *app.Contract, c contractsync.ESIContract) error {
	status, err := contractsync.ParseESIStatus(c.Status)
	if err != nil {
		return err
	}
	if !contractsync.HasChanged(o, c, status) {
		return nil
	}
	return s.st.UpdateCharacterContract(ctx, storage.UpdateCharacterContractParams{
		AcceptorID:    c.AcceptorId,
		DateAccepted:  optional.FromPtr(c.DateAccepted),
		DateCompleted: optional.FromPtr(c.DateCompleted),
		CharacterID:   characterID,
		ContractID:    c.ContractId,
		Status:        status,
	})
}

func (s *CharacterService) updateContractBids(ctx context.Context, characterID, contractPK, contractID int64) error {
	existingBidIDs, err := s.st.ListCharacterContractBidIDs(ctx, contractPK)
	if err != nil {
		return err
	}
	ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdContractsContractIdBids")
	bids, _, err := s.esiClient.ContractsAPI.GetCharactersCharacterIdContractsContractIdBids(ctx, characterID, contractID).Execute()
	if err != nil {
		return err
	}
	newBids, bidderIDs := contractsync.NewBids(existingBidIDs, xslices.Map(bids, func(x esi.CharactersCharacterIdContractsContractIdBidsGetInner) contractsync.ESIContractBid {
		return contractsync.ESIContractBid(x)
	}))
	if len(newBids) == 0 {
		return nil
	}
	if bidderIDs.Size() > 0 {
		if _, err = s.eus.AddMissingEntities(ctx, bidderIDs); err != nil {
			return err
		}
	}
	for _, b := range newBids {
		err := s.st.CreateCharacterContractBid(ctx, storage.CreateContractBidParams{
			ContractID: contractPK,
			Amount:     b.Amount,
			BidID:      b.BidId,
			BidderID:   b.BidderId,
			DateBid:    b.DateBid,
		})
		if err != nil {
			return err
		}
	}
	slog.Info("created contract bids", "characterID", characterID, "contract", contractID, "count", len(newBids))
	return nil
}

func (s *CharacterService) updateContractItemsValue(ctx context.Context, characterID int64) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("updateContractItemsValue: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	v, err := s.st.CalculateCharacterContractItemsValue(ctx, characterID)
	if err != nil {
		return wrapErr(err)
	}
	err = s.st.UpdateCharacterContractItemsValue(ctx, characterID, optional.New(v))
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (s *CharacterService) updateContractsEscrow(ctx context.Context, characterID int64) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("updateContractsEscrow: %d: %w", characterID, err)
	}
	v, err := s.st.CalculateCharacterContractsEscrow(ctx, characterID)
	if err != nil {
		return wrapErr(err)
	}
	err = s.st.UpdateCharacterContractsEscrow(ctx, characterID, optional.New(v))
	if err != nil {
		return wrapErr(err)
	}
	return nil
}
