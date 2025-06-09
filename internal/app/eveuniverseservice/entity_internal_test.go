package eveuniverseservice

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetValidEntity(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewTestService(st)
	ctx := context.Background()
	entity := factory.CreateEveEntity()
	t.Run("should return entity when id is valid", func(t *testing.T) {
		x, err := s.getValidEntity(ctx, entity.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, entity, x)
		}
	})
	t.Run("should return nil when id is not valid 1", func(t *testing.T) {
		x, err := s.getValidEntity(ctx, 0)
		if assert.NoError(t, err) {
			assert.Nil(t, x)
		}
	})
	t.Run("should return nil when id is not valid 2", func(t *testing.T) {
		x, err := s.getValidEntity(ctx, 1)
		if assert.NoError(t, err) {
			assert.Nil(t, x)
		}
	})
}
