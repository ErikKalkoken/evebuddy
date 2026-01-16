package storage

import (
	"context"
	"fmt"

	"github.com/mattn/go-sqlite3"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) CreateTag(ctx context.Context, name string) (*app.CharacterTag, error) {
	return st.createTag(ctx, st.qRW, name)
}

func (st *Storage) createTag(ctx context.Context, q *queries.Queries, name string) (*app.CharacterTag, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("createTag: %s: %w", name, err)
	}
	if name == "" {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := q.CreateCharacterTag(ctx, name)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				err = app.ErrAlreadyExists
			}
		}
		return nil, wrapErr(err)
	}
	return tagFromDBModel(r), nil
}

func (st *Storage) DeleteTag(ctx context.Context, id int64) error {
	return st.qRW.DeleteCharacterTag(ctx, id)
}

func (st *Storage) DeleteAllTags(ctx context.Context) error {
	return st.qRW.DeleteAllCharacterTags(ctx)
}

func (st *Storage) GetTag(ctx context.Context, id int64) (*app.CharacterTag, error) {
	row, err := st.qRO.GetCharacterTag(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get Tag with id %d: %w", id, convertGetError(err))
	}
	return tagFromDBModel(row), nil
}

func (st *Storage) ListTagsByName(ctx context.Context) ([]*app.CharacterTag, error) {
	rows, err := st.qRO.ListCharacterTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)

	}
	tags := make([]*app.CharacterTag, 0)
	for _, r := range rows {
		tags = append(tags, tagFromDBModel(r))
	}
	return tags, nil
}

func (st *Storage) ReplaceTags(ctx context.Context, data map[string][]int32) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceTags: %+v: %w", data, err)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)

	err = qtx.DeleteAllCharacterTags(ctx)
	if err != nil {
		return wrapErr(err)
	}
	characters, err := st.listCharacterIDs(ctx, qtx)
	if err != nil {
		return wrapErr(err)
	}
	for tag, ids := range data {
		t, err := st.createTag(ctx, qtx, tag)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if !characters.Contains(id) {
				continue // ignore non existing character
			}
			err := st.createCharactersCharacterTag(ctx, qtx, CreateCharacterTagParams{
				CharacterID: id,
				TagID:       t.ID,
			})
			if err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	return nil
}

func tagFromDBModel(r queries.CharacterTag) *app.CharacterTag {
	return &app.CharacterTag{
		ID:   r.ID,
		Name: r.Name,
	}
}

func (st *Storage) UpdateTagName(ctx context.Context, id int64, name string) error {
	err := st.qRW.UpdateCharacterTagName(ctx, queries.UpdateCharacterTagNameParams{
		ID:   id,
		Name: name,
	})
	if err != nil {
		return fmt.Errorf("update name for tag %d: %w", id, err)
	}
	return nil
}

type CreateCharacterTagParams struct {
	CharacterID int32
	TagID       int64
}

func (st *Storage) CreateCharactersCharacterTag(ctx context.Context, arg CreateCharacterTagParams) error {
	return st.createCharactersCharacterTag(ctx, st.qRW, arg)
}

func (st *Storage) createCharactersCharacterTag(ctx context.Context, q *queries.Queries, arg CreateCharacterTagParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharactersCharacterTag: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.TagID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := q.CreateCharactersCharacterTag(ctx, queries.CreateCharactersCharacterTagParams{
		CharacterID: int64(arg.CharacterID),
		TagID:       arg.TagID,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeleteCharactersCharacterTag(ctx context.Context, arg CreateCharacterTagParams) error {
	if arg.CharacterID == 0 || arg.TagID == 0 {
		return fmt.Errorf("DeleteCharactersCharacterTag: %+v: %w", arg, app.ErrInvalid)
	}
	return st.qRW.DeleteCharactersCharacterTag(ctx, queries.DeleteCharactersCharacterTagParams{
		CharacterID: int64(arg.CharacterID),
		TagID:       arg.TagID,
	})
}

func (st *Storage) ListCharactersForCharacterTag(ctx context.Context, tagID int64) ([]*app.EntityShort[int32], error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharactersForCharacterTag: %d: %w", tagID, err)
	}
	if tagID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharactersForCharacterTag(ctx, tagID)
	if err != nil {
		return nil, wrapErr(err)

	}
	cc := make([]*app.EntityShort[int32], 0)
	for _, r := range rows {
		cc = append(cc, &app.EntityShort[int32]{
			ID:   int32(r.ID),
			Name: r.Name,
		})
	}
	return cc, nil
}

func (st *Storage) ListCharacterTagsForCharacter(ctx context.Context, characterID int32) ([]*app.CharacterTag, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterTagsForCharacter: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterTagsForCharacter(ctx, int64(characterID))
	if err != nil {
		return nil, wrapErr(err)

	}
	tags := make([]*app.CharacterTag, 0)
	for _, r := range rows {
		tags = append(tags, tagFromDBModel(r))
	}
	return tags, nil
}
