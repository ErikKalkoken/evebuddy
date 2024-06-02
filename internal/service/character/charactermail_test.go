package character_test

import (
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service/character"
)

func TestSendMail(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	// ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := character.New(r, nil, nil, nil, nil, nil)
	t.Run("Can send mail", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		r := factory.CreateEveEntityCharacter(model.EveEntity{ID: 90000001})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/", c.ID),
			httpmock.NewStringResponder(201, "123"))

		// when
		mailID, err := s.SendCharacterMail(c.ID, "subject", []*model.EveEntity{r}, "body")
		// then
		if assert.NoError(t, err) {
			m, err := s.GetCharacterMail(c.ID, mailID)
			if assert.NoError(t, err) {
				assert.Equal(t, "body", m.Body)
			}
		}
	})
}
