package storage

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateEveCharacterParams struct {
	AllianceID       optional.Optional[int64]
	Birthday         time.Time
	BloodlineID      int64
	CorporationID    int64
	CorporationTitle optional.Optional[string]
	Description      optional.Optional[string]
	FactionID        optional.Optional[int64]
	Gender           string
	ID               int64
	Name             string
	RaceID           int64
	SecurityStatus   optional.Optional[float64]
}

func (st *Storage) UpdateOrCreateEveCharacter(ctx context.Context, arg CreateEveCharacterParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateEveCharacter: %+v: %w", arg, err)
	}
	if arg.ID == 0 || arg.CorporationID == 0 || arg.RaceID == 0 || arg.BloodlineID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateEveCharacter(ctx, queries.UpdateOrCreateEveCharacterParams{
		AllianceID:     optional.ToNullInt64(arg.AllianceID),
		Birthday:       arg.Birthday,
		BloodlineID:    NewNullInt64(arg.BloodlineID),
		CorporationID:  arg.CorporationID,
		Description:    arg.Description.ValueOrZero(),
		FactionID:      optional.ToNullInt64(arg.FactionID),
		Gender:         arg.Gender,
		ID:             arg.ID,
		Name:           arg.Name,
		RaceID:         arg.RaceID,
		SecurityStatus: arg.SecurityStatus.ValueOrZero(),
		Title:          arg.CorporationTitle.ValueOrZero(),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeleteEveCharacter(ctx context.Context, characterID int64) error {
	err := st.qRW.DeleteEveCharacter(ctx, characterID)
	if err != nil {
		return fmt.Errorf("delete EveCharacter %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) GetEveCharacter(ctx context.Context, characterID int64) (*app.EveCharacter, error) {
	r, err := st.qRO.GetEveCharacter(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("GetEveCharacter %d: %w", characterID, convertGetError(err))
	}
	o := eveCharacterFromDBModel(
		r.EveCharacter,
		r.EveEntity,
		r.EveRace,
		nullEveEntry{
			id:       r.EveCharacter.AllianceID,
			name:     r.AllianceName,
			category: r.AllianceCategory,
		},
		nullEveEntry{
			id:       r.EveCharacter.FactionID,
			name:     r.FactionName,
			category: r.FactionCategory,
		},
		nullEntity{
			id:   r.BloodlineID,
			name: r.BloodlineName,
		},
	)
	return o, nil
}

func (st *Storage) ListEveCharacterIDs(ctx context.Context) (set.Set[int64], error) {
	ids, err := st.qRO.ListEveCharacterIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list EveCharacterIDs: %w", err)
	}
	ids2 := set.Collect(slices.Values(ids))
	return ids2, nil
}

func (st *Storage) UpdateEveCharacterName(ctx context.Context, characterID int64, name string) error {
	if characterID == 0 || name == "" {
		return fmt.Errorf("UpdateEveCharacterName: %w", app.ErrInvalid)
	}
	if err := st.qRW.UpdateEveCharacterName(ctx, queries.UpdateEveCharacterNameParams{
		ID:   characterID,
		Name: name,
	}); err != nil {
		return fmt.Errorf("UpdateEveCharacterName %d: %w", characterID, err)
	}
	return nil
}

func eveCharacterFromDBModel(
	character queries.EveCharacter,
	corporation queries.EveEntity,
	race queries.EveRace,
	alliance nullEveEntry,
	faction nullEveEntry,
	bloodline nullEntity,

) *app.EveCharacter {
	var bloodline2 optional.Optional[*app.EntityShort]
	if bloodline.isValid() {
		bloodline2.Set(&app.EntityShort{
			ID:   bloodline.id.Int64,
			Name: bloodline.name.String,
		})
	}
	o := app.EveCharacter{
		Alliance:         eveEntityFromNullableDBModel(alliance),
		Birthday:         character.Birthday,
		Corporation:      eveEntityFromDBModel(corporation),
		Description:      optional.FromZeroValue(character.Description),
		Gender:           character.Gender,
		Faction:          eveEntityFromNullableDBModel(faction),
		ID:               character.ID,
		Name:             character.Name,
		Race:             eveRaceFromDBModel(race),
		SecurityStatus:   optional.New(character.SecurityStatus),
		CorporationTitle: optional.FromZeroValue(character.Title),
		Bloodline:        bloodline2,
	}
	return &o
}
