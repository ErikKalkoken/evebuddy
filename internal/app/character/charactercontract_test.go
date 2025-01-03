package character_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNotifyUpdatedContracts(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := newCharacterService(st)
	ctx := context.Background()
	t.Run("send notification when contract has relevant status and it was not notified", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
			CharacterID: c.ID,
			Status:      app.ContractStatusInProgress,
		})
		earliest := time.Now().Add(-24 * time.Hour)
		var sendCount int
		// when
		err := cs.NotifyUpdatedContracts(ctx, c.ID, earliest, func(title string, content string) {
			sendCount++
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, sendCount, 1)
		}
	})
	t.Run("send no notification when contract has relevant status and was already notified", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
			CharacterID:    c.ID,
			Status:         app.ContractStatusInProgress,
			StatusNotified: app.ContractStatusInProgress,
		})
		earliest := time.Now().Add(-24 * time.Hour)
		var sendCount int
		// when
		err := cs.NotifyUpdatedContracts(ctx, c.ID, earliest, func(title string, content string) {
			sendCount++
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, sendCount, 0)
		}
	})
}
