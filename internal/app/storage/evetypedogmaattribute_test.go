package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveTypeDogmaAttribute(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		et := factory.CreateEveType()
		eda := factory.CreateEveDogmaAttribute()
		arg := storage.CreateEveTypeDogmaAttributeParams{
			DogmaAttributeID: eda.ID,
			EveTypeID:        et.ID,
			Value:            123.45,
		}
		// when
		err := st.CreateEveTypeDogmaAttribute(ctx, arg)
		// then
		require.NoError(t, err)
		v, err := st.GetEveTypeDogmaAttribute(ctx, et.ID, eda.ID)
		require.NoError(t, err)
		xassert.Equal(t, 123.45, v)
	})
	t.Run("can list for type", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		et := factory.CreateEveType()
		o1 := factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID: et.ID,
		})
		o2 := factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID: et.ID,
		})
		factory.CreateEveTypeDogmaAttribute()
		// when
		oo, err := st.ListEveTypeDogmaAttributesForType(ctx, et.ID)
		// then
		require.NoError(t, err)
		// want := set.Of()
		assert.ElementsMatch(t, []*app.EveTypeDogmaAttribute{o1, o2}, oo)
	})
	t.Run("can list for skills", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategorySkill})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID, IsPublished: true})
		skill := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		o1 := factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID: skill.ID,
		})
		o2 := factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID: skill.ID,
		})
		factory.CreateEveTypeDogmaAttribute()
		// when
		oo, err := st.ListEveTypeDogmaAttributesForSkills(ctx)
		// then
		require.NoError(t, err)
		// want := set.Of()
		assert.ElementsMatch(t, []*app.EveTypeDogmaAttribute{o1, o2}, oo)
	})
}
