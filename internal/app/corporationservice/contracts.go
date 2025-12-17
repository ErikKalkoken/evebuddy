package corporationservice

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ErikKalkoken/kx/set"
	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// GetContract fetches and returns a contract from the database.
func (s *CorporationService) GetContract(ctx context.Context, corporationID, contractID int32) (*app.CorporationContract, error) {
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
	var max float32
	var top *app.CorporationContractBid
	for _, b := range bids {
		if top == nil || b.Amount > max {
			top = b
		}
	}
	return top, nil
}

func (s *CorporationService) ListCorporationContracts(ctx context.Context, corporationID int32) ([]*app.CorporationContract, error) {
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
func (s *CorporationService) updateContractsESI(ctx context.Context, arg app.CorporationSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCorporationContracts {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, arg app.CorporationSectionUpdateParams) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdContracts")
			contracts, err := xgoesi.FetchPages(
				func(pageNum int) ([]esi.GetCorporationsCorporationIdContracts200Ok, *http.Response, error) {
					return s.esiClient.ESI.ContractsApi.GetCorporationsCorporationIdContracts(
						ctx, arg.CorporationID, &esi.GetCorporationsCorporationIdContractsOpts{
							Page: esioptional.NewInt32(int32(pageNum)),
						},
					)
				},
			)
			if err != nil {
				return false, err
			}
			slog.Debug("Received contracts from ESI", "corporationID", arg.CorporationID, "count", len(contracts))
			return contracts, nil
		},
		func(ctx context.Context, arg app.CorporationSectionUpdateParams, data any) error {
			contracts := data.([]esi.GetCorporationsCorporationIdContracts200Ok)
			// fetch missing eve entities
			var entityIDs set.Set[int32]
			var locationIDs set.Set[int64]
			for _, c := range contracts {
				entityIDs.Add(c.AcceptorId, c.AssigneeId, c.IssuerId, c.IssuerCorporationId)
				locationIDs.Add(c.StartLocationId, c.EndLocationId)
			}
			err := s.eus.AddMissingEveEntitiesAndLocations(ctx, entityIDs, locationIDs)
			if err != nil {
				return err
			}
			// identify new contracts
			ii, err := s.st.ListCorporationContractIDs(ctx, arg.CorporationID)
			if err != nil {
				return err
			}
			existingIDs := set.Of(ii...)
			var existingContracts, newContracts []esi.GetCorporationsCorporationIdContracts200Ok
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
					if err := s.createNewContract(ctx, arg.CorporationID, c); err != nil {
						slog.Error("create contract", "contract", c, "error", err)
						continue
					} else {
						count++
					}
				}
				slog.Info("Stored new contracts", "corporationID", arg.CorporationID, "count", count)
			}
			if len(existingContracts) > 0 {
				var count int
				for _, c := range existingContracts {
					if err := s.updateContract(ctx, arg.CorporationID, c); err != nil {
						slog.Error("update contract", "contract", c, "error", err)
						continue
					} else {
						count++
					}
				}
				slog.Info("Updated contracts", "corporationID", arg.CorporationID, "count", count)
			}
			// add new bids for auctions
			for _, c := range contracts {
				if c.Type_ != "auction" {
					continue
				}
				err := s.updateContractBids(ctx, arg.CorporationID, c.ContractId)
				if err != nil {
					slog.Error("update contract bids", "contract", c, "error", err)
					continue
				}
			}
			return nil
		})
}

func (s *CorporationService) createNewContract(ctx context.Context, corporationID int32, c esi.GetCorporationsCorporationIdContracts200Ok) error {
	// Ensuring again all related objects are created to prevent occasional FK constraint error
	entityIDs := set.Of(c.AcceptorId, c.AssigneeId, c.IssuerId, c.IssuerCorporationId)
	locationIDs := set.Of(c.StartLocationId, c.EndLocationId)
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
	typ, ok := contractTypeFromESIValue[c.Type_]
	if !ok {
		return fmt.Errorf("unknown type: %s", c.Type_)
	}
	arg := storage.CreateCorporationContractParams{
		AcceptorID:          c.AcceptorId,
		AssigneeID:          c.AssigneeId,
		Availability:        availability,
		Buyout:              c.Buyout,
		CorporationID:       corporationID,
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
	id, err := s.st.CreateCorporationContract(ctx, arg)
	if err != nil {
		return err
	}
	ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdContractsContractIdItems")
	items, _, err := s.esiClient.ESI.ContractsApi.GetCorporationsCorporationIdContractsContractIdItems(ctx, c.ContractId, corporationID, nil)
	if err != nil {
		return err
	}
	typeIDs := set.Of(xslices.Map(items, func(x esi.GetCorporationsCorporationIdContractsContractIdItems200Ok) int32 {
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
			RawQuantity: it.RawQuantity,
			RecordID:    it.RecordId,
			TypeID:      it.TypeId,
		}
		if err := s.st.CreateCorporationContractItem(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func (s *CorporationService) updateContract(ctx context.Context, corporationID int32, c esi.GetCorporationsCorporationIdContracts200Ok) error {
	status, ok := contractStatusFromESIValue[c.Status]
	if !ok {
		return fmt.Errorf("unknown status: %s", c.Status)
	}
	o, err := s.st.GetCorporationContract(ctx, corporationID, c.ContractId)
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
	arg := storage.UpdateCorporationContractParams{
		AcceptorID:    c.AcceptorId,
		DateAccepted:  c.DateAccepted,
		DateCompleted: c.DateCompleted,
		CorporationID: corporationID,
		ContractID:    c.ContractId,
		Status:        status,
	}
	if err := s.st.UpdateCorporationContract(ctx, arg); err != nil {
		return err
	}
	return nil
}

func (s *CorporationService) updateContractBids(ctx context.Context, corporationID, contractID int32) error {
	c, err := s.st.GetCorporationContract(ctx, corporationID, contractID)
	if err != nil {
		return err
	}
	existingBidIDs, err := s.st.ListCorporationContractBidIDs(ctx, c.ID)
	if err != nil {
		return err
	}
	ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdContractsContractIdBids")
	bids, _, err := s.esiClient.ESI.ContractsApi.GetCorporationsCorporationIdContractsContractIdBids(ctx, contractID, corporationID, nil)
	if err != nil {
		return err
	}
	newBids := make([]esi.GetCorporationsCorporationIdContractsContractIdBids200Ok, 0)
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
