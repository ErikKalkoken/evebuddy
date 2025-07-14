package corporationservice

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"

	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
)

func (s *CorporationService) GetWalletName(ctx context.Context, corporationID int32, division app.Division) (string, error) {
	n, err := s.st.GetCorporationWalletName(ctx, storage.CorporationDivision{
		CorporationID: corporationID,
		DivisionID:    division.ID(),
	})
	if err != nil {
		return "", err
	}
	return n.Name, nil
}

func (s *CorporationService) ListWalletNames(ctx context.Context, corporationID int32) map[app.Division]string {
	m := map[app.Division]string{
		app.Division1: "Master Wallet",
		app.Division2: "2nd Wallet Division",
		app.Division3: "3rd Wallet Division",
		app.Division4: "4th Wallet Division",
		app.Division5: "5th Wallet Division",
		app.Division6: "6th Wallet Division",
		app.Division7: "7th Wallet Division",
	}
	oo, err := s.st.ListCorporationWalletNames(ctx, corporationID)
	if err != nil {
		slog.Error("Failed to fetch wallet names. Falling back to defaults.", "corporationID", corporationID, "error", err)
		return m
	}
	for _, o := range oo {
		if o.Name == "" {
			continue
		}
		m[app.Division(o.DivisionID)] = o.Name
	}
	return m
}

func (s *CorporationService) GetWalletBalance(ctx context.Context, corporationID int32, division app.Division) (float64, error) {
	x, err := s.st.GetCorporationWalletBalance(ctx, storage.CorporationDivision{
		CorporationID: corporationID,
		DivisionID:    division.ID(),
	})
	if err != nil {
		return 0, err
	}
	return x.Balance, nil
}

// ListCorporationWalletBalances returns a list of corporation
func (s *CorporationService) ListCorporationWalletBalances(ctx context.Context, corporationID int32) ([]app.CorporationWalletBalanceWithName, error) {
	oo, err := s.st.ListCorporationWalletBalances(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	m := make(map[app.Division]app.CorporationWalletBalanceWithName)
	for _, o := range oo {
		d := app.Division(o.DivisionID)
		m[d] = app.CorporationWalletBalanceWithName{
			Balance:       o.Balance,
			CorporationID: o.CorporationID,
			DivisionID:    o.DivisionID,
		}
	}
	for d, n := range s.ListWalletNames(ctx, corporationID) {
		x := m[d]
		x.Name = n
		m[d] = x
	}
	w := slices.SortedFunc(maps.Values(m), func(a, b app.CorporationWalletBalanceWithName) int {
		return cmp.Compare(a.DivisionID, b.DivisionID)
	})
	return w, nil
}

func (s *CorporationService) updateWalletBalancesESI(ctx context.Context, arg app.CorporationUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionCorporationWalletBalances {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, arg app.CorporationUpdateSectionParams) (any, error) {
			wallets, _, err := s.esiClient.ESI.WalletApi.GetCorporationsCorporationIdWallets(ctx, arg.CorporationID, nil)
			if err != nil {
				return false, err
			}
			return wallets, nil
		},
		func(ctx context.Context, arg app.CorporationUpdateSectionParams, data any) error {
			wallets := data.([]esi.GetCorporationsCorporationIdWallets200Ok)
			for _, w := range wallets {
				if err := s.st.UpdateOrCreateCorporationWalletBalance(ctx, storage.UpdateOrCreateCorporationWalletBalanceParams{
					CorporationID: arg.CorporationID,
					DivisionID:    w.Division,
					Balance:       w.Balance,
				}); err != nil {
					return err
				}
			}
			slog.Info("Updated corporation wallet balances", "corporationID", arg.CorporationID)
			return nil
		})
}

func (s *CorporationService) GetWalletJournalEntry(ctx context.Context, corporationID int32, division app.Division, refID int64) (*app.CorporationWalletJournalEntry, error) {
	return s.st.GetCorporationWalletJournalEntry(ctx, storage.GetCorporationWalletJournalEntryParams{
		CorporationID: corporationID,
		DivisionID:    division.ID(),
		RefID:         refID,
	})
}

func (s *CorporationService) ListWalletJournalEntries(ctx context.Context, corporationID int32, division app.Division) ([]*app.CorporationWalletJournalEntry, error) {
	return s.st.ListCorporationWalletJournalEntries(ctx, storage.CorporationDivision{
		CorporationID: corporationID,
		DivisionID:    division.ID(),
	})
}

func (s *CorporationService) updateWalletJournalESI(ctx context.Context, arg app.CorporationUpdateSectionParams) (bool, error) {
	sections := set.Of(
		app.SectionCorporationWalletJournal1,
		app.SectionCorporationWalletJournal2,
		app.SectionCorporationWalletJournal3,
		app.SectionCorporationWalletJournal4,
		app.SectionCorporationWalletJournal5,
		app.SectionCorporationWalletJournal6,
		app.SectionCorporationWalletJournal7,
	)
	if arg.CorporationID <= 0 || !sections.Contains(arg.Section) {
		return false, fmt.Errorf("updateWalletJournalESI %+v: %w", arg, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, arg app.CorporationUpdateSectionParams) (any, error) {
			entries, err := xesi.FetchWithPaging(
				func(pageNum int) ([]esi.GetCorporationsCorporationIdWalletsDivisionJournal200Ok, *http.Response, error) {
					opts := &esi.GetCorporationsCorporationIdWalletsDivisionJournalOpts{
						Page: esioptional.NewInt32(int32(pageNum)),
					}
					return s.esiClient.ESI.WalletApi.GetCorporationsCorporationIdWalletsDivisionJournal(
						ctx,
						arg.CorporationID,
						arg.Section.Division().ID(),
						opts,
					)
				})
			if err != nil {
				return false, err
			}
			slog.Debug(
				"Received wallet journal from ESI",
				"corporationID", arg.CorporationID,
				"divisionID", arg.Section.Division(),
				"entries", len(entries),
			)
			return entries, nil
		},
		func(ctx context.Context, arg app.CorporationUpdateSectionParams, data any) error {
			entries := data.([]esi.GetCorporationsCorporationIdWalletsDivisionJournal200Ok)
			existingIDs, err := s.st.ListCorporationWalletJournalEntryIDs(ctx, storage.CorporationDivision{
				CorporationID: arg.CorporationID,
				DivisionID:    arg.Section.Division().ID(),
			})
			if err != nil {
				return err
			}
			var newEntries []esi.GetCorporationsCorporationIdWalletsDivisionJournal200Ok
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
					"corporationID", arg.CorporationID,
					"divisionID", arg.Section.Division(),
				)
				return nil
			}
			ids := set.Of[int32]()
			for _, e := range newEntries {
				if e.FirstPartyId != 0 {
					ids.Add(e.FirstPartyId)
				}
				if e.SecondPartyId != 0 {
					ids.Add(e.SecondPartyId)
				}
				if e.TaxReceiverId != 0 {
					ids.Add(e.TaxReceiverId)
				}
			}
			_, err = s.eus.AddMissingEntities(ctx, ids)
			if err != nil {
				return err
			}
			for _, o := range newEntries {
				arg := storage.CreateCorporationWalletJournalEntryParams{
					Amount:        o.Amount,
					Balance:       o.Balance,
					ContextID:     o.ContextId,
					ContextIDType: o.ContextIdType,
					Date:          o.Date,
					Description:   o.Description,
					DivisionID:    arg.Section.Division().ID(),
					FirstPartyID:  o.FirstPartyId,
					RefID:         o.Id,
					CorporationID: arg.CorporationID,
					RefType:       o.RefType,
					Reason:        o.Reason,
					SecondPartyID: o.SecondPartyId,
					Tax:           o.Tax,
					TaxReceiverID: o.TaxReceiverId,
				}
				if err := s.st.CreateCorporationWalletJournalEntry(ctx, arg); err != nil {
					return err
				}
			}
			slog.Info(
				"Stored new wallet journal entries",
				"corporationID", arg.CorporationID,
				"divisionID", arg.Section.Division(),
				"entries", len(newEntries),
			)
			return nil
		})
}

const (
	maxTransactionsPerPage = 2_500 // maximum objects returned per page
)

func (s *CorporationService) GetWalletTransactions(ctx context.Context, corporationID int32, division app.Division, transactionID int64) (*app.CorporationWalletTransaction, error) {
	return s.st.GetCorporationWalletTransaction(ctx, storage.GetCorporationWalletTransactionParams{
		CorporationID: corporationID,
		DivisionID:    division.ID(),
		TransactionID: transactionID,
	})
}

func (s *CorporationService) ListWalletTransactions(ctx context.Context, corporationID int32, division app.Division) ([]*app.CorporationWalletTransaction, error) {
	return s.st.ListCorporationWalletTransactions(ctx, storage.CorporationDivision{
		CorporationID: corporationID,
		DivisionID:    division.ID(),
	})
}

// updateWalletTransactionESI updates the wallet journal from ESI and reports whether it has changed.
func (s *CorporationService) updateWalletTransactionESI(ctx context.Context, arg app.CorporationUpdateSectionParams) (bool, error) {
	sections := set.Of(
		app.SectionCorporationWalletTransactions1,
		app.SectionCorporationWalletTransactions2,
		app.SectionCorporationWalletTransactions3,
		app.SectionCorporationWalletTransactions4,
		app.SectionCorporationWalletTransactions5,
		app.SectionCorporationWalletTransactions6,
		app.SectionCorporationWalletTransactions7,
	)
	if arg.CorporationID <= 0 || !sections.Contains(arg.Section) {
		return false, fmt.Errorf("updateWalletTransactionESI %+v: %w", arg, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, arg app.CorporationUpdateSectionParams) (any, error) {
			transactions, err := s.fetchWalletTransactionsESI(ctx, arg)
			if err != nil {
				return false, err
			}
			return transactions, nil
		},
		func(ctx context.Context, arg app.CorporationUpdateSectionParams, data any) error {
			transactions := data.([]esi.GetCorporationsCorporationIdWalletsDivisionTransactions200Ok)
			existingIDs, err := s.st.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
				CorporationID: arg.CorporationID,
				DivisionID:    arg.Section.Division().ID(),
			})
			if err != nil {
				return err
			}
			var newEntries []esi.GetCorporationsCorporationIdWalletsDivisionTransactions200Ok
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
					"corporationID", arg.CorporationID,
					"divisionID", arg.Section.Division(),
				)
				return nil
			}
			var entityIDs, typeIDs set.Set[int32]
			var locationIDs set.Set[int64]
			for _, en := range newEntries {
				if en.ClientId != 0 {
					entityIDs.Add(en.ClientId)
				}
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
				return err
			}
			for _, o := range newEntries {
				arg := storage.CreateCorporationWalletTransactionParams{
					ClientID:      o.ClientId,
					CorporationID: arg.CorporationID,
					Date:          o.Date,
					DivisionID:    arg.Section.Division().ID(),
					EveTypeID:     o.TypeId,
					IsBuy:         o.IsBuy,
					JournalRefID:  o.JournalRefId,
					LocationID:    o.LocationId,
					Quantity:      o.Quantity,
					TransactionID: o.TransactionId,
					UnitPrice:     o.UnitPrice,
				}
				if err := s.st.CreateCorporationWalletTransaction(ctx, arg); err != nil {
					return err
				}
			}
			slog.Info(
				"Stored new wallet transactions",
				"corporationID", arg.CorporationID,
				"divisionID", arg.Section.Division(),
				"entries", len(newEntries),
			)
			return nil
		})
}

// fetchWalletTransactionsESI fetches wallet transactions from ESI with paging and returns them.
func (s *CorporationService) fetchWalletTransactionsESI(ctx context.Context, arg app.CorporationUpdateSectionParams) ([]esi.GetCorporationsCorporationIdWalletsDivisionTransactions200Ok, error) {
	var oo2 []esi.GetCorporationsCorporationIdWalletsDivisionTransactions200Ok
	lastID := int64(0)
	for {
		var opts *esi.GetCorporationsCorporationIdWalletsDivisionTransactionsOpts
		if lastID > 0 {
			opts = &esi.GetCorporationsCorporationIdWalletsDivisionTransactionsOpts{
				FromId: esioptional.NewInt64(lastID),
			}
		} else {
			opts = nil
		}
		oo, _, err := s.esiClient.ESI.WalletApi.GetCorporationsCorporationIdWalletsDivisionTransactions(
			ctx,
			arg.CorporationID,
			arg.Section.Division().ID(),
			opts,
		)
		if err != nil {
			return nil, err
		}
		oo2 = slices.Concat(oo2, oo)
		isLimitExceeded := (arg.MaxWalletTransactions != 0 && len(oo2)+maxTransactionsPerPage > arg.MaxWalletTransactions)
		if len(oo) < maxTransactionsPerPage || isLimitExceeded {
			break
		}
		ids := make([]int64, len(oo))
		for i, o := range oo {
			ids[i] = o.TransactionId
		}
		lastID = slices.Min(ids)
	}
	slog.Debug("Received wallet transactions", "corporationID", arg.CorporationID, "divisionID", arg.Section.Division(), "count", len(oo2))
	return oo2, nil
}
