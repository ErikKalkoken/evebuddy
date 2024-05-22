package images

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"fyne.io/fyne/v2"
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
	h := GetMD5Hash(url)
	name := filepath.Join(m.path, h+".tmp")
	dat, err := os.ReadFile(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			r, err := fyne.LoadResourceFromURLString(url)
			if err != nil {
				return nil, err
			}
			dat := r.Content()
			if err := os.WriteFile(name, dat, 0666); err != nil {
				return nil, err
			}
			return r, nil
		}
		return nil, err
	}
	r := fyne.NewStaticResource(fmt.Sprintf("image-%s", h), dat)
	return r, nil
}

func GetMD5Hash(text string) string {
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
