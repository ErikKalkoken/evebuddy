package characterservice

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

func (s *CharacterService) GetWalletJournalEntry(ctx context.Context, characterID int64, refID int64) (*app.CharacterWalletJournalEntry, error) {
	return s.st.GetCharacterWalletJournalEntry(ctx, storage.GetCharacterWalletJournalEntryParams{
		CharacterID: characterID,
		RefID:       refID,
	})
}

func (s *CharacterService) ListWalletJournalEntries(ctx context.Context, characterID int64) ([]*app.CharacterWalletJournalEntry, error) {
	return s.st.ListCharacterWalletJournalEntries(ctx, characterID)
}

// updateWalletJournalEntryESI updates the wallet journal from ESI and reports whether it has changed.
func (s *CharacterService) updateWalletJournalEntryESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterWalletJournal {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdWalletJournal")
			lastID, err := s.st.GetCharacterWalletJournalEntryLastID(ctx, characterID)
			if err != nil {
				return false, err
			}
			checkLastID := lastID > 0 && !arg.forceUpdate
			entries, err := xgoesi.FetchPagesWithStop(
				func(page int32) ([]esi.CharactersCharacterIdWalletJournalGetInner, *http.Response, error) {
					return s.esiClient.WalletAPI.GetCharactersCharacterIdWalletJournal(ctx, characterID).Page(page).Execute()
				}, func(x esi.CharactersCharacterIdWalletJournalGetInner) bool {
					return checkLastID && x.Id <= lastID
				})
			if err != nil {
				return false, err
			}
			slog.Debug("Received wallet journal from ESI", "entries", len(entries), "characterID", characterID)
			return entries, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			entries := data.([]esi.CharactersCharacterIdWalletJournalGetInner)
			existingIDs, err := s.st.ListCharacterWalletJournalEntryIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			slog.Debug("wallet journal", "existing", existingIDs.Size(), "entries", len(entries), "characterID", characterID)
			return walletsync.SyncJournal(
				ctx,
				xslices.Map(entries, func(x esi.CharactersCharacterIdWalletJournalGetInner) walletsync.ESIJournalEntry {
					return walletsync.ESIJournalEntry(x)
				}),
				existingIDs,
				func(ctx context.Context, ids set.Set[int64]) error {
					_, err := s.eus.AddMissingEntities(ctx, ids)
					return err
				},
				func(ctx context.Context, items []walletsync.ESIJournalEntry) error {
					return s.st.CreateCharacterWalletJournalEntries(ctx, xslices.Map(items, func(o walletsync.ESIJournalEntry) storage.CreateCharacterWalletJournalEntryParams {
						return storage.CreateCharacterWalletJournalEntryParams{
							Amount:        optional.FromPtr(o.Amount),
							Balance:       optional.FromPtr(o.Balance),
							ContextID:     optional.FromPtr(o.ContextId),
							ContextIDType: optional.FromPtr(o.ContextIdType),
							Date:          o.Date,
							Description:   o.Description,
							FirstPartyID:  optional.FromPtr(o.FirstPartyId),
							RefID:         o.Id,
							CharacterID:   characterID,
							RefType:       o.RefType,
							Reason:        optional.FromPtr(o.Reason),
							SecondPartyID: optional.FromPtr(o.SecondPartyId),
							Tax:           optional.FromPtr(o.Tax),
							TaxReceiverID: optional.FromPtr(o.TaxReceiverId),
						}
					}))
				},
				slog.Int64("characterID", characterID),
			)
		})
}

func (s *CharacterService) GetWalletTransactions(ctx context.Context, characterID int64, transactionID int64) (*app.CharacterWalletTransaction, error) {
	return s.st.GetCharacterWalletTransaction(ctx, storage.GetCharacterWalletTransactionParams{
		CharacterID:   characterID,
		TransactionID: transactionID,
	})
}

func (s *CharacterService) ListWalletTransactions(ctx context.Context, characterID int64) ([]*app.CharacterWalletTransaction, error) {
	return s.st.ListCharacterWalletTransactions(ctx, characterID)
}

// updateWalletTransactionESI updates the wallet journal from ESI and reports whether it has changed.
func (s *CharacterService) updateWalletTransactionESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterWalletTransactions {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true,
		func(ctx context.Context, characterID int64) (any, error) {
			lastID, err := s.st.GetCharacterWalletTransactionLastID(ctx, characterID)
			if err != nil {
				return false, err
			}
			checkLastID := lastID > 0 && !arg.forceUpdate
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdWalletTransactions")
			transactions, err := walletsync.FetchTransactions(
				func(fromID int64) ([]walletsync.ESITransaction, error) {
					req := s.esiClient.WalletAPI.GetCharactersCharacterIdWalletTransactions(ctx, characterID)
					if fromID > 0 {
						req = req.FromId(fromID)
					}
					oo, _, err := req.Execute()
					if err != nil {
						return nil, err
					}
					return xslices.Map(oo, func(x esi.CharactersCharacterIdWalletTransactionsGetInner) walletsync.ESITransaction {
						return walletsync.ESITransaction(x)
					}), nil
				},
				lastID,
				checkLastID,
			)
			if err != nil {
				return false, err
			}
			slog.Debug("Received wallet transactions", "characterID", characterID, "count", len(transactions))
			return transactions, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			transactions := data.([]walletsync.ESITransaction)
			existingIDs, err := s.st.ListCharacterWalletTransactionIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			slog.Debug("wallet transaction", "existing", existingIDs.Size(), "entries", len(transactions), "characterID", characterID)
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
					return s.st.CreateCharacterWalletTransactions(ctx, xslices.Map(items, func(o walletsync.ESITransaction) storage.CreateCharacterWalletTransactionParams {
						return storage.CreateCharacterWalletTransactionParams{
							ClientID:      o.ClientId,
							Date:          o.Date,
							EveTypeID:     o.TypeId,
							IsBuy:         o.IsBuy,
							IsPersonal:    o.IsPersonal,
							JournalRefID:  o.JournalRefId,
							LocationID:    o.LocationId,
							CharacterID:   characterID,
							Quantity:      o.Quantity,
							TransactionID: o.TransactionId,
							UnitPrice:     o.UnitPrice,
						}
					}))
				},
				slog.Int64("characterID", characterID),
			)
		})
}

func (s *CharacterService) updateWalletBalanceESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterWalletBalance {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, false,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdWallet")
			balance, _, err := s.esiClient.WalletAPI.GetCharactersCharacterIdWallet(ctx, characterID).Execute()
			if err != nil {
				return false, err
			}
			return balance, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			balance := data.(float64)
			if err := s.st.UpdateCharacterWalletBalance(ctx, characterID, optional.New(balance)); err != nil {
				return false, err
			}
			return true, nil
		})
}
