package corporationservice

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/contractsync"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// GetContract fetches and returns a contract from the database.
func (s *CorporationService) GetContract(ctx context.Context, corporationID, contractID int64) (*app.CorporationContract, error) {
	return s.st.GetCorporationContract(ctx, corporationID, contractID)
}

// GetContractBidSummary returns the number of bids on a contract and its top bid, if any.
func (s *CorporationService) GetContractBidSummary(ctx context.Context, contractID int64) (int, optional.Optional[*app.ContractBid], error) {
	bids, err := s.st.ListCorporationContractBids(ctx, contractID)
	if err != nil {
		return 0, optional.Optional[*app.ContractBid]{}, err
	}
	top, ok := app.ContractTopBid(bids)
	if !ok {
		return 0, optional.Optional[*app.ContractBid]{}, nil
	}
	return len(bids), optional.New(top), nil
}

func (s *CorporationService) ListCorporationContracts(ctx context.Context, corporationID int64) ([]*app.CorporationContract, error) {
	return s.st.ListCorporationContracts(ctx, corporationID)
}

// ListActiveCorporationContracts returns the amount of active contracts in a corporation.
func (s *CorporationService) ListActiveCorporationContracts(ctx context.Context, corporationID int64) (int64, error) {
	return s.st.CountCorporationContractsWithStatus(ctx, corporationID, app.ContractStatusActive)
}

func (s *CorporationService) ListContractItems(ctx context.Context, contractID int64) ([]*app.ContractItem, error) {
	return s.st.ListCorporationContractItems(ctx, contractID)
}

// CalculateContractsEscrow returns the sum of courier and auction contract escrow held by the corporation.
func (s *CorporationService) CalculateContractsEscrow(ctx context.Context, corporationID int64) (float64, error) {
	return s.st.CalculateCorporationContractsEscrow(ctx, corporationID)
}

// updateContractsESI updates the contracts from ESI and reports whether they have changed.
func (s *CorporationService) updateContractsESI(ctx context.Context, arg corporationSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCorporationContracts {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true, // skipping change detection because it can not detect new bids
		func(ctx context.Context, arg corporationSectionUpdateParams) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdContracts")
			contracts, err := xgoesi.FetchPages(
				func(page int32) ([]esi.CorporationsCorporationIdContractsGetInner, *http.Response, error) {
					return s.esiClient.ContractsAPI.GetCorporationsCorporationIdContracts(ctx, arg.corporationID).Page(page).Execute()
				},
			)
			if err != nil {
				return false, err
			}
			slices.SortFunc(contracts, func(a, b esi.CorporationsCorporationIdContractsGetInner) int {
				return cmp.Compare(a.ContractId, b.ContractId)
			})
			slog.Debug("Received contracts from ESI", "corporationID", arg.corporationID, "count", len(contracts))
			return contracts, nil
		},
		func(ctx context.Context, arg corporationSectionUpdateParams, data any) (bool, error) {
			contracts := xslices.Map(data.([]esi.CorporationsCorporationIdContractsGetInner), func(x esi.CorporationsCorporationIdContractsGetInner) contractsync.ESIContract {
				return contractsync.ESIContract(x)
			})
			contracts = contractsync.RemoveDeleted(contracts)
			entityIDs, locationIDs := contractsync.ExtractEntityAndLocationIDs(contracts)
			err := s.eus.AddMissingEntitiesAndLocations(ctx, entityIDs, locationIDs)
			if err != nil {
				return false, err
			}
			oo, err := s.st.ListCorporationContracts(ctx, arg.corporationID)
			if err != nil {
				return false, err
			}
			current := xslices.Map(oo, func(x *app.CorporationContract) *app.Contract {
				return &x.Contract
			})
			staleIDs, changed, err := contractsync.Sync(
				ctx,
				contracts,
				current,
				func(ctx context.Context, c contractsync.ESIContract) (int64, error) {
					return s.createNewContract(ctx, arg.corporationID, c)
				},
				func(ctx context.Context, o *app.Contract, c contractsync.ESIContract) (bool, error) {
					return s.updateContract(ctx, arg.corporationID, o, c)
				},
				func(ctx context.Context, contractPK, contractID int64) (bool, error) {
					return s.updateContractBids(ctx, arg.corporationID, contractPK, contractID)
				},
				slog.Int64("corporationID", arg.corporationID),
			)
			if err != nil {
				return false, err
			}
			if err := s.st.DeleteCorporationContracts(ctx, arg.corporationID, staleIDs); err != nil {
				return false, err
			}
			return changed, nil
		})
}

// createNewContract stores a new contract with its items and returns its ID.
// The ID is also returned when only storing the items failed.
func (s *CorporationService) createNewContract(ctx context.Context, corporationID int64, c contractsync.ESIContract) (int64, error) {
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
	var items []contractsync.ESIContractItem
	if typ.HasItems() {
		ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdContractsContractIdItems")
		items2, _, err := s.esiClient.ContractsAPI.GetCorporationsCorporationIdContractsContractIdItems(ctx, c.ContractId, corporationID).Execute()
		if err != nil {
			return 0, err
		}
		items = xslices.Map(items2, func(x esi.CharactersCharacterIdContractsContractIdItemsGetInner) contractsync.ESIContractItem {
			return contractsync.ESIContractItem(x)
		})
		typeIDs := set.Of(xslices.Map(items, func(x contractsync.ESIContractItem) int64 {
			return x.TypeId
		})...)
		if err := s.eus.AddMissingTypes(ctx, typeIDs); err != nil {
			return 0, err
		}
	}
	id, err := s.st.CreateCorporationContract(ctx, storage.CreateCorporationContractParams{
		AcceptorID:          c.AcceptorId,
		AssigneeID:          c.AssigneeId,
		Availability:        availability,
		Buyout:              optional.FromPtr(c.Buyout),
		CorporationID:       corporationID,
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
		err := s.st.CreateCorporationContractItem(ctx, storage.CreateContractItemParams{
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

// updateContract updates a stored contract from ESI and reports whether it has changed.
func (s *CorporationService) updateContract(ctx context.Context, corporationID int64, o *app.Contract, c contractsync.ESIContract) (bool, error) {
	status, err := contractsync.ParseESIStatus(c.Status)
	if err != nil {
		return false, err
	}
	if !contractsync.HasChanged(o, c, status) {
		return false, nil
	}
	err = s.st.UpdateCorporationContract(ctx, storage.UpdateCorporationContractParams{
		AcceptorID:    c.AcceptorId,
		DateAccepted:  optional.FromPtr(c.DateAccepted),
		DateCompleted: optional.FromPtr(c.DateCompleted),
		CorporationID: corporationID,
		ContractID:    c.ContractId,
		Status:        status,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// updateContractBids stores new bids of an auction from ESI and reports whether there were any.
func (s *CorporationService) updateContractBids(ctx context.Context, corporationID, contractPK, contractID int64) (bool, error) {
	existingBidIDs, err := s.st.ListCorporationContractBidIDs(ctx, contractPK)
	if err != nil {
		return false, err
	}
	ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdContractsContractIdBids")
	bids, _, err := s.esiClient.ContractsAPI.GetCorporationsCorporationIdContractsContractIdBids(ctx, contractID, corporationID).Execute()
	if err != nil {
		return false, err
	}
	newBids, bidderIDs := contractsync.NewBids(existingBidIDs, xslices.Map(bids, func(x esi.CharactersCharacterIdContractsContractIdBidsGetInner) contractsync.ESIContractBid {
		return contractsync.ESIContractBid(x)
	}))
	if len(newBids) == 0 {
		return false, nil
	}
	if bidderIDs.Size() > 0 {
		if _, err = s.eus.AddMissingEntities(ctx, bidderIDs); err != nil {
			return false, err
		}
	}
	for _, b := range newBids {
		err := s.st.CreateCorporationContractBid(ctx, storage.CreateContractBidParams{
			ContractID: contractPK,
			Amount:     b.Amount,
			BidID:      b.BidId,
			BidderID:   b.BidderId,
			DateBid:    b.DateBid,
		})
		if err != nil {
			return false, err
		}
	}
	slog.Info("created contract bids", "corporationID", corporationID, "contract", contractID, "count", len(newBids))
	return true, nil
}
