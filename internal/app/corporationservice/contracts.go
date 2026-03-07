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
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
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

func (s *CorporationService) GetContractTopBid(ctx context.Context, contractID int64) (*app.CorporationContractBid, error) {
	bids, err := s.st.ListCorporationContractBids(ctx, contractID)
	if err != nil {
		return nil, err
	}
	if len(bids) == 0 {
		return nil, app.ErrNotFound
	}
	var max float64
	var top *app.CorporationContractBid
	for _, b := range bids {
		if top == nil || b.Amount > max {
			top = b
		}
	}
	return top, nil
}

func (s *CorporationService) ListCorporationContracts(ctx context.Context, corporationID int64) ([]*app.CorporationContract, error) {
	return s.st.ListCorporationContracts(ctx, corporationID)
}

func (s *CorporationService) ListContractItems(ctx context.Context, contractID int64) ([]*app.CorporationContractItem, error) {
	return s.st.ListCorporationContractItems(ctx, contractID)
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
			contracts := data.([]esi.CorporationsCorporationIdContractsGetInner)
			// filter out unwanted contracts
			contracts = slices.DeleteFunc(contracts, func(x esi.CorporationsCorporationIdContractsGetInner) bool {
				return x.Status == "deleted"
			})
			// fetch missing eve entities
			var entityIDs set.Set[int64]
			var locationIDs set.Set[int64]
			for _, c := range contracts {
				entityIDs.Add(c.AcceptorId, c.AssigneeId, c.IssuerId, c.IssuerCorporationId)
				if c.StartLocationId != nil && c.EndLocationId != nil {
					locationIDs.Add(*c.StartLocationId, *c.EndLocationId)
				}
			}
			err := s.eus.AddMissingEveEntitiesAndLocations(ctx, entityIDs, locationIDs)
			if err != nil {
				return false, err
			}
			// identify new contracts
			current, err := s.st.ListCorporationContracts(ctx, arg.corporationID)
			if err != nil {
				return false, err
			}
			currentIDs := set.Collect(xiter.MapSlice(current, func(x *app.CorporationContract) int64 {
				return x.ContractID
			}))
			var existingContracts, newContracts []esi.CorporationsCorporationIdContractsGetInner
			for _, c := range contracts {
				if currentIDs.Contains(c.ContractId) {
					existingContracts = append(existingContracts, c)
				} else {
					newContracts = append(newContracts, c)
				}
			}
			slog.Debug("contracts", "existing", currentIDs, "entries", contracts)
			// create new entries
			if len(newContracts) > 0 {
				var count int
				for _, c := range newContracts {
					if err := s.createNewContract(ctx, arg.corporationID, c); err != nil {
						slog.Error("create contract", "contract", c, "error", err)
						continue
					} else {
						count++
					}
				}
				slog.Info("Stored new contracts", "corporationID", arg.corporationID, "count", count)
			}
			if len(existingContracts) > 0 {
				var count int
				for _, c := range existingContracts {
					if err := s.updateContract(ctx, arg.corporationID, c); err != nil {
						slog.Error("update contract", "contract", c, "error", err)
						continue
					} else {
						count++
					}
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
					slog.Error("update contract bids", "contract", c, "error", err)
					continue
				}
			}
			// delete stale local contracts
			var unfinishedIDs set.Set[int64]
			for _, o := range current {
				if !o.Status.IsFinished() {
					unfinishedIDs.Add(o.ContractID)
				}
			}
			incomingIDs := set.Collect(xiter.MapSlice(contracts, func(x esi.CorporationsCorporationIdContractsGetInner) int64 {
				return x.ContractId
			}))
			staleIDs := set.Difference(unfinishedIDs, incomingIDs)
			if err := s.st.DeleteCorporationContracts(ctx, arg.corporationID, staleIDs); err != nil {
				return false, err
			}
			return true, nil
		})
}

func (s *CorporationService) createNewContract(ctx context.Context, corporationID int64, c esi.CorporationsCorporationIdContractsGetInner) error {
	// Ensuring again all related objects are created to prevent occasional FK constraint error
	entityIDs := set.Of(c.AcceptorId, c.AssigneeId, c.IssuerId, c.IssuerCorporationId)
	var locationIDs set.Set[int64]
	if c.StartLocationId != nil && c.EndLocationId != nil {
		locationIDs.Add(*c.StartLocationId, *c.EndLocationId)
	}
	err := s.eus.AddMissingEveEntitiesAndLocations(ctx, entityIDs, locationIDs)
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
	typ, ok := contractTypeFromESIValue[c.Type]
	if !ok {
		return fmt.Errorf("unknown type: %s", c.Type)
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
	items, _, err := s.esiClient.ContractsAPI.GetCorporationsCorporationIdContractsContractIdItems(ctx, c.ContractId, corporationID).Execute()
	if err != nil {
		return err
	}
	typeIDs := set.Of(xslices.Map(items, func(x esi.CharactersCharacterIdContractsContractIdItemsGetInner) int64 {
		return x.TypeId

	})...)
	if err := s.eus.AddMissingTypes(ctx, typeIDs); err != nil {
		return err
	}
	for _, it := range items {
		arg := storage.CreateCorporationContractItemParams{
			ContractID:  id,
			IsIncluded:  it.IsIncluded,
			IsSingleton: it.IsSingleton,
			Quantity:    it.Quantity,
			RawQuantity: optional.FromPtr(it.RawQuantity),
			RecordID:    it.RecordId,
			TypeID:      it.TypeId,
		}
		if err := s.st.CreateCorporationContractItem(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func (s *CorporationService) updateContract(ctx context.Context, corporationID int64, c esi.CorporationsCorporationIdContractsGetInner) error {
	status, ok := contractStatusFromESIValue[c.Status]
	if !ok {
		return fmt.Errorf("unknown status: %s", c.Status)
	}
	o, err := s.st.GetCorporationContract(ctx, corporationID, c.ContractId)
	if err != nil {
		return err
	}
	acceptorID := optional.Map(o.Acceptor, 0, func(x *app.EveEntity) int64 {
		return x.ID
	})
	if c.AcceptorId == acceptorID &&
		optional.Equal2(optional.FromPtr(c.DateAccepted), o.DateAccepted) &&
		optional.Equal2(optional.FromPtr(c.DateCompleted), o.DateCompleted) &&
		o.Status == contractStatusFromESIValue[c.Status] {
		return nil
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
		return err
	}
	return nil
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
	var newBids []esi.CharactersCharacterIdContractsContractIdBidsGetInner
	for _, b := range bids {
		if !existingBidIDs.Contains(b.BidId) {
			newBids = append(newBids, b)
		}
	}
	if len(newBids) == 0 {
		return nil
	}
	var eeIDs set.Set[int64]
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
		arg := storage.CreateCorporationContractBidParams{
			ContractID: c.ID,
			Amount:     b.Amount,
			BidID:      b.BidId,
			BidderID:   b.BidderId,
			DateBid:    b.DateBid,
		}
		if err := s.st.CreateCorporationContractBid(ctx, arg); err != nil {
			return err
		}
	}
	slog.Info("created contract bids", "corporationID", corporationID, "contract", contractID, "count", len(newBids))
	return nil
}
