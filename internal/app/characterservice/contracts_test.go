package characterservice_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestNotifyUpdatedContracts(t *testing.T) {
	db, st, f := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})

	const characterID = 7
	earliest := time.Now().UTC().Add(-6 * time.Hour)
	now := time.Now().UTC()
	cases := []struct {
		name           string
		acceptorID     int64
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
			testutil.MustTruncateTables(db)
			if tc.acceptorID != 0 {
				f.CreateEveEntityCharacter(app.EveEntity{ID: tc.acceptorID})
			}
			ec := f.CreateEveCharacter(storage.CreateEveCharacterParams{ID: characterID})
			c := f.CreateCharacterFull(storage.CreateCharacterParams{ID: ec.ID})
			o := f.CreateCharacterContract(storage.CreateCharacterContractParams{
				AcceptorID:     tc.acceptorID,
				CharacterID:    c.ID,
				Status:         tc.status,
				StatusNotified: tc.statusNotified,
				Type:           tc.typ,
				UpdatedAt:      tc.updatedAt,
			})
			var sendCount int
			// when
			err := s.NotifyUpdatedContracts(t.Context(), o.CharacterID, earliest, func(title string, content string) {
				sendCount++
			})
			// then
			if assert.NoError(t, err) {
				xassert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}

func TestListAllCharacterContractSlotsPersonal(t *testing.T) {
	db, st, f := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})

	cases := []struct {
		name              string
		contractingLevel  int64
		advancedLevel     int64
		personalContracts int
		wantUsed          int
		wantFree          int
		wantTotal         int
	}{
		{
			name:              "contracing and open contracts",
			contractingLevel:  3,
			advancedLevel:     0,
			personalContracts: 2,
			wantUsed:          2,
			wantFree:          11,
			wantTotal:         13,
		},
		{
			name:              "contracing and no contracts",
			contractingLevel:  3,
			advancedLevel:     0,
			personalContracts: 0,
			wantUsed:          0,
			wantFree:          13,
			wantTotal:         13,
		},
		{
			name:              "no skills ",
			contractingLevel:  0,
			advancedLevel:     0,
			personalContracts: 1,
			wantUsed:          1,
			wantFree:          0,
			wantTotal:         1,
		},
		{
			name:              "no skills and no contracts",
			contractingLevel:  0,
			advancedLevel:     0,
			personalContracts: 0,
			wantUsed:          0,
			wantFree:          1,
			wantTotal:         1,
		},
		{
			name:              "advanced contracing and open contracts",
			contractingLevel:  5,
			advancedLevel:     2,
			personalContracts: 2,
			wantUsed:          2,
			wantFree:          27,
			wantTotal:         29,
		},
		{
			name:              "advanced contracing and no contracts",
			contractingLevel:  5,
			advancedLevel:     2,
			personalContracts: 0,
			wantUsed:          0,
			wantFree:          29,
			wantTotal:         29,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			c := f.CreateCharacter()
			if tc.contractingLevel > 0 {
				skill := f.CreateEveType(storage.CreateEveTypeParams{
					ID: app.EveTypeContracting,
				})
				f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
					CharacterID:      c.ID,
					TypeID:           skill.ID,
					ActiveSkillLevel: tc.contractingLevel,
				})
			}
			if tc.advancedLevel > 0 {
				skill := f.CreateEveType(storage.CreateEveTypeParams{
					ID: app.EveTypeAdvancedContracting,
				})
				f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
					CharacterID:      c.ID,
					TypeID:           skill.ID,
					ActiveSkillLevel: tc.advancedLevel,
				})
			}
			for range tc.personalContracts {
				f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
					CharacterID:    c.ID,
					ForCorporation: false,
				})
			}

			// given - contracts to ignore
			issuer := f.CreateEveEntityCharacter()
			issuerCorporation := f.CreateEveEntityCorporation()
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID:         c.ID,
				IssuerID:            issuer.ID,
				IssuerCorporationID: issuerCorporation.ID,
				ForCorporation:      false,
			})
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID:    c.ID,
				ForCorporation: true,
			})
			acceptor := f.CreateEveEntityCharacter()
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID: c.ID,
				Status:      app.ContractStatusInProgress,
				AcceptorID:  acceptor.ID,
			})
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID:    c.ID,
				ForCorporation: false,
				AssigneeID:     c.EveCharacter.Corporation.ID,
			})

			// when
			oo, err := s.ListAllCharacterContractSlotsPersonal(t.Context())

			// then
			require.NoError(t, err)
			require.Len(t, oo, 1)
			got := oo[0]
			want := app.CharacterContractSlots{
				CharacterID:     c.ID,
				CharacterName:   c.EveCharacter.Name,
				CorporationID:   c.EveCharacter.Corporation.ID,
				CorporationName: c.EveCharacter.Corporation.Name,
				Free:            tc.wantFree,
				Used:            tc.wantUsed,
				Total:           tc.wantTotal,
				IsCorporation:   false,
			}
			xassert.Equal(t, want, got)
		})
	}
}

func TestListAllCharacterContractSlotsCorporation(t *testing.T) {
	db, st, f := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})

	cases := []struct {
		name             string
		contractingLevel int64
		activeContracts  int
		wantUsed         int
		wantFree         int
		wantTotal        int
	}{
		{
			name:             "contracing and open contracts",
			contractingLevel: 3,
			activeContracts:  2,
			wantUsed:         2,
			wantFree:         38,
			wantTotal:        40,
		},
		{
			name:             "contracing and no contracts",
			contractingLevel: 3,
			activeContracts:  0,
			wantUsed:         0,
			wantFree:         40,
			wantTotal:        40,
		},
		{
			name:             "no skills",
			contractingLevel: 0,
			activeContracts:  1,
			wantUsed:         1,
			wantFree:         9,
			wantTotal:        10,
		},
		{
			name:             "no skills and no contracts",
			contractingLevel: 0,
			activeContracts:  0,
			wantUsed:         0,
			wantFree:         10,
			wantTotal:        10,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			c := f.CreateCharacter()
			if tc.contractingLevel > 0 {
				skill := f.CreateEveType(storage.CreateEveTypeParams{
					ID: app.EveTypeCorporationContracting,
				})
				f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
					CharacterID:      c.ID,
					TypeID:           skill.ID,
					ActiveSkillLevel: tc.contractingLevel,
				})
			}
			for range tc.activeContracts {
				f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
					CharacterID:    c.ID,
					ForCorporation: true,
				})
			}
			// given - contracts to ignore
			issuer := f.CreateEveEntityCharacter()
			issuerCorporation := f.CreateEveEntityCorporation()
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID:         c.ID,
				IssuerID:            issuer.ID,
				IssuerCorporationID: issuerCorporation.ID,
				ForCorporation:      true,
			})
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID:    c.ID,
				ForCorporation: false,
			})
			acceptor := f.CreateEveEntityCharacter()
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID:    c.ID,
				Status:         app.ContractStatusInProgress,
				AcceptorID:     acceptor.ID,
				ForCorporation: true,
			})

			// when
			oo, err := s.ListAllCharacterContractSlotsCorporation(t.Context())

			// then
			require.NoError(t, err)
			require.Len(t, oo, 1)
			got := oo[0]
			want := app.CharacterContractSlots{
				CharacterID:     c.ID,
				CharacterName:   c.EveCharacter.Name,
				CorporationID:   c.EveCharacter.Corporation.ID,
				CorporationName: c.EveCharacter.Corporation.Name,
				Free:            tc.wantFree,
				Used:            tc.wantUsed,
				Total:           tc.wantTotal,
				IsCorporation:   true,
			}
			xassert.Equal(t, want, got)
		})
	}
}
