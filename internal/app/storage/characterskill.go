package storage

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/xmaps"
)

func (st *Storage) GetCharacterSkill(ctx context.Context, characterID int64, typeID int64) (*app.CharacterSkill, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterSkill: %d %d: %w", characterID, typeID, err)
	}
	if characterID == 0 || typeID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCharacterSkill(ctx, queries.GetCharacterSkillParams{
		CharacterID: characterID,
		EveTypeID:   typeID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	t2 := characterSkillFromDBModel(r.CharacterSkill, r.EveType, r.EveGroup, r.EveCategory)
	return t2, nil
}

func (st *Storage) ListCharacterSkills(ctx context.Context, characterID int64) ([]*app.CharacterSkill, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterSkills: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterSkills(ctx, characterID)
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]*app.CharacterSkill, len(rows))
	for i, r := range rows {
		oo[i] = characterSkillFromDBModel(r.CharacterSkill, r.EveType, r.EveGroup, r.EveCategory)
	}
	return oo, nil
}

func (st *Storage) ListAllCharacterSkills(ctx context.Context) ([]*app.CharacterSkill, error) {
	rows, err := st.qRO.ListAllCharacterSkills(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListAllCharacterSkills: %w", err)
	}
	oo := make([]*app.CharacterSkill, len(rows))
	for i, r := range rows {
		oo[i] = characterSkillFromDBModel(r.CharacterSkill, r.EveType, r.EveGroup, r.EveCategory)
	}
	return oo, nil
}

func (st *Storage) ListAllCharactersActiveSkillLevels(ctx context.Context, typeID int64) ([]app.CharacterActiveSkillLevel, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListAllCharactersActiveSkillLevels: %d: %w", typeID, err)
	}
	if typeID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharactersActiveSkillLevels(ctx, typeID)
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]app.CharacterActiveSkillLevel, len(rows))
	for i, r := range rows {
		oo[i] = app.CharacterActiveSkillLevel{
			CharacterID: r.CharacterID,
			Level:       int(r.Level),
			TypeID:      typeID,
		}
	}
	return oo, nil

}

func (st *Storage) ListCharacterSkillIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterSkillIDs: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return set.Set[int64]{}, wrapErr(app.ErrInvalid)
	}
	ids, err := st.qRO.ListCharacterSkillIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	return set.Collect(slices.Values(ids)), nil
}

// ReplaceCharacterSkillParams is a skill of a character from ESI.
type ReplaceCharacterSkillParams struct {
	ActiveSkillLevel   int64
	SkillPointsInSkill int64
	TrainedSkillLevel  int64
	TypeID             int64
}

// ReplaceCharacterSkills replaces all stored skills of a character with skills
// and reports whether any skill was created, updated or deleted.
func (st *Storage) ReplaceCharacterSkills(ctx context.Context, characterID int64, skills []ReplaceCharacterSkillParams) (bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceCharacterSkills: character %d: %w", characterID, err)
	}
	if characterID == 0 {
		return false, wrapErr(app.ErrInvalid)
	}
	for _, arg := range skills {
		if arg.TypeID == 0 {
			return false, wrapErr(fmt.Errorf("skill with type ID 0: %w", app.ErrInvalid))
		}
	}
	rows, err := st.qRO.ListCharacterSkillStates(ctx, characterID)
	if err != nil {
		return false, wrapErr(err)
	}
	current := make(map[int64]queries.CharacterSkill, len(rows)) // by type ID
	for _, r := range rows {
		current[r.EveTypeID] = r
	}
	var changed []ReplaceCharacterSkillParams
	for _, arg := range skills {
		r, found := xmaps.Pop(current, arg.TypeID) // what remains in current is obsolete
		if found &&
			r.ActiveSkillLevel == arg.ActiveSkillLevel &&
			r.SkillPointsInSkill == arg.SkillPointsInSkill &&
			r.TrainedSkillLevel == arg.TrainedSkillLevel {
			continue
		}
		changed = append(changed, arg)
	}
	obsolete := slices.Collect(maps.Keys(current))
	if len(changed) == 0 && len(obsolete) == 0 {
		return false, nil
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return false, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	for _, arg := range changed {
		err := qtx.UpdateOrCreateCharacterSkill(ctx, queries.UpdateOrCreateCharacterSkillParams{
			ActiveSkillLevel:   arg.ActiveSkillLevel,
			EveTypeID:          arg.TypeID,
			SkillPointsInSkill: arg.SkillPointsInSkill,
			CharacterID:        characterID,
			TrainedSkillLevel:  arg.TrainedSkillLevel,
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("update or create skill %d: %w", arg.TypeID, err))
		}
	}
	for ids := range slices.Chunk(obsolete, st.MaxIDsPerQuery) {
		err := qtx.DeleteCharacterSkills(ctx, queries.DeleteCharacterSkillsParams{
			CharacterID: characterID,
			EveTypeIds:  ids,
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("delete %d skills: %w", len(ids), err))
		}
	}
	if err := tx.Commit(); err != nil {
		return false, wrapErr(err)
	}
	slog.Info("Character skills replaced", "characterID", characterID, "changed", len(changed), "deleted", len(obsolete))
	return true, nil
}

type UpdateOrCreateCharacterSkillParams struct {
	ActiveSkillLevel   int64
	TypeID             int64
	SkillPointsInSkill int64
	CharacterID        int64
	TrainedSkillLevel  int64
}

func (st *Storage) UpdateOrCreateCharacterSkill(ctx context.Context, arg UpdateOrCreateCharacterSkillParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCharacterSkill: %v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.TypeID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateCharacterSkill(ctx, queries.UpdateOrCreateCharacterSkillParams{
		ActiveSkillLevel:   arg.ActiveSkillLevel,
		EveTypeID:          arg.TypeID,
		SkillPointsInSkill: arg.SkillPointsInSkill,
		CharacterID:        arg.CharacterID,
		TrainedSkillLevel:  arg.TrainedSkillLevel,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func characterSkillFromDBModel(
	o queries.CharacterSkill,
	t queries.EveType,
	g queries.EveGroup,
	c queries.EveCategory,
) *app.CharacterSkill {
	return &app.CharacterSkill{
		ActiveSkillLevel:   o.ActiveSkillLevel,
		CharacterID:        o.CharacterID,
		Type:               eveTypeFromDBModel(t, g, c),
		SkillPointsInSkill: o.SkillPointsInSkill,
		TrainedSkillLevel:  o.TrainedSkillLevel,
	}
}
