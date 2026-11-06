package corporationservice

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/walletsync"
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
	m := app.DefaultCorporationWalletNames()
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

func (s *CorporationService) updateWalletJournalESI(ctx context.Context, arg corporationSectionUpdateParams) (bool, error) {
	if arg.corporationID <= 0 || !app.CorporationSectionsWalletJournal.Contains(arg.section) {
		return false, fmt.Errorf("updateWalletJournalESI %+v: %w", arg, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true,
		func(ctx context.Context, arg corporationSectionUpdateParams) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdWalletsDivisionJournal")
			lastID, err := s.st.GetCorporationWalletJournalLastRefID(ctx, storage.CorporationDivision{
				CorporationID: arg.corporationID,
				Division:      arg.section.Division(),
			})
			if err != nil {
				return false, err
			}
			checkLastID := lastID > 0 && !arg.forceUpdate
			entries, err := xgoesi.FetchPagesWithStop(
				func(page int32) ([]esi.CorporationsCorporationIdWalletsDivisionJournalGetInner, *http.Response, error) {
					return s.esiClient.WalletAPI.GetCorporationsCorporationIdWalletsDivisionJournal(ctx, arg.corporationID, int64(arg.section.Division())).Page(page).Execute()
				}, func(x esi.CorporationsCorporationIdWalletsDivisionJournalGetInner) bool {
					return checkLastID && x.Id <= lastID
				})
			if err != nil {
				return false, err
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
			slog.Debug("wallet journal", "existing", existingIDs, "entries", entries)
			return walletsync.SyncJournal(
				ctx,
				xslices.Map(entries, func(x esi.CorporationsCorporationIdWalletsDivisionJournalGetInner) walletsync.ESIJournalEntry {
					return walletsync.ESIJournalEntry(x)
				}),
				existingIDs,
				func(ctx context.Context, ids set.Set[int64]) error {
					_, err := s.eus.AddMissingEntities(ctx, ids)
					return err
				},
				func(ctx context.Context, items []walletsync.ESIJournalEntry) error {
					return s.st.CreateCorporationWalletJournalEntries(ctx, xslices.Map(items, func(o walletsync.ESIJournalEntry) storage.CreateCorporationWalletJournalEntryParams {
						return storage.CreateCorporationWalletJournalEntryParams{
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
						}
					}))
				},
				slog.Int64("corporationID", arg.corporationID),
				slog.Any("divisionID", arg.section.Division()),
			)
		})
}

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
			lastID, err := s.st.GetCorporationWalletLastTransactionID(ctx, storage.CorporationDivision{
				CorporationID: arg.corporationID,
				Division:      arg.section.Division(),
			})
			if err != nil {
				return false, err
			}
			checkLastID := lastID > 0 && !arg.forceUpdate
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdWalletsDivisionTransactions")
			transactions, err := walletsync.FetchTransactions(
				func(fromID int64) ([]walletsync.ESITransaction, error) {
					req := s.esiClient.WalletAPI.GetCorporationsCorporationIdWalletsDivisionTransactions(ctx, arg.corporationID, int64(arg.section.Division()))
					if fromID > 0 {
						req = req.FromId(fromID)
					}
					oo, _, err := req.Execute()
					if err != nil {
						return nil, err
					}
					return xslices.Map(oo, walletTransactionFromESI), nil
				},
				s.settings.MaxWalletTransactions(),
				lastID,
				checkLastID,
			)
			if err != nil {
				return false, err
			}
			slog.Debug("Received wallet transactions",
				slog.Int64("corporationID", arg.corporationID),
				slog.Any("divisionID", arg.section.Division()),
				slog.Any("count", len(transactions)),
			)
			return transactions, nil
		},
		func(ctx context.Context, arg corporationSectionUpdateParams, data any) (bool, error) {
			transactions := data.([]walletsync.ESITransaction)
			existingIDs, err := s.st.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
				CorporationID: arg.corporationID,
				Division:      arg.section.Division(),
			})
			if err != nil {
				return false, err
			}
			slog.Debug("wallet transaction", "existing", existingIDs, "entries", transactions)
			return walletsync.SyncTransactions(
				ctx,
				transactions,
				existingIDs,
				func(ctx context.Context, entityIDs, locationIDs, typeIDs set.Set[int64]) error {
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
					return g.Wait()
				},
				func(ctx context.Context, items []walletsync.ESITransaction) error {
					return s.st.CreateCorporationWalletTransactions(ctx, xslices.Map(items, func(o walletsync.ESITransaction) storage.CreateCorporationWalletTransactionParams {
						return storage.CreateCorporationWalletTransactionParams{
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
						}
					}))
				},
				slog.Int64("corporationID", arg.corporationID),
				slog.Any("divisionID", arg.section.Division()),
			)
		})
}

// walletTransactionFromESI maps a transaction field by field,
// because the ESI response for corporations has no IsPersonal.
func walletTransactionFromESI(x esi.CorporationsCorporationIdWalletsDivisionTransactionsGetInner) walletsync.ESITransaction {
	return walletsync.ESITransaction{
		ClientId:      x.ClientId,
		Date:          x.Date,
		IsBuy:         x.IsBuy,
		JournalRefId:  x.JournalRefId,
		LocationId:    x.LocationId,
		Quantity:      x.Quantity,
		TransactionId: x.TransactionId,
		TypeId:        x.TypeId,
		UnitPrice:     x.UnitPrice,
	}
}
