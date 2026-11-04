package characterservice

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestLocationFlagFromESIValue_MatchesDomain(t *testing.T) {
	got := set.Collect(maps.Values(locationFlagFromESIValue))
	xassert.Equal(t, app.CharacterLocationFlags, got)
	assert.Len(t, locationFlagFromESIValue, got.Size())
}

// TestLocationValues_MatchSpec compares against a stored copy of the ESI spec,
// so it can only fail when that copy is updated.
func TestLocationValues_MatchSpec(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "genratelimit", "openapi.json"))
	require.NoError(t, err)
	var spec struct {
		Components struct {
			Schemas struct {
				Assets struct {
					Items struct {
						Properties struct {
							LocationFlag struct {
								Enum []string `json:"enum"`
							} `json:"location_flag"`
							LocationType struct {
								Enum []string `json:"enum"`
							} `json:"location_type"`
						} `json:"properties"`
					} `json:"items"`
				} `json:"CharactersCharacterIdAssetsGet"`
			} `json:"schemas"`
		} `json:"components"`
	}
	require.NoError(t, json.Unmarshal(data, &spec))
	properties := spec.Components.Schemas.Assets.Items.Properties
	require.NotEmpty(t, properties.LocationFlag.Enum)
	xassert.Equal(t, set.Of(properties.LocationFlag.Enum...), set.Collect(maps.Keys(locationFlagFromESIValue)))
	require.NotEmpty(t, properties.LocationType.Enum)
	wantTypes := set.Of("") // used when ESI does not provide a type
	wantTypes.Add(properties.LocationType.Enum...)
	xassert.Equal(t, wantTypes, set.Collect(maps.Keys(locationTypeFromESIValue)))
}
