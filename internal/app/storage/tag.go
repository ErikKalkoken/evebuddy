package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/mattn/go-sqlite3"
)

type CreateTagParams struct {
	Name        string
	Description string
}

func (st *Storage) CreateTag(ctx context.Context, arg CreateTagParams) (*app.Tag, error) {
	if arg.Name == "" {
		return nil, fmt.Errorf("CreateTag: %+v: %w", arg, app.ErrInvalid)
	}
	r, err := st.qRW.CreateTag(ctx, queries.CreateTagParams{
		Name:        arg.Name,
		Description: arg.Description,
	})
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				err = app.ErrAlreadyExists
			}
		}
		return nil, fmt.Errorf("CreateTag: %+v: %w", arg, err)
	}
	return tagFromDBModel(r), nil
}

func (st *Storage) DeleteTag(ctx context.Context, id int64) error {
	return st.qRW.DeleteTag(ctx, id)
}

func (st *Storage) GetTag(ctx context.Context, id int64) (*app.Tag, error) {
	row, err := st.qRO.GetTag(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get Tag with id %d: %w", id, convertGetError(err))
	}
	return tagFromDBModel(row), nil
}

func (st *Storage) ListTagsByName(ctx context.Context) ([]*app.Tag, error) {
	rows, err := st.qRO.ListTagsByName(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)

	}
	tags := make([]*app.Tag, 0)
	for _, r := range rows {
		tags = append(tags, tagFromDBModel(r))
	}
	return tags, nil
}

func tagFromDBModel(r queries.Tag) *app.Tag {
	return &app.Tag{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
	}
}

func (st *Storage) UpdateTagName(ctx context.Context, id int64, name string) error {
	err := st.qRW.UpdateTagName(ctx, queries.UpdateTagNameParams{
		ID:   id,
		Name: name,
	})
	if err != nil {
		return fmt.Errorf("update name for tag %d: %w", id, err)
	}
	return nil
}

func (st *Storage) UpdateTagDescription(ctx context.Context, id int64, description string) error {
	err := st.qRW.UpdateTagDescription(ctx, queries.UpdateTagDescriptionParams{
		ID:          id,
		Description: description,
	})
	if err != nil {
		return fmt.Errorf("update description for tag %d: %w", id, err)
	}
	return nil
}

type CreateCharacterTagParams struct {
	CharacterID int32
	TagID       int64
}

func (st *Storage) CreateCharacterTag(ctx context.Context, arg CreateCharacterTagParams) error {
	if arg.CharacterID == 0 || arg.TagID == 0 {
		return fmt.Errorf("CreateCharacterTag: %+v: %w", arg, app.ErrInvalid)
	}
	err := st.qRW.CreateCharacterTag(ctx, queries.CreateCharacterTagParams{
		CharacterID: int64(arg.CharacterID),
		TagID:       arg.TagID,
	})
	if err != nil {
		return fmt.Errorf("CreateCharacterTag: %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) DeleteCharacterTag(ctx context.Context, arg CreateCharacterTagParams) error {
	if arg.CharacterID == 0 || arg.TagID == 0 {
		return fmt.Errorf("DeleteCharacterTag: %+v: %w", arg, app.ErrInvalid)
	}
	return st.qRW.DeleteCharacterTag(ctx, queries.DeleteCharacterTagParams{
		CharacterID: int64(arg.CharacterID),
		TagID:       arg.TagID,
	})
}

func (st *Storage) ListCharactersForTag(ctx context.Context, tagID int64) ([]*app.EntityShort[int32], error) {
	if tagID == 0 {
		return nil, fmt.Errorf("ListCharactersForTag: %d: %w", tagID, app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharactersForTag(ctx, tagID)
	if err != nil {
		return nil, fmt.Errorf("ListCharactersForTag with ID %d: %w", tagID, err)

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

func (st *Storage) ListTagsForCharacter(ctx context.Context, characterID int32) ([]*app.Tag, error) {
	if characterID == 0 {
		return nil, fmt.Errorf("ListTagsForCharacter: %d: %w", characterID, app.ErrInvalid)
	}
	rows, err := st.qRO.ListTagsForCharacter(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("ListTagsForCharacter with ID %d: %w", characterID, err)

	}
	tags := make([]*app.Tag, 0)
	for _, r := range rows {
		tags = append(tags, tagFromDBModel(r))
	}
	return tags, nil
}
