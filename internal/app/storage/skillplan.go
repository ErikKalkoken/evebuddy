package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mattn/go-sqlite3"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

// SaveSkillPlan creates or atomically replaces a plan and its requirements.
func (st *Storage) SaveSkillPlan(ctx context.Context, p app.SkillPlan) (int64, error) {
	p.Name = strings.TrimSpace(p.Name)
	if err := p.Validate(); err != nil {
		return 0, err
	}
	tx, err := st.dbRW.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	q := st.qRW.WithTx(tx)
	var saved queries.SkillPlan
	if p.ID == 0 {
		saved, err = q.CreateSkillPlan(ctx, p.Name)
	} else {
		saved, err = q.UpdateSkillPlan(ctx, queries.UpdateSkillPlanParams{ID: p.ID, Name: p.Name})
	}
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			err = app.ErrAlreadyExists
		}
		return 0, fmt.Errorf("save skill plan %q: %w", p.Name, convertGetError(err))
	}
	if err := q.ClearSkillPlanRequirements(ctx, saved.ID); err != nil {
		return 0, err
	}
	for _, r := range p.Requirements {
		et, err := q.GetEveType(ctx, r.TypeID)
		if err != nil {
			return 0, fmt.Errorf("skill %d: %w", r.TypeID, convertGetError(err))
		}
		if et.EveCategory.ID != app.EveCategorySkill {
			return 0, fmt.Errorf("type %d is not a skill: %w", r.TypeID, app.ErrInvalid)
		}
		if err := q.CreateSkillPlanRequirement(ctx, queries.CreateSkillPlanRequirementParams{
			PlanID: saved.ID,
			TypeID: r.TypeID,
			Level:  int64(r.Level),
		}); err != nil {
			return 0, fmt.Errorf("save skill plan requirement: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return saved.ID, nil
}

func (st *Storage) ListSkillPlans(ctx context.Context) ([]app.SkillPlan, error) {
	plans, err := st.qRO.ListSkillPlans(ctx)
	if err != nil {
		return nil, err
	}
	requirements, err := st.qRO.ListSkillPlanRequirements(ctx)
	if err != nil {
		return nil, err
	}
	byPlan := make(map[int64][]app.SkillPlanRequirement)
	for _, r := range requirements {
		byPlan[r.PlanID] = append(byPlan[r.PlanID], app.SkillPlanRequirement{
			TypeID: r.TypeID,
			Name:   r.Name,
			Level:  int(r.Level),
		})
	}
	result := make([]app.SkillPlan, 0, len(plans))
	for _, p := range plans {
		result = append(result, app.SkillPlan{ID: p.ID, Name: p.Name, Requirements: byPlan[p.ID]})
	}
	return result, nil
}

func (st *Storage) DeleteSkillPlan(ctx context.Context, id int64) error {
	n, err := st.qRW.DeleteSkillPlan(ctx, id)
	if err != nil {
		return fmt.Errorf("delete skill plan: %w", err)
	}
	if n == 0 {
		return app.ErrNotFound
	}
	return nil
}
