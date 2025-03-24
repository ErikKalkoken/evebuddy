package characterservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestUpdateSkillqueueESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newCharacterService(st)
	ctx := context.Background()
	t.Run("should create new queue", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		t1 := factory.CreateEveType()
		t2 := factory.CreateEveType()
		data := []map[string]any{
			{
				"finish_date":    "2016-06-29T10:47:00Z",
				"finished_level": 3,
				"queue_position": 0,
				"skill_id":       t1.ID,
				"start_date":     "2016-06-29T10:46:00Z",
			},
			{
				"finish_date":    "2016-07-15T10:47:00Z",
				"finished_level": 4,
				"queue_position": 1,
				"skill_id":       t1.ID,
				"start_date":     "2016-06-29T10:47:00Z",
			},
			{
				"finish_date":    "2016-08-30T10:47:00Z",
				"finished_level": 2,
				"queue_position": 2,
				"skill_id":       t2.ID,
				"start_date":     "2016-07-15T10:47:00Z",
			}}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/characters/%d/skillqueue/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.UpdateCharacterSkillqueueESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionSkillqueue,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			ii, err := st.ListCharacterSkillqueueItems(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ii, 3)
			}
		}
	})
}
