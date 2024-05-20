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
	"fyne.io/fyne/v2/canvas"
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

func (m *Manager) AllianceLogo(id int32, size int) fyne.Resource {
	uri, err := AllianceLogoURL(id, size)
	if err != nil {
		panic(err)
	}
	return m.image(uri)
}

func (m *Manager) CharacterPortrait(id int32, size int) fyne.Resource {
	uri, err := CharacterPortraitURL(id, size)
	if err != nil {
		panic(err)
	}
	return m.image(uri)
}

func (m *Manager) CorporationLogo(id int32, size int) fyne.Resource {
	uri, err := CorporationLogoURL(id, size)
	if err != nil {
		panic(err)
	}
	return m.image(uri)
}

func (m *Manager) FactionLogo(id int32, size int) fyne.Resource {
	uri, err := FactionLogoURL(id, size)
	if err != nil {
		panic(err)
	}
	return m.image(uri)
}

func (m *Manager) InventoryTypeRender(id int32, size int) fyne.Resource {
	uri, err := InventoryTypeRenderURL(id, size)
	if err != nil {
		panic(err)
	}
	return m.image(uri)
}

func (m *Manager) InventoryTypeIcon(id int32, size int) fyne.Resource {
	uri, err := InventoryTypeIconURL(id, size)
	if err != nil {
		panic(err)
	}
	return m.image(uri)
}

func (m *Manager) image(uri fyne.URI) fyne.Resource {
	h := GetMD5Hash(uri.String())
	name := filepath.Join(m.path, h+".tmp")
	dat, err := os.ReadFile(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			image := canvas.NewImageFromURI(uri)
			r := image.Resource
			dat := r.Content()
			err := os.WriteFile(name, dat, 0666)
			if err != nil {
				panic(err)
			}
			return r
		}
		panic(err)
	}
	r := fyne.NewStaticResource(fmt.Sprintf("image-%s", h), dat)
	return r
}

func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// ClearCache clears the images cache and returns the number of deleted entries.
func (m *Manager) ClearCache() int {
	files, err := os.ReadDir(m.path)
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		os.RemoveAll(path.Join(m.path, f.Name()))
	}
	return len(files)
}
