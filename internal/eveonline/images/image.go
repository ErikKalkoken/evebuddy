package images

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"fyne.io/fyne/v2"
)

var (
	ErrHttpError = errors.New("http error")
	ErrNoImage   = errors.New("no image from API")
)

// Manager provides cached access to images from the Eve Online image server.
type Manager struct {
	path string
}

// New returns a new Images object. path is the location of the file cache.
func New(path string) *Manager {
	m := &Manager{path: path}
	return m
}

// AllianceLogo returns the logo for an alliance.
func (m *Manager) AllianceLogo(id int32, size int) (fyne.Resource, error) {
	url, err := AllianceLogoURL(id, size)
	if err != nil {
		return nil, err
	}
	return m.image(url)
}

// CharacterPortrait returns the portrait for a character.
func (m *Manager) CharacterPortrait(id int32, size int) (fyne.Resource, error) {
	url, err := CharacterPortraitURL(id, size)
	if err != nil {
		return nil, err
	}
	return m.image(url)
}

// CorporationLogo returns the logo for a corporation.
func (m *Manager) CorporationLogo(id int32, size int) (fyne.Resource, error) {
	url, err := CorporationLogoURL(id, size)
	if err != nil {
		return nil, err
	}
	return m.image(url)
}

// FactionLogo returns the logo for a faction.
func (m *Manager) FactionLogo(id int32, size int) (fyne.Resource, error) {
	url, err := FactionLogoURL(id, size)
	if err != nil {
		return nil, err
	}
	return m.image(url)
}

// FactionLogo returns the render for a type. Note that not ever type has a render.
func (m *Manager) InventoryTypeRender(id int32, size int) (fyne.Resource, error) {
	url, err := InventoryTypeRenderURL(id, size)
	if err != nil {
		return nil, err
	}
	return m.image(url)
}

// FactionLogo returns the logo for a type.
func (m *Manager) InventoryTypeIcon(id int32, size int) (fyne.Resource, error) {
	url, err := InventoryTypeIconURL(id, size)
	if err != nil {
		return nil, err
	}
	return m.image(url)
}

func (m *Manager) image(url string) (fyne.Resource, error) {
	h := makeMD5Hash(url)
	name := filepath.Join(m.path, h+".tmp")
	dat, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		dat, err = loadDataFromURL(url)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(name, dat, 0666); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	r := fyne.NewStaticResource(fmt.Sprintf("eve-image-%s", h), dat)
	return r, nil
}

type HTTPError struct {
	StatusCode int
	Status     string
}

func (r HTTPError) Error() string {
	return fmt.Sprintf("HTTP error: %s", r.Status)
}

func loadDataFromURL(url string) ([]byte, error) {
	r, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if r.StatusCode >= 400 {
		err := HTTPError{StatusCode: r.StatusCode, Status: r.Status}
		return nil, err
	}
	if r.Body == nil {
		return nil, fmt.Errorf("%s: %w", url, ErrNoImage)
	}
	dat, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func makeMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// ClearCache clears the images cache and returns the number of deleted entries.
func (m *Manager) ClearCache() (int, error) {
	files, err := os.ReadDir(m.path)
	if err != nil {
		return 0, err
	}
	for _, f := range files {
		os.RemoveAll(path.Join(m.path, f.Name()))
	}
	return len(files), nil
}

// Size returns the total size of all image files in by bytes.
func (m *Manager) Size() (int, error) {
	files, err := os.ReadDir(m.path)
	if err != nil {
		return 0, err
	}
	var s int64
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			return 0, err
		}
		s += info.Size()
	}
	return int(s), nil
}
