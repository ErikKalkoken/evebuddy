package corporationservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestValidSections(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("return sections when valid", func(t *testing.T) {
		s := corporationservice.NewFake(st, corporationservice.Params{
			CharacterService: &corporationservice.CharacterServiceFake{
				Token: &app.CharacterToken{AccessToken: "accessToken"},
			},
		})
		c := factory.CreateCorporation()
		got, err := s.ValidSections(ctx, c.ID)
		if assert.NoError(t, err) {
			want := []app.CorporationSection{app.SectionCorporationIndustryJobs}
			assert.EqualValues(t, want, got)
		}
	})
	t.Run("return empty when no token found", func(t *testing.T) {
		s := corporationservice.NewFake(st, corporationservice.Params{
			CharacterService: &corporationservice.CharacterServiceFake{
				Error: app.ErrNotFound,
			},
		})
		c := factory.CreateCorporation()
		got, err := s.ValidSections(ctx, c.ID)
		if assert.NoError(t, err) {
			want := []app.CorporationSection{}
			assert.EqualValues(t, want, got)
		}
	})
	t.Run("return error when any other error occured", func(t *testing.T) {
		myErr := errors.New("random error")
		s := corporationservice.NewFake(st, corporationservice.Params{
			CharacterService: &corporationservice.CharacterServiceFake{
				Error: myErr,
			},
		})
		c := factory.CreateCorporation()
		_, err := s.ValidSections(ctx, c.ID)
		assert.ErrorIs(t, err, myErr)
	})
}
