package character

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestUpdateWalletTransactionESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := New(r, nil, nil, nil, nil, nil)
	ctx := context.Background()
	t.Run("should create new transaction from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		client := factory.CreateEveEntityCharacter(model.EveEntity{ID: 54321})
		location := factory.CreateLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60014719})
		eveType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})
		data := `[
			{
			  "client_id": 54321,
			  "date": "2016-10-24T09:00:00Z",
			  "is_buy": true,
			  "is_personal": true,
			  "journal_ref_id": 67890,
			  "location_id": 60014719,
			  "quantity": 1,
			  "transaction_id": 1234567890,
			  "type_id": 587,
			  "unit_price": 1.23
			}
		  ]`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/", c.ID),
			httpmock.NewStringResponder(200, data).HeaderSet(
				http.Header{"Content-Type": []string{"application/json"}}))

		// when
		changed, err := s.updateCharacterWalletTransactionESI(ctx, UpdateCharacterSectionParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionWalletTransactions,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			e, err := r.GetCharacterWalletTransaction(ctx, c.ID, 1234567890)
			if assert.NoError(t, err) {
				assert.Equal(t, client, e.Client)
				assert.Equal(t, time.Date(2016, 10, 24, 9, 0, 0, 0, time.UTC), e.Date)
				assert.True(t, e.IsBuy)
				assert.True(t, e.IsPersonal)
				assert.Equal(t, location.ID, e.Location.ID)
				assert.Equal(t, int64(67890), e.JournalRefID)
				assert.Equal(t, int32(1), e.Quantity)
				assert.Equal(t, eveType.ID, e.EveType.ID)
				assert.Equal(t, 1.23, e.UnitPrice)
			}
			ids, err := r.ListCharacterWalletTransactionIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids, 1)
			}
		}
	})
	t.Run("should add new transaction", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		client := factory.CreateEveEntityCharacter(model.EveEntity{ID: 54321})
		location := factory.CreateLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60014719})
		eveType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})
		data := `[
			{
			  "client_id": 54321,
			  "date": "2016-10-24T09:00:00Z",
			  "is_buy": true,
			  "is_personal": true,
			  "journal_ref_id": 67890,
			  "location_id": 60014719,
			  "quantity": 1,
			  "transaction_id": 1234567890,
			  "type_id": 587,
			  "unit_price": 1.23
			}
		  ]`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/", c.ID),
			httpmock.NewStringResponder(200, data).HeaderSet(
				http.Header{"Content-Type": []string{"application/json"}}))

		// when
		changed, err := s.updateCharacterWalletTransactionESI(ctx, UpdateCharacterSectionParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionWalletTransactions,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			e, err := r.GetCharacterWalletTransaction(ctx, c.ID, 1234567890)
			if assert.NoError(t, err) {
				assert.Equal(t, client, e.Client)
				assert.Equal(t, time.Date(2016, 10, 24, 9, 0, 0, 0, time.UTC), e.Date)
				assert.True(t, e.IsBuy)
				assert.True(t, e.IsPersonal)
				assert.Equal(t, location.ID, e.Location.ID)
				assert.Equal(t, int64(67890), e.JournalRefID)
				assert.Equal(t, int32(1), e.Quantity)
				assert.Equal(t, eveType.ID, e.EveType.ID)
				assert.Equal(t, 1.23, e.UnitPrice)
			}
			ids, err := r.ListCharacterWalletTransactionIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids, 2)
			}
		}
	})
	t.Run("should ignore when transaction already exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{
			CharacterID:   c.ID,
			TransactionID: 1234567890,
		})
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		data := `[
			{
			  "client_id": 54321,
			  "date": "2016-10-24T09:00:00Z",
			  "is_buy": true,
			  "is_personal": true,
			  "journal_ref_id": 67890,
			  "location_id": 60014719,
			  "quantity": 1,
			  "transaction_id": 1234567890,
			  "type_id": 587,
			  "unit_price": 1.23
			}
		  ]`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/", c.ID),
			httpmock.NewStringResponder(200, data).HeaderSet(
				http.Header{"Content-Type": []string{"application/json"}}))

		// when
		_, err := s.updateCharacterWalletTransactionESI(ctx, UpdateCharacterSectionParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionWalletTransactions,
		})
		// then
		if assert.NoError(t, err) {
			ids, err := r.ListCharacterWalletTransactionIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids, 1)
			}
		}
	})
	t.Run("should fetch multiple pages", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(model.EveEntity{ID: 54321})
		factory.CreateLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60014719})
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
			httpmock.NewJsonResponderOrPanic(200, data))
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/wallet/transactions/?from_id=1000000001", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
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
				},
			}))
		// when
		_, err := s.updateCharacterWalletTransactionESI(ctx, UpdateCharacterSectionParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionWalletTransactions,
		})
		// then
		if assert.NoError(t, err) {
			ids, err := r.ListCharacterWalletTransactionIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids, 2501)
			}
		}
	})
}

func TestListWalletTransactions(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	s := New(r, nil, nil, nil, nil, nil)
	ctx := context.Background()
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: c.ID})
		// when
		tt, err := s.ListCharacterWalletTransactions(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, tt, 3)
		}
	})
}
