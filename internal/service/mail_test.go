package service_test

import (
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"example/evebuddy/internal/helper/testutil"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/service"
)

func TestSendMail(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	// ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := service.NewService(r)
	t.Run("Can send mail", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateMyCharacter()
		factory.CreateToken(model.Token{CharacterID: c.ID})
		r := factory.CreateEveEntityCharacter(model.EveEntity{ID: 90000001})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/", c.ID),
			httpmock.NewStringResponder(201, "123"))

		// when
		mailID, err := s.SendMail(c.ID, "subject", []*model.EveEntity{r}, "body")
		// then
		if assert.NoError(t, err) {
			m, err := s.GetMail(c.ID, mailID)
			if assert.NoError(t, err) {
				assert.Equal(t, "body", m.Body)
			}
		}
	})
}
