package eveimage

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
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/eveonline/images"
)

var (
	ErrHttpError   = errors.New("http error")
	ErrNoImage     = errors.New("no image from API")
	ErrInvalidSize = errors.New("invalid size")
)

// EveImage provides cached access to images from the Eve Online image server.
type EveImage struct {
	httpClient *http.Client
	// path is where the image files are stored for caching
	path string
	sfg  *singleflight.Group
}

// New returns a new Images object. path is the location of the file cache.
func New(path string, httpClient *http.Client) *EveImage {
	m := &EveImage{
		httpClient: httpClient,
		path:       path,
		sfg:        new(singleflight.Group),
	}
	return m
}

// AllianceLogo returns the logo for an alliance.
func (m *EveImage) AllianceLogo(id int32, size int) (fyne.Resource, error) {
	url, err := images.AllianceLogoURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// CharacterPortrait returns the portrait for a character.
func (m *EveImage) CharacterPortrait(id int32, size int) (fyne.Resource, error) {
	url, err := images.CharacterPortraitURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// CorporationLogo returns the logo for a corporation.
func (m *EveImage) CorporationLogo(id int32, size int) (fyne.Resource, error) {
	url, err := images.CorporationLogoURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// FactionLogo returns the logo for a faction.
func (m *EveImage) FactionLogo(id int32, size int) (fyne.Resource, error) {
	url, err := images.FactionLogoURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// InventoryTypeRender returns the render for a type. Note that not ever type has a render.
func (m *EveImage) InventoryTypeRender(id int32, size int) (fyne.Resource, error) {
	url, err := images.InventoryTypeRenderURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// InventoryTypeIcon returns the icon for a type.
func (m *EveImage) InventoryTypeIcon(id int32, size int) (fyne.Resource, error) {
	url, err := images.InventoryTypeIconURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// InventoryTypeBPO returns the icon for a BPO type.
func (m *EveImage) InventoryTypeBPO(id int32, size int) (fyne.Resource, error) {
	url, err := images.InventoryTypeBPOURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// InventoryTypeBPC returns the icon for a BPC type.
func (m *EveImage) InventoryTypeBPC(id int32, size int) (fyne.Resource, error) {
	url, err := images.InventoryTypeBPCURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

func (m *EveImage) image(url string) (fyne.Resource, error) {
	hash := makeMD5Hash(url)
	name := filepath.Join(m.path, hash+".tmp")
	dat, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		x, err, _ := m.sfg.Do(hash, func() (any, error) {
			dat, err = loadDataFromURL(url, m.httpClient)
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(name, dat, 0666); err != nil {
				return nil, err
			}
			return dat, nil
		})
		if err != nil {
			return nil, err
		}
		dat = x.([]byte)
	} else if err != nil {
		return nil, err
	}
	r := fyne.NewStaticResource(fmt.Sprintf("eve-image-%s", hash), dat)
	return r, nil
}

type HTTPError struct {
	StatusCode int
	Status     string
}

func (r HTTPError) Error() string {
	return fmt.Sprintf("HTTP error: %s", r.Status)
}

func loadDataFromURL(url string, client *http.Client) ([]byte, error) {
	r, err := client.Get(url)
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

// Clear clears the images cache and returns the number of deleted entries.
func (m *EveImage) Clear() (int, error) {
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
func (m *EveImage) Size() (int, error) {
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

// Count returns the number of all image files.
func (m *EveImage) Count() (int, error) {
	files, err := os.ReadDir(m.path)
	if err != nil {
		return 0, err
	}
	return len(files), nil
}
