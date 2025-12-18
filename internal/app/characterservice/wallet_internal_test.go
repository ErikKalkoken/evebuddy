package characterservice

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestUpdateWalletJournalEntryESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should create new entry from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		firstParty := factory.CreateEveEntityCharacter(app.EveEntity{ID: 2112625428})
		secondParty := factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000132})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/wallet/journal/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"amount":          -100000,
				"balance":         500000.4316,
				"context_id":      4,
				"context_id_type": "contract_id",
				"date":            "2018-02-23T14:31:32Z",
				"description":     "Contract Deposit",
				"first_party_id":  2112625428,
				"id":              89,
				"ref_type":        "contract_deposit",
				"second_party_id": 1000132,
			}}),
		)
		// when
		changed, err := s.updateWalletJournalEntryESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		x, err := st.GetCharacterWalletJournalEntry(ctx, storage.GetCharacterWalletJournalEntryParams{
			CharacterID: c.ID,
			RefID:       89,
		})
		require.NoError(t, err)
		assert.Equal(t, -100000.0, x.Amount)
		assert.Equal(t, 500000.4316, x.Balance)
		assert.Equal(t, int64(4), x.ContextID)
		assert.Equal(t, "contract_id", x.ContextIDType)
		assert.Equal(t, time.Date(2018, 02, 23, 14, 31, 32, 0, time.UTC), x.Date)
		assert.Equal(t, "Contract Deposit", x.Description)
		assert.Equal(t, firstParty.ID, x.FirstParty.ID)
		assert.Equal(t, "contract_deposit", x.RefType)
		assert.Equal(t, secondParty.ID, x.SecondParty.ID)
		ids, err := st.ListCharacterWalletJournalEntryIDs(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, ids.Size())

	})
	t.Run("should add new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 2112625428})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000132})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/wallet/journal/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"amount":          -100000,
				"balance":         500000.4316,
				"context_id":      4,
				"context_id_type": "contract_id",
				"date":            "2018-02-23T14:31:32Z",
				"description":     "Contract Deposit",
				"first_party_id":  2112625428,
				"id":              89,
				"ref_type":        "contract_deposit",
				"second_party_id": 1000132,
			}}),
		)
		// when
		changed, err := s.updateWalletJournalEntryESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		x, err := st.GetCharacterWalletJournalEntry(ctx, storage.GetCharacterWalletJournalEntryParams{
			CharacterID: c.ID,
			RefID:       89,
		})
		require.NoError(t, err)
		assert.Equal(t, "Contract Deposit", x.Description)
		ids, err := st.ListCharacterWalletJournalEntryIDs(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, ids.Size())
	})
	t.Run("should ignore existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			CharacterID: c.ID,
			RefID:       89,
			Description: "existing",
		})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 2112625428})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000132})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/wallet/journal/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"amount":          -100000,
				"balance":         500000.4316,
				"context_id":      4,
				"context_id_type": "contract_id",
				"date":            "2018-02-23T14:31:32Z",
				"description":     "Contract Deposit",
				"first_party_id":  2112625428,
				"id":              89,
				"ref_type":        "contract_deposit",
				"second_party_id": 1000132,
			}}),
		)
		// when
		_, err := s.updateWalletJournalEntryESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		// then
		require.NoError(t, err)
		x, err := st.GetCharacterWalletJournalEntry(ctx, storage.GetCharacterWalletJournalEntryParams{
			CharacterID: c.ID,
			RefID:       89,
		})
		require.NoError(t, err)
		assert.Equal(t, "existing", x.Description)
		ids, err := st.ListCharacterWalletJournalEntryIDs(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, ids.Size())
	})
	t.Run("should handle empty response", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/wallet/journal/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
		)
		// when
		changed, err := s.updateWalletJournalEntryESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		ids, err := st.ListCharacterWalletJournalEntryIDs(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, ids.Size())

	})
	t.Run("should fetch multiple pages", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 2112625428})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000132})
		pages := "2"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/wallet/journal/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"amount":          -100000,
				"balance":         500000.4316,
				"context_id":      4,
				"context_id_type": "contract_id",
				"date":            "2018-02-23T14:31:32Z",
				"description":     "First",
				"first_party_id":  2112625428,
				"id":              89,
				"ref_type":        "contract_deposit",
				"second_party_id": 1000132,
			}}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/wallet/journal/?page=2", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"amount":          -110000,
				"balance":         500000.4316,
				"context_id":      4,
				"context_id_type": "contract_id",
				"date":            "2018-02-23T15:31:32Z",
				"description":     "Second",
				"first_party_id":  2112625428,
				"id":              90,
				"ref_type":        "contract_deposit",
				"second_party_id": 1000132,
			}}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		// when
		changed, err := s.updateWalletJournalEntryESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		ids, err := st.ListCharacterWalletJournalEntryIDs(ctx, c.ID)
		require.NoError(t, err)
		if assert.Equal(t, 2, ids.Size()) {
			x1, err := st.GetCharacterWalletJournalEntry(ctx, storage.GetCharacterWalletJournalEntryParams{
				CharacterID: c.ID,
				RefID:       89,
			})
			require.NoError(t, err)
			assert.Equal(t, "First", x1.Description)
		}
		x2, err := st.GetCharacterWalletJournalEntry(ctx, storage.GetCharacterWalletJournalEntryParams{
			CharacterID: c.ID,
			RefID:       90,
		})
		require.NoError(t, err)
		assert.Equal(t, "Second", x2.Description)
		assert.Equal(t, 2, httpmock.GetTotalCallCount())
	})
	t.Run("should fetch first page only when entries already fetched previouly", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 2112625428})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000132})
		pages := "2"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/wallet/journal/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"amount":          -100000,
				"balance":         500000.4316,
				"context_id":      4,
				"context_id_type": "contract_id",
				"date":            "2018-02-23T14:31:32Z",
				"description":     "First",
				"first_party_id":  2112625428,
				"id":              89,
				"ref_type":        "contract_deposit",
				"second_party_id": 1000132,
			}}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/wallet/journal/?page=2", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"amount":          -110000,
				"balance":         500000.4316,
				"context_id":      4,
				"context_id_type": "contract_id",
				"date":            "2018-02-23T15:31:32Z",
				"description":     "Second",
				"first_party_id":  2112625428,
				"id":              90,
				"ref_type":        "contract_deposit",
				"second_party_id": 1000132,
			}}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		_, err := s.updateWalletJournalEntryESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		require.NoError(t, err) // when
		httpmock.ZeroCallCounters()
		// when
		_, err = s.updateWalletJournalEntryESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		// then
		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
	})
	t.Run("should always fetch all pages when forced", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 2112625428})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000132})
		pages := "2"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/wallet/journal/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"amount":          -100000,
				"balance":         500000.4316,
				"context_id":      4,
				"context_id_type": "contract_id",
				"date":            "2018-02-23T14:31:32Z",
				"description":     "First",
				"first_party_id":  2112625428,
				"id":              89,
				"ref_type":        "contract_deposit",
				"second_party_id": 1000132,
			}}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v5/characters/%d/wallet/journal/?page=2", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"amount":          -110000,
				"balance":         500000.4316,
				"context_id":      4,
				"context_id_type": "contract_id",
				"date":            "2018-02-23T15:31:32Z",
				"description":     "Second",
				"first_party_id":  2112625428,
				"id":              90,
				"ref_type":        "contract_deposit",
				"second_party_id": 1000132,
			}}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		_, err := s.updateWalletJournalEntryESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
		})
		require.NoError(t, err) // when
		httpmock.ZeroCallCounters()
		// when
		_, err = s.updateWalletJournalEntryESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletJournal,
			ForceUpdate: true,
		})
		// then
		require.NoError(t, err)
		assert.Equal(t, 2, httpmock.GetTotalCallCount())
	})
}

func TestListWalletJournalEntries(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		// when
		ee, err := s.ListWalletJournalEntries(ctx, c.ID)
		// then
		require.NoError(t, err)
		assert.Len(t, ee, 3)
	})
}

func TestUpdateWalletTransactionESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should create new transaction from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		client := factory.CreateEveEntityCharacter(app.EveEntity{ID: 54321})
		location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60014719})
		eveType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"client_id":      54321,
				"date":           "2016-10-24T09:00:00Z",
				"is_buy":         true,
				"is_personal":    true,
				"journal_ref_id": 67890,
				"location_id":    60014719,
				"quantity":       1,
				"transaction_id": 1234567890,
				"type_id":        587,
				"unit_price":     1.23,
			}}),
		)
		// when
		changed, err := s.updateWalletTransactionESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		e, err := st.GetCharacterWalletTransaction(ctx, storage.GetCharacterWalletTransactionParams{
			CharacterID:   c.ID,
			TransactionID: 1234567890,
		})
		require.NoError(t, err)
		assert.Equal(t, client, e.Client)
		assert.Equal(t, time.Date(2016, 10, 24, 9, 0, 0, 0, time.UTC), e.Date)
		assert.True(t, e.IsBuy)
		assert.True(t, e.IsPersonal)
		assert.Equal(t, location.ID, e.Location.ID)
		assert.Equal(t, int64(67890), e.JournalRefID)
		assert.Equal(t, int32(1), e.Quantity)
		assert.Equal(t, eveType.ID, e.Type.ID)
		assert.Equal(t, 1.23, e.UnitPrice)
		ids, err := st.ListCharacterWalletTransactionIDs(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, ids.Size())
	})
	t.Run("should add new transaction", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		client := factory.CreateEveEntityCharacter(app.EveEntity{ID: 54321})
		location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60014719})
		eveType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"client_id":      54321,
				"date":           "2016-10-24T09:00:00Z",
				"is_buy":         true,
				"is_personal":    true,
				"journal_ref_id": 67890,
				"location_id":    60014719,
				"quantity":       1,
				"transaction_id": 1234567890,
				"type_id":        587,
				"unit_price":     1.23,
			}}),
		)
		// when
		changed, err := s.updateWalletTransactionESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		e, err := st.GetCharacterWalletTransaction(ctx, storage.GetCharacterWalletTransactionParams{
			CharacterID:   c.ID,
			TransactionID: 1234567890,
		})
		require.NoError(t, err)
		assert.Equal(t, client, e.Client)
		assert.Equal(t, time.Date(2016, 10, 24, 9, 0, 0, 0, time.UTC), e.Date)
		assert.True(t, e.IsBuy)
		assert.True(t, e.IsPersonal)
		assert.Equal(t, location.ID, e.Location.ID)
		assert.Equal(t, int64(67890), e.JournalRefID)
		assert.Equal(t, int32(1), e.Quantity)
		assert.Equal(t, eveType.ID, e.Type.ID)
		assert.Equal(t, 1.23, e.UnitPrice)
		ids, err := st.ListCharacterWalletTransactionIDs(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, ids.Size())
	})
	t.Run("should ignore when transaction already exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{
			CharacterID:   c.ID,
			TransactionID: 1234567890,
		})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"client_id":      54321,
				"date":           "2016-10-24T09:00:00Z",
				"is_buy":         true,
				"is_personal":    true,
				"journal_ref_id": 67890,
				"location_id":    60014719,
				"quantity":       1,
				"transaction_id": 1234567890,
				"type_id":        587,
				"unit_price":     1.23,
			}}),
		)
		// when
		_, err := s.updateWalletTransactionESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		// then
		require.NoError(t, err)
		ids, err := st.ListCharacterWalletTransactionIDs(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, ids.Size())
	})
	t.Run("should handle empty response", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
		)
		// when
		changed, err := s.updateWalletTransactionESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		ids, err := st.ListCharacterWalletTransactionIDs(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, ids.Size())
	})
	t.Run("should fetch multiple pages", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 54321})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60014719})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})
		data := make([]map[string]any, 0)
		for i := range 2500 {
			data = append(data, map[string]any{
				"client_id":      54321,
				"date":           "2016-10-24T09:00:00Z",
				"is_buy":         true,
				"is_personal":    true,
				"journal_ref_id": 67890,
				"location_id":    60014719,
				"quantity":       1,
				"transaction_id": 1000002500 - i,
				"type_id":        587,
				"unit_price":     1.23,
			})
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/?from_id=1000000001", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"client_id":      54321,
				"date":           "2016-10-24T08:00:00Z",
				"is_buy":         true,
				"is_personal":    true,
				"journal_ref_id": 67891,
				"location_id":    60014719,
				"quantity":       1,
				"transaction_id": 1000000000,
				"type_id":        587,
				"unit_price":     9.23,
			}}),
		)
		// when
		_, err := s.updateWalletTransactionESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		// then
		require.NoError(t, err)
		ids, err := st.ListCharacterWalletTransactionIDs(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, 2501, ids.Size())
		assert.Equal(t, 2, httpmock.GetTotalCallCount())
	})
	t.Run("should fetch only first page when transactions already fetched previously", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 54321})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60014719})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})
		data := make([]map[string]any, 0)
		for i := range 2500 {
			data = append(data, map[string]any{
				"client_id":      54321,
				"date":           "2016-10-24T09:00:00Z",
				"is_buy":         true,
				"is_personal":    true,
				"journal_ref_id": 67890,
				"location_id":    60014719,
				"quantity":       1,
				"transaction_id": 1000002500 - i,
				"type_id":        587,
				"unit_price":     1.23,
			})
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/?from_id=1000000001", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"client_id":      54321,
				"date":           "2016-10-24T08:00:00Z",
				"is_buy":         true,
				"is_personal":    true,
				"journal_ref_id": 67891,
				"location_id":    60014719,
				"quantity":       1,
				"transaction_id": 1000000000,
				"type_id":        587,
				"unit_price":     9.23,
			}}),
		)
		_, err := s.updateWalletTransactionESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		require.NoError(t, err)
		httpmock.ZeroCallCounters()
		// when
		_, err = s.updateWalletTransactionESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		// then
		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
	})
	t.Run("should fetch all pages when forced", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 54321})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60014719})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})
		data := make([]map[string]any, 0)
		for i := range 2500 {
			data = append(data, map[string]any{
				"client_id":      54321,
				"date":           "2016-10-24T09:00:00Z",
				"is_buy":         true,
				"is_personal":    true,
				"journal_ref_id": 67890,
				"location_id":    60014719,
				"quantity":       1,
				"transaction_id": 1000002500 - i,
				"type_id":        587,
				"unit_price":     1.23,
			})
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/?from_id=1000000001", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"client_id":      54321,
				"date":           "2016-10-24T08:00:00Z",
				"is_buy":         true,
				"is_personal":    true,
				"journal_ref_id": 67891,
				"location_id":    60014719,
				"quantity":       1,
				"transaction_id": 1000000000,
				"type_id":        587,
				"unit_price":     9.23,
			}}),
		)
		_, err := s.updateWalletTransactionESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		require.NoError(t, err)
		httpmock.ZeroCallCounters()
		// when
		_, err = s.updateWalletTransactionESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterWalletTransactions,
			ForceUpdate: true,
		})
		// then
		require.NoError(t, err)
		assert.Equal(t, 2, httpmock.GetTotalCallCount())
	})
}

func TestListWalletTransactions(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		// when
		tt, err := s.ListWalletTransactions(ctx, c.ID)
		// then
		require.NoError(t, err)
		assert.Len(t, tt, 3)
	})
}

// func TestLoadCacheInt64(t *testing.T) {
// 	t.Run("returns value when key exists and is valid", func(t *testing.T) {
// 		cache := testutil.NewCacheFake()
// 		key := "session_id"
// 		val := int64(12345)

// 		// Manually prime the fake
// 		data, _ := json.Marshal(val)
// 		cache.Set(key, data, 0)

// 		result, ok := loadCacheInt64(cache, key)

// 		assert.True(t, ok)
// 		assert.Equal(t, val, result)
// 	})

// 	t.Run("returns false when key does not exist", func(t *testing.T) {
// 		cache := testutil.NewCacheFake()

// 		result, ok := loadCacheInt64(cache, "missing_key")

// 		assert.False(t, ok)
// 		assert.Zero(t, result)
// 	})

// 	t.Run("returns false and logs error on invalid JSON", func(t *testing.T) {
// 		cache := testutil.NewCacheFake()
// 		key := "bad_data"
// 		cache.Set(key, []byte("not-an-integer"), 0)

// 		result, ok := loadCacheInt64(cache, key)

// 		assert.False(t, ok)
// 		assert.Zero(t, result)
// 	})
// }
