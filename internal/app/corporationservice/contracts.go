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

func (s *CorporationService) CountContractBids(ctx context.Context, contractID int64) (int, error) {
	x, err := s.st.ListCorporationContractBidIDs(ctx, contractID)
	if err != nil {
		return 0, err
	}
	return x.Size(), nil
}

func (s *CorporationService) GetContractTopBid(ctx context.Context, contractID int64) (*app.ContractBid, error) {
	bids, err := s.st.ListCorporationContractBids(ctx, contractID)
	if err != nil {
		return nil, err
	}
	top, ok := app.ContractTopBid(bids)
	if !ok {
		return nil, app.ErrNotFound
	}
	return top, nil
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
	v1, err := s.st.CalculateCorporationContractsCourierEscrow(ctx, corporationID)
	if err != nil {
		return 0, err
	}
	v2, err := s.st.CalculateCorporationContractsAuctionEscrow(ctx, corporationID)
	if err != nil {
		return 0, err
	}
	return v1 + v2, nil
}

// updateContractsESI updates the contracts from ESI and reports whether they have changed.
func (s *CorporationService) updateContractsESI(ctx context.Context, arg corporationSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCorporationContracts {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, false,
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
			contracts = contractsync.FilterDeleted(contracts)
			entityIDs, locationIDs := contractsync.RelatedIDs(contracts...)
			err := s.eus.AddMissingEveEntitiesAndLocations(ctx, entityIDs, locationIDs)
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
			newContracts, existingContracts := contractsync.Split(contracts, current)
			slog.Debug("contracts", "current", len(current), "entries", contracts)
			// create new entries
			if len(newContracts) > 0 {
				var count int
				for _, c := range newContracts {
					if err := s.createNewContract(ctx, arg.corporationID, c); err != nil {
						if ctx.Err() != nil {
							return false, err
						}
						slog.Error("create contract", "contract", c, "error", err)
						continue
					}
					count++
				}
				slog.Info("Stored new contracts", "corporationID", arg.corporationID, "count", count)
			}
			if len(existingContracts) > 0 {
				var count int
				for _, c := range existingContracts {
					if err := s.updateContract(ctx, arg.corporationID, c); err != nil {
						if ctx.Err() != nil {
							return false, err
						}
						slog.Error("update contract", "contract", c, "error", err)
						continue
					}
					count++
				}
				slog.Info("Updated contracts", "corporationID", arg.corporationID, "count", count)
			}
			// add new bids for auctions
			for _, c := range contracts {
				if c.Type != "auction" {
					continue
				}
				err := s.updateContractBids(ctx, arg.corporationID, c.ContractId)
				if err != nil {
					if ctx.Err() != nil {
						return false, err
					}
					slog.Error("update contract bids", "contract", c, "error", err)
					continue
				}
			}
			// delete stale local contracts
			staleIDs := contractsync.StaleIDs(current, contracts)
			if err := s.st.DeleteCorporationContracts(ctx, arg.corporationID, staleIDs); err != nil {
				return false, err
			}
			return true, nil
		})
}

func (s *CorporationService) createNewContract(ctx context.Context, corporationID int64, c contractsync.ESIContract) error {
	// Ensuring again all related objects are created to prevent occasional FK constraint error
	entityIDs, locationIDs := contractsync.RelatedIDs(c)
	err := s.eus.AddMissingEveEntitiesAndLocations(ctx, entityIDs, locationIDs)
	if err != nil {
		return err
	}
	availability, status, typ, err := contractsync.ParseESI(c)
	if err != nil {
		return err
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
		return err
	}
	ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdContractsContractIdItems")
	items2, _, err := s.esiClient.ContractsAPI.GetCorporationsCorporationIdContractsContractIdItems(ctx, c.ContractId, corporationID).Execute()
	if err != nil {
		return err
	}
	items := xslices.Map(items2, func(x esi.CharactersCharacterIdContractsContractIdItemsGetInner) contractsync.ESIContractItem {
		return contractsync.ESIContractItem(x)
	})
	typeIDs := set.Of(xslices.Map(items, func(x contractsync.ESIContractItem) int64 {
		return x.TypeId
	})...)
	if err := s.eus.AddMissingTypes(ctx, typeIDs); err != nil {
		return err
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
			return err
		}
	}
	return nil
}

func (s *CorporationService) updateContract(ctx context.Context, corporationID int64, c contractsync.ESIContract) error {
	status, err := contractsync.ParseStatus(c.Status)
	if err != nil {
		return err
	}
	o, err := s.st.GetCorporationContract(ctx, corporationID, c.ContractId)
	if err != nil {
		return err
	}
	if !contractsync.HasChanged(&o.Contract, c, status) {
		return nil
	}
	return s.st.UpdateCorporationContract(ctx, storage.UpdateCorporationContractParams{
		AcceptorID:    c.AcceptorId,
		DateAccepted:  optional.FromPtr(c.DateAccepted),
		DateCompleted: optional.FromPtr(c.DateCompleted),
		CorporationID: corporationID,
		ContractID:    c.ContractId,
		Status:        status,
	})
}

func (s *CorporationService) updateContractBids(ctx context.Context, corporationID, contractID int64) error {
	c, err := s.st.GetCorporationContract(ctx, corporationID, contractID)
	if err != nil {
		return err
	}
	existingBidIDs, err := s.st.ListCorporationContractBidIDs(ctx, c.ID)
	if err != nil {
		return err
	}
	ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdContractsContractIdBids")
	bids, _, err := s.esiClient.ContractsAPI.GetCorporationsCorporationIdContractsContractIdBids(ctx, contractID, corporationID).Execute()
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
		err := s.st.CreateCorporationContractBid(ctx, storage.CreateContractBidParams{
			ContractID: c.ID,
			Amount:     b.Amount,
			BidID:      b.BidId,
			BidderID:   b.BidderId,
			DateBid:    b.DateBid,
		})
		if err != nil {
			return err
		}
	}
	slog.Info("created contract bids", "corporationID", corporationID, "contract", contractID, "count", len(newBids))
	return nil
}
