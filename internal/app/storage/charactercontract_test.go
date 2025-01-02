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
			Availability:        "personal",
			CharacterID:         c.ID,
			ContractID:          42,
			DateExpired:         dateExpired,
			DateIssued:          dateIssued,
			IssuerCorporationID: issuerCorporation.ID,
			IssuerID:            issuer.ID,
			Status:              "outstanding",
			Type:                "courier",
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
				assert.Equal(t, app.ContractAvailabilityPersonal, o.Availability)
				assert.Equal(t, app.ContractStatusOutstanding, o.Status)
				assert.Equal(t, app.ContractTypeCourier, o.Type)
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		issuer := factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		issuerCorporation := c.EveCharacter.Corporation
		dateExpired := time.Now().Add(12 * time.Hour).UTC()
		dateIssued := time.Now().UTC()
		startLocation := factory.CreateLocationStructure()
		endLocation := factory.CreateLocationStructure()
		arg := storage.CreateCharacterContractParams{
			Availability:        "personal",
			CharacterID:         c.ID,
			ContractID:          42,
			DateExpired:         dateExpired,
			DateIssued:          dateIssued,
			IssuerCorporationID: issuerCorporation.ID,
			IssuerID:            issuer.ID,
			Status:              "outstanding",
			Type:                "courier",
			EndLocationID:       endLocation.ID,
			StartLocationID:     startLocation.ID,
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
				assert.Equal(t, app.ContractAvailabilityPersonal, o.Availability)
				assert.Equal(t, app.ContractStatusOutstanding, o.Status)
				assert.Equal(t, app.ContractTypeCourier, o.Type)
				assert.Equal(t, endLocation.ID, o.EndLocation.ID)
				assert.Equal(t, endLocation.Name, o.EndLocation.Name)
				assert.Equal(t, startLocation.ID, o.StartLocation.ID)
				assert.Equal(t, startLocation.Name, o.StartLocation.Name)
				assert.Equal(t, endLocation.SolarSystem.ID, o.EndSolarSystem.ID)
				assert.Equal(t, endLocation.SolarSystem.Name, o.EndSolarSystem.Name)
				assert.Equal(t, startLocation.SolarSystem.ID, o.StartSolarSystem.ID)
				assert.Equal(t, startLocation.SolarSystem.Name, o.StartSolarSystem.Name)
			}
		}
	})
	t.Run("can update contract", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		dateAccepted := time.Now().UTC()
		dateCompleted := time.Now().UTC()
		arg2 := storage.UpdateCharacterContractParams{
			CharacterID:   o.CharacterID,
			ContractID:    o.ContractID,
			DateAccepted:  dateAccepted,
			DateCompleted: dateCompleted,
			Status:        "finished",
		}
		// when
		err := r.UpdateCharacterContract(ctx, arg2)
		// then
		if assert.NoError(t, err) {
			o, err := r.GetCharacterContract(ctx, o.CharacterID, o.ContractID)
			if assert.NoError(t, err) {
				assert.Equal(t, app.ContractStatusFinished, o.Status)
				assert.Equal(t, optional.New(dateAccepted), o.DateAccepted)
				assert.Equal(t, optional.New(dateCompleted), o.DateCompleted)
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
