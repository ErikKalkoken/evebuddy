package eveuniverseservice_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestEveUniverseService_ListSkills(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})
	t.Run("should return list of skills", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategorySkill})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID, IsPublished: true})
		skill1 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		skill2 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		primarySkillType := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributePrimarySkillID,
		})
		primarySkillLevel := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributePrimarySkillLevel,
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        skill1.ID,
			DogmaAttributeID: primarySkillType.ID,
			Value:            float64(skill2.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        skill1.ID,
			DogmaAttributeID: primarySkillLevel.ID,
			Value:            float64(1),
		})
		factory.CreateEveTypeDogmaAttribute()
		// when
		oo, err := s.ListSkills(t.Context())
		// then
		require.NoError(t, err)
		require.Len(t, oo, 2)
		for _, o := range oo {
			switch o.Type.ID {
			case skill1.ID:
				assert.Len(t, o.Requirements, 1)
				assert.Equal(t, 1, o.Requirements[0].Level)
				assert.Equal(t, skill2.ID, o.Requirements[0].Type.ID)

			case skill2.ID:
				assert.Len(t, o.Requirements, 0)
			}
		}
	})
}
