package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/stretchr/testify/assert"
)

func TestCharacterContract(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		issuer := factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		issuerCorporation := c.EveCharacter.Corporation
		dateExpired := time.Now().Add(12 * time.Hour).UTC()
		dateIssued := time.Now().UTC()
		arg := storage.CreateCharacterContractParams{
			Availability:        app.AvailabilityPersonal,
			CharacterID:         c.ID,
			ContractID:          42,
			DateExpired:         dateExpired,
			DateIssued:          dateIssued,
			IssuerCorporationID: issuerCorporation.ID,
			IssuerID:            issuer.ID,
			Status:              app.StatusOutstanding,
			Type:                app.TypeCourier,
		}
		// when
		id, err := r.CreateCharacterContract(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := r.GetCharacterContract(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, id, o.ID)
				assert.Equal(t, issuer, o.Issuer)
				assert.Equal(t, dateExpired, o.DateExpired)
				assert.Equal(t, app.AvailabilityPersonal, o.Availability)
				assert.Equal(t, app.StatusOutstanding, o.Status)
				assert.Equal(t, app.TypeCourier, o.Type)
			}
		}
	})
	t.Run("can update contract", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		dateAccepted := optional.New(time.Now().UTC())
		dateCompleted := optional.New(time.Now().UTC())
		arg2 := storage.UpdateCharacterContractParams{
			CharacterID:   o.CharacterID,
			ContractID:    o.ContractID,
			DateAccepted:  dateAccepted,
			DateCompleted: dateCompleted,
			Status:        app.StatusFinished,
		}
		// when
		err := r.UpdateCharacterContract(ctx, arg2)
		// then
		if assert.NoError(t, err) {
			o, err := r.GetCharacterContract(ctx, o.CharacterID, o.ContractID)
			if assert.NoError(t, err) {
				assert.Equal(t, app.StatusFinished, o.Status)
				assert.Equal(t, dateAccepted, o.DateAccepted)
				assert.Equal(t, dateCompleted, o.DateCompleted)
			}
		}
	})
	t.Run("can list IDs of existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		e1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		e2 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		e3 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		// when
		ids, err := r.ListCharacterContractIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.NewFromSlice(ids)
			want := set.NewFromSlice([]int32{e1.ContractID, e2.ContractID, e3.ContractID})
			assert.Equal(t, want, got)
		}
	})
	t.Run("can list existing contracts", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		// when
		oo, err := r.ListCharacterContracts(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 3)
		}
	})
}
