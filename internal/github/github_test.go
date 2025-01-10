package github_test

import (
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/github"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestAvailableUpdate(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	data := map[string]any{
		"url":              "https://api.github.com/repos/ErikKalkoken/janice/releases/164309952",
		"assets_url":       "https://api.github.com/repos/ErikKalkoken/janice/releases/164309952/assets",
		"upload_url":       "https://uploads.github.com/repos/ErikKalkoken/janice/releases/164309952/assets{?name,label}",
		"html_url":         "https://github.com/ErikKalkoken/janice/releases/tag/v0.2.0",
		"id":               164309952,
		"node_id":          "xyz",
		"tag_name":         "v0.2.0",
		"target_commitish": "main",
		"name":             "v0.2.0",
		"draft":            false,
		"prerelease":       false,
		"created_at":       "2024-07-07T20:45:55Z",
		"published_at":     "2024-07-07T20:48:11Z",
	}
	t.Run("should return new version when available", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "https://api.github.com/repos/ErikKalkoken/janice/releases/latest",
			httpmock.NewJsonResponderOrPanic(200, data))
		v, err := github.AvailableUpdate("ErikKalkoken", "janice", "0.1.0")
		if assert.NoError(t, err) {
			assert.Equal(t, github.VersionInfo{"0.1.0", "0.2.0", "0.2.0", true}, v)
		}
	})
	t.Run("should report when remote has no newer version", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "https://api.github.com/repos/ErikKalkoken/janice/releases/latest",
			httpmock.NewJsonResponderOrPanic(200, data))
		v, err := github.AvailableUpdate("ErikKalkoken", "janice", "0.2.0")
		if assert.NoError(t, err) {
			assert.Equal(t, github.VersionInfo{"0.2.0", "0.2.0", "0.2.0", false}, v)
		}
	})
	t.Run("should report error when request failed", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "https://api.github.com/repos/ErikKalkoken/janice/releases/latest",
			httpmock.NewErrorResponder(fmt.Errorf("some error")))
		_, err := github.AvailableUpdate("ErikKalkoken", "janice", "v0.2.0")
		assert.Error(t, err)
	})
}
