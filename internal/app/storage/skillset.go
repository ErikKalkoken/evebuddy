package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

// CreateSkillSet creates a skill set with its requirements and returns its ID.
func (st *Storage) CreateSkillSet(ctx context.Context, s app.SkillSet) (int64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateSkillSet: %q: %w", s.Name, err)
	}
	if err := validateSkillSet(s); err != nil {
		return 0, wrapErr(err)
	}
	tx, err := st.dbRW.BeginTx(ctx, nil)
	if err != nil {
		return 0, wrapErr(err)
	}
	defer tx.Rollback()
	q := st.qRW.WithTx(tx)
	r, err := q.CreateSkillSet(ctx, s.Name)
	if err != nil {
		if isUniqueViolation(err) {
			err = app.ErrAlreadyExists
		}
		return 0, wrapErr(err)
	}
	if err := createSkillSetRequirements(ctx, q, r.ID, s.Requirements); err != nil {
		return 0, wrapErr(err)
	}
	if err := tx.Commit(); err != nil {
		return 0, wrapErr(err)
	}
	return r.ID, nil
}

// UpdateSkillSet updates the name and replaces all requirements of a skill set.
func (st *Storage) UpdateSkillSet(ctx context.Context, s app.SkillSet) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateSkillSet: %d: %q: %w", s.ID, s.Name, err)
	}
	if err := validateSkillSet(s); err != nil {
		return wrapErr(err)
	}
	tx, err := st.dbRW.BeginTx(ctx, nil)
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	q := st.qRW.WithTx(tx)
	_, err = q.UpdateSkillSet(ctx, queries.UpdateSkillSetParams{ID: s.ID, Name: s.Name})
	if err != nil {
		if isUniqueViolation(err) {
			err = app.ErrAlreadyExists
		}
		return wrapErr(convertGetError(err))
	}
	if err := q.ClearSkillSetRequirements(ctx, s.ID); err != nil {
		return wrapErr(err)
	}
	if err := createSkillSetRequirements(ctx, q, s.ID, s.Requirements); err != nil {
		return wrapErr(err)
	}
	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	return nil
}

func createSkillSetRequirements(ctx context.Context, q *queries.Queries, id int64, requirements []app.SkillSetRequirement) error {
	for _, r := range requirements {
		err := q.CreateSkillSetRequirement(ctx, queries.CreateSkillSetRequirementParams{
			SkillSetID: id,
			TypeID:     r.TypeID,
			Level:      int64(r.Level),
		})
		if err != nil {
			if isUniqueViolation(err) {
				return fmt.Errorf("skill %d listed twice: %w", r.TypeID, app.ErrInvalid)
			}
			return err
		}
	}
	return nil
}

func validateSkillSet(s app.SkillSet) error {
	if s.Name == "" {
		return fmt.Errorf("name can not be empty: %w", app.ErrInvalid)
	}
	for _, r := range s.Requirements {
		if r.Level == 0 {
			return fmt.Errorf("skill %d: level can not be zero: %w", r.TypeID, app.ErrInvalid)
		}
	}
	return nil
}

func (st *Storage) ListSkillSets(ctx context.Context) ([]app.SkillSet, error) {
	sets, err := st.qRO.ListSkillSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListSkillSets: %w", err)
	}
	requirements, err := st.qRO.ListSkillSetRequirements(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListSkillSets: %w", err)
	}
	bySet := make(map[int64][]app.SkillSetRequirement)
	for _, r := range requirements {
		bySet[r.SkillSetID] = append(bySet[r.SkillSetID], app.SkillSetRequirement{
			TypeID: r.TypeID,
			Name:   r.Name,
			Level:  int(r.Level),
		})
	}
	result := make([]app.SkillSet, 0, len(sets))
	for _, s := range sets {
		result = append(result, app.SkillSet{ID: s.ID, Name: s.Name, Requirements: bySet[s.ID]})
	}
	return result, nil
}

func (st *Storage) DeleteSkillSet(ctx context.Context, id int64) error {
	n, err := st.qRW.DeleteSkillSet(ctx, id)
	if err != nil {
		return fmt.Errorf("DeleteSkillSet: %d: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("DeleteSkillSet: %d: %w", id, app.ErrNotFound)
	}
	return nil
}
