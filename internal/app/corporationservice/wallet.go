package corporationservice

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func (s *CorporationService) GetWalletName(ctx context.Context, corporationID int64, division app.Division) (string, error) {
	n, err := s.st.GetCorporationWalletName(ctx, storage.CorporationDivision{
		CorporationID: corporationID,
		Division:      division,
	})
	if err != nil {
		return "", err
	}
	return n.Name, nil
}

func (s *CorporationService) ListWalletNames(ctx context.Context, corporationID int64) map[app.Division]string {
	m := make(map[app.Division]string)
	for _, d := range app.Divisions {
		m[d] = d.DefaultWalletName()
	}
	oo, err := s.st.ListCorporationWalletNames(ctx, corporationID)
	if err != nil {
		if ctx.Err() == nil {
			slog.Warn("Failed to fetch wallet names. Falling back to defaults.", "corporationID", corporationID, "error", err)
		}
		return m
	}
	for _, o := range oo {
		if o.Name == "" {
			continue
		}
		m[o.Division] = o.Name
	}
	return m
}

func (s *CorporationService) GetWalletBalance(ctx context.Context, corporationID int64, division app.Division) (float64, error) {
	enabled, err := s.PermittedSection(ctx, corporationID, app.SectionCorporationWalletBalances)
	if err != nil {
		return 0, err
	}
	if !enabled {
		return 0, app.ErrNotFound
	}
	x, err := s.st.GetCorporationWalletBalance(ctx, storage.CorporationDivision{
		CorporationID: corporationID,
		Division:      division,
	})
	if err != nil {
		return 0, err
	}
	return x.Balance, nil
}

// ListWalletBalances returns the wallet balances for all divisions.
// It returns an empty slice if there is no data.
func (s *CorporationService) ListWalletBalances(ctx context.Context, corporationID int64) ([]app.CorporationWalletBalanceWithName, error) {
	enabled, err := s.PermittedSection(ctx, corporationID, app.SectionCorporationWalletBalances)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return []app.CorporationWalletBalanceWithName{}, nil
	}
	oo, err := s.st.ListCorporationWalletBalances(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	if len(oo) == 0 {
		return []app.CorporationWalletBalanceWithName{}, nil
	}
	balances := make(map[app.Division]float64)
	for _, o := range oo {
		balances[o.Division] = o.Balance
	}
	names := s.ListWalletNames(ctx, corporationID)
	w := make([]app.CorporationWalletBalanceWithName, 0, len(app.Divisions))
	for _, d := range app.Divisions {
		w = append(w, app.CorporationWalletBalanceWithName{
			Balance:       balances[d],
			CorporationID: corporationID,
			Division:      d,
			Name:          names[d],
		})
	}
	return w, nil
}

// GetWalletBalancesTotal returns the sum of all wallet balances.
// It returns empty if there was no data.
func (s *CorporationService) GetWalletBalancesTotal(ctx context.Context, corporationID int64) (optional.Optional[float64], error) {
	var b optional.Optional[float64]
	enabled, err := s.PermittedSection(ctx, corporationID, app.SectionCorporationWalletBalances)
	if err != nil {
		return b, err
	}
	if !enabled {
		return b, nil
	}
	oo, err := s.st.ListCorporationWalletBalances(ctx, corporationID)
	if err != nil {
		return b, err
	}
	if len(oo) == 0 {
		return b, nil
	}
	for _, o := range oo {
		b = optional.SumNonEmpty(b, optional.New(o.Balance))
	}
	return b, nil
}

func (s *CorporationService) updateWalletBalancesESI(ctx context.Context, arg corporationSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCorporationWalletBalances {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, false,
		func(ctx context.Context, arg corporationSectionUpdateParams) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdWallets")
			wallets, _, err := s.esiClient.WalletAPI.GetCorporationsCorporationIdWallets(ctx, arg.corporationID).Execute()
			if err != nil {
				return false, err
			}
			return wallets, nil
		},
		func(ctx context.Context, arg corporationSectionUpdateParams, data any) (bool, error) {
			wallets := data.([]esi.CorporationsCorporationIdWalletsGetInner)
			for _, w := range wallets {
				if err := s.st.UpdateOrCreateCorporationWalletBalance(ctx, storage.UpdateOrCreateCorporationWalletBalanceParams{
					CorporationID: arg.corporationID,
					Division:      app.Division(w.Division),
					Balance:       w.Balance,
				}); err != nil {
					return false, err
				}
			}
			slog.Info("Updated corporation wallet balances", "corporationID", arg.corporationID)
			return true, nil
		})
}

func (s *CorporationService) GetWalletJournalEntry(ctx context.Context, corporationID int64, d app.Division, refID int64) (*app.CorporationWalletJournalEntry, error) {
	enabled, err := s.PermittedSection(ctx, corporationID, app.CorporationSectionWalletJournal(d))
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, app.ErrNotFound
	}
	return s.st.GetCorporationWalletJournalEntry(ctx, storage.GetCorporationWalletJournalEntryParams{
		CorporationID: corporationID,
		Division:      d,
		RefID:         refID,
	})
}

func (s *CorporationService) ListWalletJournalEntries(ctx context.Context, corporationID int64, d app.Division) ([]*app.CorporationWalletJournalEntry, error) {
	enabled, err := s.PermittedSection(ctx, corporationID, app.CorporationSectionWalletJournal(d))
	if err != nil {
		return nil, err
	}
	if !enabled {
		return []*app.CorporationWalletJournalEntry{}, nil
	}
	return s.st.ListCorporationWalletJournalEntries(ctx, storage.CorporationDivision{
		CorporationID: corporationID,
		Division:      d,
	})
}

// TODO: Only read pages with unknown journal IDs

func (s *CorporationService) updateWalletJournalESI(ctx context.Context, arg corporationSectionUpdateParams) (bool, error) {
	if arg.corporationID <= 0 || !app.CorporationSectionsWalletJournal.Contains(arg.section) {
		return false, fmt.Errorf("updateWalletJournalESI %+v: %w", arg, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true,
		func(ctx context.Context, arg corporationSectionUpdateParams) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdWalletsDivisionJournal")
			cacheKey := fmt.Sprintf("wallet-journal-last-id-%d-%d", arg.corporationID, arg.section.Division())
			lastID, found := s.cache.GetInt64(cacheKey)
			checkLastID := found && !arg.forceUpdate
			entries, err := xgoesi.FetchPagesWithStop(
				func(page int32) ([]esi.CorporationsCorporationIdWalletsDivisionJournalGetInner, *http.Response, error) {
					return s.esiClient.WalletAPI.GetCorporationsCorporationIdWalletsDivisionJournal(ctx, arg.corporationID, int64(arg.section.Division())).Page(page).Execute()
				}, func(x esi.CorporationsCorporationIdWalletsDivisionJournalGetInner) bool {
					return checkLastID && x.Id <= lastID
				})
			if err != nil {
				return false, err
			}
			ids := xslices.Map(entries, func(x esi.CorporationsCorporationIdWalletsDivisionJournalGetInner) int64 {
				return x.Id
			})
			if len(ids) > 0 {
				s.cache.SetInt64(cacheKey, slices.Max(ids), 0)
			}
			slog.Debug(
				"Received wallet journal from ESI",
				"corporationID", arg.corporationID,
				"divisionID", arg.section.Division(),
				"entries", len(entries),
			)
			return entries, nil
		},
		func(ctx context.Context, arg corporationSectionUpdateParams, data any) (bool, error) {
			entries := data.([]esi.CorporationsCorporationIdWalletsDivisionJournalGetInner)
			existingIDs, err := s.st.ListCorporationWalletJournalEntryIDs(ctx, storage.CorporationDivision{
				CorporationID: arg.corporationID,
				Division:      arg.section.Division(),
			})
			if err != nil {
				return false, err
			}
			var newEntries []esi.CorporationsCorporationIdWalletsDivisionJournalGetInner
			for _, e := range entries {
				if existingIDs.Contains(e.Id) {
					continue
				}
				newEntries = append(newEntries, e)
			}
			slog.Debug("wallet journal", "existing", existingIDs, "entries", entries)

			if len(newEntries) == 0 {
				slog.Info(
					"No new wallet journal entries",
					"corporationID", arg.corporationID,
					"divisionID", arg.section.Division(),
				)
				return false, nil
			}

			var ids set.Set[int64]
			for _, o := range newEntries {
				for _, x := range []*int64{o.FirstPartyId, o.SecondPartyId, o.TaxReceiverId} {
					if x != nil {
						ids.Add(*x)
					}
				}
			}
			_, err = s.eus.AddMissingEntities(ctx, ids)
			if err != nil {
				return false, err
			}
			for _, o := range newEntries {
				err := s.st.CreateCorporationWalletJournalEntry(ctx, storage.CreateCorporationWalletJournalEntryParams{
					Amount:        optional.FromPtr(o.Amount),
					Balance:       optional.FromPtr(o.Balance),
					ContextID:     optional.FromPtr(o.ContextId),
					ContextIDType: optional.FromPtr(o.ContextIdType),
					Date:          o.Date,
					Description:   o.Description,
					Division:      arg.section.Division(),
					FirstPartyID:  optional.FromPtr(o.FirstPartyId),
					RefID:         o.Id,
					CorporationID: arg.corporationID,
					RefType:       o.RefType,
					Reason:        optional.FromPtr(o.Reason),
					SecondPartyID: optional.FromPtr(o.SecondPartyId),
					Tax:           optional.FromPtr(o.Tax),
					TaxReceiverID: optional.FromPtr(o.TaxReceiverId),
				})
				if err != nil {
					return false, err
				}
			}
			slog.Info(
				"Stored new wallet journal entries",
				"corporationID", arg.corporationID,
				"divisionID", arg.section.Division(),
				"entries", len(newEntries),
			)
			return true, nil
		})
}

const (
	maxTransactionsPerPage = 2_500 // maximum objects returned per page
)

func (s *CorporationService) GetWalletTransaction(ctx context.Context, corporationID int64, d app.Division, transactionID int64) (*app.CorporationWalletTransaction, error) {
	enabled, err := s.PermittedSection(ctx, corporationID, app.CorporationSectionWalletTransactions(d))
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, fmt.Errorf("section not enabled: %d %d: %w", corporationID, d, app.ErrNotFound)
	}
	return s.st.GetCorporationWalletTransaction(ctx, storage.GetCorporationWalletTransactionParams{
		CorporationID: corporationID,
		Division:      d,
		TransactionID: transactionID,
	})
}

func (s *CorporationService) ListWalletTransactionsAll(ctx context.Context, corporationID int64) ([]*app.CorporationWalletTransaction, error) {
	return s.st.ListCorporationWalletTransactionsAll(ctx, corporationID)
}

func (s *CorporationService) ListWalletTransactionsForDivision(ctx context.Context, corporationID int64, d app.Division) ([]*app.CorporationWalletTransaction, error) {
	enabled, err := s.PermittedSection(ctx, corporationID, app.CorporationSectionWalletTransactions(d))
	if err != nil {
		return nil, err
	}
	if !enabled {
		return []*app.CorporationWalletTransaction{}, nil
	}
	return s.st.ListCorporationWalletTransactions(ctx, storage.CorporationDivision{
		CorporationID: corporationID,
		Division:      d,
	})
}

// updateWalletTransactionESI updates the wallet transactions from ESI and reports whether they has changed.
func (s *CorporationService) updateWalletTransactionESI(ctx context.Context, arg corporationSectionUpdateParams) (bool, error) {
	if arg.corporationID <= 0 || !app.CorporationSectionsWalletTransactions.Contains(arg.section) {
		return false, fmt.Errorf("updateWalletTransactionESI %+v: %w", arg, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true,
		func(ctx context.Context, arg corporationSectionUpdateParams) (any, error) {
			cacheKey := fmt.Sprintf("wallet-transactions-last-id-%d-%d", arg.corporationID, arg.section.Division())
			lastID, found := s.cache.GetInt64(cacheKey)
			checkLastID := found && !arg.forceUpdate
			transactions, err := s.fetchWalletTransactionsESI(ctx, arg, lastID, checkLastID)
			if err != nil {
				return false, err
			}
			ids := xslices.Map(transactions, func(x esi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner) int64 {
				return x.TransactionId
			})
			if len(ids) > 0 {
				s.cache.SetInt64(cacheKey, slices.Max(ids), 0)
			}
			return transactions, nil
		},
		func(ctx context.Context, arg corporationSectionUpdateParams, data any) (bool, error) {
			transactions := data.([]esi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner)
			existingIDs, err := s.st.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
				CorporationID: arg.corporationID,
				Division:      arg.section.Division(),
			})
			if err != nil {
				return false, err
			}
			var newEntries []esi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner
			for _, e := range transactions {
				if existingIDs.Contains(e.TransactionId) {
					continue
				}
				newEntries = append(newEntries, e)
			}
			slog.Debug("wallet transaction", "existing", existingIDs, "entries", transactions)

			if len(newEntries) == 0 {
				slog.Info(
					"No new wallet transactions",
					"corporationID", arg.corporationID,
					"divisionID", arg.section.Division(),
				)
				return false, nil
			}

			var entityIDs, typeIDs set.Set[int64]
			var locationIDs set.Set[int64]
			for _, en := range newEntries {
				entityIDs.Add(en.ClientId)
				locationIDs.Add(en.LocationId)
				typeIDs.Add(en.TypeId)
			}
			g := new(errgroup.Group)
			g.Go(func() error {
				_, err := s.eus.AddMissingEntities(ctx, entityIDs)
				return err
			})
			g.Go(func() error {
				return s.eus.AddMissingLocations(ctx, locationIDs)
			})
			g.Go(func() error {
				return s.eus.AddMissingTypes(ctx, typeIDs)
			})
			if err := g.Wait(); err != nil {
				return false, err
			}
			for _, o := range newEntries {
				err := s.st.CreateCorporationWalletTransaction(ctx, storage.CreateCorporationWalletTransactionParams{
					ClientID:      o.ClientId,
					CorporationID: arg.corporationID,
					Date:          o.Date,
					Division:      arg.section.Division(),
					EveTypeID:     o.TypeId,
					IsBuy:         o.IsBuy,
					JournalRefID:  o.JournalRefId,
					LocationID:    o.LocationId,
					Quantity:      o.Quantity,
					TransactionID: o.TransactionId,
					UnitPrice:     o.UnitPrice,
				})
				if err != nil {
					return false, err
				}
			}
			slog.Info(
				"Stored new wallet transactions",
				"corporationID", arg.corporationID,
				"divisionID", arg.section.Division(),
				"entries", len(newEntries),
			)
			return true, nil
		})
}

// fetchWalletTransactionsESI fetches wallet transactions from ESI with paging and returns them.
func (s *CorporationService) fetchWalletTransactionsESI(ctx context.Context, arg corporationSectionUpdateParams, lastID int64, checkLastID bool) ([]esi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner, error) {
	var transactions []esi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner
	var fromID int64
	maxTransactions := s.settings.MaxWalletTransactions()
	ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdWalletsDivisionTransactions")
	for {
		var oo []esi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner
		var err error
		if fromID > 0 {
			oo, _, err = s.esiClient.WalletAPI.GetCorporationsCorporationIdWalletsDivisionTransactions(ctx, arg.corporationID, int64(arg.section.Division())).FromId(fromID).Execute()
		} else {
			oo, _, err = s.esiClient.WalletAPI.GetCorporationsCorporationIdWalletsDivisionTransactions(ctx, arg.corporationID, int64(arg.section.Division())).Execute()
		}
		if err != nil {
			return nil, err
		}
		transactions = slices.Concat(transactions, oo)
		isLimitExceeded := (maxTransactions != 0 && len(transactions)+maxTransactionsPerPage > maxTransactions)
		if len(oo) < maxTransactionsPerPage || isLimitExceeded {
			break
		}
		ids := xslices.Map(oo, func(x esi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner) int64 {
			return x.TransactionId
		})
		if checkLastID && slices.Contains(ids, lastID) {
			break // stop reading once a known ID is found
		}
		fromID = slices.Min(ids)
	}
	slog.Debug("Received wallet transactions",
		slog.Int64("corporationID", arg.corporationID),
		slog.Any("divisionID", arg.section.Division()),
		slog.Any("count", len(transactions)),
	)
	return transactions, nil
}
