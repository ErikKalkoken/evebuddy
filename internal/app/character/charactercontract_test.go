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
	const characterID = 7
	earliest := time.Now().UTC().Add(-6 * time.Hour)
	now := time.Now().UTC()
	cases := []struct {
		name           string
		acceptorID     int32
		status         app.ContractStatus
		statusNotified app.ContractStatus
		typ            app.ContractType
		updatedAt      time.Time
		shouldNotify   bool
	}{
		{"notify new courier 1", 42, app.ContractStatusInProgress, app.ContractStatusUndefined, app.ContractTypeCourier, now, true},
		{"notify new courier 2", 42, app.ContractStatusFinished, app.ContractStatusUndefined, app.ContractTypeCourier, now, true},
		{"notify new courier 3", 42, app.ContractStatusFailed, app.ContractStatusUndefined, app.ContractTypeCourier, now, true},
		{"don't notify courier", 0, app.ContractStatusOutstanding, app.ContractStatusUndefined, app.ContractTypeCourier, now, false},
		{"notify new item exchange", 42, app.ContractStatusFinished, app.ContractStatusUndefined, app.ContractTypeItemExchange, now, true},
		{"don't notify again", 42, app.ContractStatusInProgress, app.ContractStatusInProgress, app.ContractTypeCourier, now, false},
		{"don't notify when acceptor is character", characterID, app.ContractStatusInProgress, app.ContractStatusUndefined, app.ContractTypeCourier, now, false},
		{"don't notify when contract is too old", 42, app.ContractStatusInProgress, app.ContractStatusUndefined, app.ContractTypeCourier, now.Add(-12 * time.Hour), false},
		{"don't notify item exchange", 0, app.ContractStatusOutstanding, app.ContractStatusUndefined, app.ContractTypeItemExchange, now, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.TruncateTables(db)
			if tc.acceptorID != 0 {
				factory.CreateEveEntityCharacter(app.EveEntity{ID: tc.acceptorID})
			}
			ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{ID: characterID})
			c := factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{ID: ec.ID})
			o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
				AcceptorID:     tc.acceptorID,
				CharacterID:    c.ID,
				Status:         tc.status,
				StatusNotified: tc.statusNotified,
				Type:           tc.typ,
				UpdatedAt:      tc.updatedAt,
			})
			var sendCount int
			// when
			err := cs.NotifyUpdatedContracts(ctx, o.CharacterID, earliest, func(title string, content string) {
				sendCount++
			})
			// then
			if assert.NoError(t, err) {
				assert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}
