// Package github contains features for accessing repos on Github.
package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/go-version"
)

var ErrHTTPError = errors.New("HTTP error")

// VersionInfo represents the version information. All versions are normalized.
type VersionInfo struct {
	Local         string // version of local installation
	Remote        string // latest version available remote
	Latest        string // latest version
	IsRemoteNewer bool
}

// NormalizeVersion returns a normalized version number, e.g. it removes a leading v.
func NormalizeVersion(v string) (string, error) {
	version, err := version.NewVersion(v)
	if err != nil {
		return "", err
	}
	return version.String(), nil
}

// AvailableUpdate return the version of the latest release and reports whether the update is newer.
func AvailableUpdate(gitHubOwner, githubRepo, localVersion string) (VersionInfo, error) {
	return availableUpdate(gitHubOwner, githubRepo, localVersion, fetchGitHubLatest)
}

func availableUpdate(owner, repo, localVersion string, githubLatest func(owner, repo string) (string, error)) (VersionInfo, error) {
	local, err := version.NewVersion(localVersion)
	if err != nil {
		return VersionInfo{}, err
	}
	r, err := githubLatest(owner, repo)
	if err != nil {
		return VersionInfo{}, err
	}
	remote, err := version.NewVersion(r)
	if err != nil {
		return VersionInfo{}, err
	}
	v := VersionInfo{
		Local:  local.String(),
		Remote: remote.String(),
	}
	if local.LessThan(remote) {
		v.Latest = remote.String()
		v.IsRemoteNewer = true
	} else {
		v.Latest = local.String()
		v.IsRemoteNewer = false
	}
	return v, nil
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func fetchGitHubLatest(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	r, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	if r.StatusCode >= 400 {
		return "", fmt.Errorf("%s: %w", r.Status, ErrHTTPError)
	}
	var info githubRelease
	if err := json.Unmarshal(data, &info); err != nil {
		return "", err
	}
	return info.TagName, nil
}
