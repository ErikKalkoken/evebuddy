package service_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
)

func TestUpdateSkillqueueESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := service.NewService(r)
	ctx := context.Background()
	t.Run("should create new queue", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateMyCharacter()
		factory.CreateToken(model.Token{CharacterID: c.ID})
		factory.CreateEveType(model.EveType{ID: 100})
		factory.CreateEveType(model.EveType{ID: 101})
		data := `[
			{
			  "finish_date": "2016-06-29T10:47:00Z",
			  "finished_level": 3,
			  "queue_position": 0,
			  "skill_id": 100,
			  "start_date": "2016-06-29T10:46:00Z"
			},
			{
			  "finish_date": "2016-07-15T10:47:00Z",
			  "finished_level": 4,
			  "queue_position": 1,
			  "skill_id": 100,
			  "start_date": "2016-06-29T10:47:00Z"
			},
			{
			  "finish_date": "2016-08-30T10:47:00Z",
			  "finished_level": 2,
			  "queue_position": 2,
			  "skill_id": 101,
			  "start_date": "2016-07-15T10:47:00Z"
			}
		  ]`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/characters/%d/skillqueue/", c.ID),
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		err := s.UpdateSkillqueueESI(c.ID)
		// then
		if assert.NoError(t, err) {
			ii, err := r.ListSkillqueueItems(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ii, 3)
			}
		}
	})
}