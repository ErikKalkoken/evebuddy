package characterservice

import (
	"context"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func (s *CharacterService) CreateTag(ctx context.Context, name string) (*app.CharacterTag, error) {
	return s.st.CreateTag(ctx, name)
}

func (s *CharacterService) DeleteTag(ctx context.Context, id int64) error {
	return s.st.DeleteTag(ctx, id)
}

func (s *CharacterService) DeleteAllTags(ctx context.Context) error {
	return s.st.DeleteAllTags(ctx)
}

func (s *CharacterService) ListTagsByName(ctx context.Context) ([]*app.CharacterTag, error) {
	return s.st.ListTagsByName(ctx)
}

func (s *CharacterService) RenameTag(ctx context.Context, id int64, name string) error {
	return s.st.UpdateTagName(ctx, id, name)
}

func (s *CharacterService) AddTagToCharacter(ctx context.Context, characterID int32, tagID int64) error {
	return s.st.CreateCharactersCharacterTag(ctx, storage.CreateCharacterTagParams{
		CharacterID: characterID,
		TagID:       tagID,
	})
}

func (s *CharacterService) RemoveTagFromCharacter(ctx context.Context, characterID int32, tagID int64) error {
	return s.st.DeleteCharactersCharacterTag(ctx, storage.CreateCharacterTagParams{
		CharacterID: characterID,
		TagID:       tagID,
	})
}

func (s *CharacterService) ListCharactersForTag(ctx context.Context, tagID int64) (tagged []*app.EntityShort[int32], others []*app.EntityShort[int32], err error) {
	tagged, err = s.st.ListCharactersForCharacterTag(ctx, tagID)
	if err != nil {
		return
	}
	taggedIDs := set.Of(xslices.Map(tagged, func(x *app.EntityShort[int32]) int32 {
		return x.ID
	})...)
	cc, err := s.st.ListCharactersShort(ctx)
	if err != nil {
		return nil, nil, err
	}
	others = xslices.Filter(cc, func(x *app.EntityShort[int32]) bool {
		return !taggedIDs.Contains(x.ID)
	})
	return
}

func (s *CharacterService) ListTagsForCharacter(ctx context.Context, characterID int32) (set.Set[string], error) {
	oo, err := s.st.ListCharacterTagsForCharacter(ctx, characterID)
	if err != nil {
		return set.Set[string]{}, err
	}
	tags := set.Collect(xiter.MapSlice(oo, func(x *app.CharacterTag) string {
		return x.Name
	}))
	return tags, nil
}

func (s *CharacterService) ExportTags(ctx context.Context) (map[string][]int32, error) {
	tags, err := s.st.ListTagsByName(ctx)
	if err != nil {
		return nil, err
	}
	v := make(map[string][]int32)
	for _, t := range tags {
		characters, err := s.st.ListCharactersForCharacterTag(ctx, t.ID)
		if err != nil {
			return nil, err
		}
		v[t.Name] = xslices.Map(characters, func(x *app.EntityShort[int32]) int32 {
			return x.ID
		})
	}
	return v, nil
}

func (s *CharacterService) ImportTags(ctx context.Context, v map[string][]int32) error {
	err := s.st.DeleteAllTags(ctx)
	if err != nil {
		return err
	}
	cc, err := s.st.ListCharacterIDs(ctx)
	if err != nil {
		return err
	}
	for tag, ids := range v {
		t, err := s.st.CreateTag(ctx, tag)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if !cc.Contains(id) {
				continue // ignore non existing character
			}
			err := s.st.CreateCharactersCharacterTag(ctx, storage.CreateCharacterTagParams{
				CharacterID: id,
				TagID:       t.ID,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
