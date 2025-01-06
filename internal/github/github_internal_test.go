package github

import (
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestAvailableUpdateInternal(t *testing.T) {
	cases := []struct {
		name       string
		local      string
		remote     string
		updateInfo VersionInfo
		hasError   bool
	}{
		{"github is newer", "0.1.0", "v0.2.0", VersionInfo{"0.1.0", "0.2.0", "0.2.0", true}, false},
		{"github is older", "0.2.0", "v0.1.0", VersionInfo{"0.2.0", "0.1.0", "0.2.0", false}, false},
		{"github is same", "0.1.0", "v0.1.0", VersionInfo{"0.1.0", "0.1.0", "0.1.0", false}, false},
		{"normalizes all versions", "v0.1.0", "v0.2.0", VersionInfo{"0.1.0", "0.2.0", "0.2.0", true}, false},
		{"current version invalid", "", "v0.2.0", VersionInfo{}, true},
		{"remote version invalid", "0.1.0", "", VersionInfo{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := availableUpdate("owner", "repo", tc.local, func(owner, repo string) (string, error) {
				return tc.remote, nil
			})
			if tc.hasError {
				assert.Error(t, err)
			} else {
				if assert.NoError(t, err) {
					assert.Equal(t, tc.updateInfo, u)
				}
			}
		})
	}
}

func TestFetchGithubLatest(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Run("should return new version when available", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "https://api.github.com/repos/ErikKalkoken/janice/releases/latest",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"tag_name": "v0.2.0",
			}))
		r, err := fetchGitHubLatest("ErikKalkoken", "janice")
		if assert.NoError(t, err) {
			assert.Equal(t, "v0.2.0", r)
		}
	})
	t.Run("should return zero string when version not found", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "https://api.github.com/repos/ErikKalkoken/janice/releases/latest",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{}))
		r, err := fetchGitHubLatest("ErikKalkoken", "janice")
		if assert.NoError(t, err) {
			assert.Equal(t, "", r)
		}
	})
	t.Run("should report error when request failed", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "https://api.github.com/repos/ErikKalkoken/janice/releases/latest",
			httpmock.NewErrorResponder(fmt.Errorf("some error")))
		_, err := fetchGitHubLatest("ErikKalkoken", "janice")
		assert.Error(t, err)
	})
	t.Run("should report error when no release found", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "https://api.github.com/repos/ErikKalkoken/janice/releases/latest",
			httpmock.NewJsonResponderOrPanic(404, map[string]any{"message": "Not found"}))
		_, err := fetchGitHubLatest("ErikKalkoken", "janice")
		assert.ErrorIs(t, err, ErrHttpError)
	})
	t.Run("should report error when json unmarshaling failed", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "https://api.github.com/repos/ErikKalkoken/janice/releases/latest",
			httpmock.NewStringResponder(200, "invalid"))
		_, err := fetchGitHubLatest("ErikKalkoken", "janice")
		assert.Error(t, err)
	})
}
