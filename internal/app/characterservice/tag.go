package characterservice

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func (s *CharacterService) CreateTag(ctx context.Context, name string) (*app.CharacterTag, error) {
	return s.st.CreateTag(ctx, name)
}

func (s *CharacterService) DeleteTag(ctx context.Context, id int64) error {
	return s.st.DeleteTag(ctx, id)
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
