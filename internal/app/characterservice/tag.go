package characterservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ErikKalkoken/go-set"
	"github.com/hashicorp/go-version"

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

// TagsExported represents the data structure for exported character tags.
type TagsExported struct {
	Info    string             // file information
	Tags    map[string][]int32 // mapping of tags to character IDs
	Version string             // app version that created an export
}

func (s *CharacterService) WriteTags(ctx context.Context, writer io.Writer, version string) error {
	v, err := s.compileTags(ctx, version)
	if err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = writer.Write(b)
	if err != nil {
		return err
	}
	return nil
}

func (s *CharacterService) compileTags(ctx context.Context, version string) (TagsExported, error) {
	data := TagsExported{
		Tags:    make(map[string][]int32),
		Info:    "Created by EVE Buddy",
		Version: version,
	}
	tags, err := s.st.ListTagsByName(ctx)
	if err != nil {
		return data, err
	}
	for _, t := range tags {
		characters, err := s.st.ListCharactersForCharacterTag(ctx, t.ID)
		if err != nil {
			return data, err
		}
		data.Tags[t.Name] = xslices.Map(characters, func(x *app.EntityShort[int32]) int32 {
			return x.ID
		})
	}
	return data, nil
}

// ReadAndReplaceTags reads a tags definition from reader and replaces the current tags with it.
func (s *CharacterService) ReadAndReplaceTags(ctx context.Context, reader io.Reader, version string) error {
	b, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	var data TagsExported
	err = json.Unmarshal(b, &data)
	if err != nil {
		return fmt.Errorf("unrecognized format")
	}
	err = s.replaceTags(ctx, data, version)
	if err != nil {
		return err
	}
	return nil
}

func (s *CharacterService) replaceTags(ctx context.Context, data TagsExported, v string) error {
	appVersion, err := version.NewVersion(v)
	if err != nil {
		return err
	}
	fileVersion, err := version.NewVersion(data.Version)
	if err != nil {
		return err
	}
	similar := func() bool {
		fvs := fileVersion.Segments()
		if len(fvs) != 3 {
			return false
		}
		avs := appVersion.Segments()
		if len(avs) != 3 {
			return false
		}
		if fvs[0] != avs[0] {
			return false
		}
		if fvs[1] != avs[1] {
			return false
		}
		return true
	}()
	if !similar {
		return fmt.Errorf("file was created with a different version")
	}
	err = s.st.ReplaceTags(ctx, data.Tags)
	if err != nil {
		return err
	}
	return nil
}
