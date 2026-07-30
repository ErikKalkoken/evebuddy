package storage

import (
	"context"
	"database/sql"
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
	c := eveCharacterFromDBModel(eveCharacterFromDBModelParams{
		character:   r.EveCharacter,
		corporation: r.EveEntity,
		race:        r.EveRace,
		alliance: nullEveEntry{
			id:       r.EveCharacter.AllianceID,
			name:     r.AllianceName,
			category: r.AllianceCategory,
		},
		faction: nullEveEntry{
			id:       r.EveCharacter.FactionID,
			name:     r.FactionName,
			category: r.FactionCategory,
		},
		bloodline: nullEntity{
			id:   r.BloodlineID,
			name: r.BloodlineName,
		},
	})
	return c, nil
}

func (st *Storage) ListEveCharacterIDs(ctx context.Context) (set.Set[int64], error) {
	ids, err := st.qRO.ListEveCharacterIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list EveCharacterIDs: %w", err)
	}
	ids2 := set.Collect(slices.Values(ids))
	return ids2, nil
}

// TODO: Remove unused method: UpdateEveCharacter

func (st *Storage) UpdateEveCharacter(ctx context.Context, c *app.EveCharacter) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateEveCharacter: %+v: %w", c, err)
	}
	if c.ID == 0 || c.Corporation == nil {
		return wrapErr(app.ErrInvalid)
	}
	allianceID := optional.Map(c.Alliance, 0, func(x *app.EveEntity) int64 {
		return x.ID
	})
	factionID := optional.Map(c.Faction, 0, func(x *app.EveEntity) int64 {
		return x.ID
	})
	var bloodlineID sql.NullInt64
	if v, ok := c.Bloodline.Value(); ok && v != nil {
		bloodlineID = NewNullInt64(v.ID)
	}
	err := st.qRW.UpdateEveCharacter(ctx, queries.UpdateEveCharacterParams{
		AllianceID:     NewNullInt64(allianceID),
		Birthday:       c.Birthday,
		BloodlineID:    bloodlineID,
		CorporationID:  c.Corporation.ID,
		Description:    c.Description.ValueOrZero(),
		FactionID:      NewNullInt64(factionID),
		Gender:         c.Gender,
		ID:             c.ID,
		Name:           c.Name,
		RaceID:         c.Race.ID,
		SecurityStatus: c.SecurityStatus.ValueOrZero(),
		Title:          c.CorporationTitle.ValueOrZero(),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
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

type eveCharacterFromDBModelParams struct {
	character   queries.EveCharacter
	corporation queries.EveEntity
	race        queries.EveRace
	alliance    nullEveEntry
	faction     nullEveEntry
	bloodline   nullEntity
}

func eveCharacterFromDBModel(arg eveCharacterFromDBModelParams) *app.EveCharacter {
	var bloodline optional.Optional[*app.EntityShort]
	if arg.bloodline.isValid() {
		bloodline.Set(&app.EntityShort{
			ID:   arg.bloodline.id.Int64,
			Name: arg.bloodline.name.String,
		})
	}
	o := app.EveCharacter{
		Alliance:         eveEntityFromNullableDBModel(arg.alliance),
		Birthday:         arg.character.Birthday,
		Corporation:      eveEntityFromDBModel(arg.corporation),
		Description:      optional.FromZeroValue(arg.character.Description),
		Gender:           arg.character.Gender,
		Faction:          eveEntityFromNullableDBModel(arg.faction),
		ID:               arg.character.ID,
		Name:             arg.character.Name,
		Race:             eveRaceFromDBModel(arg.race),
		SecurityStatus:   optional.FromZeroValue(arg.character.SecurityStatus),
		CorporationTitle: optional.FromZeroValue(arg.character.Title),
		Bloodline:        bloodline,
	}
	return &o
}
